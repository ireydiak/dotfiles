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
	cmd := "brew bundle dump --force --file=" + exec.Quote(c.DumpPath())
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
		"brew bundle install --no-upgrade --file=" + exec.Quote(c.File),
		"brew update",
		"brew upgrade",
	}
	if strings.Join(f.Calls, "|") != strings.Join(want, "|") {
		t.Fatalf("calls = %v", f.Calls)
	}
}

func TestPathsWithSpacesAreQuoted(t *testing.T) {
	f := exec.NewFake()
	c := Client{R: f, File: "/tmp/my files/Brewfile", TempDir: "/tmp/my files"}
	c.Install(context.Background(), nil)
	c.Dump(context.Background())
	if f.Calls[0] != "brew bundle install --no-upgrade --file='/tmp/my files/Brewfile'" || f.Calls[1] != "brew bundle dump --force --file='/tmp/my files/Brewfile.dump'" {
		t.Fatalf("calls = %q", f.Calls)
	}
}
