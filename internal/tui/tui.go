package tui

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ireydiak/dotfiles/internal/app"
	"github.com/ireydiak/dotfiles/internal/link"
	"github.com/ireydiak/dotfiles/internal/manual"
	"github.com/ireydiak/dotfiles/internal/result"
	"github.com/ireydiak/dotfiles/internal/status"
)

type screen int

const (
	screenDashboard screen = iota
	screenConfirmTrust
	screenPicker
)

type (
	reportMsg  status.Report
	outputMsg  string
	actionDone struct {
		Name    string
		Summary result.Summary
	}
	tapsMsg struct {
		Taps []string
		Err  error
	}
	exportDiffMsg struct {
		Plan *app.ExportPlan
		Err  error
	}
)

type Model struct {
	app     *app.App
	report  status.Report
	loaded  bool
	screen  screen
	running bool
	action  string
	lines   []string
	out     viewport.Model
	outCh   chan string
	doneCh  chan actionDone
	taps    []string
	plan    *app.ExportPlan
	picker  Picker
	err     string
	width   int
	height  int
}

func New(a *app.App) Model {
	return Model{app: a, out: viewport.New(80, 10)}
}

// Run starts the dashboard in the alternate screen.
func Run(a *app.App) error {
	_, err := tea.NewProgram(New(a), tea.WithAltScreen()).Run()
	return err
}

func (m Model) Init() tea.Cmd { return m.refresh() }

func (m Model) refresh() tea.Cmd {
	a := m.app
	return func() tea.Msg { return reportMsg(a.Status()) }
}

func (m Model) resize(w, h int) (Model, tea.Cmd) {
	m.width, m.height = w, h
	m.out.Width = w
	m.out.Height = max(5, h/3)
	return m, nil
}

// lineWriter turns a stream of bytes into one channel message per line.
type lineWriter struct {
	ch  chan<- string
	buf []byte
}

func (w *lineWriter) Write(p []byte) (int, error) {
	w.buf = append(w.buf, p...)
	for {
		i := bytes.IndexByte(w.buf, '\n')
		if i < 0 {
			return len(p), nil
		}
		w.ch <- string(w.buf[:i])
		w.buf = w.buf[i+1:]
	}
}

func (w *lineWriter) flush() {
	if len(w.buf) > 0 {
		w.ch <- string(w.buf)
		w.buf = nil
	}
}

func (m Model) start(name string, fn func(io.Writer) result.Summary) (Model, tea.Cmd) {
	m.running, m.action, m.lines, m.err = true, name, nil, ""
	m.outCh = make(chan string, 64)
	m.doneCh = make(chan actionDone, 1)
	outCh, doneCh := m.outCh, m.doneCh
	go func() {
		w := &lineWriter{ch: outCh}
		s := fn(w)
		w.flush()
		close(outCh)
		doneCh <- actionDone{Name: name, Summary: s}
	}()
	return m, tea.Batch(waitLine(outCh), waitDone(doneCh))
}

func waitLine(ch <-chan string) tea.Cmd {
	return func() tea.Msg {
		l, ok := <-ch
		if !ok {
			return nil
		}
		return outputMsg(l)
	}
}

func waitDone(ch <-chan actionDone) tea.Cmd {
	return func() tea.Msg { return <-ch }
}

