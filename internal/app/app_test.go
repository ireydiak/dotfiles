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
	h.fake.Scripts["git -C "+exec.Quote(h.root)+" status --porcelain"] = exec.Result{Stdout: " M home/.zshrc\n"}
	h.fake.Scripts["git -C "+exec.Quote(h.root)+" diff --numstat HEAD -- Brewfile"] = exec.Result{Stdout: ""}
	h.fake.Scripts["git -C "+exec.Quote(h.root)+" remote"] = exec.Result{Stdout: "origin\n"}
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

func TestExportDiffHonoursBrewIgnore(t *testing.T) {
	h := newHarness(t, "brew \"jq\"\n")
	h.app.Manifest.Brew.Ignore = []string{"go tcurl/cmd/tcurl"}
	h.dump(t, "brew \"jq\"\ngo \"tcurl/cmd/tcurl\"\nnpm \"yarn\"\n")
	p, err := h.app.ExportDiff()
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Diff.Added) != 1 || p.Diff.Added[0].Name != "yarn" {
		t.Fatalf("ignored entry must not be offered: %+v", p.Diff.Added)
	}
}
