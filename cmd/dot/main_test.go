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
	root, _ = filepath.EvalSymlinks(root)
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
