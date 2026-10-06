// Package manual evaluates the checklist of steps only a person can perform.
package manual

import (
	"context"

	"github.com/ireydiak/dotfiles/internal/exec"
	"github.com/ireydiak/dotfiles/internal/manifest"
)

type State int

const (
	Done State = iota
	Pending
	Unverifiable
)

func (s State) String() string {
	return [...]string{"done", "pending", "verify"}[s]
}

// MarshalText renders the state by name in JSON.
func (s State) MarshalText() ([]byte, error) { return []byte(s.String()), nil }

type Item struct {
	ID    string `json:"id"`
	State State  `json:"state"`
	How   string `json:"how"`
}

// Status runs each item's check when it has one.
func Status(ctx context.Context, r exec.Runner, list []manifest.Manual) []Item {
	items := make([]Item, 0, len(list))
	for _, m := range list {
		it := Item{ID: m.ID, How: m.How, State: Unverifiable}
		if m.Check != "" {
			it.State = Pending
			if res, err := r.Run(ctx, m.Check, nil); err == nil && res.ExitCode == 0 {
				it.State = Done
			}
		}
		items = append(items, it)
	}
	return items
}

// Remaining returns the items a person still has to look at.
func Remaining(items []Item) []Item {
	var out []Item
	for _, it := range items {
		if it.State != Done {
			out = append(out, it)
		}
	}
	return out
}