func (m Model) exportDiff() tea.Cmd {
	a := m.app
	return func() tea.Msg {
		plan, err := a.ExportDiff()
		return exportDiffMsg{Plan: plan, Err: err}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m.resize(msg.Width, msg.Height)
	case reportMsg:
		m.report, m.loaded = status.Report(msg), true
		return m, nil
	case outputMsg:
		m.lines = append(m.lines, string(msg))
		m.out.SetContent(strings.Join(m.lines, "\n"))
		m.out.GotoBottom()
		return m, waitLine(m.outCh)
	case actionDone:
		var buf bytes.Buffer
		msg.Summary.Print(&buf)
		m.lines = append(m.lines, strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")...)
		m.out.SetContent(strings.Join(m.lines, "\n"))
		m.out.GotoBottom()
		m.running = false
		return m, m.refresh()
	case tapsMsg:
		if msg.Err != nil {
			m.err = msg.Err.Error()
			return m, nil
		}
		if len(msg.Taps) == 0 {
			return m, m.exportDiff()
		}
		m.taps, m.screen = msg.Taps, screenConfirmTrust
		return m, nil
	case exportDiffMsg:
		if msg.Err != nil {
			m.err, m.screen = msg.Err.Error(), screenDashboard
			return m, nil
		}
		m.plan, m.picker, m.screen = msg.Plan, NewPicker(msg.Plan.Diff), screenPicker
		return m, nil
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m Model) handleKey(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := k.String()
	if key == "ctrl+c" || (key == "q" && m.screen == screenDashboard) {
		return m, tea.Quit
	}
	switch m.screen {
	case screenConfirmTrust:
		switch key {
		case "y":
			a, taps := m.app, m.taps
			m.screen = screenDashboard
			return m, func() tea.Msg {
				var buf bytes.Buffer
				if s := a.TrustTaps(taps, &buf); s.Failed() {
					return exportDiffMsg{Err: fmt.Errorf("trusting taps failed: %s", result.LastLine(buf.String()))}
				}
				plan, err := a.ExportDiff()
				return exportDiffMsg{Plan: plan, Err: err}
			}
		case "n", "esc":
			m.screen = screenDashboard
		}
		return m, nil
	case screenPicker:
		switch key {
		case "up", "k":
			m.picker = m.picker.Up()
		case "down", "j":
			m.picker = m.picker.Down()
		case " ":
			m.picker = m.picker.Toggle()
		case "a":
			m.picker = m.picker.ToggleGroup()
		case "esc":
			m.screen = screenDashboard
		case "enter":
			m.screen = screenDashboard
			a, plan := m.app, m.plan
			keep, drop := m.picker.Selections()
			return m.start("export", func(w io.Writer) result.Summary {
				if err := a.ExportWrite(plan, keep, drop, w); err != nil {
					return result.Summary{{Section: "brew", ID: "Brewfile", Status: result.Failed, Detail: err.Error()}}
				}
				return result.Summary{{Section: "brew", ID: "Brewfile", Status: result.Ok, Detail: fmt.Sprintf("%d added, %d dropped", len(keep), len(drop))}}
			})
		}
		return m, nil
	}
	if m.running {
		return m, nil
	}
	a := m.app
	switch key {
	case "i":
		return m.start("install", a.Install)
	case "l":
		return m.start("link", a.Link)
	case "u":
		return m.start("update", a.Update)
	case "c":
		return m.start("commit", func(w io.Writer) result.Summary { return a.Commit("", true, w) })
	case "e":
		return m, func() tea.Msg {
			taps, err := a.UntrustedTaps()
			return tapsMsg{Taps: taps, Err: err}
		}
	case "r":
		return m, m.refresh()
	}
	return m, nil
}

var (
	titleStyle   = lipgloss.NewStyle().Bold(true)
	sectionStyle = lipgloss.NewStyle().Bold(true).MarginTop(1)
	okStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	badStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	warnStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	footerStyle  = lipgloss.NewStyle().Faint(true).MarginTop(1)
)

func (m Model) View() string {
	switch m.screen {
	case screenConfirmTrust:
		return "Trust these taps so brew bundle can see their packages?\n\n  " +
			strings.Join(m.taps, "\n  ") + "\n\n" + footerStyle.Render("y yes · n no")
	case screenPicker:
		return m.picker.View(m.width)
	}

	var b strings.Builder
	b.WriteString(titleStyle.Render("dot  "+m.app.Root) + "\n")
	if !m.loaded {
		b.WriteString("\nloading status…\n")
	} else {
		m.writeReport(&b)
	}
	if m.err != "" {
		b.WriteString("\n" + badStyle.Render("error: "+m.err) + "\n")
	}
	if len(m.lines) > 0 {
		label := m.action
		if m.running {
			label += " (running…)"
		}
		b.WriteString(sectionStyle.Render(label) + "\n" + m.out.View() + "\n")
	}
	b.WriteString(footerStyle.Render("i install · l link · e export · u update · c commit · r refresh · q quit"))
	return b.String()
}

func (m Model) writeReport(b *strings.Builder) {
	r := m.report
	b.WriteString(sectionStyle.Render("Packages") + "\n")
	switch {
	case r.Packages.Err != "":
		b.WriteString("  " + badStyle.Render("error") + "  " + r.Packages.Err + "\n")
	case len(r.Packages.Missing) == 0 && len(r.Packages.Unrecorded) == 0:
		b.WriteString("  " + okStyle.Render("✓") + " all Brewfile entries installed, nothing unrecorded\n")
	}
	for _, e := range r.Packages.Missing {
		b.WriteString("  " + badStyle.Render("missing") + "     " + e.Line() + "\n")
	}
	for _, e := range r.Packages.Unrecorded {
		b.WriteString("  " + warnStyle.Render("unrecorded") + "  " + e.Line() + "\n")
	}

	b.WriteString(sectionStyle.Render("Links") + "\n")
	for _, it := range r.Links {
		st := okStyle.Render(fmt.Sprintf("%-13s", it.State))
		if it.State != link.Ok {
			st = badStyle.Render(fmt.Sprintf("%-13s", it.State))
		}
		detail := ""
		if it.Detail != "" {
			detail = "  (" + it.Detail + ")"
		}
		b.WriteString("  " + st + " " + it.Key + detail + "\n")
	}

	b.WriteString(sectionStyle.Render("Services") + "\n")
	for _, s := range r.Services {
		st := okStyle.Render(fmt.Sprintf("%-13s", "loaded"))
		if !s.Loaded {
			st = badStyle.Render(fmt.Sprintf("%-13s", "not loaded"))
		}
		b.WriteString("  " + st + " " + s.ID + "\n")
	}

	b.WriteString(sectionStyle.Render("Manual") + "\n")
	for _, it := range r.Manual {
		st := okStyle.Render(fmt.Sprintf("%-13s", it.State))
		how := ""
		if it.State != manual.Done {
			st = warnStyle.Render(fmt.Sprintf("%-13s", it.State))
			how = "  → " + it.How
		}
		b.WriteString("  " + st + " " + it.ID + how + "\n")
	}

	b.WriteString(sectionStyle.Render("Repo") + "\n")
	switch {
	case r.Repo.Err != "":
		b.WriteString("  " + badStyle.Render("error") + "  " + r.Repo.Err + "\n")
	case !r.Repo.Dirty:
		b.WriteString("  " + okStyle.Render("clean") + "\n")
	default:
		b.WriteString("  " + warnStyle.Render(fmt.Sprintf("%d uncommitted", len(r.Repo.Changes))) + "  " + strings.Join(r.Repo.Changes, ", ") + "\n")
	}
}
