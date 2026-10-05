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
