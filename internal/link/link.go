// Package link inspects, plans and applies the symlinks from home/ into $HOME.
package link

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ireydiak/dotfiles/internal/manifest"
	"github.com/ireydiak/dotfiles/internal/result"
)

type State int

const (
	Ok State = iota
	Missing
	WrongTarget
	Conflict
	BrokenManifest
)

func (s State) String() string {
	return [...]string{"ok", "missing", "wrong-target", "conflict", "broken-manifest"}[s]
}

// MarshalText renders the state by name in JSON.
func (s State) MarshalText() ([]byte, error) { return []byte(s.String()), nil }

type Item struct {
	Key    string `json:"key"`
	Src    string `json:"src"`
	Dst    string `json:"dst"`
	State  State  `json:"state"`
	Detail string `json:"detail,omitempty"`
}

type Action struct {
	Item   Item
	Backup string
}

// Inspect classifies every link. Items are sorted by Key.
func Inspect(links map[string]string, repoRoot, home string) []Item {
	items := make([]Item, 0, len(links))
	for key, val := range links {
		it := Item{
			Key: key,
			Src: filepath.Clean(filepath.Join(repoRoot, "home", key)),
			Dst: manifest.Expand(val, home),
		}
		it.State, it.Detail = classify(it.Src, it.Dst)
		items = append(items, it)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Key < items[j].Key })
	return items
}

func classify(src, dst string) (State, string) {
	if _, err := os.Lstat(src); err != nil {
		return BrokenManifest, "source missing in repo"
	}
	fi, err := os.Lstat(dst)
	if os.IsNotExist(err) {
		return Missing, ""
	}
	if err != nil {
		return Conflict, err.Error()
	}
	if fi.Mode()&os.ModeSymlink == 0 {
		if fi.IsDir() {
			return Conflict, "directory exists"
		}
		return Conflict, "regular file exists"
	}
	target, err := os.Readlink(dst)
	if err != nil {
		return WrongTarget, err.Error()
	}
	if !filepath.IsAbs(target) {
		target = filepath.Join(filepath.Dir(dst), target)
	}
	if filepath.Clean(target) == src {
		return Ok, ""
	}
	return WrongTarget, "points to " + target
}

// Plan returns the actions for items that need work.
func Plan(items []Item, backupDir, home string) []Action {
	var actions []Action
	for _, it := range items {
		switch it.State {
		case Missing, WrongTarget:
			actions = append(actions, Action{Item: it})
		case Conflict:
			rel, err := filepath.Rel(home, it.Dst)
			if err != nil || strings.HasPrefix(rel, "..") {
				rel = filepath.Base(it.Dst)
			}
			actions = append(actions, Action{Item: it, Backup: filepath.Join(backupDir, rel)})
		}
	}
	return actions
}

// Apply performs the actions. With dryRun it only reports them.
func Apply(actions []Action, dryRun bool, out io.Writer) result.Summary {
	var s result.Summary
	for _, a := range actions {
		detail := describe(a)
		if dryRun {
			fmt.Fprintf(out, "would %s %s\n", detail, a.Item.Key)
			s = append(s, result.Entry{Section: "link", ID: a.Item.Key, Status: result.Skipped, Detail: "would " + detail})
			continue
		}
		if err := apply(a); err != nil {
			s = append(s, result.Entry{Section: "link", ID: a.Item.Key, Status: result.Failed, Detail: err.Error()})
			continue
		}
		fmt.Fprintf(out, "%s %s\n", detail, a.Item.Key)
		s = append(s, result.Entry{Section: "link", ID: a.Item.Key, Status: result.Ok, Detail: detail})
	}
	return s
}

func describe(a Action) string {
	switch {
	case a.Backup != "":
		return "back up to " + a.Backup + " and link"
	case a.Item.State == WrongTarget:
		return "replace link (" + a.Item.Detail + ")"
	default:
		return "link"
	}
}

func apply(a Action) error {
	dst := a.Item.Dst
	if a.Backup != "" {
		if err := os.MkdirAll(filepath.Dir(a.Backup), 0o755); err != nil {
			return err
		}
		if err := os.Rename(dst, a.Backup); err != nil {
			return fmt.Errorf("backup: %w", err)
		}
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	tmp := dst + ".dot-tmp"
	_ = os.Remove(tmp) // left behind by an interrupted run
	if err := os.Symlink(a.Item.Src, tmp); err != nil {
		return err
	}
	if err := os.Rename(tmp, dst); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}
