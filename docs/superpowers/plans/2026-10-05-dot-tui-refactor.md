# `dot` TUI Refactor Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the bash installer with a Go binary, `dot`, that symlinks configs from `home/` into `$HOME`, wraps `brew bundle` with Homebrew 7 tap trust, runs idempotent install steps, reports status, and offers a Bubble Tea dashboard, then migrate this Mac onto it.

**Architecture:** Small focused packages under `internal/` (exec runner, manifest, link planner, brew, steps, services, manual, repo/git, status) are wired by `internal/app`, which exposes one function per action. `cmd/dot` is a thin flag parser over `app`; `internal/tui` is a thin Bubble Tea view over the same `app` functions. Every external command goes through `exec.Runner` so tests use a scripted fake and never touch brew, launchctl, git or the network.

**Tech Stack:** Go 1.25, `github.com/pelletier/go-toml/v2`, `github.com/charmbracelet/bubbletea`, `github.com/charmbracelet/bubbles` (viewport), `github.com/charmbracelet/lipgloss`, standard `flag` and `testing` packages only otherwise.

**Spec:** `docs/superpowers/specs/2026-10-05-dot-tui-refactor-design.md`

## Global Constraints

- Module path `github.com/ireydiak/dotfiles`; Go 1.25; binary built from `./cmd/dot`.
- macOS only. Homebrew prefix is `/opt/homebrew`. Shell commands run via `/bin/zsh -c` with `PATH` prefixed by `/opt/homebrew/bin` and `HOME` set.
- No cobra. Subcommand dispatch uses the standard `flag` package. Flags follow the command: `dot <command> [flags]`.
- Exit codes: 0 all ok or no-op, 1 any action failed, 2 usage or manifest error.
- Nothing is ever deleted. Conflicting files move to `~/.local/state/dot/backup/<YYYYMMDD-HHMMSS>/<path relative to $HOME>`.
- `--dry-run` applies to install, link, export, update and commit: full plan, no side effects, no tap trust, no brew upgrade, no git commands.
- Symlinks are created with absolute targets. "Resolving to src" means `os.Readlink(dst)`, made absolute and cleaned, equals the cleaned `src`.
- `brew bundle install` is always invoked with `--no-upgrade`. `brew bundle cleanup` is never run.
- Secrets never enter the repo. The tracked `home/.zshrc` sources `~/.zshrc.local`, which `dot` never reads, writes or lists.
- No test shells out to real `brew`, `launchctl`, `git` or the network. Tests use `exec.Fake` and `t.TempDir()`.
- The TUI contains no logic of its own; it calls `internal/app` functions and renders results.

## Review Focus

Each line names an input the spec implies but does not spell out, and the behavior a person would expect. Each has a test in the task that owns the code.

1. A Brewfile with comments, blank lines, and option suffixes such as `brew "openssl@3", link: true` or `cask "x", args: { no_quarantine: true }` must parse, keep the suffix verbatim and round-trip unchanged. (Task 6)
2. A dangling symlink or a leftover `<dst>.dot-tmp` symlink at a link destination must be replaced cleanly, not reported as a conflict and not crash. (Task 5)
3. The first `dot export` on a repo with no Brewfile must treat every dumped entry as added, with no file-not-found error. (Task 7 for `Committed`, Task 12 for `ExportDiff`)
4. `dot commit` on a clean tree must report "nothing to commit", exit 0 and never create an empty commit. (Task 12)
5. `dot status` when brew is missing or fails must still show links, services and manual items, with the package section carrying the error text. (Task 11)

## File Structure

| Path | Responsibility |
|---|---|
| `go.mod`, `.gitignore` | Module and ignore rules |
| `cmd/dot/main.go` | Parse command and flags, build `app.App`, dispatch, map results to exit codes |
| `internal/exec/exec.go`, `fake.go` | `Runner` interface, real `Shell`, scripted `Fake` |
| `internal/result/result.go` | `Status`, `Entry`, `Summary`, printing, `LastLine` |
| `internal/manifest/manifest.go` | TOML schema, strict parse, validation, `Expand`, `LinkKeys` |
| `internal/link/link.go` | `Inspect` states, `Plan` actions with backups, `Apply` |
| `internal/brew/brewfile.go` | `Entry`, `Parse`, `Write`, `Compare`, `Filter`, `Taps` |
| `internal/brew/client.go` | `Client`: committed, dump, tapped, trusted, trust, install, update, upgrade |
| `internal/steps/steps.go` | `Install` (check, run, recheck) and `Update` |
| `internal/services/services.go` | launchd `Status` and `Start` |
| `internal/manual/manual.go` | Manual checklist `Status` |
| `internal/repo/repo.go` | `Resolve` repo root, `Git` operations, commit `Message` |
| `internal/status/status.go` | `Report`, `Collect`, `Print`, `JSON` |
| `internal/app/app.go` | `App` wiring and every action: Status, Link, Install, Update, UntrustedTaps, TrustTaps, ExportDiff, ExportWrite, Commit, ManualStatus |
| `internal/tui/picker.go` | Export checklist model |
| `internal/tui/tui.go` | Dashboard model, screens, streaming output pane |
| `manifest.toml`, `Brewfile`, `home/**` | Managed content (Tasks 15 and 16) |
| `bootstrap.sh`, `README.md` | Fresh-machine entry and docs (Task 17) |

Every task runs from the repo root `~/Dev/git/dotfiles` on branch `dot-tui-refactor`. Commit messages end with `Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>`.

---

### Task 1: Go module scaffold

**Files:**
- Create: `go.mod`, `.gitignore`, `cmd/dot/main.go`

**Interfaces:**
- Produces: a buildable `./cmd/dot` that prints usage and exits 2. Later tasks replace `main.go` wholesale in Task 13.

- [ ] **Step 1: Initialize the module and ignore rules**

```bash
go mod init github.com/ireydiak/dotfiles
cat > .gitignore <<'EOF'
.claude/
.DS_Store
*.dot-tmp
/dot
home/.config/sketchybar/helper/helper
EOF
```

- [ ] **Step 2: Write the stub entry point**

```go
// cmd/dot/main.go
package main

import (
	"fmt"
	"os"
)

const usage = `usage: dot <command> [flags]

commands:
  status   report packages, links, services, manual steps and repo state
  install  trust taps, brew bundle install, link, run steps, start services
  link     create or repair symlinks from home/ into $HOME
  export   dump installed packages and update the Brewfile
  update   brew update, brew upgrade, then step updates
  commit   git add -A, commit and push
  (none)   open the dashboard
`

func main() {
	fmt.Fprint(os.Stderr, usage)
	os.Exit(2)
}
```

- [ ] **Step 3: Verify it builds and exits 2**

Run: `go build ./... && go run ./cmd/dot; echo "exit=$?"`
Expected: usage text on stderr, then `exit=2`

- [ ] **Step 4: Commit**

