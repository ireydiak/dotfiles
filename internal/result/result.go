// Package result holds the shared ok / skipped / failed outcome type.
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
