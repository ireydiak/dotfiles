// Package app wires the engines together. The CLI and the TUI both call it.
package app

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/ireydiak/dotfiles/internal/brew"
	"github.com/ireydiak/dotfiles/internal/exec"
	"github.com/ireydiak/dotfiles/internal/link"
	"github.com/ireydiak/dotfiles/internal/manifest"
	"github.com/ireydiak/dotfiles/internal/manual"
	"github.com/ireydiak/dotfiles/internal/repo"
	"github.com/ireydiak/dotfiles/internal/result"
	"github.com/ireydiak/dotfiles/internal/services"
	"github.com/ireydiak/dotfiles/internal/status"
	"github.com/ireydiak/dotfiles/internal/steps"
)

type Options struct {
	RepoFlag string
	Env      string
	Home     string
	Cwd      string
	UID      string
	DryRun   bool
	Yes      bool
	Runner   exec.Runner
	Now      func() time.Time
}

type App struct {
	Ctx      context.Context
	Root     string
	Home     string
	UID      string
	Manifest *manifest.Manifest
	Runner   exec.Runner
	Brew     brew.Client
	Git      repo.Git
	DryRun   bool
	Yes      bool
	Now      func() time.Time
}

func New(ctx context.Context, o Options) (*App, error) {
	var err error
	home := o.Home
	if home == "" {
		if home, err = os.UserHomeDir(); err != nil {
			return nil, err
		}
	}
	cwd := o.Cwd
	if cwd == "" {
		if cwd, err = os.Getwd(); err != nil {
			return nil, err
		}
	}
	uid := o.UID
	if uid == "" {
		uid = strconv.Itoa(os.Getuid())
	}
	runner := o.Runner
	if runner == nil {
		runner = exec.NewShell(home)
	}
	now := o.Now
	if now == nil {
		now = time.Now
	}
	root, err := repo.Resolve(repo.ResolveOptions{
		Flag: o.RepoFlag, Env: o.Env, Cwd: cwd, Home: home, GitTop: repo.GitTop(ctx, runner),
	})
	if err != nil {
		return nil, err
	}
	m, err := manifest.Load(root)
	if err != nil {
		return nil, err
	}
	return &App{
		Ctx: ctx, Root: root, Home: home, UID: uid, Manifest: m, Runner: runner,
		Brew:   brew.Client{R: runner, File: filepath.Join(root, "Brewfile"), TempDir: os.TempDir()},
		Git:    repo.Git{Root: root, R: runner},
		DryRun: o.DryRun, Yes: o.Yes, Now: now,
	}, nil
}

func (a *App) BackupDir() string {
	return filepath.Join(a.Home, ".local", "state", "dot", "backup", a.Now().Format("20060102-150405"))
}

func (a *App) Status() status.Report {
	return status.Collect(status.Deps{
		Ctx: a.Ctx, Manifest: a.Manifest, RepoRoot: a.Root, Home: a.Home, UID: a.UID,
		Runner: a.Runner, Brew: a.Brew, Git: a.Git,
	})
}

func (a *App) ManualStatus() []manual.Item {
	return manual.Status(a.Ctx, a.Runner, a.Manifest.Manual)
}

func (a *App) Link(out io.Writer) result.Summary {
	items := link.Inspect(a.Manifest.Links, a.Root, a.Home)
	s := link.Apply(link.Plan(items, a.BackupDir(), a.Home), a.DryRun, out)
	for _, it := range items {
		switch it.State {
		case link.Ok:
			s = append(s, result.Entry{Section: "link", ID: it.Key, Status: result.Skipped, Detail: "already linked"})
		case link.BrokenManifest:
			s = append(s, result.Entry{Section: "link", ID: it.Key, Status: result.Failed, Detail: it.Detail})
		}
	}
	return s
}

func (a *App) TrustTaps(taps []string, out io.Writer) result.Summary {
	var s result.Summary
	for _, tap := range taps {
		e := result.Entry{Section: "brew", ID: tap}
		trusted, err := a.Brew.Trusted(a.Ctx, tap)
		switch {
		case err != nil:
			e.Status, e.Detail = result.Failed, err.Error()
		case trusted:
			e.Status, e.Detail = result.Skipped, "already trusted"
		case a.DryRun:
			e.Status, e.Detail = result.Skipped, "would trust"
		default:
			if err := a.Brew.Trust(a.Ctx, tap, out); err != nil {
				e.Status, e.Detail = result.Failed, err.Error()
			} else {
				e.Status, e.Detail = result.Ok, "trusted"
			}
		}
		s = append(s, e)
	}
	return s
}

func (a *App) Install(out io.Writer) result.Summary {
	var s result.Summary
	committed, err := a.Brew.Committed()
	if err != nil {
		s = append(s, result.Entry{Section: "brew", ID: "Brewfile", Status: result.Failed, Detail: err.Error()})
	} else {
		s = append(s, a.TrustTaps(brew.Taps(committed), out)...)
		s = append(s, a.brewInstall(committed, out))
	}
	s = append(s, a.Link(out)...)
	s = append(s, steps.Install(a.Ctx, a.Runner, a.Manifest.Steps, a.DryRun, out)...)
	s = append(s, services.Start(a.Ctx, a.Runner, a.Manifest.Services, a.UID, a.DryRun, out)...)
	return s
}

