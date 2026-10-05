// Package repo resolves the dotfiles repo root and runs the git operations dot needs.
package repo

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ireydiak/dotfiles/internal/exec"
	"github.com/ireydiak/dotfiles/internal/result"
)

type ResolveOptions struct {
	Flag   string
	Env    string
	Cwd    string
	Home   string
	GitTop func(dir string) (string, error)
}

func hasManifest(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, "manifest.toml"))
	return err == nil
}

// Resolve picks the repo root: --repo, then DOTFILES_DIR, then the git top
// level of the working directory, then ~/.dotfiles. The result is canonical.
func Resolve(o ResolveOptions) (string, error) {
	for _, c := range []struct{ name, path string }{{"--repo", o.Flag}, {"DOTFILES_DIR", o.Env}} {
		if c.path == "" {
			continue
		}
		abs, err := filepath.Abs(c.path)
		if err != nil {
			return "", fmt.Errorf("%s=%s: %w", c.name, c.path, err)
		}
		real, err := filepath.EvalSymlinks(abs)
		if err != nil || !hasManifest(real) {
			return "", fmt.Errorf("%s=%s does not contain manifest.toml", c.name, c.path)
		}
		return real, nil
	}
	if o.GitTop != nil && o.Cwd != "" {
		if top, err := o.GitTop(o.Cwd); err == nil {
			if real, err := filepath.EvalSymlinks(top); err == nil && hasManifest(real) {
				return real, nil
			}
		}
	}
	def := filepath.Join(o.Home, ".dotfiles")
	if real, err := filepath.EvalSymlinks(def); err == nil && hasManifest(real) {
		return real, nil
	}
	return "", fmt.Errorf("no dotfiles repo found: pass --repo, set DOTFILES_DIR, run inside the repo, or clone it to %s", def)
}

// GitTop returns a function that finds the git top level of a directory.
func GitTop(ctx context.Context, r exec.Runner) func(dir string) (string, error) {
	return func(dir string) (string, error) {
		res, err := r.Run(ctx, "git -C "+dir+" rev-parse --show-toplevel", nil)
		if err != nil {
			return "", err
		}
		if res.ExitCode != 0 {
			return "", fmt.Errorf("%s is not inside a git repository", dir)
		}
		return strings.TrimSpace(res.Stdout), nil
	}
}

func ShellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

type Git struct {
	Root string
	R    exec.Runner
}

func (g Git) run(ctx context.Context, args string, out io.Writer) (exec.Result, error) {
	cmd := "git -C " + g.Root + " " + args
	res, err := g.R.Run(ctx, cmd, out)
	if err != nil {
		return res, fmt.Errorf("%s: %w", cmd, err)
	}
	if res.ExitCode != 0 {
		return res, fmt.Errorf("%s: exit %d: %s", cmd, res.ExitCode, result.LastLine(res.Output()))
	}
	return res, nil
}

// Changes lists paths with uncommitted changes.
func (g Git) Changes(ctx context.Context) ([]string, error) {
	res, err := g.run(ctx, "status --porcelain", nil)
	if err != nil {
		return nil, err
	}
	var paths []string
	for _, line := range strings.Split(res.Stdout, "\n") {
		if len(line) < 4 {
			continue
		}
		p := line[3:]
		if i := strings.Index(p, " -> "); i >= 0 {
			p = p[i+4:]
		}
		paths = append(paths, p)
	}
	return paths, nil
}

func (g Git) HasRemote(ctx context.Context) (bool, error) {
	res, err := g.run(ctx, "remote", nil)
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(res.Stdout) != "", nil
}

// BrewfileNumstat returns lines added and removed in Brewfile versus HEAD.
func (g Git) BrewfileNumstat(ctx context.Context) (int, int, error) {
	res, err := g.run(ctx, "diff --numstat HEAD -- Brewfile", nil)
	if err != nil {
		return 0, 0, err
	}
	fields := strings.Fields(res.Stdout)
	if len(fields) < 2 {
		return 0, 0, nil
	}
	a, _ := strconv.Atoi(fields[0])
	r, _ := strconv.Atoi(fields[1])
	return a, r, nil
}

func (g Git) CommitAll(ctx context.Context, msg string, out io.Writer) error {
	_, err := g.run(ctx, "add -A && git -C "+g.Root+" commit -m "+ShellQuote(msg), out)
	return err
}

func (g Git) Push(ctx context.Context, out io.Writer) error {
	_, err := g.run(ctx, "push", out)
	return err
}

// Message summarizes changed paths into a commit message.
func Message(changes []string, brewAdded, brewRemoved int) string {
	var areas []string
	seen := map[string]bool{}
	for _, p := range changes {
		area := areaOf(p)
		if area == "" || seen[area] {
			continue
		}
		seen[area] = true
		areas = append(areas, area)
	}
	msg := "dot: update"
	if len(areas) > 0 {
		msg += " " + strings.Join(areas, ", ")
	}
	if brewAdded != 0 || brewRemoved != 0 {
		msg += fmt.Sprintf("; Brewfile +%d -%d", brewAdded, brewRemoved)
	}
	return msg
}

func areaOf(p string) string {
	if p == "Brewfile" {
		return ""
	}
	rel, ok := strings.CutPrefix(p, "home/")
	if !ok {
		return strings.SplitN(p, "/", 2)[0]
	}
	parts := strings.Split(rel, "/")
	switch {
	case parts[0] == ".config" && len(parts) > 1:
		return ".config/" + parts[1]
	case parts[0] == "Library" && len(parts) > 2:
		return strings.Join(parts[:3], "/")
	default:
		return parts[0]
	}
}
