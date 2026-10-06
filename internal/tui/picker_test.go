package tui

import (
	"strings"
	"testing"

	"github.com/ireydiak/dotfiles/internal/brew"
)

func diff() brew.Diff {
	return brew.Diff{
		Added:   []brew.Entry{{Kind: "brew", Name: "new1"}, {Kind: "cask", Name: "new2"}},
		Removed: []brew.Entry{{Kind: "brew", Name: "gone"}},
	}
}

func TestNewPickerDefaults(t *testing.T) {
	p := NewPicker(diff())
	if len(p.Items) != 3 || !p.Items[0].Checked || !p.Items[1].Checked || p.Items[2].Checked {
		t.Fatalf("items = %+v", p.Items)
	}
	if p.Items[0].Group != "Added" || p.Items[2].Group != "Removed" {
		t.Fatalf("groups = %+v", p.Items)
	}
	keep, drop := p.Selections()
	if len(keep) != 2 || len(drop) != 0 {
		t.Fatalf("selections = %v %v", keep, drop)
	}
}

func TestPickerNavigationAndToggle(t *testing.T) {
	p := NewPicker(diff())
	p = p.Up() // clamps at 0
	if p.Cursor != 0 {
		t.Fatal("cursor should clamp at 0")
	}
	p = p.Down().Down().Down() // clamps at last
	if p.Cursor != 2 {
		t.Fatalf("cursor = %d", p.Cursor)
	}
	p = p.Toggle()
	keep, drop := p.Selections()
	if !drop["brew gone"] || len(keep) != 2 {
		t.Fatalf("toggle on removed item: %v %v", keep, drop)
	}
	p = p.Up().Up().ToggleGroup() // cursor on new1 (checked) -> group becomes unchecked
	keep, _ = p.Selections()
	if len(keep) != 0 {
		t.Fatalf("ToggleGroup should uncheck all added: %v", keep)
	}
	p = p.ToggleGroup()
	keep, _ = p.Selections()
	if len(keep) != 2 {
		t.Fatalf("ToggleGroup again should re-check all added: %v", keep)
	}
}

func TestPickerView(t *testing.T) {
	v := NewPicker(diff()).View(80)
	for _, want := range []string{"Added", "Removed", "[x]", "[ ]", `brew "new1"`, `brew "gone"`} {
		if !strings.Contains(v, want) {
			t.Fatalf("view missing %q:\n%s", want, v)
		}
	}
}