func (a *App) brewInstall(committed []brew.Entry, out io.Writer) result.Entry {
	e := result.Entry{Section: "brew", ID: "bundle install"}
	if a.DryRun {
		dump, err := a.Brew.Dump(a.Ctx)
		if err != nil {
			e.Status, e.Detail = result.Failed, err.Error()
			return e
		}
		if missing := len(brew.Compare(committed, dump).Removed); missing == 0 {
			e.Status, e.Detail = result.Skipped, "nothing missing"
		} else {
			e.Status, e.Detail = result.Skipped, fmt.Sprintf("would install %d missing", missing)
		}
		return e
	}
	if err := a.Brew.Install(a.Ctx, out); err != nil {
		e.Status, e.Detail = result.Failed, err.Error()
		return e
	}
	e.Status, e.Detail = result.Ok, "brew bundle install completed"
	return e
}

func (a *App) Update(out io.Writer) result.Summary {
	var s result.Summary
	brewStep := func(id string, fn func(context.Context, io.Writer) error) {
		e := result.Entry{Section: "brew", ID: id}
		switch {
		case a.DryRun:
			e.Status, e.Detail = result.Skipped, "would run brew "+id
		default:
			if err := fn(a.Ctx, out); err != nil {
				e.Status, e.Detail = result.Failed, err.Error()
			} else {
				e.Status, e.Detail = result.Ok, "brew "+id+" completed"
			}
		}
		s = append(s, e)
	}
	brewStep("update", a.Brew.Update)
	brewStep("upgrade", a.Brew.Upgrade)
	s = append(s, steps.Update(a.Ctx, a.Runner, a.Manifest.Steps, a.DryRun, out)...)
	return s
}

func (a *App) UntrustedTaps() ([]string, error) {
	taps, err := a.Brew.Tapped(a.Ctx)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, tap := range taps {
		if strings.HasPrefix(tap, "homebrew/") {
			continue
		}
		trusted, err := a.Brew.Trusted(a.Ctx, tap)
		if err != nil {
			return nil, err
		}
		if !trusted {
			out = append(out, tap)
		}
	}
	return out, nil
}

type ExportPlan struct {
	Committed []brew.Entry
	Dump      []brew.Entry
	Diff      brew.Diff
}

func (a *App) ExportDiff() (*ExportPlan, error) {
	committed, err := a.Brew.Committed()
	if err != nil {
		return nil, err
	}
	dump, err := a.Brew.Dump(a.Ctx)
	if err != nil {
		return nil, err
	}
	return &ExportPlan{Committed: committed, Dump: dump, Diff: brew.Compare(committed, dump)}, nil
}

func (a *App) ExportWrite(p *ExportPlan, keepAdded, dropRemoved map[string]bool, out io.Writer) error {
	entries := brew.Filter(p.Dump, p.Committed, keepAdded, dropRemoved)
	if a.DryRun {
		fmt.Fprintf(out, "would write %d entries to %s\n", len(entries), a.Brew.File)
		return nil
	}
	tmp := a.Brew.File + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	if err := brew.Write(f, entries); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, a.Brew.File)
}

func (a *App) Commit(msg string, push bool, out io.Writer) result.Summary {
	entry := func(id string, st result.Status, detail string) result.Entry {
		return result.Entry{Section: "git", ID: id, Status: st, Detail: detail}
	}
	changes, err := a.Git.Changes(a.Ctx)
	if err != nil {
		return result.Summary{entry("commit", result.Failed, err.Error())}
	}
	if len(changes) == 0 {
		return result.Summary{entry("commit", result.Skipped, "nothing to commit")}
	}
	if msg == "" {
		added, removed, _ := a.Git.BrewfileNumstat(a.Ctx)
		msg = repo.Message(changes, added, removed)
	}
	if a.DryRun {
		return result.Summary{entry("commit", result.Skipped, "would commit: "+msg)}
	}
	if err := a.Git.CommitAll(a.Ctx, msg, out); err != nil {
		return result.Summary{entry("commit", result.Failed, err.Error())}
	}
	s := result.Summary{entry("commit", result.Ok, msg)}
	if !push {
		return s
	}
	has, err := a.Git.HasRemote(a.Ctx)
	switch {
	case err != nil:
		s = append(s, entry("push", result.Failed, err.Error()))
	case !has:
		s = append(s, entry("push", result.Skipped, "no remote"))
	default:
		if err := a.Git.Push(a.Ctx, out); err != nil {
			s = append(s, entry("push", result.Failed, err.Error()))
		} else {
			s = append(s, entry("push", result.Ok, "pushed"))
		}
	}
	return s
}
