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
	f.Scripts["git -C '/r' status --porcelain"] = exec.Result{Stdout: " M home/.config/nvim/lua/plugins/lsp.lua\n M home/.config/tmux/tmux.conf\n?? home/.zshrc\nM  Brewfile\nR  old.md -> manifest.toml\n"}
	f.Scripts["git -C '/r' diff --numstat HEAD -- Brewfile"] = exec.Result{Stdout: "3\t1\tBrewfile\n"}
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
	f.Scripts["git -C '/r' remote"] = exec.Result{Stdout: "origin\n"}
	if has, _ := g.HasRemote(context.Background()); !has {
		t.Fatal("expected remote")
	}
	if err := g.CommitAll(context.Background(), "it's done", nil); err != nil {
		t.Fatal(err)
	}
	if err := g.Push(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	want := `git -C '/r' add -A && git -C '/r' commit -m 'it'\''s done'`
	if f.Calls[1] != want || f.Calls[2] != "git -C '/r' push" {
		t.Fatalf("calls = %q", f.Calls)
	}
	f.Scripts[want] = exec.Result{ExitCode: 1, Stderr: "nothing added\n"}
	if err := g.CommitAll(context.Background(), "it's done", nil); err == nil || !strings.Contains(err.Error(), "nothing added") {
		t.Fatalf("commit failure should surface, got %v", err)
	}
}

func TestGitTop(t *testing.T) {
	f := exec.NewFake()
	f.Scripts["git -C '/some/dir' rev-parse --show-toplevel"] = exec.Result{Stdout: "/some\n"}
	top, err := GitTop(context.Background(), f)("/some/dir")
	if err != nil || top != "/some" {
		t.Fatalf("GitTop = %q %v", top, err)
	}
	f.Default = exec.Result{ExitCode: 128}
	if _, err := GitTop(context.Background(), f)("/elsewhere"); err == nil {
		t.Fatal("non-repo must error")
	}
}

func TestGitQuotesRoot(t *testing.T) {
	f := exec.NewFake()
	g := Git{Root: "/my repo", R: f}
	g.Push(context.Background(), nil)
	GitTop(context.Background(), f)("/some dir")
	if f.Calls[0] != "git -C '/my repo' push" || f.Calls[1] != "git -C '/some dir' rev-parse --show-toplevel" {
		t.Fatalf("calls = %q", f.Calls)
	}
}
