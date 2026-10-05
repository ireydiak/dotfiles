// Package steps runs the manifest's non-brew install and update commands.
package steps

import (
	"context"
	"io"

	"github.com/ireydiak/dotfiles/internal/exec"
	"github.com/ireydiak/dotfiles/internal/manifest"
	"github.com/ireydiak/dotfiles/internal/result"
)

func entry(id string, st result.Status, detail string) result.Entry {
	return result.Entry{Section: "step", ID: id, Status: st, Detail: detail}
}

func passes(ctx context.Context, r exec.Runner, check string) bool {
	res, err := r.Run(ctx, check, nil)
	return err == nil && res.ExitCode == 0
}

// Install runs each step whose check fails, in manifest order.
func Install(ctx context.Context, r exec.Runner, list []manifest.Step, dryRun bool, out io.Writer) result.Summary {
	var s result.Summary
	for _, st := range list {
		if passes(ctx, r, st.Check) {
			s = append(s, entry(st.ID, result.Skipped, "already satisfied"))
			continue
		}
		if dryRun {
			s = append(s, entry(st.ID, result.Skipped, "would run"))
			continue
		}
		res, err := r.Run(ctx, st.Run, out)
		if err != nil {
			s = append(s, entry(st.ID, result.Failed, err.Error()))
			continue
		}
		if passes(ctx, r, st.Check) {
			s = append(s, entry(st.ID, result.Ok, "installed"))
			continue
		}
		detail := result.LastLine(res.Output())
		if detail == "" {
			detail = "check still failing after run"
		}
		s = append(s, entry(st.ID, result.Failed, detail))
	}
	return s
}

// Update runs the update command of every step that defines one.
func Update(ctx context.Context, r exec.Runner, list []manifest.Step, dryRun bool, out io.Writer) result.Summary {
	var s result.Summary
	for _, st := range list {
		if st.Update == "" {
			continue
		}
		if dryRun {
			s = append(s, entry(st.ID, result.Skipped, "would update"))
			continue
		}
		res, err := r.Run(ctx, st.Update, out)
		switch {
		case err != nil:
			s = append(s, entry(st.ID, result.Failed, err.Error()))
		case res.ExitCode != 0:
			s = append(s, entry(st.ID, result.Failed, result.LastLine(res.Output())))
		default:
			s = append(s, entry(st.ID, result.Ok, "updated"))
		}
	}
	return s
}
