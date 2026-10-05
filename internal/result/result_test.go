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
