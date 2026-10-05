// Command dot installs and reproduces the macOS development setup.
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ireydiak/dotfiles/internal/app"
	"github.com/ireydiak/dotfiles/internal/manual"
	"github.com/ireydiak/dotfiles/internal/result"
	"github.com/ireydiak/dotfiles/internal/tui"
)

const usage = `usage: dot <command> [flags]

commands:
  status   report packages, links, services, manual steps and repo state
  install  trust taps, brew bundle install, link, run steps, start services
  link     create or repair symlinks from home/ into $HOME
  export   dump installed packages and update the Brewfile
  update   brew update, brew upgrade, then step updates
  commit   git add -A, commit and push
  (none)   open the dashboard

flags:
  --repo <path>   dotfiles repo (default: $DOTFILES_DIR, the current git repo, or ~/.dotfiles)
  --dry-run       print the plan without changing anything
  --yes           accept prompts (export)
  --json          machine-readable output (status)
  -m <msg>        commit message (commit)
  --no-push       do not push after committing (commit)
`

type newAppFunc func(ctx context.Context, o app.Options) (*app.App, error)

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr, app.New))
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer, newApp newAppFunc) int {
	cmd := ""
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		cmd, args = args[0], args[1:]
	}
	switch cmd {
	case "", "status", "install", "link", "export", "update", "commit":
	case "help", "-h", "--help":
		fmt.Fprint(stdout, usage)
		return 0
	default:
		fmt.Fprintf(stderr, "unknown command %q\n\n%s", cmd, usage)
		return 2
	}

	fs := flag.NewFlagSet("dot", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() { fmt.Fprint(stderr, usage) }
	repoFlag := fs.String("repo", "", "")
	dryRun := fs.Bool("dry-run", false, "")
	yes := fs.Bool("yes", false, "")
	asJSON := fs.Bool("json", false, "")
	msg := fs.String("m", "", "")
	noPush := fs.Bool("no-push", false, "")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	a, err := newApp(context.Background(), app.Options{
		RepoFlag: *repoFlag, Env: os.Getenv("DOTFILES_DIR"), DryRun: *dryRun, Yes: *yes,
	})
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 2
	}

	switch cmd {
	case "":
		if err := tui.Run(a); err != nil {
			fmt.Fprintln(stderr, "error:", err)
			return 1
		}
		return 0
	case "status":
		r := a.Status()
		if *asJSON {
			if err := r.JSON(stdout); err != nil {
				fmt.Fprintln(stderr, "error:", err)
				return 1
			}
			return 0
		}
		r.Print(stdout)
		return 0
	case "install":
		s := a.Install(stdout)
		s.Print(stdout)
		printPending(stdout, a.ManualStatus())
		return exitFor(s)
	case "link":
		return finish(stdout, a.Link(stdout))
	case "update":
		return finish(stdout, a.Update(stdout))
	case "commit":
		return finish(stdout, a.Commit(*msg, !*noPush, stdout))
	case "export":
		return runExport(a, bufio.NewScanner(stdin), stdout, stderr, *yes)
	}
	return 0
}

func finish(w io.Writer, s result.Summary) int {
	s.Print(w)
	return exitFor(s)
}

func exitFor(s result.Summary) int {
	if s.Failed() {
		return 1
	}
	return 0
}

func printPending(w io.Writer, items []manual.Item) {
	pending := manual.Remaining(items)
	if len(pending) == 0 {
		return
	}
	fmt.Fprintln(w, "\nPending manual steps:")
	for _, it := range pending {
		fmt.Fprintf(w, "  %s: %s\n", it.ID, it.How)
	}
}

func confirm(in *bufio.Scanner, out io.Writer, question string) bool {
	fmt.Fprintf(out, "%s [y/N] ", question)
	if !in.Scan() {
		fmt.Fprintln(out)
		return false
	}
	ans := strings.ToLower(strings.TrimSpace(in.Text()))
	return ans == "y" || ans == "yes"
}

func runExport(a *app.App, in *bufio.Scanner, stdout, stderr io.Writer, yes bool) int {
	taps, err := a.UntrustedTaps()
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}
	if len(taps) > 0 {
		fmt.Fprintf(stdout, "Untrusted taps: %s\n", strings.Join(taps, ", "))
		if !yes && !confirm(in, stdout, "Trust them so brew bundle can see their packages?") {
			fmt.Fprintln(stdout, "export cancelled")
			return 0
		}
		s := a.TrustTaps(taps, stdout)
		s.Print(stdout)
		if s.Failed() {
			return 1
		}
	}
	plan, err := a.ExportDiff()
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}
	for _, e := range plan.Diff.Added {
		fmt.Fprintf(stdout, "  + %s\n", e.Line())
	}
	for _, e := range plan.Diff.Removed {
		fmt.Fprintf(stdout, "  - %s (no longer installed; kept)\n", e.Line())
	}
	if len(plan.Diff.Added) == 0 && len(plan.Diff.Removed) == 0 {
		fmt.Fprintln(stdout, "Brewfile is up to date")
		return 0
	}
	if !yes && !confirm(in, stdout, fmt.Sprintf("Write Brewfile with %d additions?", len(plan.Diff.Added))) {
		fmt.Fprintln(stdout, "export cancelled")
		return 0
	}
	keep := map[string]bool{}
	for _, e := range plan.Diff.Added {
		keep[e.Key()] = true
	}
	if err := a.ExportWrite(plan, keep, map[string]bool{}, stdout); err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}
	fmt.Fprintln(stdout, "Brewfile written")
	return 0
}
