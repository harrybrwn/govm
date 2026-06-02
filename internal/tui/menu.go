package tui

import (
	"fmt"
	"log/slog"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/harrybrwn/govm/internal/nerdfont"
)

type MenuOption[T any] struct {
	ID       string
	Display  string
	Value    T
	selected bool
}

type Menu[T any] struct {
	Options     []MenuOption[T]
	OnSelect    func(index int, option *MenuOption[T])
	Prompt      string
	Keys        MenuKeys
	Styles      MenuStyles
	QuitCmd     tea.Cmd
	hasSelected bool
	height      int
	// cursor      int
	selected int // currently selected entry index
	cursor   int // cursor's y-axis terminal cell position
	help     *Help
}

type MenuKeys struct {
	Keys
	Select,
	PageUp,
	PageDown,
	GotoTop,
	GotoBottom key.Binding
}

func DefaultMenuKeys() MenuKeys {
	return MenuKeys{
		Keys: DefaultKeys(),
		Select: key.NewBinding(
			key.WithKeys("enter", " "),
			key.WithHelp("enter/space", "select current option"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("u", "ctrl+u"),
			key.WithHelp("u", "page up"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("d", "ctrl+d"),
			key.WithHelp("d", "page down"),
		),
		GotoTop: key.NewBinding(
			key.WithKeys("g"),
			key.WithHelp("g", "goto top"),
		),
		GotoBottom: key.NewBinding(
			key.WithKeys("G"),
			key.WithHelp("G", "goto bottom"),
		),
	}
}

func (k *MenuKeys) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Quit, k.Esc, k.ToggleHelp, k.Select},
		{k.Up, k.Down, k.PageUp, k.PageDown},
		{k.GotoTop, k.GotoBottom},
	}
}

type MenuStyles struct {
	// View renders the entire menu. Good for borders or background colors.
	View,
	Cursor,
	Selected,
	Prompt lipgloss.Style
}

func DefaultMenuStyles() MenuStyles {
	selectedBg := lipgloss.ANSIColor(55)
	return MenuStyles{
		View: lipgloss.NewStyle().
			Bold(false),
		Cursor: lipgloss.NewStyle().
			// Background(lipgloss.ANSIColor(55)).
			Background(selectedBg).
			Foreground(lipgloss.ANSIColor(198)).
			Bold(true),
		Selected: lipgloss.NewStyle().
			// Background(lipgloss.Color("55")).
			Background(selectedBg).
			Foreground(lipgloss.Color("7")).
			Bold(false),
		Prompt: lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Bold(true),
	}
}

func (m *Menu[T]) Selected() bool { return m.hasSelected }

func (m *Menu[T]) GetSelected() []*MenuOption[T] {
	selected := make([]*MenuOption[T], 0)
	for i, o := range m.Options {
		if o.selected {
			selected = append(selected, &m.Options[i])
		}
	}
	return selected
}

func (m *Menu[T]) Init() tea.Cmd {
	m.help = NewHelp(&m.Keys)
	logger := slog.With("model", fmt.Sprintf("%T", m))
	logger.Info("init")
	for i, o := range m.Options {
		if len(o.Display) != 0 {
			continue
		}
		if len(o.ID) > 0 {
			m.Options[i].Display = o.ID
		} else {
			switch s := any(o.Value).(type) {
			case interface{ String() string }:
				m.Options[i].Display = s.String()
			case error:
				m.Options[i].Display = s.Error()
			default:
				m.Options[i].Display = fmt.Sprintf("%v", o.Value)
			}
		}
	}
	if m.QuitCmd == nil {
		m.QuitCmd = tea.Quit
	}
	return nil
}

func (m *Menu[T]) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		slog.Info("window size", "width", msg.Width, "height", msg.Height)
		m.height = msg.Height
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.Keys.Quit, m.Keys.Esc):
			return m, tea.Quit
		case key.Matches(msg, m.Keys.Up):
			m.up(1)
		case key.Matches(msg, m.Keys.Down):
			m.down(1)
		case key.Matches(msg, m.Keys.PageUp):
			m.up(m.height / 2)
		case key.Matches(msg, m.Keys.PageDown):
			m.down(m.height / 2)
		case key.Matches(msg, m.Keys.GotoTop):
			m.cursor = 0
			m.selected = 0
		case key.Matches(msg, m.Keys.GotoBottom):
			m.cursor = m.getHeight() - 1 - m.promptHeight() - m.help.Height()
			m.selected = len(m.Options) - 1
		case key.Matches(msg, m.Keys.Select):
			slog.Info("menu selection", "cursor", m.cursor)
			if m.OnSelect != nil {
				m.OnSelect(m.selected, &m.Options[m.selected])
			}
			m.Options[m.selected].selected = true
			m.hasSelected = true // mark the menu as having at least one selection
			return m, m.QuitCmd
		case key.Matches(msg, m.Keys.ToggleHelp):
			m.help.Toggle()
		}
	}
	return m, nil
}

func (m *Menu[T]) up(n int) {
	m.cursor = max(m.cursor-n, 0)
	m.selected = max(m.selected-n, 0)
}

func (m *Menu[T]) down(n int) {
	bh := m.borderHeight()
	height := m.height - bh - m.promptHeight() - m.help.Height()
	if bh == 0 {
		height--
	}
	m.cursor = min(m.cursor+n, height, len(m.Options)-1)
	m.selected = min(m.selected+n, len(m.Options)-1)
}

func (m *Menu[T]) View() tea.View {
	var (
		b      strings.Builder
		prompt string
	)
	if len(m.Prompt) > 0 {
		prompt = m.Styles.Prompt.Render(m.Prompt)
		fmt.Fprintf(&b, "%s\n", prompt)
	}

	var (
		h     = m.getHeight() - m.help.Height() - 1 - lipgloss.Height(prompt)
		start = max(m.selected-m.cursor, 0)
		end   = min(start+h, len(m.Options)-1)
	)
	for i := start; i <= end; i++ {
		option := m.Options[i]
		if i == m.selected {
			// b.WriteString(m.Styles.Cursor.Render(nerdfont.CodArrowRight))
			b.WriteString(m.Styles.Cursor.Render(nerdfont.CodChevronRight))
			b.WriteString(m.Styles.Selected.Render(" " + option.Display))
		} else {
			fmt.Fprintf(&b, "  %s", option.Display)
		}
		if i < end {
			b.WriteByte('\n')
		}
	}

	view := tea.NewView(lipgloss.JoinVertical(
		lipgloss.Top,
		m.Styles.View.Render(b.String()),
		m.help.View(),
	))
	if len(m.Options) > h {
		view.AltScreen = true
	}
	return view
}

func (m *Menu[T]) promptHeight() int {
	if len(m.Prompt) == 0 {
		return 0
	}
	return lipgloss.Height(m.Styles.Prompt.Render(m.Prompt))
}

func (m *Menu[T]) getHeight() int {
	return m.height - m.borderHeight()
}

func (m *Menu[T]) borderHeight() int {
	return m.Styles.View.GetBorderBottomSize() + m.Styles.View.GetBorderTopSize()
}

func (m *Menu[T]) ChainView() tea.View {
	selected := m.GetSelected()
	if len(selected) == 0 {
		return tea.View{}
	}
	return tea.NewView(fmt.Sprintf("Selected: %v", selected[0].Display))
}

var _ tea.Model = (*Menu[any])(nil)
