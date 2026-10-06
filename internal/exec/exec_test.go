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

func TestQuote(t *testing.T) {
	if got := Quote("/a b/c"); got != "'/a b/c'" {
		t.Fatalf("Quote(space) = %q", got)
	}
	if got := Quote("it's"); got != `'it'\''s'` {
		t.Fatalf("Quote(apostrophe) = %q", got)
	}
}
