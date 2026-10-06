// Package manifest defines what dot manages: links, steps, services and manual checks.
package manifest

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

type Manifest struct {
	Links    map[string]string `toml:"links"`
	Brew     BrewOptions       `toml:"brew"`
	Steps    []Step            `toml:"steps"`
	Services []Service         `toml:"services"`
	Manual   []Manual          `toml:"manual"`
}

// BrewOptions tunes how dot treats brew bundle output.
type BrewOptions struct {
	// Ignore lists "<kind> <name>" entries that brew bundle dump reports but
	// that must never reach the Brewfile, status or export (local-only builds).
	Ignore []string `toml:"ignore"`
}

type Step struct {
	ID     string `toml:"id"`
	Check  string `toml:"check"`
	Run    string `toml:"run"`
	Update string `toml:"update"`
}

type Service struct {
	ID    string `toml:"id"`
	Label string `toml:"label"`
	Start string `toml:"start"`
}

type Manual struct {
	ID    string `toml:"id"`
	Check string `toml:"check"`
	How   string `toml:"how"`
}

// Load reads and validates <repoRoot>/manifest.toml.
func Load(repoRoot string) (*Manifest, error) {
	f, err := os.Open(filepath.Join(repoRoot, "manifest.toml"))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	m, err := Parse(f)
	if err != nil {
		return nil, fmt.Errorf("manifest.toml: %w", err)
	}
	if err := m.Validate(repoRoot); err != nil {
		return nil, fmt.Errorf("manifest.toml: %w", err)
	}
	return m, nil
}

// Parse decodes TOML strictly: unknown keys are errors.
func Parse(r io.Reader) (*Manifest, error) {
	var m Manifest
	dec := toml.NewDecoder(r)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&m); err != nil {
		var strict *toml.StrictMissingError
		if errors.As(err, &strict) {
			return nil, fmt.Errorf("unknown key: %s", strict.String())
		}
		return nil, err
	}
	if m.Links == nil {
		m.Links = map[string]string{}
	}
	return &m, nil
}

func (m *Manifest) LinkKeys() []string {
	keys := make([]string, 0, len(m.Links))
	for k := range m.Links {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// Expand replaces a leading "~/" with home. It is the only expansion performed.
func Expand(p, home string) string {
	if strings.HasPrefix(p, "~/") {
		return filepath.Join(home, p[2:])
	}
	return p
}

func (m *Manifest) Validate(repoRoot string) error {
	var errs []error
	for _, key := range m.LinkKeys() {
		if key == "" || filepath.IsAbs(key) || filepath.Clean(key) != key || strings.HasPrefix(key, "..") {
			errs = append(errs, fmt.Errorf("links: key %q must be a clean relative path", key))
		} else if _, err := os.Lstat(filepath.Join(repoRoot, "home", key)); err != nil {
			errs = append(errs, fmt.Errorf("links: %q not found under home/", key))
		}
		if val := m.Links[key]; !strings.HasPrefix(val, "~/") || len(val) < 3 {
			errs = append(errs, fmt.Errorf("links: target for %q must start with ~/", key))
		}
	}

	for _, ig := range m.Brew.Ignore {
		if len(strings.Fields(ig)) != 2 {
			errs = append(errs, fmt.Errorf("brew.ignore: %q must be '<kind> <name>' as in the Brewfile", ig))
		}
	}

	seen := map[string]bool{}
	for _, s := range m.Steps {
		if s.ID == "" || s.Check == "" || s.Run == "" {
			errs = append(errs, fmt.Errorf("steps: %q needs id, check and run", s.ID))
		}
		errs = appendDup(errs, seen, "steps", s.ID)
	}
	seen = map[string]bool{}
	for _, s := range m.Services {
		if s.ID == "" || s.Label == "" || s.Start == "" {
			errs = append(errs, fmt.Errorf("services: %q needs id, label and start", s.ID))
		}
		errs = appendDup(errs, seen, "services", s.ID)
	}
	seen = map[string]bool{}
	for _, s := range m.Manual {
		if s.ID == "" || s.How == "" {
			errs = append(errs, fmt.Errorf("manual: %q needs id and how", s.ID))
		}
		errs = appendDup(errs, seen, "manual", s.ID)
	}
	return errors.Join(errs...)
}

func appendDup(errs []error, seen map[string]bool, section, id string) []error {
	if seen[id] {
		return append(errs, fmt.Errorf("%s: duplicate id %q", section, id))
	}
	seen[id] = true
	return errs
}
