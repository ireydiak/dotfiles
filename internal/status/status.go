// Package status aggregates packages, links, services, manual checks and repo state.
package status

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/ireydiak/dotfiles/internal/brew"
	"github.com/ireydiak/dotfiles/internal/exec"
	"github.com/ireydiak/dotfiles/internal/link"
	"github.com/ireydiak/dotfiles/internal/manifest"
	"github.com/ireydiak/dotfiles/internal/manual"
	"github.com/ireydiak/dotfiles/internal/repo"
	"github.com/ireydiak/dotfiles/internal/services"
)

type Packages struct {
	Missing    []brew.Entry `json:"missing"`
	Unrecorded []brew.Entry `json:"unrecorded"`
	Err        string       `json:"error,omitempty"`
}

type RepoState struct {
	Dirty   bool     `json:"dirty"`
	Changes []string `json:"changes"`
	Err     string   `json:"error,omitempty"`
}

type Report struct {
	Packages Packages        `json:"packages"`
	Links    []link.Item     `json:"links"`
	Services []services.Item `json:"services"`
	Manual   []manual.Item   `json:"manual"`
	Repo     RepoState       `json:"repo"`
}

type Deps struct {
	Ctx      context.Context
	Manifest *manifest.Manifest
	RepoRoot string
	Home     string
	UID      string
	Runner   exec.Runner
	Brew     brew.Client
	Git      repo.Git
}

// Collect gathers every section. A failing section records its error and
// never prevents the others from being collected.
func Collect(d Deps) Report {
	var r Report
	r.Packages = collectPackages(d)
	r.Links = link.Inspect(d.Manifest.Links, d.RepoRoot, d.Home)
	r.Services = services.Status(d.Ctx, d.Runner, d.Manifest.Services, d.UID)
	r.Manual = manual.Status(d.Ctx, d.Runner, d.Manifest.Manual)
	changes, err := d.Git.Changes(d.Ctx)
	if err != nil {
		r.Repo.Err = err.Error()
	}
	r.Repo.Changes = changes
	r.Repo.Dirty = len(changes) > 0
	return r
}

func collectPackages(d Deps) Packages {
	committed, err := d.Brew.Committed()
	if err != nil {
		return Packages{Err: err.Error()}
	}
	dump, err := d.Brew.Dump(d.Ctx)
	if err != nil {
		return Packages{Err: err.Error()}
	}
	diff := brew.Compare(committed, dump)
	return Packages{Missing: diff.Removed, Unrecorded: diff.Added}
}

func (r Report) JSON(w io.Writer) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(r)
}

func (r Report) Print(w io.Writer) {
	fmt.Fprintln(w, "Packages")
	switch {
	case r.Packages.Err != "":
		fmt.Fprintf(w, "  error         %s\n", r.Packages.Err)
	case len(r.Packages.Missing) == 0 && len(r.Packages.Unrecorded) == 0:
		fmt.Fprintln(w, "  all Brewfile entries installed, nothing unrecorded")
	}
	for _, e := range r.Packages.Missing {
		fmt.Fprintf(w, "  missing       %s\n", e.Line())
	}
	for _, e := range r.Packages.Unrecorded {
		fmt.Fprintf(w, "  unrecorded    %s\n", e.Line())
	}

	fmt.Fprintln(w, "Links")
	for _, it := range r.Links {
		detail := ""
		if it.Detail != "" {
			detail = "  (" + it.Detail + ")"
		}
		fmt.Fprintf(w, "  %-13s %s%s\n", it.State, it.Key, detail)
	}

	fmt.Fprintln(w, "Services")
	for _, s := range r.Services {
		state := "not loaded"
		if s.Loaded {
			state = "loaded"
		}
		fmt.Fprintf(w, "  %-13s %s\n", state, s.ID)
	}

	fmt.Fprintln(w, "Manual")
	for _, m := range r.Manual {
		how := ""
		if m.State != manual.Done {
			how = "  → " + m.How
		}
		fmt.Fprintf(w, "  %-13s %s%s\n", m.State, m.ID, how)
	}

	fmt.Fprintln(w, "Repo")
	switch {
	case r.Repo.Err != "":
		fmt.Fprintf(w, "  error         %s\n", r.Repo.Err)
	case !r.Repo.Dirty:
		fmt.Fprintln(w, "  clean")
	default:
		fmt.Fprintf(w, "  %d uncommitted: %s\n", len(r.Repo.Changes), strings.Join(r.Repo.Changes, ", "))
	}
}
