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