```bash
git add go.mod .gitignore cmd/dot/main.go
git commit -m "chore: scaffold dot Go module

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

---

### Task 2: Command runner (`internal/exec`)

**Files:**
- Create: `internal/exec/exec.go`, `internal/exec/fake.go`
- Test: `internal/exec/exec_test.go`

**Interfaces:**
- Produces:
  - `type Result struct { Stdout, Stderr string; ExitCode int }` with `func (r Result) Output() string` (stdout then stderr)
  - `type Runner interface { Run(ctx context.Context, command string, out io.Writer) (Result, error) }`. A non-zero exit is reported in `Result.ExitCode` with a nil error; `error` is non-nil only when the command could not be started.
  - `func NewShell(home string) *Shell` and `func (s *Shell) Run(...)`
  - `type Fake struct { Scripts map[string]Result; Queue map[string][]Result; Default Result; Calls []string }`, `func NewFake() *Fake`. `Queue` entries are consumed first in order, then `Scripts`, then `Default`. Every command string is appended to `Calls`.

- [ ] **Step 1: Write the failing tests**

```go
// internal/exec/exec_test.go
package exec

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestShellCapturesOutputAndExitCode(t *testing.T) {
	res, err := NewShell(t.TempDir()).Run(context.Background(), "echo hi; echo err >&2; exit 3", nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Stdout != "hi\n" || res.Stderr != "err\n" || res.ExitCode != 3 {
		t.Fatalf("unexpected result: %+v", res)
	}
	if res.Output() != "hi\nerr\n" {
		t.Fatalf("Output() = %q", res.Output())
	}
}

func TestShellEnvironment(t *testing.T) {
	home := t.TempDir()
	res, err := NewShell(home).Run(context.Background(), `printf '%s\n%s' "$HOME" "$PATH"`, nil)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(res.Stdout, "\n")
	if lines[0] != home {
		t.Fatalf("HOME = %q, want %q", lines[0], home)
	}
	if !strings.HasPrefix(lines[1], "/opt/homebrew/bin:") {
		t.Fatalf("PATH = %q, want /opt/homebrew/bin prefix", lines[1])
	}
}

func TestShellStreamsToWriter(t *testing.T) {
	var buf bytes.Buffer
	if _, err := NewShell(t.TempDir()).Run(context.Background(), "echo streamed", &buf); err != nil {
		t.Fatal(err)
	}
	if buf.String() != "streamed\n" {
		t.Fatalf("streamed output = %q", buf.String())
	}
}

func TestFakeScriptsQueueDefaultAndCalls(t *testing.T) {
	f := NewFake()
	f.Scripts["a"] = Result{Stdout: "A"}
	f.Queue["b"] = []Result{{ExitCode: 1}, {ExitCode: 0}}
	f.Default = Result{ExitCode: 7}
	ctx := context.Background()
	r1, _ := f.Run(ctx, "a", nil)
	r2, _ := f.Run(ctx, "b", nil)
	r3, _ := f.Run(ctx, "b", nil)
	r4, _ := f.Run(ctx, "zzz", nil)
	if r1.Stdout != "A" || r2.ExitCode != 1 || r3.ExitCode != 0 || r4.ExitCode != 7 {
		t.Fatalf("got %+v %+v %+v %+v", r1, r2, r3, r4)
	}
	if got := strings.Join(f.Calls, ","); got != "a,b,b,zzz" {
		t.Fatalf("Calls = %q", got)
	}
	var buf bytes.Buffer
	f.Run(ctx, "a", &buf)
	if buf.String() != "A" {
		t.Fatalf("fake did not stream: %q", buf.String())
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/exec/`
Expected: FAIL, `undefined: NewShell`, `undefined: NewFake`

- [ ] **Step 3: Implement the runner and fake**

```go
// internal/exec/exec.go
package exec

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	osexec "os/exec"
	"strings"
)

// Result is the outcome of one shell command. A non-zero ExitCode is not an error.
type Result struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// Output returns stdout followed by stderr.
func (r Result) Output() string { return r.Stdout + r.Stderr }

// Runner executes a shell command string. When out is non-nil, combined
// output is streamed to it as well as captured in the Result.
type Runner interface {
	Run(ctx context.Context, command string, out io.Writer) (Result, error)
}

// Shell runs commands through /bin/zsh -c with a fixed environment.
type Shell struct {
	Env []string
}

// NewShell returns a Shell whose HOME is home and whose PATH starts with
// /opt/homebrew/bin. All other variables are inherited from the process.
func NewShell(home string) *Shell {
	env := []string{"HOME=" + home, "PATH=/opt/homebrew/bin:" + os.Getenv("PATH")}
	for _, kv := range os.Environ() {
		if strings.HasPrefix(kv, "HOME=") || strings.HasPrefix(kv, "PATH=") {
			continue
		}
		env = append(env, kv)
	}
	return &Shell{Env: env}
}

func (s *Shell) Run(ctx context.Context, command string, out io.Writer) (Result, error) {
	cmd := osexec.CommandContext(ctx, "/bin/zsh", "-c", command)
	cmd.Env = s.Env
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if out != nil {
		cmd.Stdout = io.MultiWriter(&stdout, out)
		cmd.Stderr = io.MultiWriter(&stderr, out)
	}
	err := cmd.Run()
	res := Result{Stdout: stdout.String(), Stderr: stderr.String()}
	var exitErr *osexec.ExitError
	if errors.As(err, &exitErr) {
		res.ExitCode = exitErr.ExitCode()
		return res, nil
	}
	return res, err
}
```

```go
// internal/exec/fake.go
package exec

import (
	"context"
	"io"
)

// Fake is a Runner for tests. Queue entries for a command are consumed first,
// in order; then Scripts; then Default. Every command is recorded in Calls.
type Fake struct {
	Scripts map[string]Result
	Queue   map[string][]Result
	Default Result
	Calls   []string
}

func NewFake() *Fake {
	return &Fake{Scripts: map[string]Result{}, Queue: map[string][]Result{}}
}

func (f *Fake) Run(_ context.Context, command string, out io.Writer) (Result, error) {
	f.Calls = append(f.Calls, command)
	var res Result
	switch {
	case len(f.Queue[command]) > 0:
		res = f.Queue[command][0]
		f.Queue[command] = f.Queue[command][1:]
	default:
		var ok bool
		if res, ok = f.Scripts[command]; !ok {
			res = f.Default
		}
	}
	if out != nil {
		io.WriteString(out, res.Output())
	}
	return res, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/exec/`
Expected: `ok`

- [ ] **Step 5: Commit**

```bash
git add internal/exec
git commit -m "feat(exec): shell runner and scripted fake

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

---

### Task 3: Shared result type (`internal/result`)

**Files:**
- Create: `internal/result/result.go`
- Test: `internal/result/result_test.go`

**Interfaces:**
- Produces:
  - `type Status int` with constants `Ok`, `Skipped`, `Failed` and `String()` returning `ok`, `skipped`, `failed`
  - `type Entry struct { Section, ID string; Status Status; Detail string }`
  - `type Summary []Entry` with `Failed() bool`, `Counts() (ok, skipped, failed int)`, `Print(w io.Writer)`
  - `func LastLine(s string) string`: last non-empty line, trimmed, at most 120 runes

- [ ] **Step 1: Write the failing tests**

```go
// internal/result/result_test.go
package result

import (
	"bytes"
	"strings"
	"testing"
)

func TestSummaryFailedAndCounts(t *testing.T) {
	s := Summary{
		{Section: "link", ID: "a", Status: Ok},
		{Section: "step", ID: "b", Status: Skipped, Detail: "already satisfied"},
	}
	if s.Failed() {
		t.Fatal("Failed() should be false without failures")
	}
	s = append(s, Entry{Section: "brew", ID: "c", Status: Failed, Detail: "boom"})
	if !s.Failed() {
		t.Fatal("Failed() should be true")
	}
	ok, sk, f := s.Counts()
	if ok != 1 || sk != 1 || f != 1 {
		t.Fatalf("Counts = %d %d %d", ok, sk, f)
	}
}

func TestSummaryPrint(t *testing.T) {
	var buf bytes.Buffer
	Summary{{Section: "link", ID: ".zshrc", Status: Ok, Detail: "linked"}}.Print(&buf)
	out := buf.String()
	for _, want := range []string{"ok", "link", ".zshrc", "linked", "1 ok, 0 skipped, 0 failed"} {
		if !strings.Contains(out, want) {
			t.Fatalf("Print output missing %q:\n%s", want, out)
		}
	}
}

func TestLastLine(t *testing.T) {
	if got := LastLine("a\nb\n\n"); got != "b" {
		t.Fatalf("LastLine = %q", got)
	}
	if got := LastLine(""); got != "" {
		t.Fatalf("LastLine(empty) = %q", got)
	}
	long := strings.Repeat("x", 200)
	if got := LastLine(long); len([]rune(got)) != 120 {
		t.Fatalf("LastLine should truncate to 120 runes, got %d", len([]rune(got)))
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/result/`
Expected: FAIL, `undefined: Summary`

- [ ] **Step 3: Implement**

```go
// internal/result/result.go
package result

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
)

type Status int

const (
	Ok Status = iota
	Skipped
	Failed
)

func (s Status) String() string {
	switch s {
	case Ok:
		return "ok"
	case Skipped:
		return "skipped"
	default:
		return "failed"
	}
}

// Entry is the outcome of one unit of work.
type Entry struct {
	Section string // brew, link, step, service, git
	ID      string
	Status  Status
	Detail  string // one line: reason or last line of output
}

type Summary []Entry

func (s Summary) Failed() bool {
	for _, e := range s {
		if e.Status == Failed {
			return true
		}
	}
	return false
}

func (s Summary) Counts() (ok, skipped, failed int) {
	for _, e := range s {
		switch e.Status {
		case Ok:
			ok++
		case Skipped:
			skipped++
		default:
			failed++
		}
	}
	return
}

// Print writes one row per entry and a totals line.
func (s Summary) Print(w io.Writer) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	for _, e := range s {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", e.Status, e.Section, e.ID, e.Detail)
	}
	tw.Flush()
	ok, sk, f := s.Counts()
	fmt.Fprintf(w, "%d ok, %d skipped, %d failed\n", ok, sk, f)
}

// LastLine returns the last non-empty line of s, trimmed and capped at 120 runes.
func LastLine(s string) string {
	lines := strings.Split(s, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		l := strings.TrimSpace(lines[i])
		if l == "" {
			continue
		}
		if r := []rune(l); len(r) > 120 {
			return string(r[:120])
		}
		return l
	}
	return ""
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/result/`
Expected: `ok`

- [ ] **Step 5: Commit**

```bash
git add internal/result
git commit -m "feat(result): shared ok/skipped/failed summary

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

---

### Task 4: Manifest schema and validation (`internal/manifest`)

**Files:**
- Create: `internal/manifest/manifest.go`
- Test: `internal/manifest/manifest_test.go`
- Modify: `go.mod` (adds `github.com/pelletier/go-toml/v2`)

**Interfaces:**
- Produces:
  - `type Manifest struct { Links map[string]string; Steps []Step; Services []Service; Manual []Manual }`
  - `type Step struct { ID, Check, Run, Update string }`, `type Service struct { ID, Label, Start string }`, `type Manual struct { ID, Check, How string }`
  - `func Load(repoRoot string) (*Manifest, error)`: reads `<repoRoot>/manifest.toml`, parses strictly, validates
  - `func Parse(r io.Reader) (*Manifest, error)`: strict decode, unknown keys are errors
  - `func (m *Manifest) Validate(repoRoot string) error`: all rules below, errors joined
  - `func (m *Manifest) LinkKeys() []string`: sorted keys
  - `func Expand(p, home string) string`: replaces a leading `~/` with home; nothing else

Validation rules: link key is a clean relative path (no `..`, no leading `/`, `filepath.Clean(key) == key`) and exists under `<repoRoot>/home/`; link value starts with `~/` and has something after it; ids unique per section and non-empty; step `check` and `run` non-empty (`update` optional); service `label` and `start` non-empty; manual `how` non-empty (`check` optional).

- [ ] **Step 1: Add the dependency**

Run: `go get github.com/pelletier/go-toml/v2@latest`

- [ ] **Step 2: Write the failing tests**

```go
// internal/manifest/manifest_test.go
package manifest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeRepo creates <tmp>/manifest.toml plus the given files under <tmp>/home.
func writeRepo(t *testing.T, manifest string, homeFiles ...string) string {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "manifest.toml"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, f := range homeFiles {
		p := filepath.Join(root, "home", f)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

const valid = `
[links]
".zshrc" = "~/.zshrc"
"Library/Application Support/lazygit/config.yml" = "~/Library/Application Support/lazygit/config.yml"

[[steps]]
id = "omz"
check = "test -d ~/.oh-my-zsh"
run = "echo install"
update = "echo update"

[[steps]]
id = "nvm"
check = "test -s ~/.nvm/nvm.sh"
run = "echo install"

[[services]]
id = "yabai"
label = "com.koekeishiya.yabai"
start = "yabai --start-service"

[[manual]]
id = "xcode"
check = "xcode-select -p"
how = "xcode-select --install"

[[manual]]
id = "accessibility"
how = "System Settings"
`

func TestLoadValid(t *testing.T) {
	root := writeRepo(t, valid, ".zshrc", "Library/Application Support/lazygit/config.yml")
	m, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if got := m.LinkKeys(); len(got) != 2 || got[0] != ".zshrc" {
		t.Fatalf("LinkKeys = %v", got)
	}
	if len(m.Steps) != 2 || m.Steps[0].Update != "echo update" || m.Steps[1].Update != "" {
		t.Fatalf("steps = %+v", m.Steps)
	}
	if len(m.Services) != 1 || m.Services[0].Label != "com.koekeishiya.yabai" {
		t.Fatalf("services = %+v", m.Services)
	}
	if len(m.Manual) != 2 || m.Manual[1].Check != "" {
		t.Fatalf("manual = %+v", m.Manual)
	}
}

func TestParseRejectsUnknownKey(t *testing.T) {
	_, err := Parse(strings.NewReader("[links]\n\".zshrc\" = \"~/.zshrc\"\n[[steps]]\nid = \"a\"\ncheck = \"true\"\nrun = \"true\"\nrunn = \"typo\"\n"))
	if err == nil || !strings.Contains(err.Error(), "unknown key") {
		t.Fatalf("expected unknown key error, got %v", err)
	}
}

func TestValidateErrors(t *testing.T) {
	cases := map[string]struct {
		manifest string
		files    []string
		want     string
	}{
		"missing source": {"[links]\n\".zshrc\" = \"~/.zshrc\"\n", nil, "not found under home/"},
		"bad target":     {"[links]\n\".zshrc\" = \"/etc/zshrc\"\n", []string{".zshrc"}, "must start with ~/"},
		"dotdot key":     {"[links]\n\"../x\" = \"~/x\"\n", nil, "clean relative path"},
		"absolute key":   {"[links]\n\"/x\" = \"~/x\"\n", nil, "clean relative path"},
		"dup step id":    {"[[steps]]\nid=\"a\"\ncheck=\"t\"\nrun=\"t\"\n[[steps]]\nid=\"a\"\ncheck=\"t\"\nrun=\"t\"\n", nil, "duplicate"},
		"step no run":    {"[[steps]]\nid=\"a\"\ncheck=\"t\"\n", nil, "needs id, check and run"},
		"service no label": {"[[services]]\nid=\"a\"\nstart=\"t\"\n", nil, "needs id, label and start"},
		"manual no how":  {"[[manual]]\nid=\"a\"\n", nil, "needs id and how"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			root := writeRepo(t, c.manifest, c.files...)
			_, err := Load(root)
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("want error containing %q, got %v", c.want, err)
			}
		})
	}
}

func TestExpand(t *testing.T) {
	if got := Expand("~/.config/nvim", "/Users/x"); got != "/Users/x/.config/nvim" {
		t.Fatalf("Expand = %q", got)
	}
	if got := Expand("/abs", "/Users/x"); got != "/abs" {
		t.Fatalf("Expand should leave non-tilde paths alone, got %q", got)
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `go test ./internal/manifest/`
Expected: FAIL, `undefined: Load`

- [ ] **Step 4: Implement**

```go
// internal/manifest/manifest.go
package manifest

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

type Manifest struct {
	Links    map[string]string `toml:"links"`
	Steps    []Step            `toml:"steps"`
	Services []Service         `toml:"services"`
	Manual   []Manual          `toml:"manual"`
}

type Step struct {
	ID     string `toml:"id"`
	Check  string `toml:"check"`
	Run    string `toml:"run"`
	Update string `toml:"update"`
}

type Service struct {
	ID    string `toml:"id"`
	Label string `toml:"label"`
	Start string `toml:"start"`
}

type Manual struct {
	ID    string `toml:"id"`
	Check string `toml:"check"`
	How   string `toml:"how"`
}

// Load reads and validates <repoRoot>/manifest.toml.
func Load(repoRoot string) (*Manifest, error) {
	f, err := os.Open(filepath.Join(repoRoot, "manifest.toml"))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	m, err := Parse(f)
	if err != nil {
		return nil, fmt.Errorf("manifest.toml: %w", err)
	}
	if err := m.Validate(repoRoot); err != nil {
		return nil, fmt.Errorf("manifest.toml: %w", err)
	}
	return m, nil
}

// Parse decodes TOML strictly: unknown keys are errors.
func Parse(r io.Reader) (*Manifest, error) {
	var m Manifest
	dec := toml.NewDecoder(r)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&m); err != nil {
		var strict *toml.StrictMissingError
		if errors.As(err, &strict) {
			return nil, fmt.Errorf("unknown key: %s", strict.String())
		}
		return nil, err
	}
	if m.Links == nil {
		m.Links = map[string]string{}
	}
	return &m, nil
}

func (m *Manifest) LinkKeys() []string {
	keys := make([]string, 0, len(m.Links))
	for k := range m.Links {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// Expand replaces a leading "~/" with home. It is the only expansion performed.
func Expand(p, home string) string {
	if strings.HasPrefix(p, "~/") {
		return filepath.Join(home, p[2:])
	}
	return p
}

func (m *Manifest) Validate(repoRoot string) error {
	var errs []error
	for _, key := range m.LinkKeys() {
		if key == "" || filepath.IsAbs(key) || filepath.Clean(key) != key || strings.HasPrefix(key, "..") {
			errs = append(errs, fmt.Errorf("links: key %q must be a clean relative path", key))
		} else if _, err := os.Lstat(filepath.Join(repoRoot, "home", key)); err != nil {
			errs = append(errs, fmt.Errorf("links: %q not found under home/", key))
		}
		if val := m.Links[key]; !strings.HasPrefix(val, "~/") || len(val) < 3 {
			errs = append(errs, fmt.Errorf("links: target for %q must start with ~/", key))
		}
	}

	seen := map[string]bool{}
	for _, s := range m.Steps {
		if s.ID == "" || s.Check == "" || s.Run == "" {
			errs = append(errs, fmt.Errorf("steps: %q needs id, check and run", s.ID))
		}
		errs = appendDup(errs, seen, "steps", s.ID)
	}
	seen = map[string]bool{}
	for _, s := range m.Services {
		if s.ID == "" || s.Label == "" || s.Start == "" {
			errs = append(errs, fmt.Errorf("services: %q needs id, label and start", s.ID))
		}
		errs = appendDup(errs, seen, "services", s.ID)
	}
	seen = map[string]bool{}
	for _, s := range m.Manual {
		if s.ID == "" || s.How == "" {
			errs = append(errs, fmt.Errorf("manual: %q needs id and how", s.ID))
		}
		errs = appendDup(errs, seen, "manual", s.ID)
	}
	return errors.Join(errs...)
}

func appendDup(errs []error, seen map[string]bool, section, id string) []error {
	if seen[id] {
		return append(errs, fmt.Errorf("%s: duplicate id %q", section, id))
	}
	seen[id] = true
	return errs
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/manifest/`
Expected: `ok`

- [ ] **Step 6: Commit**

```bash
git add go.mod go.sum internal/manifest
git commit -m "feat(manifest): strict TOML schema with validation

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

---

### Task 5: Link engine (`internal/link`)

**Files:**
- Create: `internal/link/link.go`
- Test: `internal/link/link_test.go`

**Interfaces:**
- Consumes: `manifest.Expand`, `result.Entry`, `result.Summary`
- Produces:
  - `type State int` with `Ok`, `Missing`, `WrongTarget`, `Conflict`, `BrokenManifest`; `String()` gives `ok`, `missing`, `wrong-target`, `conflict`, `broken-manifest`
  - `type Item struct { Key, Src, Dst string; State State; Detail string }`
  - `func Inspect(links map[string]string, repoRoot, home string) []Item` sorted by Key; `Src = <repoRoot>/home/<key>`, `Dst = Expand(value, home)`
  - `type Action struct { Item Item; Backup string }` (Backup non-empty only for Conflict)
  - `func Plan(items []Item, backupDir, home string) []Action` for Missing, WrongTarget and Conflict items
  - `func Apply(actions []Action, dryRun bool, out io.Writer) result.Summary` with Section `link`, ID = Key

Apply per action: move conflict to `<backupDir>/<dst relative to home>`; `MkdirAll` the destination's parent; remove a stale `<dst>.dot-tmp`; `os.Symlink(src, tmp)`; `os.Rename(tmp, dst)`.

- [ ] **Step 1: Write the failing tests**

```go
// internal/link/link_test.go
package link

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/ireydiak/dotfiles/internal/result"
)

type fixture struct {
	repo, home, backup string
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	f := fixture{repo: t.TempDir(), home: t.TempDir(), backup: filepath.Join(t.TempDir(), "20260101-120000")}
	for _, rel := range []string{".zshrc", ".config/nvim/init.lua", "Library/Application Support/lazygit/config.yml"} {
		p := filepath.Join(f.repo, "home", rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("repo:"+rel), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return f
}

var links = map[string]string{
	".zshrc":       "~/.zshrc",
	".config/nvim": "~/.config/nvim",
	"Library/Application Support/lazygit/config.yml": "~/Library/Application Support/lazygit/config.yml",
}

func (f fixture) inspect() map[string]Item {
	out := map[string]Item{}
	for _, it := range Inspect(links, f.repo, f.home) {
		out[it.Key] = it
	}
	return out
}

func (f fixture) apply(t *testing.T, dryRun bool) result.Summary {
	t.Helper()
	var buf bytes.Buffer
	return Apply(Plan(Inspect(links, f.repo, f.home), f.backup, f.home), dryRun, &buf)
}

func readlink(t *testing.T, p string) string {
	t.Helper()
	target, err := os.Readlink(p)
	if err != nil {
		t.Fatalf("readlink %s: %v", p, err)
	}
	return target
}

func TestInspectSortedAndMissing(t *testing.T) {
	f := newFixture(t)
	items := Inspect(links, f.repo, f.home)
	if len(items) != 3 || items[0].Key != ".config/nvim" || items[1].Key != ".zshrc" {
		t.Fatalf("items not sorted: %+v", items)
	}
	for _, it := range items {
		if it.State != Missing {
			t.Fatalf("%s state = %s, want missing", it.Key, it.State)
		}
	}
}

func TestApplyCreatesLinksForMissing(t *testing.T) {
	f := newFixture(t)
	s := f.apply(t, false)
	if s.Failed() || len(s) != 3 {
		t.Fatalf("summary = %+v", s)
	}
	want := filepath.Join(f.repo, "home", ".config/nvim")
	if got := readlink(t, filepath.Join(f.home, ".config/nvim")); got != want {
		t.Fatalf("link target = %q, want %q", got, want)
	}
	if got := readlink(t, filepath.Join(f.home, "Library/Application Support/lazygit/config.yml")); got != filepath.Join(f.repo, "home", "Library/Application Support/lazygit/config.yml") {
		t.Fatalf("path with spaces linked to %q", got)
	}
	for _, it := range Inspect(links, f.repo, f.home) {
		if it.State != Ok {
			t.Fatalf("after apply %s = %s", it.Key, it.State)
		}
	}
}

func TestApplyIsIdempotent(t *testing.T) {
	f := newFixture(t)
	f.apply(t, false)
	s := f.apply(t, false)
	if len(s) != 0 {
		t.Fatalf("second apply should plan nothing, got %+v", s)
	}
}

func TestConflictFileIsBackedUp(t *testing.T) {
	f := newFixture(t)
	live := filepath.Join(f.home, ".zshrc")
	os.WriteFile(live, []byte("live"), 0o644)
	if f.inspect()[".zshrc"].State != Conflict {
		t.Fatal("regular file should be a conflict")
	}
	f.apply(t, false)
	backed, err := os.ReadFile(filepath.Join(f.backup, ".zshrc"))
	if err != nil || string(backed) != "live" {
		t.Fatalf("backup missing or wrong: %v %q", err, backed)
	}
	if readlink(t, live) != filepath.Join(f.repo, "home", ".zshrc") {
		t.Fatal("link not created after backup")
	}
}

func TestConflictDirectoryIsBackedUp(t *testing.T) {
	f := newFixture(t)
	live := filepath.Join(f.home, ".config/nvim")
	os.MkdirAll(live, 0o755)
	os.WriteFile(filepath.Join(live, "old.lua"), []byte("old"), 0o644)
	f.apply(t, false)
	if _, err := os.Stat(filepath.Join(f.backup, ".config/nvim/old.lua")); err != nil {
		t.Fatalf("directory not moved to backup: %v", err)
	}
	if readlink(t, live) != filepath.Join(f.repo, "home", ".config/nvim") {
		t.Fatal("directory conflict not replaced by link")
	}
}

func TestWrongTargetAndDanglingAreReplaced(t *testing.T) {
	f := newFixture(t)
	os.Symlink("/etc/zshrc", filepath.Join(f.home, ".zshrc"))
	os.MkdirAll(filepath.Join(f.home, ".config"), 0o755)
	os.Symlink(filepath.Join(f.home, "gone"), filepath.Join(f.home, ".config/nvim")) // dangling
	items := f.inspect()
	if items[".zshrc"].State != WrongTarget || items[".config/nvim"].State != WrongTarget {
		t.Fatalf("states = %s %s", items[".zshrc"].State, items[".config/nvim"].State)
	}
	s := f.apply(t, false)
	if s.Failed() {
		t.Fatalf("apply failed: %+v", s)
	}
	if readlink(t, filepath.Join(f.home, ".config/nvim")) != filepath.Join(f.repo, "home", ".config/nvim") {
		t.Fatal("dangling link not replaced")
	}
	if _, err := os.Stat(filepath.Join(f.backup, ".zshrc")); err == nil {
		t.Fatal("replacing a symlink must not create a backup")
	}
}

func TestStaleTmpLinkDoesNotBlock(t *testing.T) {
	f := newFixture(t)
	os.Symlink("/nowhere", filepath.Join(f.home, ".zshrc.dot-tmp"))
	s := f.apply(t, false)
	if s.Failed() {
		t.Fatalf("apply failed with stale tmp: %+v", s)
	}
	if _, err := os.Lstat(filepath.Join(f.home, ".zshrc.dot-tmp")); err == nil {
		t.Fatal("stale tmp link should be gone")
	}
}

func TestBrokenManifestSourceIsReported(t *testing.T) {
	f := newFixture(t)
	os.RemoveAll(filepath.Join(f.repo, "home", ".zshrc"))
	items := f.inspect()
	if items[".zshrc"].State != BrokenManifest {
		t.Fatalf("state = %s", items[".zshrc"].State)
	}
	if n := len(Plan(Inspect(links, f.repo, f.home), f.backup, f.home)); n != 2 {
		t.Fatalf("broken item must not be planned, got %d actions", n)
	}
}

func TestDryRunChangesNothing(t *testing.T) {
	f := newFixture(t)
	os.WriteFile(filepath.Join(f.home, ".zshrc"), []byte("live"), 0o644)
	s := f.apply(t, true)
	if len(s) != 3 {
		t.Fatalf("dry run should report 3 planned actions, got %+v", s)
	}
	for _, e := range s {
		if e.Status != result.Skipped {
			t.Fatalf("dry run entries must be skipped: %+v", e)
		}
	}
	if fi, err := os.Lstat(filepath.Join(f.home, ".zshrc")); err != nil || fi.Mode()&os.ModeSymlink != 0 {
		t.Fatal("dry run must not touch the live file")
	}
	if _, err := os.Stat(f.backup); err == nil {
		t.Fatal("dry run must not create the backup directory")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/link/`
Expected: FAIL, `undefined: Inspect`

- [ ] **Step 3: Implement**

```go
// internal/link/link.go
package link

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ireydiak/dotfiles/internal/manifest"
	"github.com/ireydiak/dotfiles/internal/result"
)

type State int

const (
	Ok State = iota
	Missing
	WrongTarget
	Conflict
	BrokenManifest
)

func (s State) String() string {
	return [...]string{"ok", "missing", "wrong-target", "conflict", "broken-manifest"}[s]
}

type Item struct {
	Key    string
	Src    string
	Dst    string
	State  State
	Detail string
}

type Action struct {
	Item   Item
	Backup string
}

// Inspect classifies every link. Items are sorted by Key.
func Inspect(links map[string]string, repoRoot, home string) []Item {
	items := make([]Item, 0, len(links))
	for key, val := range links {
		it := Item{
			Key: key,
			Src: filepath.Clean(filepath.Join(repoRoot, "home", key)),
			Dst: manifest.Expand(val, home),
		}
		it.State, it.Detail = classify(it.Src, it.Dst)
		items = append(items, it)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Key < items[j].Key })
	return items
}

func classify(src, dst string) (State, string) {
	if _, err := os.Lstat(src); err != nil {
		return BrokenManifest, "source missing in repo"
	}
	fi, err := os.Lstat(dst)
	if os.IsNotExist(err) {
		return Missing, ""
	}
	if err != nil {
		return Conflict, err.Error()
	}
	if fi.Mode()&os.ModeSymlink == 0 {
		if fi.IsDir() {
			return Conflict, "directory exists"
		}
		return Conflict, "regular file exists"
	}
	target, err := os.Readlink(dst)
	if err != nil {
		return WrongTarget, err.Error()
	}
	if !filepath.IsAbs(target) {
		target = filepath.Join(filepath.Dir(dst), target)
	}
	if filepath.Clean(target) == src {
		return Ok, ""
	}
	return WrongTarget, "points to " + target
}

// Plan returns the actions for items that need work.
func Plan(items []Item, backupDir, home string) []Action {
	var actions []Action
	for _, it := range items {
		switch it.State {
		case Missing, WrongTarget:
			actions = append(actions, Action{Item: it})
		case Conflict:
			rel, err := filepath.Rel(home, it.Dst)
			if err != nil || strings.HasPrefix(rel, "..") {
				rel = filepath.Base(it.Dst)
			}
			actions = append(actions, Action{Item: it, Backup: filepath.Join(backupDir, rel)})
		}
	}
	return actions
}

// Apply performs the actions. With dryRun it only reports them.
func Apply(actions []Action, dryRun bool, out io.Writer) result.Summary {
	var s result.Summary
	for _, a := range actions {
		detail := describe(a)
		if dryRun {
			fmt.Fprintf(out, "would %s %s\n", detail, a.Item.Key)
			s = append(s, result.Entry{Section: "link", ID: a.Item.Key, Status: result.Skipped, Detail: "would " + detail})
			continue
		}
		if err := apply(a); err != nil {
			s = append(s, result.Entry{Section: "link", ID: a.Item.Key, Status: result.Failed, Detail: err.Error()})
			continue
		}
		fmt.Fprintf(out, "%s %s\n", detail, a.Item.Key)
		s = append(s, result.Entry{Section: "link", ID: a.Item.Key, Status: result.Ok, Detail: detail})
	}
	return s
}

func describe(a Action) string {
	switch {
	case a.Backup != "":
		return "back up to " + a.Backup + " and link"
	case a.Item.State == WrongTarget:
		return "replace link (" + a.Item.Detail + ")"
	default:
		return "link"
	}
}

func apply(a Action) error {
	dst := a.Item.Dst
	if a.Backup != "" {
		if err := os.MkdirAll(filepath.Dir(a.Backup), 0o755); err != nil {
			return err
		}
		if err := os.Rename(dst, a.Backup); err != nil {
			return fmt.Errorf("backup: %w", err)
		}
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	tmp := dst + ".dot-tmp"
	_ = os.Remove(tmp) // left behind by an interrupted run
	if err := os.Symlink(a.Item.Src, tmp); err != nil {
		return err
	}
	if err := os.Rename(tmp, dst); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/link/`
Expected: `ok`

- [ ] **Step 5: Commit**

```bash
git add internal/link
git commit -m "feat(link): inspect, plan and apply symlinks with backups

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

---

### Task 6: Brewfile parsing and diffing (`internal/brew/brewfile.go`)

**Files:**
- Create: `internal/brew/brewfile.go`
- Test: `internal/brew/brewfile_test.go`

**Interfaces:**
- Produces:
  - `type Entry struct { Kind, Name, Rest string }` with `Key() string` (`kind + " " + name`) and `Line() string` (`kind "name"` + Rest verbatim)
  - `func Parse(r io.Reader) ([]Entry, error)`: skips blank lines and `#` comments; any other line not matching `<kind> "<name>"...` errors with its line number
  - `func Write(w io.Writer, entries []Entry) error`: one `Line()` per entry, newline terminated
  - `type Diff struct { Added, Removed []Entry }`; `func Compare(committed, dump []Entry) Diff` (Added = in dump not committed, dump order; Removed = committed not in dump, committed order)
  - `func Filter(dump, committed []Entry, keepAdded, dropRemoved map[string]bool) []Entry`: dump entries whose Key is committed or in keepAdded, in dump order, then committed entries absent from the dump and not in dropRemoved, in committed order
  - `func Taps(entries []Entry) []string`: names of `tap` entries in order

- [ ] **Step 1: Write the failing tests**

```go
// internal/brew/brewfile_test.go
package brew

import (
	"bytes"
	"strings"
	"testing"
)

const sample = `# generated by brew bundle dump
tap "felixkratz/formulae", "https://github.com/FelixKratz/homebrew-formulae"

brew "openssl@3", link: true
brew "jq"
cask "ghostty", args: { no_quarantine: true }
mas "Xcode", id: 497799835
`

func TestParseRoundTripKeepsOptionsVerbatim(t *testing.T) {
	entries, err := Parse(strings.NewReader(sample))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 5 {
		t.Fatalf("got %d entries: %+v", len(entries), entries)
	}
	if entries[0].Kind != "tap" || entries[0].Name != "felixkratz/formulae" || entries[0].Rest != `, "https://github.com/FelixKratz/homebrew-formulae"` {
		t.Fatalf("tap entry = %+v", entries[0])
	}
	if entries[1].Key() != "brew openssl@3" || entries[1].Rest != ", link: true" {
		t.Fatalf("brew entry = %+v", entries[1])
	}
	if entries[3].Line() != `cask "ghostty", args: { no_quarantine: true }` {
		t.Fatalf("Line() = %q", entries[3].Line())
	}
	var buf bytes.Buffer
	if err := Write(&buf, entries); err != nil {
		t.Fatal(err)
	}
	want := strings.Join([]string{
		`tap "felixkratz/formulae", "https://github.com/FelixKratz/homebrew-formulae"`,
		`brew "openssl@3", link: true`,
		`brew "jq"`,
		`cask "ghostty", args: { no_quarantine: true }`,
		`mas "Xcode", id: 497799835`,
	}, "\n") + "\n"
	if buf.String() != want {
		t.Fatalf("Write =\n%s\nwant\n%s", buf.String(), want)
	}
}

func TestParseRejectsGarbageWithLineNumber(t *testing.T) {
	_, err := Parse(strings.NewReader("brew \"jq\"\nthis is not a brewfile line\n"))
	if err == nil || !strings.Contains(err.Error(), "line 2") {
		t.Fatalf("want line 2 error, got %v", err)
	}
}

func e(kind, name string) Entry { return Entry{Kind: kind, Name: name} }

func TestCompare(t *testing.T) {
	committed := []Entry{e("tap", "a/b"), e("brew", "jq"), e("brew", "gone"), e("cask", "ghostty")}
	dump := []Entry{e("tap", "a/b"), e("brew", "jq"), e("brew", "new1"), e("cask", "ghostty"), e("cask", "new2")}
	d := Compare(committed, dump)
	if len(d.Added) != 2 || d.Added[0].Name != "new1" || d.Added[1].Name != "new2" {
		t.Fatalf("Added = %+v", d.Added)
	}
	if len(d.Removed) != 1 || d.Removed[0].Name != "gone" {
		t.Fatalf("Removed = %+v", d.Removed)
	}
}

func TestFilterKeepsCommittedOrderForRetainedRemovals(t *testing.T) {
	committed := []Entry{e("brew", "jq"), e("brew", "gone-keep"), e("brew", "gone-drop")}
	dump := []Entry{e("brew", "jq"), e("brew", "new-keep"), e("brew", "new-skip")}
	got := Filter(dump, committed,
		map[string]bool{"brew new-keep": true},
		map[string]bool{"brew gone-drop": true})
	var names []string
	for _, x := range got {
		names = append(names, x.Name)
	}
	if strings.Join(names, ",") != "jq,new-keep,gone-keep" {
		t.Fatalf("Filter = %v", names)
	}
}

func TestFilterWithNoCommittedKeepsOnlyChosen(t *testing.T) {
	dump := []Entry{e("brew", "a"), e("brew", "b")}
	got := Filter(dump, nil, map[string]bool{"brew b": true}, nil)
	if len(got) != 1 || got[0].Name != "b" {
		t.Fatalf("Filter = %+v", got)
	}
}

func TestTaps(t *testing.T) {
	got := Taps([]Entry{e("tap", "x/y"), e("brew", "jq"), e("tap", "z/w")})
	if strings.Join(got, ",") != "x/y,z/w" {
		t.Fatalf("Taps = %v", got)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/brew/`
Expected: FAIL, `undefined: Parse`

- [ ] **Step 3: Implement**

```go
// internal/brew/brewfile.go
package brew

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strings"
)

// Entry is one Brewfile line. Rest is everything after the closing quote of
// Name, kept verbatim so options such as `, link: true` survive a round trip.
type Entry struct {
	Kind string
	Name string
	Rest string
}

func (e Entry) Key() string  { return e.Kind + " " + e.Name }
func (e Entry) Line() string { return e.Kind + ` "` + e.Name + `"` + e.Rest }

var lineRE = regexp.MustCompile(`^([a-z_]+)\s+"([^"]+)"(.*)$`)

// Parse reads Brewfile entries, skipping blank lines and comments.
func Parse(r io.Reader) ([]Entry, error) {
	var entries []Entry
	sc := bufio.NewScanner(r)
	n := 0
	for sc.Scan() {
		n++
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		m := lineRE.FindStringSubmatch(line)
		if m == nil {
			return nil, fmt.Errorf("Brewfile line %d: cannot parse %q", n, line)
		}
		entries = append(entries, Entry{Kind: m[1], Name: m[2], Rest: m[3]})
	}
	return entries, sc.Err()
}

func Write(w io.Writer, entries []Entry) error {
	for _, e := range entries {
		if _, err := io.WriteString(w, e.Line()+"\n"); err != nil {
			return err
		}
	}
	return nil
}

type Diff struct {
	Added   []Entry
	Removed []Entry
}

func keySet(entries []Entry) map[string]bool {
	s := make(map[string]bool, len(entries))
	for _, e := range entries {
		s[e.Key()] = true
	}
	return s
}

// Compare reports what the dump has that the committed file lacks and vice versa.
func Compare(committed, dump []Entry) Diff {
	c, d := keySet(committed), keySet(dump)
	var diff Diff
	for _, e := range dump {
		if !c[e.Key()] {
			diff.Added = append(diff.Added, e)
		}
	}
	for _, e := range committed {
		if !d[e.Key()] {
			diff.Removed = append(diff.Removed, e)
		}
	}
	return diff
}

// Filter builds the new Brewfile content: dump entries that are committed or
// chosen, in dump order, then committed entries no longer installed and not
// chosen for removal, in committed order.
func Filter(dump, committed []Entry, keepAdded, dropRemoved map[string]bool) []Entry {
	c, d := keySet(committed), keySet(dump)
	var out []Entry
	for _, e := range dump {
		if c[e.Key()] || keepAdded[e.Key()] {
			out = append(out, e)
		}
	}
	for _, e := range committed {
		if !d[e.Key()] && !dropRemoved[e.Key()] {
			out = append(out, e)
		}
	}
	return out
}

func Taps(entries []Entry) []string {
	var taps []string
	for _, e := range entries {
		if e.Kind == "tap" {
			taps = append(taps, e.Name)
		}
	}
	return taps
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/brew/`
Expected: `ok`

- [ ] **Step 5: Commit**

```bash
git add internal/brew
git commit -m "feat(brew): Brewfile parse, diff and filter

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

---

### Task 7: Homebrew client (`internal/brew/client.go`)

**Files:**
- Create: `internal/brew/client.go`
- Test: `internal/brew/client_test.go`

**Interfaces:**
- Consumes: `exec.Runner`, `exec.Result`, `result.LastLine`, `Parse`
- Produces `type Client struct { R exec.Runner; File string; TempDir string }` with:
  - `Committed() ([]Entry, error)`: parses `File`; a missing file returns an empty slice and nil error
  - `DumpPath() string`: `<TempDir>/Brewfile.dump`
  - `Dump(ctx) ([]Entry, error)`: runs `brew bundle dump --force --file=<DumpPath>`, parses it, removes it
  - `Tapped(ctx) ([]string, error)`: `brew tap`, one per non-empty line
  - `Trusted(ctx, tap string) (bool, error)`: `brew tap-info --json <tap>`, reads `trusted` from the first array element
  - `Trust(ctx, tap string, out io.Writer) error`: `brew trust --tap <tap>`
  - `Install(ctx, out) error`: `brew bundle install --no-upgrade --file=<File>`
  - `Update(ctx, out) error`: `brew update`
  - `Upgrade(ctx, out) error`: `brew upgrade`
- Non-zero exits become `fmt.Errorf("%s: exit %d: %s", command, code, result.LastLine(output))`.

- [ ] **Step 1: Write the failing tests**

```go
// internal/brew/client_test.go
package brew

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ireydiak/dotfiles/internal/exec"
)

func newClient(t *testing.T) (Client, *exec.Fake) {
	t.Helper()
	f := exec.NewFake()
	dir := t.TempDir()
	return Client{R: f, File: filepath.Join(dir, "Brewfile"), TempDir: dir}, f
}

func TestCommittedMissingFileIsEmpty(t *testing.T) {
	c, _ := newClient(t)
	entries, err := c.Committed()
	if err != nil || len(entries) != 0 {
		t.Fatalf("got %v %v", entries, err)
	}
}

func TestCommittedParsesFile(t *testing.T) {
	c, _ := newClient(t)
	os.WriteFile(c.File, []byte("brew \"jq\"\n"), 0o644)
	entries, err := c.Committed()
	if err != nil || len(entries) != 1 || entries[0].Name != "jq" {
		t.Fatalf("got %v %v", entries, err)
	}
}

func TestDumpRunsBrewAndParsesTempFile(t *testing.T) {
	c, f := newClient(t)
	cmd := "brew bundle dump --force --file=" + c.DumpPath()
	os.WriteFile(c.DumpPath(), []byte("tap \"a/b\"\nbrew \"jq\"\n"), 0o644)
	f.Scripts[cmd] = exec.Result{}
	entries, err := c.Dump(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 || f.Calls[0] != cmd {
		t.Fatalf("entries=%v calls=%v", entries, f.Calls)
	}
	if _, err := os.Stat(c.DumpPath()); err == nil {
		t.Fatal("dump file should be removed after parsing")
	}
}

func TestDumpFailureIncludesLastLine(t *testing.T) {
	c, f := newClient(t)
	f.Default = exec.Result{ExitCode: 1, Stderr: "Error: brew exploded\n"}
	_, err := c.Dump(context.Background())
	if err == nil || !strings.Contains(err.Error(), "exit 1") || !strings.Contains(err.Error(), "brew exploded") {
		t.Fatalf("err = %v", err)
	}
}

func TestTappedAndTrusted(t *testing.T) {
	c, f := newClient(t)
	f.Scripts["brew tap"] = exec.Result{Stdout: "felixkratz/formulae\nhomebrew/services\n"}
	f.Scripts["brew tap-info --json felixkratz/formulae"] = exec.Result{Stdout: `[{"name":"felixkratz/formulae","trusted":false}]`}
	f.Scripts["brew tap-info --json homebrew/services"] = exec.Result{Stdout: `[{"name":"homebrew/services","trusted":true}]`}
	taps, err := c.Tapped(context.Background())
	if err != nil || strings.Join(taps, ",") != "felixkratz/formulae,homebrew/services" {
		t.Fatalf("Tapped = %v %v", taps, err)
	}
	if ok, _ := c.Trusted(context.Background(), "felixkratz/formulae"); ok {
		t.Fatal("felixkratz should be untrusted")
	}
	if ok, _ := c.Trusted(context.Background(), "homebrew/services"); !ok {
		t.Fatal("homebrew/services should be trusted")
	}
}

func TestCommandStrings(t *testing.T) {
	c, f := newClient(t)
	ctx := context.Background()
	var out bytes.Buffer
	c.Trust(ctx, "x/y", &out)
	c.Install(ctx, &out)
	c.Update(ctx, &out)
	c.Upgrade(ctx, &out)
	want := []string{
		"brew trust --tap x/y",
		"brew bundle install --no-upgrade --file=" + c.File,
		"brew update",
		"brew upgrade",
	}
	if strings.Join(f.Calls, "|") != strings.Join(want, "|") {
		t.Fatalf("calls = %v", f.Calls)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/brew/`
Expected: FAIL, `undefined: Client`

- [ ] **Step 3: Implement**

```go
// internal/brew/client.go
package brew

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/ireydiak/dotfiles/internal/exec"
	"github.com/ireydiak/dotfiles/internal/result"
)

// Client wraps the brew commands dot needs. File is the committed Brewfile.
type Client struct {
	R       exec.Runner
	File    string
	TempDir string
}

func (c Client) run(ctx context.Context, command string, out io.Writer) (exec.Result, error) {
	res, err := c.R.Run(ctx, command, out)
	if err != nil {
		return res, fmt.Errorf("%s: %w", command, err)
	}
	if res.ExitCode != 0 {
		return res, fmt.Errorf("%s: exit %d: %s", command, res.ExitCode, result.LastLine(res.Output()))
	}
	return res, nil
}

// Committed parses the committed Brewfile. A missing file is an empty list.
func (c Client) Committed() ([]Entry, error) {
	f, err := os.Open(c.File)
	if os.IsNotExist(err) {
		return []Entry{}, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return Parse(f)
}

func (c Client) DumpPath() string { return filepath.Join(c.TempDir, "Brewfile.dump") }

// Dump asks brew for the installed set and parses it.
func (c Client) Dump(ctx context.Context) ([]Entry, error) {
	path := c.DumpPath()
	defer os.Remove(path)
	if _, err := c.run(ctx, "brew bundle dump --force --file="+path, nil); err != nil {
		return nil, err
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("brew bundle dump produced no file: %w", err)
	}
	defer f.Close()
	return Parse(f)
}

func (c Client) Tapped(ctx context.Context) ([]string, error) {
	res, err := c.run(ctx, "brew tap", nil)
	if err != nil {
		return nil, err
	}
	var taps []string
	for _, l := range strings.Split(res.Stdout, "\n") {
		if l = strings.TrimSpace(l); l != "" {
			taps = append(taps, l)
		}
	}
	return taps, nil
}

func (c Client) Trusted(ctx context.Context, tap string) (bool, error) {
	res, err := c.run(ctx, "brew tap-info --json "+tap, nil)
	if err != nil {
		return false, err
	}
	var info []struct {
		Trusted bool `json:"trusted"`
	}
	if err := json.Unmarshal([]byte(res.Stdout), &info); err != nil {
		return false, fmt.Errorf("brew tap-info %s: %w", tap, err)
	}
	if len(info) == 0 {
		return false, fmt.Errorf("brew tap-info %s: empty response", tap)
	}
	return info[0].Trusted, nil
}

func (c Client) Trust(ctx context.Context, tap string, out io.Writer) error {
	_, err := c.run(ctx, "brew trust --tap "+tap, out)
	return err
}

func (c Client) Install(ctx context.Context, out io.Writer) error {
	_, err := c.run(ctx, "brew bundle install --no-upgrade --file="+c.File, out)
	return err
}

func (c Client) Update(ctx context.Context, out io.Writer) error {
	_, err := c.run(ctx, "brew update", out)
	return err
}

func (c Client) Upgrade(ctx context.Context, out io.Writer) error {
	_, err := c.run(ctx, "brew upgrade", out)
	return err
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/brew/`
Expected: `ok`

- [ ] **Step 5: Commit**

```bash
git add internal/brew
git commit -m "feat(brew): client for dump, tap trust, install and upgrade

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

---

### Task 8: Step runner (`internal/steps`)

**Files:**
- Create: `internal/steps/steps.go`
- Test: `internal/steps/steps_test.go`

**Interfaces:**
- Consumes: `exec.Runner`, `manifest.Step`, `result.*`
- Produces:
  - `func Install(ctx context.Context, r exec.Runner, list []manifest.Step, dryRun bool, out io.Writer) result.Summary`: per step run `Check`; exit 0 is Skipped "already satisfied"; else dryRun is Skipped "would run"; else run `Run` streaming to out, run `Check` again; exit 0 is Ok "installed", otherwise Failed with the last line of the run output or "check still failing after run"
  - `func Update(ctx context.Context, r exec.Runner, list []manifest.Step, dryRun bool, out io.Writer) result.Summary`: steps without `Update` produce no entry; dryRun is Skipped "would update"; else run `Update`, exit 0 is Ok "updated", otherwise Failed with last line
- Section is `step`, ID is the step id.

- [ ] **Step 1: Write the failing tests**

```go
// internal/steps/steps_test.go
package steps

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/ireydiak/dotfiles/internal/exec"
	"github.com/ireydiak/dotfiles/internal/manifest"
	"github.com/ireydiak/dotfiles/internal/result"
)

var list = []manifest.Step{
	{ID: "a", Check: "check-a", Run: "run-a", Update: "update-a"},
	{ID: "b", Check: "check-b", Run: "run-b"},
}

func TestInstallSkipsSatisfiedSteps(t *testing.T) {
	f := exec.NewFake() // Default exit 0: every check passes
	s := Install(context.Background(), f, list, false, &bytes.Buffer{})
	if len(s) != 2 || s[0].Status != result.Skipped || s[1].Status != result.Skipped {
		t.Fatalf("summary = %+v", s)
	}
	if strings.Join(f.Calls, ",") != "check-a,check-b" {
		t.Fatalf("run must not be called: %v", f.Calls)
	}
}

func TestInstallRunsThenRechecks(t *testing.T) {
	f := exec.NewFake()
	f.Queue["check-a"] = []exec.Result{{ExitCode: 1}, {ExitCode: 0}}
	f.Scripts["run-a"] = exec.Result{Stdout: "installing...\n"}
	var out bytes.Buffer
	s := Install(context.Background(), f, list[:1], false, &out)
	if s[0].Status != result.Ok || s[0].Detail != "installed" {
		t.Fatalf("entry = %+v", s[0])
	}
	if strings.Join(f.Calls, ",") != "check-a,run-a,check-a" {
		t.Fatalf("calls = %v", f.Calls)
	}
	if !strings.Contains(out.String(), "installing...") {
		t.Fatal("run output should stream to out")
	}
}

func TestInstallFailsWhenRecheckFails(t *testing.T) {
	f := exec.NewFake()
	f.Scripts["check-a"] = exec.Result{ExitCode: 1}
	f.Scripts["run-a"] = exec.Result{ExitCode: 2, Stderr: "curl: (6) could not resolve\n"}
	s := Install(context.Background(), f, list[:1], false, &bytes.Buffer{})
	if s[0].Status != result.Failed || !strings.Contains(s[0].Detail, "could not resolve") {
		t.Fatalf("entry = %+v", s[0])
	}
}

func TestInstallDryRunDoesNotRun(t *testing.T) {
	f := exec.NewFake()
	f.Default = exec.Result{ExitCode: 1}
	s := Install(context.Background(), f, list, true, &bytes.Buffer{})
	for i, e := range s {
		if e.Status != result.Skipped || e.Detail != "would run" {
			t.Fatalf("entry %d = %+v", i, e)
		}
	}
	if strings.Join(f.Calls, ",") != "check-a,check-b" {
		t.Fatalf("dry run must only check: %v", f.Calls)
	}
}

func TestUpdateOnlyForStepsThatDefineIt(t *testing.T) {
	f := exec.NewFake()
	f.Scripts["check-a"] = exec.Result{ExitCode: 1} // must be ignored by Update
	s := Update(context.Background(), f, list, false, &bytes.Buffer{})
	if len(s) != 1 || s[0].ID != "a" || s[0].Status != result.Ok || s[0].Detail != "updated" {
		t.Fatalf("summary = %+v", s)
	}
	if strings.Join(f.Calls, ",") != "update-a" {
		t.Fatalf("calls = %v", f.Calls)
	}
}

func TestUpdateDryRunAndFailure(t *testing.T) {
	f := exec.NewFake()
	s := Update(context.Background(), f, list, true, &bytes.Buffer{})
	if len(s) != 1 || s[0].Status != result.Skipped || s[0].Detail != "would update" || len(f.Calls) != 0 {
		t.Fatalf("dry run = %+v calls=%v", s, f.Calls)
	}
	f.Scripts["update-a"] = exec.Result{ExitCode: 1, Stderr: "nope\n"}
	s = Update(context.Background(), f, list, false, &bytes.Buffer{})
	if s[0].Status != result.Failed || s[0].Detail != "nope" {
		t.Fatalf("failure = %+v", s[0])
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/steps/`
Expected: FAIL, `undefined: Install`

- [ ] **Step 3: Implement**

```go
// internal/steps/steps.go
package steps

import (
	"context"
	"io"

	"github.com/ireydiak/dotfiles/internal/exec"
	"github.com/ireydiak/dotfiles/internal/manifest"
	"github.com/ireydiak/dotfiles/internal/result"
)

func entry(id string, st result.Status, detail string) result.Entry {
	return result.Entry{Section: "step", ID: id, Status: st, Detail: detail}
}

func passes(ctx context.Context, r exec.Runner, check string) bool {
	res, err := r.Run(ctx, check, nil)
	return err == nil && res.ExitCode == 0
}

// Install runs each step whose check fails, in manifest order.
func Install(ctx context.Context, r exec.Runner, list []manifest.Step, dryRun bool, out io.Writer) result.Summary {
	var s result.Summary
	for _, st := range list {
		if passes(ctx, r, st.Check) {
			s = append(s, entry(st.ID, result.Skipped, "already satisfied"))
			continue
		}
		if dryRun {
			s = append(s, entry(st.ID, result.Skipped, "would run"))
			continue
		}
		res, err := r.Run(ctx, st.Run, out)
		if err != nil {
			s = append(s, entry(st.ID, result.Failed, err.Error()))
			continue
		}
		if passes(ctx, r, st.Check) {
			s = append(s, entry(st.ID, result.Ok, "installed"))
			continue
		}
		detail := result.LastLine(res.Output())
		if detail == "" {
			detail = "check still failing after run"
		}
		s = append(s, entry(st.ID, result.Failed, detail))
	}
	return s
}

// Update runs the update command of every step that defines one.
func Update(ctx context.Context, r exec.Runner, list []manifest.Step, dryRun bool, out io.Writer) result.Summary {
	var s result.Summary
	for _, st := range list {
		if st.Update == "" {
			continue
		}
		if dryRun {
			s = append(s, entry(st.ID, result.Skipped, "would update"))
			continue
		}
		res, err := r.Run(ctx, st.Update, out)
		switch {
		case err != nil:
			s = append(s, entry(st.ID, result.Failed, err.Error()))
		case res.ExitCode != 0:
			s = append(s, entry(st.ID, result.Failed, result.LastLine(res.Output())))
		default:
			s = append(s, entry(st.ID, result.Ok, "updated"))
		}
	}
	return s
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/steps/`
Expected: `ok`

- [ ] **Step 5: Commit**

```bash
git add internal/steps
git commit -m "feat(steps): idempotent install and update runner

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

---

### Task 9: Services and manual checks (`internal/services`, `internal/manual`)

**Files:**
- Create: `internal/services/services.go`, `internal/manual/manual.go`
- Test: `internal/services/services_test.go`, `internal/manual/manual_test.go`

**Interfaces:**
- Consumes: `exec.Runner`, `manifest.Service`, `manifest.Manual`, `result.*`
- Produces in `services`:
  - `type Item struct { ID, Label string; Loaded bool }`
  - `func Command(uid, label string) string` returning `launchctl print gui/<uid>/<label>`
  - `func Status(ctx, r exec.Runner, list []manifest.Service, uid string) []Item` (Loaded when the command exits 0)
  - `func Start(ctx, r exec.Runner, list []manifest.Service, uid string, dryRun bool, out io.Writer) result.Summary`: loaded is Skipped "already loaded"; dryRun is Skipped "would start"; else run `Start`, recheck, Ok "started" or Failed with last line. Section `service`.
- Produces in `manual`:
  - `type State int` with `Done`, `Pending`, `Unverifiable`; `String()` gives `done`, `pending`, `verify`
  - `type Item struct { ID string; State State; How string }`
  - `func Status(ctx, r exec.Runner, list []manifest.Manual) []Item`: no check is Unverifiable; exit 0 is Done; otherwise Pending
  - `func Pending(items []Item) []Item`: Pending and Unverifiable items, in order

- [ ] **Step 1: Write the failing tests**

```go
// internal/services/services_test.go
package services

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/ireydiak/dotfiles/internal/exec"
	"github.com/ireydiak/dotfiles/internal/manifest"
	"github.com/ireydiak/dotfiles/internal/result"
)

var list = []manifest.Service{
	{ID: "yabai", Label: "com.koekeishiya.yabai", Start: "yabai --start-service"},
	{ID: "skhd", Label: "com.jackielii.skhd", Start: "skhd --start-service"},
}

func TestStatus(t *testing.T) {
	f := exec.NewFake()
	f.Scripts[Command("501", "com.jackielii.skhd")] = exec.Result{ExitCode: 113, Stderr: "Could not find service\n"}
	items := Status(context.Background(), f, list, "501")
	if !items[0].Loaded || items[1].Loaded {
		t.Fatalf("items = %+v", items)
	}
	if f.Calls[0] != "launchctl print gui/501/com.koekeishiya.yabai" {
		t.Fatalf("command = %q", f.Calls[0])
	}
}

func TestStartOnlyUnloaded(t *testing.T) {
	f := exec.NewFake()
	f.Queue[Command("501", "com.jackielii.skhd")] = []exec.Result{{ExitCode: 113}, {ExitCode: 0}}
	s := Start(context.Background(), f, list, "501", false, &bytes.Buffer{})
	if s[0].Status != result.Skipped || s[0].Detail != "already loaded" {
		t.Fatalf("yabai = %+v", s[0])
	}
	if s[1].Status != result.Ok || s[1].Detail != "started" {
		t.Fatalf("skhd = %+v", s[1])
	}
	if !strings.Contains(strings.Join(f.Calls, ","), "skhd --start-service") || strings.Contains(strings.Join(f.Calls, ","), "yabai --start-service") {
		t.Fatalf("calls = %v", f.Calls)
	}
}

func TestStartDryRunAndFailure(t *testing.T) {
	f := exec.NewFake()
	f.Default = exec.Result{ExitCode: 113}
	s := Start(context.Background(), f, list[:1], "501", true, &bytes.Buffer{})
	if s[0].Status != result.Skipped || s[0].Detail != "would start" {
		t.Fatalf("dry run = %+v", s[0])
	}
	f.Scripts["yabai --start-service"] = exec.Result{ExitCode: 1, Stderr: "yabai: permission denied\n"}
	s = Start(context.Background(), f, list[:1], "501", false, &bytes.Buffer{})
	if s[0].Status != result.Failed || !strings.Contains(s[0].Detail, "permission denied") {
		t.Fatalf("failure = %+v", s[0])
	}
}
```

```go
// internal/manual/manual_test.go
package manual

import (
	"context"
	"testing"

	"github.com/ireydiak/dotfiles/internal/exec"
	"github.com/ireydiak/dotfiles/internal/manifest"
)

func TestStatusAndPending(t *testing.T) {
	f := exec.NewFake()
	f.Scripts["csrutil-check"] = exec.Result{ExitCode: 1}
	list := []manifest.Manual{
		{ID: "xcode", Check: "xcode-select -p", How: "xcode-select --install"},
		{ID: "sip", Check: "csrutil-check", How: "recovery mode"},
		{ID: "accessibility", How: "System Settings"},
	}
	items := Status(context.Background(), f, list)
	if items[0].State != Done || items[1].State != Pending || items[2].State != Unverifiable {
		t.Fatalf("items = %+v", items)
	}
	if items[2].State.String() != "verify" || items[1].How != "recovery mode" {
		t.Fatalf("string/how = %q %q", items[2].State, items[1].How)
	}
	p := Pending(items)
	if len(p) != 2 || p[0].ID != "sip" || p[1].ID != "accessibility" {
		t.Fatalf("Pending = %+v", p)
	}
	if len(f.Calls) != 2 {
		t.Fatalf("items without check must not run anything: %v", f.Calls)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/services/ ./internal/manual/`
Expected: FAIL, `undefined: Command`, `undefined: Status`

- [ ] **Step 3: Implement**

```go
// internal/services/services.go
package services

import (
	"context"
	"io"

	"github.com/ireydiak/dotfiles/internal/exec"
	"github.com/ireydiak/dotfiles/internal/manifest"
	"github.com/ireydiak/dotfiles/internal/result"
)

type Item struct {
	ID     string
	Label  string
	Loaded bool
}

// Command is the launchctl query whose exit code says whether label is loaded.
func Command(uid, label string) string {
	return "launchctl print gui/" + uid + "/" + label
}

func loaded(ctx context.Context, r exec.Runner, uid, label string) bool {
	res, err := r.Run(ctx, Command(uid, label), nil)
	return err == nil && res.ExitCode == 0
}

func Status(ctx context.Context, r exec.Runner, list []manifest.Service, uid string) []Item {
	items := make([]Item, 0, len(list))
	for _, svc := range list {
		items = append(items, Item{ID: svc.ID, Label: svc.Label, Loaded: loaded(ctx, r, uid, svc.Label)})
	}
	return items
}

// Start starts every service that is not loaded.
func Start(ctx context.Context, r exec.Runner, list []manifest.Service, uid string, dryRun bool, out io.Writer) result.Summary {
	var s result.Summary
	add := func(id string, st result.Status, detail string) {
		s = append(s, result.Entry{Section: "service", ID: id, Status: st, Detail: detail})
	}
	for _, svc := range list {
		if loaded(ctx, r, uid, svc.Label) {
			add(svc.ID, result.Skipped, "already loaded")
			continue
		}
		if dryRun {
			add(svc.ID, result.Skipped, "would start")
			continue
		}
		res, err := r.Run(ctx, svc.Start, out)
		switch {
		case err != nil:
			add(svc.ID, result.Failed, err.Error())
		case res.ExitCode != 0:
			add(svc.ID, result.Failed, result.LastLine(res.Output()))
		case loaded(ctx, r, uid, svc.Label):
			add(svc.ID, result.Ok, "started")
		default:
			add(svc.ID, result.Failed, "not loaded after start")
		}
	}
	return s
}
```

```go
// internal/manual/manual.go
package manual

import (
	"context"

	"github.com/ireydiak/dotfiles/internal/exec"
	"github.com/ireydiak/dotfiles/internal/manifest"
)

type State int

const (
	Done State = iota
	Pending
	Unverifiable
)

func (s State) String() string {
	return [...]string{"done", "pending", "verify"}[s]
}

type Item struct {
	ID    string
	State State
	How   string
}

// Status runs each item's check when it has one.
func Status(ctx context.Context, r exec.Runner, list []manifest.Manual) []Item {
	items := make([]Item, 0, len(list))
	for _, m := range list {
		it := Item{ID: m.ID, How: m.How, State: Unverifiable}
		if m.Check != "" {
			it.State = Pending
			if res, err := r.Run(ctx, m.Check, nil); err == nil && res.ExitCode == 0 {
				it.State = Done
			}
		}
		items = append(items, it)
	}
	return items
}

// Pending returns the items a person still has to look at.
func Pending(items []Item) []Item {
	var out []Item
	for _, it := range items {
		if it.State != Done {
			out = append(out, it)
		}
	}
	return out
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/services/ ./internal/manual/`
Expected: `ok` for both

- [ ] **Step 5: Commit**

```bash
git add internal/services internal/manual
git commit -m "feat: launchd service status/start and manual checklist

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

---

### Task 10: Repo root and git operations (`internal/repo`)

**Files:**
- Create: `internal/repo/repo.go`
- Test: `internal/repo/repo_test.go`

**Interfaces:**
- Consumes: `exec.Runner`, `result.LastLine`
- Produces:
  - `type ResolveOptions struct { Flag, Env, Cwd, Home string; GitTop func(dir string) (string, error) }`
  - `func Resolve(o ResolveOptions) (string, error)`: Flag, then Env (each, when set, must contain `manifest.toml` or Resolve errors), then `GitTop(Cwd)` if it contains `manifest.toml`, then `<Home>/.dotfiles`. The chosen path is canonicalized with `filepath.EvalSymlinks`.
  - `func GitTop(ctx context.Context, r exec.Runner) func(dir string) (string, error)`: runs `git -C <dir> rev-parse --show-toplevel`
  - `type Git struct { Root string; R exec.Runner }` with `Changes(ctx) ([]string, error)` (paths from `git -C <root> status --porcelain`, column 4 onward, renames keep the new name), `HasRemote(ctx) (bool, error)`, `BrewfileNumstat(ctx) (added, removed int, err error)` from `git -C <root> diff --numstat HEAD -- Brewfile`, `CommitAll(ctx, msg string, out io.Writer) error` (`git -C <root> add -A && git -C <root> commit -m '<msg>'` with single quotes escaped), `Push(ctx, out) error`
  - `func Message(changes []string, brewAdded, brewRemoved int) string`: `dot: update <areas>` where an area is `.config/<x>` or `Library/Application Support/<x>` or the first path component under `home/`, plus top-level files other than `Brewfile`; appends `; Brewfile +a -r` when either count is non-zero; `dot: update` when nothing else applies
  - `func ShellQuote(s string) string`: wraps in single quotes, escaping embedded single quotes as `'\''`

- [ ] **Step 1: Write the failing tests**

```go
// internal/repo/repo_test.go
package repo

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ireydiak/dotfiles/internal/exec"
)

func repoDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "manifest.toml"), []byte(""), 0o644)
	real, _ := filepath.EvalSymlinks(dir)
	return real
}

func TestResolveOrder(t *testing.T) {
	flagRepo, envRepo, gitRepo := repoDir(t), repoDir(t), repoDir(t)
	home := t.TempDir()
	gitTop := func(string) (string, error) { return gitRepo, nil }

	if got, _ := Resolve(ResolveOptions{Flag: flagRepo, Env: envRepo, Cwd: "/x", Home: home, GitTop: gitTop}); got != flagRepo {
		t.Fatalf("flag should win, got %q", got)
	}
	if got, _ := Resolve(ResolveOptions{Env: envRepo, Cwd: "/x", Home: home, GitTop: gitTop}); got != envRepo {
		t.Fatalf("env should win over git, got %q", got)
	}
	if got, _ := Resolve(ResolveOptions{Cwd: "/x", Home: home, GitTop: gitTop}); got != gitRepo {
		t.Fatalf("git top should be used, got %q", got)
	}
	noGit := func(string) (string, error) { return "", errors.New("not a repo") }
	defRepo := repoDir(t)
	os.Symlink(defRepo, filepath.Join(home, ".dotfiles"))
	if got, _ := Resolve(ResolveOptions{Cwd: "/x", Home: home, GitTop: noGit}); got != defRepo {
		t.Fatalf("default ~/.dotfiles symlink should resolve to real dir, got %q", got)
	}
}

func TestResolveErrors(t *testing.T) {
	home := t.TempDir()
	noGit := func(string) (string, error) { return "", errors.New("not a repo") }
	if _, err := Resolve(ResolveOptions{Flag: t.TempDir(), Home: home, GitTop: noGit}); err == nil || !strings.Contains(err.Error(), "manifest.toml") {
		t.Fatalf("flag without manifest must error, got %v", err)
	}
	if _, err := Resolve(ResolveOptions{Cwd: "/x", Home: home, GitTop: noGit}); err == nil {
		t.Fatal("no candidates must error")
	}
}

func TestResolveMakesFlagAbsolute(t *testing.T) {
	flagRepo := repoDir(t)
	t.Chdir(flagRepo)
	got, err := Resolve(ResolveOptions{Flag: ".", Home: t.TempDir()})
	if err != nil || got != flagRepo {
		t.Fatalf("Resolve(.) = %q %v, want %q", got, err, flagRepo)
	}
}

func TestGitChangesAndMessage(t *testing.T) {
	f := exec.NewFake()
	g := Git{Root: "/r", R: f}
	f.Scripts["git -C /r status --porcelain"] = exec.Result{Stdout: " M home/.config/nvim/lua/plugins/lsp.lua\n M home/.config/tmux/tmux.conf\n?? home/.zshrc\nM  Brewfile\nR  old.md -> manifest.toml\n"}
	f.Scripts["git -C /r diff --numstat HEAD -- Brewfile"] = exec.Result{Stdout: "3\t1\tBrewfile\n"}
	changes, err := g.Changes(context.Background())
	if err != nil || len(changes) != 5 || changes[4] != "manifest.toml" {
		t.Fatalf("changes = %v %v", changes, err)
	}
	a, r, _ := g.BrewfileNumstat(context.Background())
	if a != 3 || r != 1 {
		t.Fatalf("numstat = %d %d", a, r)
	}
	msg := Message(changes, a, r)
	if msg != "dot: update .config/nvim, .config/tmux, .zshrc, manifest.toml; Brewfile +3 -1" {
		t.Fatalf("Message = %q", msg)
	}
	if got := Message(nil, 0, 0); got != "dot: update" {
		t.Fatalf("empty Message = %q", got)
	}
	if got := Message([]string{"home/Library/Application Support/lazygit/config.yml"}, 0, 0); got != "dot: update Library/Application Support/lazygit" {
		t.Fatalf("Message = %q", got)
	}
}

func TestGitCommitPushAndRemote(t *testing.T) {
	f := exec.NewFake()
	g := Git{Root: "/r", R: f}
	f.Scripts["git -C /r remote"] = exec.Result{Stdout: "origin\n"}
	if has, _ := g.HasRemote(context.Background()); !has {
		t.Fatal("expected remote")
	}
	if err := g.CommitAll(context.Background(), "it's done", nil); err != nil {
		t.Fatal(err)
	}
	if err := g.Push(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	want := `git -C /r add -A && git -C /r commit -m 'it'\''s done'`
	if f.Calls[1] != want || f.Calls[2] != "git -C /r push" {
		t.Fatalf("calls = %q", f.Calls)
	}
	f.Scripts[want] = exec.Result{ExitCode: 1, Stderr: "nothing added\n"}
	if err := g.CommitAll(context.Background(), "it's done", nil); err == nil || !strings.Contains(err.Error(), "nothing added") {
		t.Fatalf("commit failure should surface, got %v", err)
	}
}

func TestGitTop(t *testing.T) {
	f := exec.NewFake()
	f.Scripts["git -C /some/dir rev-parse --show-toplevel"] = exec.Result{Stdout: "/some\n"}
	top, err := GitTop(context.Background(), f)("/some/dir")
	if err != nil || top != "/some" {
		t.Fatalf("GitTop = %q %v", top, err)
	}
	f.Default = exec.Result{ExitCode: 128}
	if _, err := GitTop(context.Background(), f)("/elsewhere"); err == nil {
		t.Fatal("non-repo must error")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/repo/`
Expected: FAIL, `undefined: Resolve`

- [ ] **Step 3: Implement**

```go
// internal/repo/repo.go
package repo

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ireydiak/dotfiles/internal/exec"
	"github.com/ireydiak/dotfiles/internal/result"
)

type ResolveOptions struct {
	Flag   string
	Env    string
	Cwd    string
	Home   string
	GitTop func(dir string) (string, error)
}

func hasManifest(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, "manifest.toml"))
	return err == nil
}

// Resolve picks the repo root: --repo, then DOTFILES_DIR, then the git top
// level of the working directory, then ~/.dotfiles. The result is canonical.
func Resolve(o ResolveOptions) (string, error) {
	for _, c := range []struct{ name, path string }{{"--repo", o.Flag}, {"DOTFILES_DIR", o.Env}} {
		if c.path == "" {
			continue
		}
		abs, err := filepath.Abs(c.path)
		if err != nil {
			return "", fmt.Errorf("%s=%s: %w", c.name, c.path, err)
		}
		real, err := filepath.EvalSymlinks(abs)
		if err != nil || !hasManifest(real) {
			return "", fmt.Errorf("%s=%s does not contain manifest.toml", c.name, c.path)
		}
		return real, nil
	}
	if o.GitTop != nil && o.Cwd != "" {
		if top, err := o.GitTop(o.Cwd); err == nil {
			if real, err := filepath.EvalSymlinks(top); err == nil && hasManifest(real) {
				return real, nil
			}
		}
	}
	def := filepath.Join(o.Home, ".dotfiles")
	if real, err := filepath.EvalSymlinks(def); err == nil && hasManifest(real) {
		return real, nil
	}
	return "", fmt.Errorf("no dotfiles repo found: pass --repo, set DOTFILES_DIR, run inside the repo, or clone it to %s", def)
}

// GitTop returns a function that finds the git top level of a directory.
func GitTop(ctx context.Context, r exec.Runner) func(dir string) (string, error) {
	return func(dir string) (string, error) {
		res, err := r.Run(ctx, "git -C "+dir+" rev-parse --show-toplevel", nil)
		if err != nil {
			return "", err
		}
		if res.ExitCode != 0 {
			return "", fmt.Errorf("%s is not inside a git repository", dir)
		}
		return strings.TrimSpace(res.Stdout), nil
	}
}

func ShellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

type Git struct {
	Root string
	R    exec.Runner
}

func (g Git) run(ctx context.Context, args string, out io.Writer) (exec.Result, error) {
	cmd := "git -C " + g.Root + " " + args
	res, err := g.R.Run(ctx, cmd, out)
	if err != nil {
		return res, fmt.Errorf("%s: %w", cmd, err)
	}
	if res.ExitCode != 0 {
		return res, fmt.Errorf("%s: exit %d: %s", cmd, res.ExitCode, result.LastLine(res.Output()))
	}
	return res, nil
}

// Changes lists paths with uncommitted changes.
func (g Git) Changes(ctx context.Context) ([]string, error) {
	res, err := g.run(ctx, "status --porcelain", nil)
	if err != nil {
		return nil, err
	}
	var paths []string
	for _, line := range strings.Split(res.Stdout, "\n") {
		if len(line) < 4 {
			continue
		}
		p := line[3:]
		if i := strings.Index(p, " -> "); i >= 0 {
			p = p[i+4:]
		}
		paths = append(paths, p)
	}
	return paths, nil
}

func (g Git) HasRemote(ctx context.Context) (bool, error) {
	res, err := g.run(ctx, "remote", nil)
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(res.Stdout) != "", nil
}

// BrewfileNumstat returns lines added and removed in Brewfile versus HEAD.
func (g Git) BrewfileNumstat(ctx context.Context) (int, int, error) {
	res, err := g.run(ctx, "diff --numstat HEAD -- Brewfile", nil)
	if err != nil {
		return 0, 0, err
	}
	fields := strings.Fields(res.Stdout)
	if len(fields) < 2 {
		return 0, 0, nil
	}
	a, _ := strconv.Atoi(fields[0])
	r, _ := strconv.Atoi(fields[1])
	return a, r, nil
}

func (g Git) CommitAll(ctx context.Context, msg string, out io.Writer) error {
	_, err := g.run(ctx, "add -A && git -C "+g.Root+" commit -m "+ShellQuote(msg), out)
	return err
}

func (g Git) Push(ctx context.Context, out io.Writer) error {
	_, err := g.run(ctx, "push", out)
	return err
}

// Message summarizes changed paths into a commit message.
func Message(changes []string, brewAdded, brewRemoved int) string {
	var areas []string
	seen := map[string]bool{}
	for _, p := range changes {
		area := areaOf(p)
		if area == "" || seen[area] {
			continue
		}
		seen[area] = true
		areas = append(areas, area)
	}
	msg := "dot: update"
	if len(areas) > 0 {
		msg += " " + strings.Join(areas, ", ")
	}
	if brewAdded != 0 || brewRemoved != 0 {
		msg += fmt.Sprintf("; Brewfile +%d -%d", brewAdded, brewRemoved)
	}
	return msg
}

func areaOf(p string) string {
	if p == "Brewfile" {
		return ""
	}
	rel, ok := strings.CutPrefix(p, "home/")
	if !ok {
		return strings.SplitN(p, "/", 2)[0]
	}
	parts := strings.Split(rel, "/")
	switch {
	case parts[0] == ".config" && len(parts) > 1:
		return ".config/" + parts[1]
	case parts[0] == "Library" && len(parts) > 2:
		return strings.Join(parts[:3], "/")
	default:
		return parts[0]
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/repo/`
Expected: `ok`

- [ ] **Step 5: Commit**

```bash
git add internal/repo
git commit -m "feat(repo): root resolution, git status/commit, message builder

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

---

### Task 11: Status report (`internal/status`)

**Files:**
- Create: `internal/status/status.go`
- Test: `internal/status/status_test.go`

**Interfaces:**
- Consumes: `brew.Client`, `brew.Compare`, `link.Inspect`, `services.Status`, `manual.Status`, `repo.Git`, `manifest.Manifest`
- Produces:
  - `type Packages struct { Missing, Unrecorded []brew.Entry; Err string }`
  - `type RepoState struct { Dirty bool; Changes []string; Err string }`
  - `type Report struct { Packages Packages; Links []link.Item; Services []services.Item; Manual []manual.Item; Repo RepoState }` with json tags `packages`, `links`, `services`, `manual`, `repo`
  - `type Deps struct { Ctx context.Context; Manifest *manifest.Manifest; RepoRoot, Home, UID string; Runner exec.Runner; Brew brew.Client; Git repo.Git }`
  - `func Collect(d Deps) Report`: Missing is `Compare(committed, dump).Removed`, Unrecorded is `.Added`; any brew error goes to `Packages.Err` and the rest still runs
  - `func (r Report) Print(w io.Writer)` and `func (r Report) JSON(w io.Writer) error`

- [ ] **Step 1: Write the failing tests**

```go
// internal/status/status_test.go
package status

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ireydiak/dotfiles/internal/brew"
	"github.com/ireydiak/dotfiles/internal/exec"
	"github.com/ireydiak/dotfiles/internal/manifest"
	"github.com/ireydiak/dotfiles/internal/repo"
)

func deps(t *testing.T, f *exec.Fake) Deps {
	t.Helper()
	root, home := t.TempDir(), t.TempDir()
	os.MkdirAll(filepath.Join(root, "home"), 0o755)
	os.WriteFile(filepath.Join(root, "home", ".zshrc"), []byte("x"), 0o644)
	m := &manifest.Manifest{
		Links:    map[string]string{".zshrc": "~/.zshrc"},
		Services: []manifest.Service{{ID: "yabai", Label: "com.koekeishiya.yabai", Start: "yabai --start-service"}},
		Manual:   []manifest.Manual{{ID: "accessibility", How: "System Settings"}},
	}
	return Deps{
		Ctx: context.Background(), Manifest: m, RepoRoot: root, Home: home, UID: "501", Runner: f,
		Brew: brew.Client{R: f, File: filepath.Join(root, "Brewfile"), TempDir: root},
		Git:  repo.Git{Root: root, R: f},
	}
}

func TestCollectWithWorkingBrew(t *testing.T) {
	f := exec.NewFake()
	d := deps(t, f)
	os.WriteFile(d.Brew.File, []byte("brew \"jq\"\nbrew \"gone\"\n"), 0o644)
	os.WriteFile(d.Brew.DumpPath(), []byte("brew \"jq\"\nbrew \"new\"\n"), 0o644)
	f.Scripts["git -C "+d.RepoRoot+" status --porcelain"] = exec.Result{Stdout: " M home/.zshrc\n"}
	r := Collect(d)
	if r.Packages.Err != "" || len(r.Packages.Missing) != 1 || r.Packages.Missing[0].Name != "gone" || len(r.Packages.Unrecorded) != 1 {
		t.Fatalf("packages = %+v", r.Packages)
	}
	if len(r.Links) != 1 || r.Links[0].Key != ".zshrc" {
		t.Fatalf("links = %+v", r.Links)
	}
	if len(r.Services) != 1 || !r.Services[0].Loaded {
		t.Fatalf("services = %+v", r.Services)
	}
	if !r.Repo.Dirty || len(r.Repo.Changes) != 1 {
		t.Fatalf("repo = %+v", r.Repo)
	}
}

func TestCollectSurvivesBrewFailure(t *testing.T) {
	f := exec.NewFake()
	d := deps(t, f)
	f.Scripts["brew bundle dump --force --file="+d.Brew.DumpPath()] = exec.Result{ExitCode: 127, Stderr: "zsh: command not found: brew\n"}
	r := Collect(d)
	if !strings.Contains(r.Packages.Err, "command not found") {
		t.Fatalf("Packages.Err = %q", r.Packages.Err)
	}
	if len(r.Links) != 1 || len(r.Services) != 1 || len(r.Manual) != 1 {
		t.Fatal("other sections must still be collected when brew fails")
	}
}

func TestPrintAndJSON(t *testing.T) {
	f := exec.NewFake()
	d := deps(t, f)
	os.WriteFile(d.Brew.DumpPath(), []byte(""), 0o644)
	r := Collect(d)
	var buf bytes.Buffer
	r.Print(&buf)
	out := buf.String()
	for _, want := range []string{"Packages", "Links", "missing", ".zshrc", "Services", "loaded", "yabai", "Manual", "verify", "accessibility", "Repo", "clean"} {
		if !strings.Contains(out, want) {
			t.Fatalf("Print missing %q:\n%s", want, out)
		}
	}
	buf.Reset()
	if err := r.JSON(&buf); err != nil {
		t.Fatal(err)
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(buf.Bytes(), &m); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"packages", "links", "services", "manual", "repo"} {
		if _, ok := m[k]; !ok {
			t.Fatalf("JSON missing key %q", k)
		}
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/status/`
Expected: FAIL, `undefined: Deps`

- [ ] **Step 3: Implement**

```go
// internal/status/status.go
package status

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/ireydiak/dotfiles/internal/brew"
	"github.com/ireydiak/dotfiles/internal/exec"
	"github.com/ireydiak/dotfiles/internal/link"
	"github.com/ireydiak/dotfiles/internal/manifest"
	"github.com/ireydiak/dotfiles/internal/manual"
	"github.com/ireydiak/dotfiles/internal/repo"
	"github.com/ireydiak/dotfiles/internal/services"
)

type Packages struct {
	Missing    []brew.Entry `json:"missing"`
	Unrecorded []brew.Entry `json:"unrecorded"`
	Err        string       `json:"error,omitempty"`
}

type RepoState struct {
	Dirty   bool     `json:"dirty"`
	Changes []string `json:"changes"`
	Err     string   `json:"error,omitempty"`
}

type Report struct {
	Packages Packages         `json:"packages"`
	Links    []link.Item      `json:"links"`
	Services []services.Item  `json:"services"`
	Manual   []manual.Item    `json:"manual"`
	Repo     RepoState        `json:"repo"`
}

type Deps struct {
	Ctx      context.Context
	Manifest *manifest.Manifest
	RepoRoot string
	Home     string
	UID      string
	Runner   exec.Runner
	Brew     brew.Client
	Git      repo.Git
}

// Collect gathers every section. A failing section records its error and
// never prevents the others from being collected.
func Collect(d Deps) Report {
	var r Report
	r.Packages = collectPackages(d)
	r.Links = link.Inspect(d.Manifest.Links, d.RepoRoot, d.Home)
	r.Services = services.Status(d.Ctx, d.Runner, d.Manifest.Services, d.UID)
	r.Manual = manual.Status(d.Ctx, d.Runner, d.Manifest.Manual)
	changes, err := d.Git.Changes(d.Ctx)
	if err != nil {
		r.Repo.Err = err.Error()
	}
	r.Repo.Changes = changes
	r.Repo.Dirty = len(changes) > 0
	return r
}

func collectPackages(d Deps) Packages {
	committed, err := d.Brew.Committed()
	if err != nil {
		return Packages{Err: err.Error()}
	}
	dump, err := d.Brew.Dump(d.Ctx)
	if err != nil {
		return Packages{Err: err.Error()}
	}
	diff := brew.Compare(committed, dump)
	return Packages{Missing: diff.Removed, Unrecorded: diff.Added}
}

func (r Report) JSON(w io.Writer) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(r)
}

func (r Report) Print(w io.Writer) {
	fmt.Fprintln(w, "Packages")
	switch {
	case r.Packages.Err != "":
		fmt.Fprintf(w, "  error         %s\n", r.Packages.Err)
	case len(r.Packages.Missing) == 0 && len(r.Packages.Unrecorded) == 0:
		fmt.Fprintln(w, "  all Brewfile entries installed, nothing unrecorded")
	}
	for _, e := range r.Packages.Missing {
		fmt.Fprintf(w, "  missing       %s\n", e.Line())
	}
	for _, e := range r.Packages.Unrecorded {
		fmt.Fprintf(w, "  unrecorded    %s\n", e.Line())
	}

	fmt.Fprintln(w, "Links")
	for _, it := range r.Links {
		detail := ""
		if it.Detail != "" {
			detail = "  (" + it.Detail + ")"
		}
		fmt.Fprintf(w, "  %-13s %s%s\n", it.State, it.Key, detail)
	}

	fmt.Fprintln(w, "Services")
	for _, s := range r.Services {
		state := "not loaded"
		if s.Loaded {
			state = "loaded"
		}
		fmt.Fprintf(w, "  %-13s %s\n", state, s.ID)
	}

	fmt.Fprintln(w, "Manual")
	for _, m := range r.Manual {
		how := ""
		if m.State != manual.Done {
			how = "  → " + m.How
		}
		fmt.Fprintf(w, "  %-13s %s%s\n", m.State, m.ID, how)
	}

	fmt.Fprintln(w, "Repo")
	switch {
	case r.Repo.Err != "":
		fmt.Fprintf(w, "  error         %s\n", r.Repo.Err)
	case !r.Repo.Dirty:
		fmt.Fprintln(w, "  clean")
	default:
		fmt.Fprintf(w, "  %d uncommitted: %s\n", len(r.Repo.Changes), strings.Join(r.Repo.Changes, ", "))
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/status/`
Expected: `ok`

- [ ] **Step 5: Commit**

```bash
git add internal/status
git commit -m "feat(status): aggregate report with text and JSON output

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

---

### Task 12: Application layer (`internal/app`)

**Files:**
- Create: `internal/app/app.go`
- Test: `internal/app/app_test.go`

**Interfaces:**
- Consumes: everything from Tasks 2 to 11
- Produces:
  - `type Options struct { RepoFlag, Env, Home, Cwd, UID string; DryRun, Yes bool; Runner exec.Runner; Now func() time.Time }`. Empty `Home`, `Cwd`, `UID`, nil `Runner` and nil `Now` default to the real environment (`os.UserHomeDir`, `os.Getwd`, `os.Getuid`, `exec.NewShell(home)`, `time.Now`).
  - `type App struct { Ctx context.Context; Root, Home, UID string; Manifest *manifest.Manifest; Runner exec.Runner; Brew brew.Client; Git repo.Git; DryRun, Yes bool; Now func() time.Time }`
  - `func New(ctx context.Context, o Options) (*App, error)`
  - `func (a *App) BackupDir() string`: `<Home>/.local/state/dot/backup/<Now().Format("20060102-150405")>`
  - `func (a *App) Status() status.Report`
  - `func (a *App) ManualStatus() []manual.Item`
  - `func (a *App) Link(out io.Writer) result.Summary`: Ok items are reported as Skipped "already linked"; BrokenManifest items as Failed
  - `func (a *App) TrustTaps(taps []string, out io.Writer) result.Summary`: Section `brew`, ID = tap; already trusted is Skipped "already trusted"; dryRun is Skipped "would trust"
  - `func (a *App) Install(out io.Writer) result.Summary`: taps from committed Brewfile, `brew bundle install` (dryRun: Skipped "nothing missing" or "would install N missing" using a dump diff), Link, `steps.Install`, `services.Start`, in that order
  - `func (a *App) Update(out io.Writer) result.Summary`: `brew update`, `brew upgrade` (dryRun: Skipped "would run ..."), then `steps.Update`
  - `func (a *App) UntrustedTaps() ([]string, error)`: tapped taps not starting with `homebrew/` whose `Trusted` is false
  - `type ExportPlan struct { Committed, Dump []brew.Entry; Diff brew.Diff }`; `func (a *App) ExportDiff() (*ExportPlan, error)`
  - `func (a *App) ExportWrite(p *ExportPlan, keepAdded, dropRemoved map[string]bool, out io.Writer) error`: `brew.Filter`, written to `Brewfile.tmp` then renamed over `Brewfile`; dryRun prints "would write N entries" only
  - `func (a *App) Commit(msg string, push bool, out io.Writer) result.Summary`: Section `git`; clean tree is a single Skipped "nothing to commit"; empty msg uses `repo.Message`; dryRun is Skipped "would commit: <msg>"; push is skipped with "no remote" when there is none

- [ ] **Step 1: Write the failing tests**

```go
// internal/app/app_test.go
package app

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ireydiak/dotfiles/internal/exec"
	"github.com/ireydiak/dotfiles/internal/result"
)

const manifestText = `
[links]
".zshrc" = "~/.zshrc"

[[steps]]
id = "omz"
check = "check-omz"
run = "run-omz"
update = "update-omz"

[[services]]
id = "yabai"
label = "com.koekeishiya.yabai"
start = "yabai --start-service"

[[manual]]
id = "accessibility"
how = "System Settings"
`

type harness struct {
	app  *App
	fake *exec.Fake
	root string
	home string
}

func newHarness(t *testing.T, brewfile string) harness {
	t.Helper()
	root, home := t.TempDir(), t.TempDir()
	os.MkdirAll(filepath.Join(root, "home"), 0o755)
	os.WriteFile(filepath.Join(root, "home", ".zshrc"), []byte("# zsh"), 0o644)
	os.WriteFile(filepath.Join(root, "manifest.toml"), []byte(manifestText), 0o644)
	if brewfile != "" {
		os.WriteFile(filepath.Join(root, "Brewfile"), []byte(brewfile), 0o644)
	}
	f := exec.NewFake()
	a, err := New(context.Background(), Options{
		RepoFlag: root, Home: home, Cwd: root, UID: "501", Runner: f,
		Now: func() time.Time { return time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC) },
	})
	if err != nil {
		t.Fatal(err)
	}
	a.Brew.TempDir = root
	return harness{app: a, fake: f, root: a.Root, home: home}
}

func (h harness) dump(t *testing.T, content string) {
	t.Helper()
	os.WriteFile(h.app.Brew.DumpPath(), []byte(content), 0o644)
}

func TestBackupDirUsesTimestamp(t *testing.T) {
	h := newHarness(t, "")
	want := filepath.Join(h.home, ".local/state/dot/backup/20260102-030405")
	if h.app.BackupDir() != want {
		t.Fatalf("BackupDir = %q", h.app.BackupDir())
	}
}

func TestInstallOrder(t *testing.T) {
	h := newHarness(t, "tap \"a/b\"\nbrew \"jq\"\n")
	h.fake.Scripts["brew tap-info --json a/b"] = exec.Result{Stdout: `[{"trusted":false}]`}
	h.fake.Scripts["check-omz"] = exec.Result{ExitCode: 1}
	h.fake.Scripts["launchctl print gui/501/com.koekeishiya.yabai"] = exec.Result{ExitCode: 113}
	var out bytes.Buffer
	s := h.app.Install(&out)
	calls := strings.Join(h.fake.Calls, "|")
	order := []string{"brew trust --tap a/b", "brew bundle install --no-upgrade", "run-omz", "yabai --start-service"}
	last := -1
	for _, c := range order {
		i := strings.Index(calls, c)
		if i < 0 || i < last {
			t.Fatalf("call %q missing or out of order in %s", c, calls)
		}
		last = i
	}
	if target, err := os.Readlink(filepath.Join(h.home, ".zshrc")); err != nil || target != filepath.Join(h.root, "home", ".zshrc") {
		t.Fatalf("link not created before steps: %v %q", err, target)
	}
	if !s.Failed() {
		t.Fatal("yabai recheck stays unloaded (exit 113), so install must report a failure")
	}
}

func TestInstallDryRunReportsNothingMissing(t *testing.T) {
	h := newHarness(t, "brew \"jq\"\n")
	h.app.DryRun = true
	h.dump(t, "brew \"jq\"\n")
	s := h.app.Install(&bytes.Buffer{})
	for _, e := range s {
		if e.Status == result.Ok || e.Status == result.Failed {
			t.Fatalf("dry run must only skip: %+v", e)
		}
	}
	var found bool
	for _, e := range s {
		found = found || e.Detail == "nothing missing"
	}
	if !found {
		t.Fatalf("expected 'nothing missing', got %+v", s)
	}
	if _, err := os.Lstat(filepath.Join(h.home, ".zshrc")); err == nil {
		t.Fatal("dry run must not link")
	}
}

func TestLinkReportsAlreadyLinked(t *testing.T) {
	h := newHarness(t, "")
	h.app.Link(&bytes.Buffer{})
	s := h.app.Link(&bytes.Buffer{})
	if len(s) != 1 || s[0].Status != result.Skipped || s[0].Detail != "already linked" {
		t.Fatalf("second link = %+v", s)
	}
}

func TestExportDiffWithoutBrewfileIsAllAdded(t *testing.T) {
	h := newHarness(t, "")
	h.dump(t, "tap \"a/b\"\nbrew \"jq\"\n")
	p, err := h.app.ExportDiff()
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Diff.Added) != 2 || len(p.Diff.Removed) != 0 || len(p.Committed) != 0 {
		t.Fatalf("plan = %+v", p.Diff)
	}
}

func TestExportWriteIsAtomicAndFiltered(t *testing.T) {
	h := newHarness(t, "brew \"jq\"\nbrew \"gone\"\n")
	h.dump(t, "brew \"jq\"\nbrew \"new\"\n")
	p, _ := h.app.ExportDiff()
	if err := h.app.ExportWrite(p, map[string]bool{"brew new": true}, map[string]bool{}, &bytes.Buffer{}); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(h.app.Brew.File)
	if string(got) != "brew \"jq\"\nbrew \"new\"\nbrew \"gone\"\n" {
		t.Fatalf("Brewfile =\n%s", got)
	}
	if _, err := os.Stat(h.app.Brew.File + ".tmp"); err == nil {
		t.Fatal("temp file must be renamed away")
	}
}

func TestUntrustedTapsSkipsHomebrew(t *testing.T) {
	h := newHarness(t, "")
	h.fake.Scripts["brew tap"] = exec.Result{Stdout: "homebrew/services\nfelixkratz/formulae\nasmvik/formulae\n"}
	h.fake.Scripts["brew tap-info --json felixkratz/formulae"] = exec.Result{Stdout: `[{"trusted":true}]`}
	h.fake.Scripts["brew tap-info --json asmvik/formulae"] = exec.Result{Stdout: `[{"trusted":false}]`}
	taps, err := h.app.UntrustedTaps()
	if err != nil || strings.Join(taps, ",") != "asmvik/formulae" {
		t.Fatalf("taps = %v %v", taps, err)
	}
}

func TestCommitCleanTreeIsSkipped(t *testing.T) {
	h := newHarness(t, "")
	s := h.app.Commit("", true, &bytes.Buffer{})
	if len(s) != 1 || s[0].Status != result.Skipped || s[0].Detail != "nothing to commit" {
		t.Fatalf("commit = %+v", s)
	}
	for _, c := range h.fake.Calls {
		if strings.Contains(c, "commit -m") {
			t.Fatal("must not commit a clean tree")
		}
	}
}

func TestCommitGeneratesMessageAndPushes(t *testing.T) {
	h := newHarness(t, "")
	h.fake.Scripts["git -C "+h.root+" status --porcelain"] = exec.Result{Stdout: " M home/.zshrc\n"}
	h.fake.Scripts["git -C "+h.root+" diff --numstat HEAD -- Brewfile"] = exec.Result{Stdout: ""}
	h.fake.Scripts["git -C "+h.root+" remote"] = exec.Result{Stdout: "origin\n"}
	s := h.app.Commit("", true, &bytes.Buffer{})
	if s.Failed() || len(s) != 2 || s[0].Detail != "dot: update .zshrc" || s[1].Detail != "pushed" {
		t.Fatalf("commit = %+v", s)
	}
	h.app.DryRun = true
	s = h.app.Commit("", true, &bytes.Buffer{})
	if s[0].Status != result.Skipped || !strings.HasPrefix(s[0].Detail, "would commit: ") {
		t.Fatalf("dry run commit = %+v", s)
	}
}

func TestUpdateRunsBrewThenSteps(t *testing.T) {
	h := newHarness(t, "")
	h.fake.Scripts["brew upgrade"] = exec.Result{ExitCode: 1, Stderr: "upgrade failed\n"}
	s := h.app.Update(&bytes.Buffer{})
	if strings.Join(h.fake.Calls, "|") != "brew update|brew upgrade|update-omz" {
		t.Fatalf("calls = %v", h.fake.Calls)
	}
	if !s.Failed() || s[2].Status != result.Ok {
		t.Fatalf("a failing brew upgrade must not stop step updates: %+v", s)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/app/`
Expected: FAIL, `undefined: New`

- [ ] **Step 3: Implement**

```go
// internal/app/app.go
package app

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/ireydiak/dotfiles/internal/brew"
	"github.com/ireydiak/dotfiles/internal/exec"
	"github.com/ireydiak/dotfiles/internal/link"
	"github.com/ireydiak/dotfiles/internal/manifest"
	"github.com/ireydiak/dotfiles/internal/manual"
	"github.com/ireydiak/dotfiles/internal/repo"
	"github.com/ireydiak/dotfiles/internal/result"
	"github.com/ireydiak/dotfiles/internal/services"
	"github.com/ireydiak/dotfiles/internal/status"
	"github.com/ireydiak/dotfiles/internal/steps"
)

type Options struct {
	RepoFlag string
	Env      string
	Home     string
	Cwd      string
	UID      string
	DryRun   bool
	Yes      bool
	Runner   exec.Runner
	Now      func() time.Time
}

type App struct {
	Ctx      context.Context
	Root     string
	Home     string
	UID      string
	Manifest *manifest.Manifest
	Runner   exec.Runner
	Brew     brew.Client
	Git      repo.Git
	DryRun   bool
	Yes      bool
	Now      func() time.Time
}

func New(ctx context.Context, o Options) (*App, error) {
	var err error
	home := o.Home
	if home == "" {
		if home, err = os.UserHomeDir(); err != nil {
			return nil, err
		}
	}
	cwd := o.Cwd
	if cwd == "" {
		if cwd, err = os.Getwd(); err != nil {
			return nil, err
		}
	}
	uid := o.UID
	if uid == "" {
		uid = strconv.Itoa(os.Getuid())
	}
	runner := o.Runner
	if runner == nil {
		runner = exec.NewShell(home)
	}
	now := o.Now
	if now == nil {
		now = time.Now
	}
	root, err := repo.Resolve(repo.ResolveOptions{
		Flag: o.RepoFlag, Env: o.Env, Cwd: cwd, Home: home, GitTop: repo.GitTop(ctx, runner),
	})
	if err != nil {
		return nil, err
	}
	m, err := manifest.Load(root)
	if err != nil {
		return nil, err
	}
	return &App{
		Ctx: ctx, Root: root, Home: home, UID: uid, Manifest: m, Runner: runner,
		Brew:   brew.Client{R: runner, File: filepath.Join(root, "Brewfile"), TempDir: os.TempDir()},
		Git:    repo.Git{Root: root, R: runner},
		DryRun: o.DryRun, Yes: o.Yes, Now: now,
	}, nil
}

func (a *App) BackupDir() string {
	return filepath.Join(a.Home, ".local", "state", "dot", "backup", a.Now().Format("20060102-150405"))
}

func (a *App) Status() status.Report {
	return status.Collect(status.Deps{
		Ctx: a.Ctx, Manifest: a.Manifest, RepoRoot: a.Root, Home: a.Home, UID: a.UID,
		Runner: a.Runner, Brew: a.Brew, Git: a.Git,
	})
}

func (a *App) ManualStatus() []manual.Item {
	return manual.Status(a.Ctx, a.Runner, a.Manifest.Manual)
}

func (a *App) Link(out io.Writer) result.Summary {
	items := link.Inspect(a.Manifest.Links, a.Root, a.Home)
	s := link.Apply(link.Plan(items, a.BackupDir(), a.Home), a.DryRun, out)
	for _, it := range items {
		switch it.State {
		case link.Ok:
			s = append(s, result.Entry{Section: "link", ID: it.Key, Status: result.Skipped, Detail: "already linked"})
		case link.BrokenManifest:
			s = append(s, result.Entry{Section: "link", ID: it.Key, Status: result.Failed, Detail: it.Detail})
		}
	}
	return s
}

func (a *App) TrustTaps(taps []string, out io.Writer) result.Summary {
	var s result.Summary
	for _, tap := range taps {
		e := result.Entry{Section: "brew", ID: tap}
		trusted, err := a.Brew.Trusted(a.Ctx, tap)
		switch {
		case err != nil:
			e.Status, e.Detail = result.Failed, err.Error()
		case trusted:
			e.Status, e.Detail = result.Skipped, "already trusted"
		case a.DryRun:
			e.Status, e.Detail = result.Skipped, "would trust"
		default:
			if err := a.Brew.Trust(a.Ctx, tap, out); err != nil {
				e.Status, e.Detail = result.Failed, err.Error()
			} else {
				e.Status, e.Detail = result.Ok, "trusted"
			}
		}
		s = append(s, e)
	}
	return s
}

func (a *App) Install(out io.Writer) result.Summary {
	var s result.Summary
	committed, err := a.Brew.Committed()
	if err != nil {
		s = append(s, result.Entry{Section: "brew", ID: "Brewfile", Status: result.Failed, Detail: err.Error()})
	} else {
		s = append(s, a.TrustTaps(brew.Taps(committed), out)...)
		s = append(s, a.brewInstall(committed, out))
	}
	s = append(s, a.Link(out)...)
	s = append(s, steps.Install(a.Ctx, a.Runner, a.Manifest.Steps, a.DryRun, out)...)
	s = append(s, services.Start(a.Ctx, a.Runner, a.Manifest.Services, a.UID, a.DryRun, out)...)
	return s
}

func (a *App) brewInstall(committed []brew.Entry, out io.Writer) result.Entry {
	e := result.Entry{Section: "brew", ID: "bundle install"}
	if a.DryRun {
		dump, err := a.Brew.Dump(a.Ctx)
		if err != nil {
			e.Status, e.Detail = result.Failed, err.Error()
			return e
		}
		if missing := len(brew.Compare(committed, dump).Removed); missing == 0 {
			e.Status, e.Detail = result.Skipped, "nothing missing"
		} else {
			e.Status, e.Detail = result.Skipped, fmt.Sprintf("would install %d missing", missing)
		}
		return e
	}
	if err := a.Brew.Install(a.Ctx, out); err != nil {
		e.Status, e.Detail = result.Failed, err.Error()
		return e
	}
	e.Status, e.Detail = result.Ok, "brew bundle install completed"
	return e
}

func (a *App) Update(out io.Writer) result.Summary {
	var s result.Summary
	brewStep := func(id string, fn func(context.Context, io.Writer) error) {
		e := result.Entry{Section: "brew", ID: id}
		switch {
		case a.DryRun:
			e.Status, e.Detail = result.Skipped, "would run brew "+id
		default:
			if err := fn(a.Ctx, out); err != nil {
				e.Status, e.Detail = result.Failed, err.Error()
			} else {
				e.Status, e.Detail = result.Ok, "brew "+id+" completed"
			}
		}
		s = append(s, e)
	}
	brewStep("update", a.Brew.Update)
	brewStep("upgrade", a.Brew.Upgrade)
	s = append(s, steps.Update(a.Ctx, a.Runner, a.Manifest.Steps, a.DryRun, out)...)
	return s
}

func (a *App) UntrustedTaps() ([]string, error) {
	taps, err := a.Brew.Tapped(a.Ctx)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, tap := range taps {
		if strings.HasPrefix(tap, "homebrew/") {
			continue
		}
		trusted, err := a.Brew.Trusted(a.Ctx, tap)
		if err != nil {
			return nil, err
		}
		if !trusted {
			out = append(out, tap)
		}
	}
	return out, nil
}

type ExportPlan struct {
	Committed []brew.Entry
	Dump      []brew.Entry
	Diff      brew.Diff
}

func (a *App) ExportDiff() (*ExportPlan, error) {
	committed, err := a.Brew.Committed()
	if err != nil {
		return nil, err
	}
	dump, err := a.Brew.Dump(a.Ctx)
	if err != nil {
		return nil, err
	}
	return &ExportPlan{Committed: committed, Dump: dump, Diff: brew.Compare(committed, dump)}, nil
}

func (a *App) ExportWrite(p *ExportPlan, keepAdded, dropRemoved map[string]bool, out io.Writer) error {
	entries := brew.Filter(p.Dump, p.Committed, keepAdded, dropRemoved)
	if a.DryRun {
		fmt.Fprintf(out, "would write %d entries to %s\n", len(entries), a.Brew.File)
		return nil
	}
	tmp := a.Brew.File + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	if err := brew.Write(f, entries); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, a.Brew.File)
}

func (a *App) Commit(msg string, push bool, out io.Writer) result.Summary {
	entry := func(id string, st result.Status, detail string) result.Entry {
		return result.Entry{Section: "git", ID: id, Status: st, Detail: detail}
	}
	changes, err := a.Git.Changes(a.Ctx)
	if err != nil {
		return result.Summary{entry("commit", result.Failed, err.Error())}
	}
	if len(changes) == 0 {
		return result.Summary{entry("commit", result.Skipped, "nothing to commit")}
	}
	if msg == "" {
		added, removed, _ := a.Git.BrewfileNumstat(a.Ctx)
		msg = repo.Message(changes, added, removed)
	}
	if a.DryRun {
		return result.Summary{entry("commit", result.Skipped, "would commit: "+msg)}
	}
	if err := a.Git.CommitAll(a.Ctx, msg, out); err != nil {
		return result.Summary{entry("commit", result.Failed, err.Error())}
	}
	s := result.Summary{entry("commit", result.Ok, msg)}
	if !push {
		return s
	}
	has, err := a.Git.HasRemote(a.Ctx)
	switch {
	case err != nil:
		s = append(s, entry("push", result.Failed, err.Error()))
	case !has:
		s = append(s, entry("push", result.Skipped, "no remote"))
	default:
		if err := a.Git.Push(a.Ctx, out); err != nil {
			s = append(s, entry("push", result.Failed, err.Error()))
		} else {
			s = append(s, entry("push", result.Ok, "pushed"))
		}
	}
	return s
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/app/`
Expected: `ok`

- [ ] **Step 5: Commit**

```bash
git add internal/app
git commit -m "feat(app): wire engines into install, link, export, update, commit

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

---

### Task 13: Command line (`cmd/dot`)

**Files:**
- Modify: `cmd/dot/main.go` (replace the Task 1 stub entirely)
- Test: `cmd/dot/main_test.go`

**Interfaces:**
- Consumes: `app.New`, `app.App` methods, `tui.Run` (Task 14; until then `main.go` calls a placeholder described in Step 3)
- Produces:
  - `type newAppFunc func(ctx context.Context, o app.Options) (*app.App, error)`
  - `func run(args []string, stdin io.Reader, stdout, stderr io.Writer, newApp newAppFunc) int`
  - `main()` calls `run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr, app.New)` and exits with the result
- Flags parsed after the command: `--repo`, `--dry-run`, `--yes`, `--json` (status), `-m` and `--no-push` (commit). Exit 2 for unknown commands, flag errors and app construction errors; 1 when a summary has a failure; 0 otherwise.
- `export` flow: `UntrustedTaps`; when non-empty print them and, unless `--yes`, ask `Trust them so brew bundle can see their packages? [y/N]`; `TrustTaps`; `ExportDiff`; print `+ line` for added and `- line (no longer installed; kept)` for removed; when both empty print `Brewfile is up to date`; unless `--yes` ask `Write Brewfile with N additions? [y/N]`; `ExportWrite` keeping every added entry and dropping nothing.
- `install` prints the summary, then `Pending manual steps:` with `  <id>: <how>` lines from `ManualStatus`, when any.

- [ ] **Step 1: Write the failing tests**

```go
// cmd/dot/main_test.go
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ireydiak/dotfiles/internal/app"
	"github.com/ireydiak/dotfiles/internal/exec"
)

const manifestText = `
[links]
".zshrc" = "~/.zshrc"

[[steps]]
id = "omz"
check = "check-omz"
run = "run-omz"

[[manual]]
id = "accessibility"
how = "System Settings"
`

func testApp(t *testing.T, f *exec.Fake) (newAppFunc, string) {
	t.Helper()
	root, home := t.TempDir(), t.TempDir()
	os.MkdirAll(filepath.Join(root, "home"), 0o755)
	os.WriteFile(filepath.Join(root, "home", ".zshrc"), []byte("x"), 0o644)
	os.WriteFile(filepath.Join(root, "manifest.toml"), []byte(manifestText), 0o644)
	os.WriteFile(filepath.Join(root, "Brewfile"), []byte("brew \"jq\"\n"), 0o644)
	return func(ctx context.Context, o app.Options) (*app.App, error) {
		o.RepoFlag, o.Home, o.Cwd, o.UID, o.Runner = root, home, root, "501", f
		a, err := app.New(ctx, o)
		if err == nil {
			a.Brew.TempDir = root
		}
		return a, err
	}, root
}

func TestUnknownCommandExits2(t *testing.T) {
	var out, errOut bytes.Buffer
	code := run([]string{"frobnicate"}, strings.NewReader(""), &out, &errOut, nil)
	if code != 2 || !strings.Contains(errOut.String(), "unknown command") {
		t.Fatalf("code=%d stderr=%s", code, errOut.String())
	}
}

func TestStatusJSON(t *testing.T) {
	f := exec.NewFake()
	newApp, root := testApp(t, f)
	os.WriteFile(filepath.Join(root, "Brewfile.dump"), []byte("brew \"jq\"\n"), 0o644)
	var out, errOut bytes.Buffer
	code := run([]string{"status", "--json"}, strings.NewReader(""), &out, &errOut, newApp)
	if code != 0 {
		t.Fatalf("code=%d stderr=%s", code, errOut.String())
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(out.Bytes(), &m); err != nil || m["links"] == nil {
		t.Fatalf("invalid status JSON: %v\n%s", err, out.String())
	}
}

func TestInstallFailureExits1AndListsManual(t *testing.T) {
	f := exec.NewFake()
	f.Scripts["check-omz"] = exec.Result{ExitCode: 1}
	f.Scripts["run-omz"] = exec.Result{ExitCode: 1, Stderr: "boom\n"}
	newApp, _ := testApp(t, f)
	var out, errOut bytes.Buffer
	code := run([]string{"install"}, strings.NewReader(""), &out, &errOut, newApp)
	if code != 1 {
		t.Fatalf("code=%d\n%s%s", code, out.String(), errOut.String())
	}
	if !strings.Contains(out.String(), "Pending manual steps:") || !strings.Contains(out.String(), "accessibility: System Settings") {
		t.Fatalf("manual steps not listed:\n%s", out.String())
	}
}

func TestExportPromptsAndWrites(t *testing.T) {
	f := exec.NewFake()
	newApp, root := testApp(t, f)
	f.Scripts["brew tap"] = exec.Result{Stdout: "asmvik/formulae\n"}
	f.Scripts["brew tap-info --json asmvik/formulae"] = exec.Result{Stdout: `[{"trusted":false}]`}
	os.WriteFile(filepath.Join(root, "Brewfile.dump"), []byte("tap \"asmvik/formulae\"\nbrew \"jq\"\nbrew \"yabai\"\n"), 0o644)
	var out, errOut bytes.Buffer
	code := run([]string{"export"}, strings.NewReader("y\ny\n"), &out, &errOut, newApp)
	if code != 0 {
		t.Fatalf("code=%d\n%s%s", code, out.String(), errOut.String())
	}
	if !strings.Contains(strings.Join(f.Calls, "|"), "brew trust --tap asmvik/formulae") {
		t.Fatalf("tap not trusted: %v", f.Calls)
	}
	got, _ := os.ReadFile(filepath.Join(root, "Brewfile"))
	if string(got) != "tap \"asmvik/formulae\"\nbrew \"jq\"\nbrew \"yabai\"\n" {
		t.Fatalf("Brewfile =\n%s", got)
	}
}

func TestExportDeclinedWritesNothing(t *testing.T) {
	f := exec.NewFake()
	newApp, root := testApp(t, f)
	f.Scripts["brew tap"] = exec.Result{Stdout: ""}
	os.WriteFile(filepath.Join(root, "Brewfile.dump"), []byte("brew \"jq\"\nbrew \"new\"\n"), 0o644)
	var out bytes.Buffer
	code := run([]string{"export"}, strings.NewReader("n\n"), &out, &bytes.Buffer{}, newApp)
	if code != 0 || !strings.Contains(out.String(), "cancelled") {
		t.Fatalf("code=%d out=%s", code, out.String())
	}
	got, _ := os.ReadFile(filepath.Join(root, "Brewfile"))
	if string(got) != "brew \"jq\"\n" {
		t.Fatalf("Brewfile must be unchanged, got %q", got)
	}
}

func TestCommitFlags(t *testing.T) {
	f := exec.NewFake()
	newApp, root := testApp(t, f)
	f.Scripts["git -C "+root+" status --porcelain"] = exec.Result{Stdout: " M home/.zshrc\n"}
	var out bytes.Buffer
	code := run([]string{"commit", "-m", "hello", "--no-push"}, strings.NewReader(""), &out, &bytes.Buffer{}, newApp)
	if code != 0 {
		t.Fatalf("code=%d out=%s", code, out.String())
	}
	calls := strings.Join(f.Calls, "|")
	if !strings.Contains(calls, "commit -m 'hello'") || strings.Contains(calls, "git -C "+root+" push") {
		t.Fatalf("calls = %v", f.Calls)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./cmd/dot/`
Expected: FAIL, `undefined: run`, `undefined: newAppFunc`

- [ ] **Step 3: Implement**

Until Task 14 exists, define the dashboard call as a local placeholder so the package compiles:

```go
// cmd/dot/tui_placeholder.go  (DELETE in Task 14)
package main

import (
	"errors"

	"github.com/ireydiak/dotfiles/internal/app"
)

func runTUI(_ *app.App) error { return errors.New("dashboard not built yet; use a subcommand") }
```

```go
// cmd/dot/main.go
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ireydiak/dotfiles/internal/app"
	"github.com/ireydiak/dotfiles/internal/manual"
	"github.com/ireydiak/dotfiles/internal/result"
)

const usage = `usage: dot <command> [flags]

commands:
  status   report packages, links, services, manual steps and repo state
  install  trust taps, brew bundle install, link, run steps, start services
  link     create or repair symlinks from home/ into $HOME
  export   dump installed packages and update the Brewfile
  update   brew update, brew upgrade, then step updates
  commit   git add -A, commit and push
  (none)   open the dashboard

flags:
  --repo <path>   dotfiles repo (default: $DOTFILES_DIR, the current git repo, or ~/.dotfiles)
  --dry-run       print the plan without changing anything
  --yes           accept prompts (export)
  --json          machine-readable output (status)
  -m <msg>        commit message (commit)
  --no-push       do not push after committing (commit)
`

type newAppFunc func(ctx context.Context, o app.Options) (*app.App, error)

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr, app.New))
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer, newApp newAppFunc) int {
	cmd := ""
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		cmd, args = args[0], args[1:]
	}
	switch cmd {
	case "", "status", "install", "link", "export", "update", "commit":
	case "help", "-h", "--help":
		fmt.Fprint(stdout, usage)
		return 0
	default:
		fmt.Fprintf(stderr, "unknown command %q\n\n%s", cmd, usage)
		return 2
	}

	fs := flag.NewFlagSet("dot", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() { fmt.Fprint(stderr, usage) }
	repoFlag := fs.String("repo", "", "")
	dryRun := fs.Bool("dry-run", false, "")
	yes := fs.Bool("yes", false, "")
	asJSON := fs.Bool("json", false, "")
	msg := fs.String("m", "", "")
	noPush := fs.Bool("no-push", false, "")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	a, err := newApp(context.Background(), app.Options{
		RepoFlag: *repoFlag, Env: os.Getenv("DOTFILES_DIR"), DryRun: *dryRun, Yes: *yes,
	})
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 2
	}

	switch cmd {
	case "":
		if err := runTUI(a); err != nil {
			fmt.Fprintln(stderr, "error:", err)
			return 1
		}
		return 0
	case "status":
		r := a.Status()
		if *asJSON {
			if err := r.JSON(stdout); err != nil {
				fmt.Fprintln(stderr, "error:", err)
				return 1
			}
			return 0
		}
		r.Print(stdout)
		return 0
	case "install":
		s := a.Install(stdout)
		s.Print(stdout)
		printPending(stdout, a.ManualStatus())
		return exitFor(s)
	case "link":
		return finish(stdout, a.Link(stdout))
	case "update":
		return finish(stdout, a.Update(stdout))
	case "commit":
		return finish(stdout, a.Commit(*msg, !*noPush, stdout))
	case "export":
		return runExport(a, stdin, stdout, stderr, *yes)
	}
	return 0
}

func finish(w io.Writer, s result.Summary) int {
	s.Print(w)
	return exitFor(s)
}

func exitFor(s result.Summary) int {
	if s.Failed() {
		return 1
	}
	return 0
}

func printPending(w io.Writer, items []manual.Item) {
	pending := manual.Pending(items)
	if len(pending) == 0 {
		return
	}
	fmt.Fprintln(w, "\nPending manual steps:")
	for _, it := range pending {
		fmt.Fprintf(w, "  %s: %s\n", it.ID, it.How)
	}
}

func confirm(stdin io.Reader, out io.Writer, question string) bool {
	fmt.Fprintf(out, "%s [y/N] ", question)
	sc := bufio.NewScanner(stdin)
	if !sc.Scan() {
		fmt.Fprintln(out)
		return false
	}
	ans := strings.ToLower(strings.TrimSpace(sc.Text()))
	return ans == "y" || ans == "yes"
}

func runExport(a *app.App, stdin io.Reader, stdout, stderr io.Writer, yes bool) int {
	taps, err := a.UntrustedTaps()
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}
	if len(taps) > 0 {
		fmt.Fprintf(stdout, "Untrusted taps: %s\n", strings.Join(taps, ", "))
		if !yes && !confirm(stdin, stdout, "Trust them so brew bundle can see their packages?") {
			fmt.Fprintln(stdout, "export cancelled")
			return 0
		}
		s := a.TrustTaps(taps, stdout)
		s.Print(stdout)
		if s.Failed() {
			return 1
		}
	}
	plan, err := a.ExportDiff()
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}
	for _, e := range plan.Diff.Added {
		fmt.Fprintf(stdout, "  + %s\n", e.Line())
	}
	for _, e := range plan.Diff.Removed {
		fmt.Fprintf(stdout, "  - %s (no longer installed; kept)\n", e.Line())
	}
	if len(plan.Diff.Added) == 0 && len(plan.Diff.Removed) == 0 {
		fmt.Fprintln(stdout, "Brewfile is up to date")
		return 0
	}
	if !yes && !confirm(stdin, stdout, fmt.Sprintf("Write Brewfile with %d additions?", len(plan.Diff.Added))) {
		fmt.Fprintln(stdout, "export cancelled")
		return 0
	}
	keep := map[string]bool{}
	for _, e := range plan.Diff.Added {
		keep[e.Key()] = true
	}
	if err := a.ExportWrite(plan, keep, map[string]bool{}, stdout); err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}
	fmt.Fprintln(stdout, "Brewfile written")
	return 0
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./cmd/dot/ && go vet ./...`
Expected: `ok`, no vet output

- [ ] **Step 5: Commit**

```bash
git add cmd/dot
git commit -m "feat(cli): subcommands, flags, export prompts and exit codes

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

---

### Task 14: Dashboard TUI (`internal/tui`)

**Files:**
- Create: `internal/tui/picker.go`, `internal/tui/tui.go`
- Test: `internal/tui/picker_test.go`, `internal/tui/tui_test.go`
- Modify: `cmd/dot/main.go` (call `tui.Run`), delete `cmd/dot/tui_placeholder.go`
- Modify: `go.mod` (adds bubbletea, bubbles, lipgloss)

**Interfaces:**
- Consumes: `app.App` methods, `brew.Diff`, `brew.Entry`, `result.Summary`, `status.Report`
- Produces in `picker.go`:
  - `type PickerItem struct { Entry brew.Entry; Group string; Checked bool }` (Group is `Added` or `Removed`)
  - `type Picker struct { Items []PickerItem; Cursor int }`
  - `func NewPicker(d brew.Diff) Picker`: Added items first and checked, Removed items after and unchecked
  - methods `Up()`, `Down()`, `Toggle()`, `ToggleGroup()` (flips every item in the cursor item's group to the opposite of the cursor item's current state), `Selections() (keepAdded, dropRemoved map[string]bool)`, `View(width int) string`
- Produces in `tui.go`:
  - `type Model struct` implementing `tea.Model`, with unexported fields; `func New(a *app.App) Model`; `func Run(a *app.App) error`
  - message types `reportMsg status.Report`, `outputMsg string`, `actionDone struct { Name string; Summary result.Summary }`, `tapsMsg struct { Taps []string; Err error }`, `exportDiffMsg struct { Plan *app.ExportPlan; Err error }`
  - screens `screenDashboard`, `screenConfirmTrust`, `screenPicker`
  - Dashboard keys: `i` install, `l` link, `e` export, `u` update, `c` commit (push on), `r` refresh, `q`/`ctrl+c` quit. Keys other than `q` are ignored while an action runs.
  - Confirm screen keys: `y` trusts then diffs, `n`/`esc` back to dashboard.
  - Picker keys: `up`/`k`, `down`/`j`, `space` toggle, `a` toggle group, `enter` write and return, `esc` cancel.
- Streaming: actions run in a goroutine writing lines to a channel; each line arrives as `outputMsg` and is appended to a `viewport`; `actionDone` clears the running flag and triggers a refresh.

- [ ] **Step 1: Add dependencies**

Run: `go get github.com/charmbracelet/bubbletea@latest github.com/charmbracelet/bubbles@latest github.com/charmbracelet/lipgloss@latest`

- [ ] **Step 2: Write the failing picker tests**

```go
// internal/tui/picker_test.go
package tui

import (
	"strings"
	"testing"

	"github.com/ireydiak/dotfiles/internal/brew"
)

func diff() brew.Diff {
	return brew.Diff{
		Added:   []brew.Entry{{Kind: "brew", Name: "new1"}, {Kind: "cask", Name: "new2"}},
		Removed: []brew.Entry{{Kind: "brew", Name: "gone"}},
	}
}

func TestNewPickerDefaults(t *testing.T) {
	p := NewPicker(diff())
	if len(p.Items) != 3 || !p.Items[0].Checked || !p.Items[1].Checked || p.Items[2].Checked {
		t.Fatalf("items = %+v", p.Items)
	}
	if p.Items[0].Group != "Added" || p.Items[2].Group != "Removed" {
		t.Fatalf("groups = %+v", p.Items)
	}
	keep, drop := p.Selections()
	if len(keep) != 2 || len(drop) != 0 {
		t.Fatalf("selections = %v %v", keep, drop)
	}
}

func TestPickerNavigationAndToggle(t *testing.T) {
	p := NewPicker(diff())
	p = p.Up() // clamps at 0
	if p.Cursor != 0 {
		t.Fatal("cursor should clamp at 0")
	}
	p = p.Down().Down().Down() // clamps at last
	if p.Cursor != 2 {
		t.Fatalf("cursor = %d", p.Cursor)
	}
	p = p.Toggle()
	keep, drop := p.Selections()
	if !drop["brew gone"] || len(keep) != 2 {
		t.Fatalf("toggle on removed item: %v %v", keep, drop)
	}
	p = p.Up().Up().ToggleGroup() // cursor on new1 (checked) -> group becomes unchecked
	keep, _ = p.Selections()
	if len(keep) != 0 {
		t.Fatalf("ToggleGroup should uncheck all added: %v", keep)
	}
	p = p.ToggleGroup()
	keep, _ = p.Selections()
	if len(keep) != 2 {
		t.Fatalf("ToggleGroup again should re-check all added: %v", keep)
	}
}

func TestPickerView(t *testing.T) {
	v := NewPicker(diff()).View(80)
	for _, want := range []string{"Added", "Removed", "[x]", "[ ]", `brew "new1"`, `brew "gone"`} {
		if !strings.Contains(v, want) {
			t.Fatalf("view missing %q:\n%s", want, v)
		}
	}
}
```

- [ ] **Step 3: Run picker tests to verify they fail**

Run: `go test ./internal/tui/ -run Picker`
Expected: FAIL, `undefined: NewPicker`

- [ ] **Step 4: Implement the picker**

```go
// internal/tui/picker.go
package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/ireydiak/dotfiles/internal/brew"
)

type PickerItem struct {
	Entry   brew.Entry
	Group   string
	Checked bool
}

type Picker struct {
	Items  []PickerItem
	Cursor int
}

// NewPicker lists added entries first (checked) then removed ones (unchecked).
func NewPicker(d brew.Diff) Picker {
	var p Picker
	for _, e := range d.Added {
		p.Items = append(p.Items, PickerItem{Entry: e, Group: "Added", Checked: true})
	}
	for _, e := range d.Removed {
		p.Items = append(p.Items, PickerItem{Entry: e, Group: "Removed"})
	}
	return p
}

func (p Picker) Up() Picker {
	if p.Cursor > 0 {
		p.Cursor--
	}
	return p
}

func (p Picker) Down() Picker {
	if p.Cursor < len(p.Items)-1 {
		p.Cursor++
	}
	return p
}

func (p Picker) Toggle() Picker {
	if len(p.Items) == 0 {
		return p
	}
	items := append([]PickerItem(nil), p.Items...)
	items[p.Cursor].Checked = !items[p.Cursor].Checked
	p.Items = items
	return p
}

// ToggleGroup sets every item in the cursor's group to the opposite of the cursor item.
func (p Picker) ToggleGroup() Picker {
	if len(p.Items) == 0 {
		return p
	}
	items := append([]PickerItem(nil), p.Items...)
	group, target := items[p.Cursor].Group, !items[p.Cursor].Checked
	for i := range items {
		if items[i].Group == group {
			items[i].Checked = target
		}
	}
	p.Items = items
	return p
}

// Selections returns the chosen additions and removals keyed by Entry.Key().
func (p Picker) Selections() (keepAdded, dropRemoved map[string]bool) {
	keepAdded, dropRemoved = map[string]bool{}, map[string]bool{}
	for _, it := range p.Items {
		if !it.Checked {
			continue
		}
		if it.Group == "Added" {
			keepAdded[it.Entry.Key()] = true
		} else {
			dropRemoved[it.Entry.Key()] = true
		}
	}
	return keepAdded, dropRemoved
}

var (
	groupStyle  = lipgloss.NewStyle().Bold(true).Underline(true)
	cursorStyle = lipgloss.NewStyle().Reverse(true)
	dimStyle    = lipgloss.NewStyle().Faint(true)
)

func (p Picker) View(width int) string {
	var b strings.Builder
	b.WriteString("Export: choose what to record in the Brewfile\n\n")
	lastGroup := ""
	for i, it := range p.Items {
		if it.Group != lastGroup {
			hint := "installed, not in Brewfile"
			if it.Group == "Removed" {
				hint = "in Brewfile, no longer installed; check to drop"
			}
			b.WriteString(groupStyle.Render(it.Group) + dimStyle.Render("  "+hint) + "\n")
			lastGroup = it.Group
		}
		box := "[ ]"
		if it.Checked {
			box = "[x]"
		}
		line := "  " + box + " " + it.Entry.Line()
		if i == p.Cursor {
			line = cursorStyle.Render(line)
		}
		b.WriteString(line + "\n")
	}
	if len(p.Items) == 0 {
		b.WriteString(dimStyle.Render("Brewfile is up to date") + "\n")
	}
	b.WriteString("\n" + dimStyle.Render("space toggle · a toggle group · enter write · esc cancel"))
	return lipgloss.NewStyle().Width(width).Render(b.String())
}
```

- [ ] **Step 5: Run picker tests to verify they pass**

Run: `go test ./internal/tui/ -run Picker`
Expected: `ok`

- [ ] **Step 6: Write the failing model tests**

```go
// internal/tui/tui_test.go
package tui

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ireydiak/dotfiles/internal/app"
	"github.com/ireydiak/dotfiles/internal/brew"
	"github.com/ireydiak/dotfiles/internal/exec"
	"github.com/ireydiak/dotfiles/internal/result"
	"github.com/ireydiak/dotfiles/internal/status"
)

func testModel(t *testing.T) (Model, *exec.Fake) {
	t.Helper()
	root, home := t.TempDir(), t.TempDir()
	os.MkdirAll(filepath.Join(root, "home"), 0o755)
	os.WriteFile(filepath.Join(root, "home", ".zshrc"), []byte("x"), 0o644)
	os.WriteFile(filepath.Join(root, "manifest.toml"), []byte("[links]\n\".zshrc\" = \"~/.zshrc\"\n"), 0o644)
	f := exec.NewFake()
	a, err := app.New(context.Background(), app.Options{RepoFlag: root, Home: home, Cwd: root, UID: "501", Runner: f})
	if err != nil {
		t.Fatal(err)
	}
	a.Brew.TempDir = root
	m := New(a)
	m, _ = m.resize(100, 40)
	return m, f
}

func key(s string) tea.KeyMsg {
	if s == " " {
		return tea.KeyMsg{Type: tea.KeySpace}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func TestQuitKey(t *testing.T) {
	m, _ := testModel(t)
	_, cmd := m.Update(key("q"))
	if cmd == nil {
		t.Fatal("q should produce a quit command")
	}
	if msg := cmd(); msg == nil {
		t.Fatal("quit command should produce a message")
	} else if _, ok := msg.(tea.QuitMsg); !ok {
		t.Fatalf("expected tea.QuitMsg, got %T", msg)
	}
}

func TestActionKeysSetRunningAndIgnoreOthersWhileRunning(t *testing.T) {
	m, _ := testModel(t)
	for _, k := range []string{"i", "l", "u", "c"} {
		next, cmd := m.Update(key(k))
		nm := next.(Model)
		if !nm.running || cmd == nil {
			t.Fatalf("key %q should start an action", k)
		}
		again, cmd2 := nm.Update(key("l"))
		if cmd2 != nil || again.(Model).action != nm.action {
			t.Fatalf("keys must be ignored while %q runs", nm.action)
		}
	}
}

func TestOutputAndDoneMessages(t *testing.T) {
	m, _ := testModel(t)
	next, _ := m.Update(key("l"))
	nm := next.(Model)
	next, cmd := nm.Update(outputMsg("linked .zshrc"))
	nm = next.(Model)
	if len(nm.lines) != 1 || nm.lines[0] != "linked .zshrc" || cmd == nil {
		t.Fatalf("output not appended or no follow-up wait: %+v", nm.lines)
	}
	next, cmd = nm.Update(actionDone{Name: "link", Summary: result.Summary{{Section: "link", ID: ".zshrc", Status: result.Ok, Detail: "linked"}}})
	nm = next.(Model)
	if nm.running || cmd == nil {
		t.Fatal("done should clear running and schedule a refresh")
	}
	if !strings.Contains(strings.Join(nm.lines, "\n"), "1 ok, 0 skipped, 0 failed") {
		t.Fatalf("summary not appended: %v", nm.lines)
	}
}

func TestExportFlowScreens(t *testing.T) {
	m, _ := testModel(t)
	next, _ := m.Update(tapsMsg{Taps: []string{"asmvik/formulae"}})
	nm := next.(Model)
	if nm.screen != screenConfirmTrust {
		t.Fatalf("screen = %v, want confirm", nm.screen)
	}
	next, _ = nm.Update(key("n"))
	if next.(Model).screen != screenDashboard {
		t.Fatal("n should return to the dashboard")
	}
	next, _ = m.Update(tapsMsg{})
	if _, cmd := next.(Model).Update(tapsMsg{}); cmd == nil {
		t.Fatal("no untrusted taps should go straight to the diff command")
	}
	plan := &app.ExportPlan{Diff: brew.Diff{Added: []brew.Entry{{Kind: "brew", Name: "new"}}}}
	next, _ = m.Update(exportDiffMsg{Plan: plan})
	nm = next.(Model)
	if nm.screen != screenPicker || len(nm.picker.Items) != 1 {
		t.Fatalf("screen = %v items=%d", nm.screen, len(nm.picker.Items))
	}
	next, _ = nm.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if next.(Model).screen != screenDashboard {
		t.Fatal("esc should cancel the picker")
	}
	next, cmd := nm.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if next.(Model).screen != screenDashboard || !next.(Model).running || cmd == nil {
		t.Fatal("enter should start the export action")
	}
}

func TestViewShowsSections(t *testing.T) {
	m, _ := testModel(t)
	next, _ := m.Update(reportMsg(status.Report{}))
	v := next.(Model).View()
	for _, want := range []string{"Packages", "Links", "Services", "Manual", "i install", "u update", "q quit"} {
		if !strings.Contains(v, want) {
			t.Fatalf("view missing %q:\n%s", want, v)
		}
	}
}
```

- [ ] **Step 7: Run model tests to verify they fail**

Run: `go test ./internal/tui/`
Expected: FAIL, `undefined: New`, `undefined: outputMsg`

- [ ] **Step 8: Implement the model**

```go
// internal/tui/tui.go
package tui

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ireydiak/dotfiles/internal/app"
	"github.com/ireydiak/dotfiles/internal/link"
	"github.com/ireydiak/dotfiles/internal/manual"
	"github.com/ireydiak/dotfiles/internal/result"
	"github.com/ireydiak/dotfiles/internal/status"
)

type screen int

const (
	screenDashboard screen = iota
	screenConfirmTrust
	screenPicker
)

type (
	reportMsg     status.Report
	outputMsg     string
	actionDone    struct {
		Name    string
		Summary result.Summary
	}
	tapsMsg struct {
		Taps []string
		Err  error
	}
	exportDiffMsg struct {
		Plan *app.ExportPlan
		Err  error
	}
)

type Model struct {
	app     *app.App
	report  status.Report
	loaded  bool
	screen  screen
	running bool
	action  string
	lines   []string
	out     viewport.Model
	outCh   chan string
	doneCh  chan actionDone
	taps    []string
	plan    *app.ExportPlan
	picker  Picker
	err     string
	width   int
	height  int
}

func New(a *app.App) Model {
	return Model{app: a, out: viewport.New(80, 10)}
}

// Run starts the dashboard in the alternate screen.
func Run(a *app.App) error {
	_, err := tea.NewProgram(New(a), tea.WithAltScreen()).Run()
	return err
}

func (m Model) Init() tea.Cmd { return m.refresh() }

func (m Model) refresh() tea.Cmd {
	a := m.app
	return func() tea.Msg { return reportMsg(a.Status()) }
}

func (m Model) resize(w, h int) (Model, tea.Cmd) {
	m.width, m.height = w, h
	m.out.Width = w
	m.out.Height = max(5, h/3)
	return m, nil
}

// lineWriter turns a stream of bytes into one channel message per line.
type lineWriter struct {
	ch  chan<- string
	buf []byte
}

func (w *lineWriter) Write(p []byte) (int, error) {
	w.buf = append(w.buf, p...)
	for {
		i := bytes.IndexByte(w.buf, '\n')
		if i < 0 {
			return len(p), nil
		}
		w.ch <- string(w.buf[:i])
		w.buf = w.buf[i+1:]
	}
}

func (w *lineWriter) flush() {
	if len(w.buf) > 0 {
		w.ch <- string(w.buf)
		w.buf = nil
	}
}

func (m Model) start(name string, fn func(io.Writer) result.Summary) (Model, tea.Cmd) {
	m.running, m.action, m.lines, m.err = true, name, nil, ""
	m.outCh = make(chan string, 64)
	m.doneCh = make(chan actionDone, 1)
	outCh, doneCh := m.outCh, m.doneCh
	go func() {
		w := &lineWriter{ch: outCh}
		s := fn(w)
		w.flush()
		close(outCh)
		doneCh <- actionDone{Name: name, Summary: s}
	}()
	return m, tea.Batch(waitLine(outCh), waitDone(doneCh))
}

func waitLine(ch <-chan string) tea.Cmd {
	return func() tea.Msg {
		l, ok := <-ch
		if !ok {
			return nil
		}
		return outputMsg(l)
	}
}

func waitDone(ch <-chan actionDone) tea.Cmd {
	return func() tea.Msg { return <-ch }
}

func (m Model) exportDiff() tea.Cmd {
	a := m.app
	return func() tea.Msg {
		plan, err := a.ExportDiff()
		return exportDiffMsg{Plan: plan, Err: err}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m.resize(msg.Width, msg.Height)
	case reportMsg:
		m.report, m.loaded = status.Report(msg), true
		return m, nil
	case outputMsg:
		m.lines = append(m.lines, string(msg))
		m.out.SetContent(strings.Join(m.lines, "\n"))
		m.out.GotoBottom()
		return m, waitLine(m.outCh)
	case actionDone:
		var buf bytes.Buffer
		msg.Summary.Print(&buf)
		m.lines = append(m.lines, strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")...)
		m.out.SetContent(strings.Join(m.lines, "\n"))
		m.out.GotoBottom()
		m.running = false
		return m, m.refresh()
	case tapsMsg:
		if msg.Err != nil {
			m.err = msg.Err.Error()
			return m, nil
		}
		if len(msg.Taps) == 0 {
			return m, m.exportDiff()
		}
		m.taps, m.screen = msg.Taps, screenConfirmTrust
		return m, nil
	case exportDiffMsg:
		if msg.Err != nil {
			m.err, m.screen = msg.Err.Error(), screenDashboard
			return m, nil
		}
		m.plan, m.picker, m.screen = msg.Plan, NewPicker(msg.Plan.Diff), screenPicker
		return m, nil
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m Model) handleKey(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := k.String()
	if key == "ctrl+c" || (key == "q" && m.screen == screenDashboard) {
		return m, tea.Quit
	}
	switch m.screen {
	case screenConfirmTrust:
		switch key {
		case "y":
			a, taps := m.app, m.taps
			m.screen = screenDashboard
			return m, func() tea.Msg {
				var buf bytes.Buffer
				if s := a.TrustTaps(taps, &buf); s.Failed() {
					return exportDiffMsg{Err: fmt.Errorf("trusting taps failed: %s", result.LastLine(buf.String()))}
				}
				plan, err := a.ExportDiff()
				return exportDiffMsg{Plan: plan, Err: err}
			}
		case "n", "esc":
			m.screen = screenDashboard
		}
		return m, nil
	case screenPicker:
		switch key {
		case "up", "k":
			m.picker = m.picker.Up()
		case "down", "j":
			m.picker = m.picker.Down()
		case " ":
			m.picker = m.picker.Toggle()
		case "a":
			m.picker = m.picker.ToggleGroup()
		case "esc":
			m.screen = screenDashboard
		case "enter":
			m.screen = screenDashboard
			a, plan := m.app, m.plan
			keep, drop := m.picker.Selections()
			return m.start("export", func(w io.Writer) result.Summary {
				if err := a.ExportWrite(plan, keep, drop, w); err != nil {
					return result.Summary{{Section: "brew", ID: "Brewfile", Status: result.Failed, Detail: err.Error()}}
				}
				return result.Summary{{Section: "brew", ID: "Brewfile", Status: result.Ok, Detail: fmt.Sprintf("%d added, %d dropped", len(keep), len(drop))}}
			})
		}
		return m, nil
	}
	if m.running {
		return m, nil
	}
	a := m.app
	switch key {
	case "i":
		return m.start("install", a.Install)
	case "l":
		return m.start("link", a.Link)
	case "u":
		return m.start("update", a.Update)
	case "c":
		return m.start("commit", func(w io.Writer) result.Summary { return a.Commit("", true, w) })
	case "e":
		return m, func() tea.Msg {
			taps, err := a.UntrustedTaps()
			return tapsMsg{Taps: taps, Err: err}
		}
	case "r":
		return m, m.refresh()
	}
	return m, nil
}

var (
	titleStyle   = lipgloss.NewStyle().Bold(true)
	sectionStyle = lipgloss.NewStyle().Bold(true).MarginTop(1)
	okStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	badStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	warnStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	footerStyle  = lipgloss.NewStyle().Faint(true).MarginTop(1)
)

func (m Model) View() string {
	switch m.screen {
	case screenConfirmTrust:
		return "Trust these taps so brew bundle can see their packages?\n\n  " +
			strings.Join(m.taps, "\n  ") + "\n\n" + footerStyle.Render("y yes · n no")
	case screenPicker:
		return m.picker.View(m.width)
	}

	var b strings.Builder
	b.WriteString(titleStyle.Render("dot  "+m.app.Root) + "\n")
	if !m.loaded {
		b.WriteString("\nloading status…\n")
	} else {
		m.writeReport(&b)
	}
	if m.err != "" {
		b.WriteString("\n" + badStyle.Render("error: "+m.err) + "\n")
	}
	if len(m.lines) > 0 {
		label := m.action
		if m.running {
			label += " (running…)"
		}
		b.WriteString(sectionStyle.Render(label) + "\n" + m.out.View() + "\n")
	}
	b.WriteString(footerStyle.Render("i install · l link · e export · u update · c commit · r refresh · q quit"))
	return b.String()
}

func (m Model) writeReport(b *strings.Builder) {
	r := m.report
	b.WriteString(sectionStyle.Render("Packages") + "\n")
	switch {
	case r.Packages.Err != "":
		b.WriteString("  " + badStyle.Render("error") + "  " + r.Packages.Err + "\n")
	case len(r.Packages.Missing) == 0 && len(r.Packages.Unrecorded) == 0:
		b.WriteString("  " + okStyle.Render("✓") + " all Brewfile entries installed, nothing unrecorded\n")
	}
	for _, e := range r.Packages.Missing {
		b.WriteString("  " + badStyle.Render("missing") + "     " + e.Line() + "\n")
	}
	for _, e := range r.Packages.Unrecorded {
		b.WriteString("  " + warnStyle.Render("unrecorded") + "  " + e.Line() + "\n")
	}

	b.WriteString(sectionStyle.Render("Links") + "\n")
	for _, it := range r.Links {
		st := okStyle.Render(fmt.Sprintf("%-13s", it.State))
		if it.State != link.Ok {
			st = badStyle.Render(fmt.Sprintf("%-13s", it.State))
		}
		detail := ""
		if it.Detail != "" {
			detail = "  (" + it.Detail + ")"
		}
		b.WriteString("  " + st + " " + it.Key + detail + "\n")
	}

	b.WriteString(sectionStyle.Render("Services") + "\n")
	for _, s := range r.Services {
		st := okStyle.Render(fmt.Sprintf("%-13s", "loaded"))
		if !s.Loaded {
			st = badStyle.Render(fmt.Sprintf("%-13s", "not loaded"))
		}
		b.WriteString("  " + st + " " + s.ID + "\n")
	}

	b.WriteString(sectionStyle.Render("Manual") + "\n")
	for _, it := range r.Manual {
		st := okStyle.Render(fmt.Sprintf("%-13s", it.State))
		how := ""
		if it.State != manual.Done {
			st = warnStyle.Render(fmt.Sprintf("%-13s", it.State))
			how = "  → " + it.How
		}
		b.WriteString("  " + st + " " + it.ID + how + "\n")
	}

	b.WriteString(sectionStyle.Render("Repo") + "\n")
	switch {
	case r.Repo.Err != "":
		b.WriteString("  " + badStyle.Render("error") + "  " + r.Repo.Err + "\n")
	case !r.Repo.Dirty:
		b.WriteString("  " + okStyle.Render("clean") + "\n")
	default:
		b.WriteString("  " + warnStyle.Render(fmt.Sprintf("%d uncommitted", len(r.Repo.Changes))) + "  " + strings.Join(r.Repo.Changes, ", ") + "\n")
	}
}
```

- [ ] **Step 9: Wire the CLI to the real dashboard**

Delete `cmd/dot/tui_placeholder.go`. In `cmd/dot/main.go` add the import `"github.com/ireydiak/dotfiles/internal/tui"` and replace the `runTUI(a)` call with `tui.Run(a)`.

- [ ] **Step 10: Run all tests and vet**

Run: `go test ./... && go vet ./... && gofmt -l .`
Expected: every package `ok`, no vet output, no files listed by gofmt

- [ ] **Step 11: Commit**

```bash
git add go.mod go.sum internal/tui cmd/dot
git commit -m "feat(tui): Bubble Tea dashboard with export picker and streaming output

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

---

### Task 15: Migrate this Mac, part 1: populate `home/`, split secrets, write the manifest

This task edits live files in `$HOME`. Every command runs from `~/Dev/git/dotfiles`. Nothing here is committed until the secret check in Step 9 passes.

**Files:**
- Create: `manifest.toml`, `home/**` (copied from the live machine)
- Modify: `~/.zshrc.local` (created, outside the repo)

**Interfaces:**
- Consumes: the `dot` binary built from Task 14
- Produces: a repo where `manifest.Load` succeeds and `dot status` runs; a `~/.dotfiles` symlink to this clone; `~/.local/bin/dot`

- [ ] **Step 1: Create the target directories**

```bash
mkdir -p home/.config "home/Library/Application Support/com.mitchellh.ghostty" "home/Library/Application Support/lazygit"
```

- [ ] **Step 2: Split `.zshrc` into tracked and local parts**

The eleven credential variables move to `~/.zshrc.local`. Hardcoded `/Users/jean-charles.verdier` paths become `$HOME` so the file works on any machine.

```bash
SECRET_RE='^export (NPM_TOKEN|GH_TOKEN|FLARE_API_KEY|FLR_DEMO_SECRET|TORCHE_SECRET_KEY|TORCHE_CLIENT_ID|TORCHE_TENANT_ID|SENTINEL_SECRET_KEY|AZ_RESOURCE_GROUP|VMRAY_API_KEY|FLARE_TENANT_ID)='
touch ~/.zshrc.local && chmod 600 ~/.zshrc.local
grep -E "$SECRET_RE" ~/.zshrc >> ~/.zshrc.local
grep -vE "$SECRET_RE" ~/.zshrc > home/.zshrc
sed -i '' 's#/Users/jean-charles.verdier#$HOME#g' home/.zshrc
printf '\n# Machine-local settings and secrets. Not tracked.\n[[ -f ~/.zshrc.local ]] && source ~/.zshrc.local\n' >> home/.zshrc
```

- [ ] **Step 3: Verify the split**

Run: `grep -cE "$SECRET_RE" home/.zshrc; grep -cE "$SECRET_RE" ~/.zshrc.local; grep -c 'jean-charles.verdier' home/.zshrc; tail -2 home/.zshrc`
Expected: `0`, `11`, `0`, and the two source lines

- [ ] **Step 4: Copy the remaining live configs into `home/`**

```bash
cp ~/.zprofile home/.zprofile
cp ~/.gitconfig home/.gitconfig
cp -R ~/.config/git home/.config/git
cp -R ~/.config/tmux home/.config/tmux
cp -R ~/.config/nvim home/.config/nvim
cp -R ~/.config/skhd home/.config/skhd
cp -R ~/.config/yabai home/.config/yabai
cp -R ~/.config/sketchybar home/.config/sketchybar
cp "$HOME/Library/Application Support/com.mitchellh.ghostty/config" "home/Library/Application Support/com.mitchellh.ghostty/config"
cp "$HOME/Library/Application Support/lazygit/config.yml" "home/Library/Application Support/lazygit/config.yml"
rm -rf home/.config/sketchybar/sketchybar-app-font home/.config/nvim/.claude
```

- [ ] **Step 5: Verify the copy is clean**

Run: `find home -name .git -type d; ls home/.config/nvim/lazy-lock.json home/.config/nvim/lazyvim.json home/.config/sketchybar/helper/makefile; git check-ignore -q home/.config/sketchybar/helper/helper && echo "helper binary ignored"`
Expected: no `.git` directories, the three files listed, `helper binary ignored`

- [ ] **Step 6: Write `manifest.toml`**

```toml
# What dot manages. Paths may start with ~/ ; nothing else is expanded.

[links]
".zshrc"                        = "~/.zshrc"
".zprofile"                     = "~/.zprofile"
".gitconfig"                    = "~/.gitconfig"
".config/git"                   = "~/.config/git"
".config/tmux"                  = "~/.config/tmux"
".config/nvim"                  = "~/.config/nvim"
".config/skhd"                  = "~/.config/skhd"
".config/yabai"                 = "~/.config/yabai"
".config/sketchybar"            = "~/.config/sketchybar"
"Library/Application Support/com.mitchellh.ghostty/config" = "~/Library/Application Support/com.mitchellh.ghostty/config"
"Library/Application Support/lazygit/config.yml"           = "~/Library/Application Support/lazygit/config.yml"

# Non-brew installs. Run in order; `run` executes only when `check` fails.
# `update` is optional and used by `dot update`.
[[steps]]
id     = "oh-my-zsh"
check  = "test -d ~/.oh-my-zsh"
run    = "sh -c \"$(curl -fsSL https://raw.githubusercontent.com/ohmyzsh/ohmyzsh/master/tools/install.sh)\" '' --unattended --keep-zshrc"
update = "ZSH=~/.oh-my-zsh zsh ~/.oh-my-zsh/tools/upgrade.sh"

[[steps]]
id    = "nvm"
check = "test -s ~/.nvm/nvm.sh"
run   = "curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.40.3/install.sh | PROFILE=/dev/null bash"

[[steps]]
id    = "sketchybar-helper"
check = "test -x ~/.config/sketchybar/helper/helper"
run   = "make -C ~/.config/sketchybar/helper"

[[steps]]
id     = "nvim-plugins"
check  = "test -d ~/.local/share/nvim/lazy/lazy.nvim"
run    = "nvim --headless '+Lazy! restore' +qa"
update = "nvim --headless '+Lazy! sync' +qa"

# launchd agents. Loaded when `launchctl print gui/<uid>/<label>` exits 0.
[[services]]
id    = "yabai"
label = "com.koekeishiya.yabai"
start = "yabai --start-service"

[[services]]
id    = "skhd"
label = "com.jackielii.skhd"
start = "skhd --install-service && skhd --start-service"

[[services]]
id    = "sketchybar"
label = "homebrew.mxcl.sketchybar"
start = "brew services start sketchybar"

# Things only a person can do. `check` is optional.
[[manual]]
id    = "xcode-clt"
check = "xcode-select -p"
how   = "xcode-select --install"

[[manual]]
id    = "sip-for-yabai"
check = "csrutil status | grep -q 'Filesystem Protections: disabled'"
how   = "Boot to Recovery, open Terminal, run: csrutil enable --without fs --without debug --without nvram"

[[manual]]
id    = "yabai-sudoers"
check = "sudo -n -l $(which yabai) --load-sa"
how   = "echo \"$(whoami) ALL=(root) NOPASSWD: sha256:$(shasum -a 256 $(which yabai) | cut -d' ' -f1) $(which yabai) --load-sa\" | sudo tee /private/etc/sudoers.d/yabai"

[[manual]]
id  = "accessibility"
how = "System Settings > Privacy & Security > Accessibility: enable yabai and skhd"

[[manual]]
id  = "ssh-and-gpg"
how = "Restore SSH and GPG keys, then: git -C ~/.dotfiles remote set-url origin git@github.com:ireydiak/dotfiles.git"
```

- [ ] **Step 7: Build the binary and make the default repo path work**

```bash
mkdir -p ~/.local/bin && go build -o ~/.local/bin/dot ./cmd/dot
ln -sfn "$PWD" ~/.dotfiles
```

- [ ] **Step 8: Verify the manifest loads and status runs**

Run: `~/.local/bin/dot status`
Expected: no manifest error. `Links` shows all 11 entries as `conflict` (the live files still exist as regular files or directories). `Services` shows yabai, skhd and sketchybar loaded. `Packages` lists every installed package as `unrecorded` because there is no Brewfile yet. `Manual` shows `xcode-clt` done.

- [ ] **Step 9: Stage and check for secrets before committing**

```bash
git add home manifest.toml
git diff --cached -U0 | grep -cE 'NPM_TOKEN|GH_TOKEN|FLARE_API_KEY|FLR_DEMO_SECRET|TORCHE_|SENTINEL_SECRET_KEY|VMRAY_API_KEY'
```
Expected: `0`. If anything else is printed, run `git reset` and redo Step 2 before continuing.

- [ ] **Step 10: Commit**

```bash
git commit -m "feat: move live configs into home/ and add the dot manifest

Live versions of nvim, tmux, skhd, yabai and sketchybar configs replace the
old repo copies. zsh credentials moved to the untracked ~/.zshrc.local.

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

---

### Task 16: Migrate this Mac, part 2: export the Brewfile, link, verify

**Files:**
- Create: `Brewfile`
- Modify: live `$HOME` paths become symlinks; `~/.sketchybarrc` and `~/.tmux.conf` move to the backup directory

**Interfaces:**
- Consumes: `~/.local/bin/dot` from Task 15
- Produces: the acceptance state from spec section 13

- [ ] **Step 1: Trust the taps and write the first Brewfile**

`--yes` trusts every currently tapped third-party tap. On this machine that is `anomalyco/tap`, `asmvik/formulae`, `derailed/k9s`, `felixkratz/formulae`, `hashicorp/tap`, `jackielii/tap` and `tinygo-org/tools`, all added by the owner.

Run: `~/.local/bin/dot export --yes`
Expected: a `trusted` line per tap, then `+` lines for every installed package, then `Brewfile written`

- [ ] **Step 2: Verify the previously hidden packages are now recorded and add the font casks**

```bash
grep -E 'yabai|sketchybar|skhd' Brewfile
printf 'cask "font-jetbrains-mono-nerd-font"\ncask "font-sketchybar-app-font"\n' >> Brewfile
```
Expected from grep: three `brew` lines naming `asmvik/formulae/yabai`, `felixkratz/formulae/sketchybar` and `jackielii/tap/skhd-zig`

- [ ] **Step 3: Link everything**

Run: `~/.local/bin/dot link`
Expected: 11 `ok` entries, each `back up to ~/.local/state/dot/backup/<timestamp>/... and link`; `11 ok, 0 skipped, 0 failed`

- [ ] **Step 4: Verify the links and the shell**

```bash
readlink ~/.zshrc ~/.config/nvim ~/.config/sketchybar "$HOME/Library/Application Support/com.mitchellh.ghostty/config"
TMUX=skip zsh -ic 'echo "$ZSH"; env | grep -c FLARE_API_KEY; alias pyro' 2>/dev/null
```
`TMUX=skip` stops the oh-my-zsh tmux plugin from autostarting a session in this check.
Expected: four paths under `~/Dev/git/dotfiles/home/`; then `/Users/jean-charles.verdier/.oh-my-zsh`, `1`, and the `pyro` alias definition

- [ ] **Step 5: Run install for real**

Run: `~/.local/bin/dot install`
Expected: taps `already trusted`, `brew bundle install completed` (it installs the two font casks), links `already linked`, all four steps `already satisfied`, all three services `already loaded`; exit 0. Pending manual steps printed: `sip-for-yabai` or `yabai-sudoers` only if their checks fail here, plus `accessibility` and `ssh-and-gpg` as verify items.

- [ ] **Step 6: Acceptance checks**

```bash
~/.local/bin/dot status
~/.local/bin/dot install --dry-run; echo "exit=$?"
```
Expected from status: `Packages` says all Brewfile entries installed, nothing unrecorded; every link `ok`; every service `loaded`. Expected from dry run: every entry `skipped` with `already trusted`, `nothing missing`, `already linked`, `already satisfied` or `already loaded`; `exit=0`.

- [ ] **Step 7: Retire the duplicate rc files**

```bash
BK=~/.local/state/dot/backup/manual-$(date +%Y%m%d-%H%M%S); mkdir -p "$BK"
mv ~/.sketchybarrc ~/.tmux.conf "$BK"/
tmux -L dottest start-server \; show-options -gv base-index \; kill-server
```
Expected from tmux: `1`, proving `~/.config/tmux/tmux.conf` is still loaded without `~/.tmux.conf`

- [ ] **Step 8: Confirm the desktop stack still works**

Run: `pgrep -fl 'yabai|sketchybar'; yabai -m query --spaces | head -c 200; echo; sketchybar --query bar | head -c 200; echo`
Expected: both processes listed and both queries return JSON

- [ ] **Step 9: Commit the Brewfile**

```bash
git add Brewfile
git commit -m "feat: export Brewfile from this machine

Includes the taps Homebrew 7 needed trusted (yabai, sketchybar, skhd-zig)
and the two font casks that replace the curl and vendored installs.

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

---

### Task 17: Bootstrap script, README, remove the old installer

**Files:**
- Create: `bootstrap.sh`, `README.md`
- Delete: `macos/`, `shared/`, `themes/`, `linux/`

**Interfaces:**
- Consumes: the finished binary and repo layout
- Produces: the fresh-machine entry point described in spec section 11

- [ ] **Step 1: Write `bootstrap.sh`**

```bash
#!/bin/bash
# Fresh-Mac entry point for ireydiak/dotfiles.
# Prerequisite: xcode-select --install (and wait for it to finish).
set -euo pipefail

REPO_URL="https://github.com/ireydiak/dotfiles"
REPO_DIR="$HOME/.dotfiles"

if ! xcode-select -p >/dev/null 2>&1; then
  echo "Xcode Command Line Tools are missing. Run: xcode-select --install, then rerun this script." >&2
  exit 1
fi

if [ ! -x /opt/homebrew/bin/brew ]; then
  /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
fi
eval "$(/opt/homebrew/bin/brew shellenv)"

command -v go >/dev/null 2>&1 || brew install go

if [ ! -d "$REPO_DIR/.git" ]; then
  git clone "$REPO_URL" "$REPO_DIR"
fi

mkdir -p "$HOME/.local/bin"
(cd "$REPO_DIR" && go build -o "$HOME/.local/bin/dot" ./cmd/dot)

"$HOME/.local/bin/dot" install

echo
echo "Done. Open a new terminal to pick up the shell configuration."
```

Run: `chmod +x bootstrap.sh && bash -n bootstrap.sh && shellcheck bootstrap.sh`
Expected: no output

- [ ] **Step 2: Write `README.md`**

````markdown
# dotfiles

My macOS development setup, reproduced by a small Go tool called `dot`.
Configs live in `home/` and are symlinked into `$HOME`, so editing a live
config edits the repo. Packages live in the `Brewfile`.

## New Mac

```bash
xcode-select --install          # wait for it to finish
curl -fsSL https://raw.githubusercontent.com/ireydiak/dotfiles/main/bootstrap.sh | bash
```

Then work through the manual checklist that `dot install` prints:

1. Accessibility permissions for yabai and skhd (System Settings > Privacy & Security).
2. Partial SIP disable for yabai, from Recovery: `csrutil enable --without fs --without debug --without nvram`.
3. The yabai sudoers line (printed by `dot status`).
4. Restore SSH and GPG keys, then switch the remote to SSH:
   `git -C ~/.dotfiles remote set-url origin git@github.com:ireydiak/dotfiles.git`.
5. Create `~/.zshrc.local` with machine-specific exports. It is sourced by
   `.zshrc` and never tracked.

## Daily use

| Command | What it does |
|---|---|
| `dot` | Dashboard: packages, links, services, manual steps, repo state. Keys: `i` install, `l` link, `e` export, `u` update, `c` commit, `r` refresh, `q` quit |
| `dot status` | Same report as text; `--json` for machines |
| `dot install` | Trust taps, `brew bundle install`, link, run steps, start services. Safe to rerun |
| `dot link` | Create or repair symlinks. Existing files are moved to `~/.local/state/dot/backup/<timestamp>/` |
| `dot export` | Record newly installed brew packages into the Brewfile |
| `dot update` | `brew update`, `brew upgrade`, oh-my-zsh upgrade, nvim plugin sync |
| `dot commit` | `git add -A`, commit with a generated message, push |

Every command accepts `--dry-run`. Rebuild after pulling repo changes with
`go build -o ~/.local/bin/dot ./cmd/dot`.

## Adding a config

1. Move the file or directory under `home/` at the path it has under `$HOME`.
2. Add one line to `[links]` in `manifest.toml`.
3. `dot link`, then `dot commit`.

## Layout

```
Brewfile        taps, formulae, casks (brew bundle format)
manifest.toml   links, install steps, launchd services, manual checklist
home/           mirrors $HOME
cmd/dot         the binary
internal/       exec, manifest, link, brew, steps, services, manual, repo, status, app, tui
```
````

- [ ] **Step 3: Remove the old installer and theme files**

```bash
git rm -r -q macos shared themes
rm -rf linux
git status --short | head -5
```
Expected: `D` lines for the removed files; no leftover `linux/`

- [ ] **Step 4: Full verification**

Run: `gofmt -l . ; go vet ./... && go test ./... && ~/.local/bin/dot status | head -20`
Expected: gofmt prints nothing, vet prints nothing, every package `ok`, status still green

- [ ] **Step 5: Commit**

```bash
git add bootstrap.sh README.md
git commit -m "feat: bootstrap script and README; remove the bash installer

Co-Authored-By: Claude Fable 5.1 <noreply@anthropic.com>"
```

- [ ] **Step 6: Report**

Do not push. Report the branch state (`git log --oneline main..HEAD`), the backup directory path from Task 16, and the pending manual items from the last `dot status`, then stop. Merging and pushing are the owner's call.
