package brew

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/ireydiak/dotfiles/internal/exec"
	"github.com/ireydiak/dotfiles/internal/result"
)

// Client wraps the brew commands dot needs. File is the committed Brewfile.
type Client struct {
	R       exec.Runner
	File    string
	TempDir string
}

func (c Client) run(ctx context.Context, command string, out io.Writer) (exec.Result, error) {
	res, err := c.R.Run(ctx, command, out)
	if err != nil {
		return res, fmt.Errorf("%s: %w", command, err)
	}
	if res.ExitCode != 0 {
		return res, fmt.Errorf("%s: exit %d: %s", command, res.ExitCode, result.LastLine(res.Output()))
	}
	return res, nil
}

// Committed parses the committed Brewfile. A missing file is an empty list.
func (c Client) Committed() ([]Entry, error) {
	f, err := os.Open(c.File)
	if os.IsNotExist(err) {
		return []Entry{}, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return Parse(f)
}

func (c Client) DumpPath() string { return filepath.Join(c.TempDir, "Brewfile.dump") }

// Dump asks brew for the installed set and parses it.
func (c Client) Dump(ctx context.Context) ([]Entry, error) {
	path := c.DumpPath()
	defer os.Remove(path)
	if _, err := c.run(ctx, "brew bundle dump --force --file="+exec.Quote(path), nil); err != nil {
		return nil, err
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("brew bundle dump produced no file: %w", err)
	}
	defer f.Close()
	return Parse(f)
}

func (c Client) Tapped(ctx context.Context) ([]string, error) {
	res, err := c.run(ctx, "brew tap", nil)
	if err != nil {
		return nil, err
	}
	var taps []string
	for _, l := range strings.Split(res.Stdout, "\n") {
		if l = strings.TrimSpace(l); l != "" {
			taps = append(taps, l)
		}
	}
	return taps, nil
}

func (c Client) Trusted(ctx context.Context, tap string) (bool, error) {
	res, err := c.run(ctx, "brew tap-info --json "+tap, nil)
	if err != nil {
		return false, err
	}
	var info []struct {
		Trusted bool `json:"trusted"`
	}
	if err := json.Unmarshal([]byte(res.Stdout), &info); err != nil {
		return false, fmt.Errorf("brew tap-info %s: %w", tap, err)
	}
	if len(info) == 0 {
		return false, fmt.Errorf("brew tap-info %s: empty response", tap)
	}
	return info[0].Trusted, nil
}

func (c Client) Trust(ctx context.Context, tap string, out io.Writer) error {
	_, err := c.run(ctx, "brew trust --tap "+tap, out)
	return err
}

func (c Client) Install(ctx context.Context, out io.Writer) error {
	_, err := c.run(ctx, "brew bundle install --no-upgrade --file="+exec.Quote(c.File), out)
	return err
}

func (c Client) Update(ctx context.Context, out io.Writer) error {
	_, err := c.run(ctx, "brew update", out)
	return err
}

func (c Client) Upgrade(ctx context.Context, out io.Writer) error {
	_, err := c.run(ctx, "brew upgrade", out)
	return err
}
