package manifest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeRepo creates <tmp>/manifest.toml plus the given files under <tmp>/home.
func writeRepo(t *testing.T, manifest string, homeFiles ...string) string {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "manifest.toml"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, f := range homeFiles {
		p := filepath.Join(root, "home", f)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

const valid = `
[links]
".zshrc" = "~/.zshrc"
"Library/Application Support/lazygit/config.yml" = "~/Library/Application Support/lazygit/config.yml"

[[steps]]
id = "omz"
check = "test -d ~/.oh-my-zsh"
run = "echo install"
update = "echo update"

[[steps]]
id = "nvm"
check = "test -s ~/.nvm/nvm.sh"
run = "echo install"

[[services]]
id = "yabai"
label = "com.koekeishiya.yabai"
start = "yabai --start-service"

[[manual]]
id = "xcode"
check = "xcode-select -p"
how = "xcode-select --install"

[[manual]]
id = "accessibility"
how = "System Settings"
`

func TestLoadValid(t *testing.T) {
	root := writeRepo(t, valid, ".zshrc", "Library/Application Support/lazygit/config.yml")
	m, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if got := m.LinkKeys(); len(got) != 2 || got[0] != ".zshrc" {
		t.Fatalf("LinkKeys = %v", got)
	}
	if len(m.Steps) != 2 || m.Steps[0].Update != "echo update" || m.Steps[1].Update != "" {
		t.Fatalf("steps = %+v", m.Steps)
	}
	if len(m.Services) != 1 || m.Services[0].Label != "com.koekeishiya.yabai" {
		t.Fatalf("services = %+v", m.Services)
	}
	if len(m.Manual) != 2 || m.Manual[1].Check != "" {
		t.Fatalf("manual = %+v", m.Manual)
	}
}

func TestParseRejectsUnknownKey(t *testing.T) {
	_, err := Parse(strings.NewReader("[links]\n\".zshrc\" = \"~/.zshrc\"\n[[steps]]\nid = \"a\"\ncheck = \"true\"\nrun = \"true\"\nrunn = \"typo\"\n"))
	if err == nil || !strings.Contains(err.Error(), "unknown key") {
		t.Fatalf("expected unknown key error, got %v", err)
	}
}

func TestValidateErrors(t *testing.T) {
	cases := map[string]struct {
		manifest string
		files    []string
		want     string
	}{
		"missing source":   {"[links]\n\".zshrc\" = \"~/.zshrc\"\n", nil, "not found under home/"},
		"bad target":       {"[links]\n\".zshrc\" = \"/etc/zshrc\"\n", []string{".zshrc"}, "must start with ~/"},
		"dotdot key":       {"[links]\n\"../x\" = \"~/x\"\n", nil, "clean relative path"},
		"absolute key":     {"[links]\n\"/x\" = \"~/x\"\n", nil, "clean relative path"},
		"dup step id":      {"[[steps]]\nid=\"a\"\ncheck=\"t\"\nrun=\"t\"\n[[steps]]\nid=\"a\"\ncheck=\"t\"\nrun=\"t\"\n", nil, "duplicate"},
		"step no run":      {"[[steps]]\nid=\"a\"\ncheck=\"t\"\n", nil, "needs id, check and run"},
		"service no label": {"[[services]]\nid=\"a\"\nstart=\"t\"\n", nil, "needs id, label and start"},
		"manual no how":    {"[[manual]]\nid=\"a\"\n", nil, "needs id and how"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			root := writeRepo(t, c.manifest, c.files...)
			_, err := Load(root)
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("want error containing %q, got %v", c.want, err)
			}
		})
	}
}

func TestExpand(t *testing.T) {
	if got := Expand("~/.config/nvim", "/Users/x"); got != "/Users/x/.config/nvim" {
		t.Fatalf("Expand = %q", got)
	}
	if got := Expand("/abs", "/Users/x"); got != "/abs" {
		t.Fatalf("Expand should leave non-tilde paths alone, got %q", got)
	}
}

func TestBrewIgnore(t *testing.T) {
	m, err := Parse(strings.NewReader("[brew]\nignore = [\"go example.com/local/tool\", \"npm yarn\"]\n"))
	if err != nil || len(m.Brew.Ignore) != 2 || m.Brew.Ignore[0] != "go example.com/local/tool" {
		t.Fatalf("Brew.Ignore = %v %v", m, err)
	}
	root := writeRepo(t, "[brew]\nignore = [\"tool\"]\n")
	if _, err := Load(root); err == nil || !strings.Contains(err.Error(), "<kind> <name>") {
		t.Fatalf("ignore entries must be '<kind> <name>', got %v", err)
	}
}
