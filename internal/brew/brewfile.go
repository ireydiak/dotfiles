// Package brew parses Brewfiles and wraps the brew commands dot needs.
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
