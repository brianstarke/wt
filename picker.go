package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// worktreeItem adapts a Worktree to bubbles/list.
type worktreeItem struct {
	wt  Worktree
	dir string // worktrees dir, for display shortening
}

func (i worktreeItem) FilterValue() string {
	return i.wt.Name() + " " + i.wt.Path
}

type delegate struct{ dir string }

func (d delegate) Height() int  { return 2 }
func (d delegate) Spacing() int { return 0 }
func (d delegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd {
	return nil
}

func (d delegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	i, ok := item.(worktreeItem)
	if !ok {
		return
	}
	name := i.wt.Name()
	path := i.wt.Path
	if rel, err := filepath.Rel(d.dir, path); err == nil && !strings.HasPrefix(rel, "..") {
		path = d.dir + string(os.PathSeparator) + rel
	}
	head := i.wt.Head
	if len(head) > 8 {
		head = head[:8]
	}

	nameLine := branchStyle.Render(name)
	pathLine := pathStyle.Render(path) + "  " + headStyle.Render(head)
	if index == m.Index() {
		nameLine = lipgloss.NewStyle().
			Foreground(accent2).Bold(true).
			PaddingLeft(1).
			BorderStyle(lipgloss.ThickBorder()).
			BorderLeft(true).
			BorderForeground(accent).
			Render(" " + name)
		pathLine = lipgloss.NewStyle().PaddingLeft(4).Render(pathLine)
	} else {
		nameLine = lipgloss.NewStyle().PaddingLeft(3).Render(nameLine)
		pathLine = lipgloss.NewStyle().PaddingLeft(4).Render(pathLine)
	}
	fmt.Fprintf(w, "%s\n%s", nameLine, pathLine)
}

func newPicker(title string, wts []Worktree, dir string) list.Model {
	items := make([]list.Item, len(wts))
	for i, w := range wts {
		items[i] = worktreeItem{wt: w, dir: dir}
	}
	l := list.New(items, delegate{dir: dir}, 0, 0)
	l.Title = title
	l.Styles.Title = titleStyle
	l.Styles.TitleBar = lipgloss.NewStyle()
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.FilterInput.PromptStyle = lipgloss.NewStyle().Foreground(accent2)
	l.FilterInput.Cursor.Style = lipgloss.NewStyle().Foreground(accent2)
	return l
}

type pickerModel struct {
	list     list.Model
	choice   *Worktree
	quitting bool
}

func (m pickerModel) Init() tea.Cmd { return nil }

func (m pickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h, v := lipgloss.NewStyle().Margin(1, 2).GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)
	case tea.KeyMsg:
		if m.list.SettingFilter() {
			break
		}
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.quitting = true
			return m, tea.Quit
		case "enter":
			if i, ok := m.list.SelectedItem().(worktreeItem); ok {
				wt := i.wt
				m.choice = &wt
			}
			return m, tea.Quit
		}
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m pickerModel) View() string {
	return lipgloss.NewStyle().Margin(1, 2).Render(m.list.View())
}

// pickWorktree runs an interactive picker on stderr (so stdout stays clean
// for command substitution). Returns nil if the user aborted.
func pickWorktree(title string, wts []Worktree, dir string) *Worktree {
	if len(wts) == 0 {
		return nil
	}
	m := pickerModel{list: newPicker(title, wts, dir)}
	p := tea.NewProgram(m, tea.WithOutput(os.Stderr))
	final, err := p.Run()
	if err != nil {
		return nil
	}
	if pm, ok := final.(pickerModel); ok {
		return pm.choice
	}
	return nil
}

// --- multi-select picker ---

type multiModel struct {
	list     list.Model
	selected map[string]bool
	done     bool
	aborted  bool
}

func (m multiModel) Init() tea.Cmd { return nil }

func (m multiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h, v := lipgloss.NewStyle().Margin(1, 2).GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)
	case tea.KeyMsg:
		if m.list.SettingFilter() {
			break
		}
		switch msg.String() {
		case "ctrl+c", "esc":
			m.aborted = true
			return m, tea.Quit
		case " ":
			if i, ok := m.list.SelectedItem().(worktreeItem); ok {
				p := i.wt.Path
				m.selected[p] = !m.selected[p]
			}
			return m, nil
		case "enter":
			m.done = true
			return m, tea.Quit
		}
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m multiModel) View() string {
	return lipgloss.NewStyle().Margin(1, 2).Render(m.list.View()) +
		"\n" + lipgloss.NewStyle().Margin(0, 2).Render(
			dimStyle.Render("space: toggle · enter: confirm · esc: cancel"))
}

// multiDelegate renders a checkbox per row.
type multiDelegate struct {
	delegate
	selected func(path string) bool
}

func (d multiDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	i, ok := item.(worktreeItem)
	if !ok {
		return
	}
	box := dimStyle.Render("[ ]")
	if d.selected(i.wt.Path) {
		box = okStyle.Render("[✓]")
	}
	var sb strings.Builder
	d.delegate.Render(&sb, m, index, item)
	lines := strings.Split(sb.String(), "\n")
	fmt.Fprintf(w, "%s %s", box, strings.Join(lines, "\n   "))
}

func multiPickWorktrees(title string, wts []Worktree, dir string) []Worktree {
	if len(wts) == 0 {
		return nil
	}
	selected := map[string]bool{}
	items := make([]list.Item, len(wts))
	for i, w := range wts {
		items[i] = worktreeItem{wt: w, dir: dir}
	}
	d := multiDelegate{
		delegate: delegate{dir: dir},
		selected: func(p string) bool { return selected[p] },
	}
	l := list.New(items, d, 0, 0)
	l.Title = title
	l.Styles.Title = titleStyle
	l.Styles.TitleBar = lipgloss.NewStyle()
	m := multiModel{list: l, selected: selected}
	p := tea.NewProgram(m, tea.WithOutput(os.Stderr))
	final, err := p.Run()
	if err != nil {
		return nil
	}
	fm, ok := final.(multiModel)
	if !ok || fm.aborted {
		return nil
	}
	var out []Worktree
	for _, w := range wts {
		if fm.selected[w.Path] {
			out = append(out, w)
		}
	}
	return out
}

// confirm asks a y/N question on stderr.
func confirm(prompt string) bool {
	fmt.Fprintf(os.Stderr, "%s %s ",
		warnStyle.Render("?"),
		lipgloss.NewStyle().Bold(true).Render(prompt+" [y/N]"))
	var reply string
	fmt.Scanln(&reply)
	return strings.HasPrefix(strings.ToLower(reply), "y")
}
