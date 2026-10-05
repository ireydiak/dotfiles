// Package services reports and starts the launchd agents dot manages.
package services

import (
	"context"
	"io"

	"github.com/ireydiak/dotfiles/internal/exec"
	"github.com/ireydiak/dotfiles/internal/manifest"
	"github.com/ireydiak/dotfiles/internal/result"
)

type Item struct {
	ID     string
	Label  string
	Loaded bool
}

// Command is the launchctl query whose exit code says whether label is loaded.
func Command(uid, label string) string {
	return "launchctl print gui/" + uid + "/" + label
}

func loaded(ctx context.Context, r exec.Runner, uid, label string) bool {
	res, err := r.Run(ctx, Command(uid, label), nil)
	return err == nil && res.ExitCode == 0
}

func Status(ctx context.Context, r exec.Runner, list []manifest.Service, uid string) []Item {
	items := make([]Item, 0, len(list))
	for _, svc := range list {
		items = append(items, Item{ID: svc.ID, Label: svc.Label, Loaded: loaded(ctx, r, uid, svc.Label)})
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
		if loaded(ctx, r, uid, svc.Label) {
			add(svc.ID, result.Skipped, "already loaded")
			continue
		}
		if dryRun {
			add(svc.ID, result.Skipped, "would start")
			continue
		}
		res, err := r.Run(ctx, svc.Start, out)
		switch {
		case err != nil:
			add(svc.ID, result.Failed, err.Error())
		case res.ExitCode != 0:
			add(svc.ID, result.Failed, result.LastLine(res.Output()))
		case loaded(ctx, r, uid, svc.Label):
			add(svc.ID, result.Ok, "started")
		default:
			add(svc.ID, result.Failed, "not loaded after start")
		}
	}
	return s
}
