// Package tui is the Bubble Tea dashboard. It renders app results and holds no logic of its own.
package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/ireydiak/dotfiles/internal/brew"
)

type PickerItem struct {
	Entry   brew.Entry
	Group   string
	Checked bool
}

type Picker struct {
	Items  []PickerItem
	Cursor int
}

// NewPicker lists added entries first (checked) then removed ones (unchecked).
func NewPicker(d brew.Diff) Picker {
	var p Picker
	for _, e := range d.Added {
		p.Items = append(p.Items, PickerItem{Entry: e, Group: "Added", Checked: true})
	}
	for _, e := range d.Removed {
		p.Items = append(p.Items, PickerItem{Entry: e, Group: "Removed"})
	}
	return p
}

func (p Picker) Up() Picker {
	if p.Cursor > 0 {
		p.Cursor--
	}
	return p
}

func (p Picker) Down() Picker {
	if p.Cursor < len(p.Items)-1 {
		p.Cursor++
	}
	return p
}

func (p Picker) Toggle() Picker {
	if len(p.Items) == 0 {
		return p
	}
	items := append([]PickerItem(nil), p.Items...)
	items[p.Cursor].Checked = !items[p.Cursor].Checked
	p.Items = items
	return p
}

// ToggleGroup sets every item in the cursor's group to the opposite of the cursor item.
func (p Picker) ToggleGroup() Picker {
	if len(p.Items) == 0 {
		return p
	}
	items := append([]PickerItem(nil), p.Items...)
	group, target := items[p.Cursor].Group, !items[p.Cursor].Checked
	for i := range items {
		if items[i].Group == group {
			items[i].Checked = target
		}
	}
	p.Items = items
	return p
}

// Selections returns the chosen additions and removals keyed by Entry.Key().
func (p Picker) Selections() (keepAdded, dropRemoved map[string]bool) {
	keepAdded, dropRemoved = map[string]bool{}, map[string]bool{}
	for _, it := range p.Items {
		if !it.Checked {
			continue
		}
		if it.Group == "Added" {
			keepAdded[it.Entry.Key()] = true
		} else {
			dropRemoved[it.Entry.Key()] = true
		}
	}
	return keepAdded, dropRemoved
}

var (
	groupStyle  = lipgloss.NewStyle().Bold(true).Underline(true)
	cursorStyle = lipgloss.NewStyle().Reverse(true)
	dimStyle    = lipgloss.NewStyle().Faint(true)
)

func (p Picker) View(width int) string {
	var b strings.Builder
	b.WriteString("Export: choose what to record in the Brewfile\n\n")
	lastGroup := ""
	for i, it := range p.Items {
		if it.Group != lastGroup {
			hint := "installed, not in Brewfile"
			if it.Group == "Removed" {
				hint = "in Brewfile, no longer installed; check to drop"
			}
			b.WriteString(groupStyle.Render(it.Group) + dimStyle.Render("  "+hint) + "\n")
			lastGroup = it.Group
		}
		box := "[ ]"
		if it.Checked {
			box = "[x]"
		}
		line := "  " + box + " " + it.Entry.Line()
		if i == p.Cursor {
			line = cursorStyle.Render(line)
		}
		b.WriteString(line + "\n")
	}
	if len(p.Items) == 0 {
		b.WriteString(dimStyle.Render("Brewfile is up to date") + "\n")
	}
	b.WriteString("\n" + dimStyle.Render("space toggle · a toggle group · enter write · esc cancel"))
	return lipgloss.NewStyle().Width(width).Render(b.String())
}
