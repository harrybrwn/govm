package tui

import (
	"fmt"
	"log/slog"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

type Confirm struct {
	Prompt    string
	PromptFn  func() string
	Keys      ConfirmKeys
	Yes       bool
	completed bool
}

type ConfirmKeys struct {
	Keys
	Yes, No key.Binding
}

func DefaultConfirmKeys() ConfirmKeys {
	return ConfirmKeys{
		Keys: DefaultKeys(),
		Yes: key.NewBinding(
			key.WithKeys("Y", "y", "enter"),
		),
		No: key.NewBinding(
			key.WithKeys("N", "n"),
		),
	}
}

func (c *Confirm) Init() tea.Cmd {
	logger := slog.With("model", fmt.Sprintf("%T", c))
	logger.Info("init")
	return nil
}

func (c *Confirm) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		var cmd tea.Cmd
		switch {
		case key.Matches(msg, c.Keys.Quit, c.Keys.Esc):
			return c, tea.Quit
		case key.Matches(msg, c.Keys.Yes):
			c.Yes = true
			cmd = NextChainedModel
		case key.Matches(msg, c.Keys.No):
			c.Yes = false
			cmd = PrevChainedModel
		default:
			slog.Error("confirm: unknown key", "key", msg.String())
			cmd = tea.Quit
		}
		c.completed = true
		return c, cmd
	default:
		return c, nil
	}
}

func (c *Confirm) View() tea.View {
	var b strings.Builder
	if c.PromptFn == nil {
		fmt.Fprintf(&b, "%s: (Y/n)\n", c.Prompt)
	} else {
		fmt.Fprintf(&b, "%s: (Y/n)\n", c.PromptFn())
	}
	return tea.NewView(b.String())
}

func (c *Confirm) ChainView() tea.View {
	if !c.completed {
		return tea.View{}
	}
	var b strings.Builder
	if c.PromptFn == nil {
		b.WriteString(c.Prompt)
	} else {
		b.WriteString(c.PromptFn())
	}
	b.Write([]byte{':', ' '})
	if c.Yes {
		b.WriteString("yes")
	} else {
		b.WriteString("no")
	}
	return tea.NewView(b.String())
}
