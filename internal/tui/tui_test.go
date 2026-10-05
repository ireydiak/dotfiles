package tui

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ireydiak/dotfiles/internal/app"
	"github.com/ireydiak/dotfiles/internal/brew"
	"github.com/ireydiak/dotfiles/internal/exec"
	"github.com/ireydiak/dotfiles/internal/result"
	"github.com/ireydiak/dotfiles/internal/status"
)

func testModel(t *testing.T) (Model, *exec.Fake) {
	t.Helper()
	root, home := t.TempDir(), t.TempDir()
	os.MkdirAll(filepath.Join(root, "home"), 0o755)
	os.WriteFile(filepath.Join(root, "home", ".zshrc"), []byte("x"), 0o644)
	os.WriteFile(filepath.Join(root, "manifest.toml"), []byte("[links]\n\".zshrc\" = \"~/.zshrc\"\n"), 0o644)
	f := exec.NewFake()
	a, err := app.New(context.Background(), app.Options{RepoFlag: root, Home: home, Cwd: root, UID: "501", Runner: f})
	if err != nil {
		t.Fatal(err)
	}
	a.Brew.TempDir = root
	m := New(a)
	m, _ = m.resize(100, 40)
	return m, f
}

func key(s string) tea.KeyMsg {
	if s == " " {
		return tea.KeyMsg{Type: tea.KeySpace}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func TestQuitKey(t *testing.T) {
	m, _ := testModel(t)
	_, cmd := m.Update(key("q"))
	if cmd == nil {
		t.Fatal("q should produce a quit command")
	}
	if msg := cmd(); msg == nil {
		t.Fatal("quit command should produce a message")
	} else if _, ok := msg.(tea.QuitMsg); !ok {
		t.Fatalf("expected tea.QuitMsg, got %T", msg)
	}
}

func TestActionKeysSetRunningAndIgnoreOthersWhileRunning(t *testing.T) {
	m, _ := testModel(t)
	for _, k := range []string{"i", "l", "u", "c"} {
		next, cmd := m.Update(key(k))
		nm := next.(Model)
		if !nm.running || cmd == nil {
			t.Fatalf("key %q should start an action", k)
		}
		again, cmd2 := nm.Update(key("l"))
		if cmd2 != nil || again.(Model).action != nm.action {
			t.Fatalf("keys must be ignored while %q runs", nm.action)
		}
	}
}

func TestOutputAndDoneMessages(t *testing.T) {
	m, _ := testModel(t)
	next, _ := m.Update(key("l"))
	nm := next.(Model)
	next, cmd := nm.Update(outputMsg("linked .zshrc"))
	nm = next.(Model)
	if len(nm.lines) != 1 || nm.lines[0] != "linked .zshrc" || cmd == nil {
		t.Fatalf("output not appended or no follow-up wait: %+v", nm.lines)
	}
	next, cmd = nm.Update(actionDone{Name: "link", Summary: result.Summary{{Section: "link", ID: ".zshrc", Status: result.Ok, Detail: "linked"}}})
	nm = next.(Model)
	if nm.running || cmd == nil {
		t.Fatal("done should clear running and schedule a refresh")
	}
	if !strings.Contains(strings.Join(nm.lines, "\n"), "1 ok, 0 skipped, 0 failed") {
		t.Fatalf("summary not appended: %v", nm.lines)
	}
}

func TestExportFlowScreens(t *testing.T) {
	m, _ := testModel(t)
	next, _ := m.Update(tapsMsg{Taps: []string{"asmvik/formulae"}})
	nm := next.(Model)
	if nm.screen != screenConfirmTrust {
		t.Fatalf("screen = %v, want confirm", nm.screen)
	}
	next, _ = nm.Update(key("n"))
	if next.(Model).screen != screenDashboard {
		t.Fatal("n should return to the dashboard")
	}
	next, _ = m.Update(tapsMsg{})
	if _, cmd := next.(Model).Update(tapsMsg{}); cmd == nil {
		t.Fatal("no untrusted taps should go straight to the diff command")
	}
	plan := &app.ExportPlan{Diff: brew.Diff{Added: []brew.Entry{{Kind: "brew", Name: "new"}}}}
	next, _ = m.Update(exportDiffMsg{Plan: plan})
	nm = next.(Model)
	if nm.screen != screenPicker || len(nm.picker.Items) != 1 {
		t.Fatalf("screen = %v items=%d", nm.screen, len(nm.picker.Items))
	}
	next, _ = nm.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if next.(Model).screen != screenDashboard {
		t.Fatal("esc should cancel the picker")
	}
	next, cmd := nm.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if next.(Model).screen != screenDashboard || !next.(Model).running || cmd == nil {
		t.Fatal("enter should start the export action")
	}
}

func TestViewShowsSections(t *testing.T) {
	m, _ := testModel(t)
	next, _ := m.Update(reportMsg(status.Report{}))
	v := next.(Model).View()
	for _, want := range []string{"Packages", "Links", "Services", "Manual", "i install", "u update", "q quit"} {
		if !strings.Contains(v, want) {
			t.Fatalf("view missing %q:\n%s", want, v)
		}
	}
}
