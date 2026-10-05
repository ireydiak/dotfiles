// Package services reports and starts the launchd agents dot manages.
package services

import (
	"context"
	"io"
	"strings"

	"github.com/ireydiak/dotfiles/internal/exec"
	"github.com/ireydiak/dotfiles/internal/manifest"
	"github.com/ireydiak/dotfiles/internal/result"
)

// Item is one launchd agent. Loaded means launchd knows the job; Running
// means its process is alive right now rather than crash-looping.
type Item struct {
	ID      string `json:"id"`
	Label   string `json:"label"`
	Loaded  bool   `json:"loaded"`
	Running bool   `json:"running"`
}

// Command is the launchctl query whose exit code says whether label is loaded.
func Command(uid, label string) string {
	return "launchctl print gui/" + uid + "/" + label
}

func query(ctx context.Context, r exec.Runner, uid, label string) (loaded, running bool) {
	res, err := r.Run(ctx, Command(uid, label), nil)
	if err != nil || res.ExitCode != 0 {
		return false, false
	}
	return true, strings.Contains(res.Stdout, "state = running")
}

func Status(ctx context.Context, r exec.Runner, list []manifest.Service, uid string) []Item {
	items := make([]Item, 0, len(list))
	for _, svc := range list {
		loaded, running := query(ctx, r, uid, svc.Label)
		items = append(items, Item{ID: svc.ID, Label: svc.Label, Loaded: loaded, Running: running})
	}
	return items
}

// Start starts every service that is not loaded.
func Start(ctx context.Context, r exec.Runner, list []manifest.Service, uid string, dryRun bool, out io.Writer) result.Summary {
	var s result.Summary
	add := func(id string, st result.Status, detail string) {
		s = append(s, result.Entry{Section: "service", ID: id, Status: st, Detail: detail})
	}
	for _, svc := range list {
		if loaded, _ := query(ctx, r, uid, svc.Label); loaded {
			add(svc.ID, result.Skipped, "already loaded")
			continue
		}
		if dryRun {
			add(svc.ID, result.Skipped, "would start")
			continue
		}
		res, err := r.Run(ctx, svc.Start, out)
		loaded, _ := query(ctx, r, uid, svc.Label)
		switch {
		case err != nil:
			add(svc.ID, result.Failed, err.Error())
		case res.ExitCode != 0:
			add(svc.ID, result.Failed, result.LastLine(res.Output()))
		case loaded:
			add(svc.ID, result.Ok, "started")
		default:
			add(svc.ID, result.Failed, "not loaded after start")
		}
	}
	return s
}

// Describe is the one-word-ish state used by status output and the TUI.
func Describe(it Item) string {
	switch {
	case !it.Loaded:
		return "not loaded"
	case !it.Running:
		return "loaded, not running"
	default:
		return "loaded"
	}
}
