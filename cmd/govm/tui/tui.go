// Package tui holds the tui models.
package tui

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/harrybrwn/govm"
)

func Run(model tea.Model) error {
	_, err := tea.NewProgram(model).Run()
	return err
}

type Program struct {
	VersionSelectionMenu Menu[govm.Version]
}

func New() *Program {
	return &Program{}
}

func (p *Program) Run() error {
	return Run(&p.VersionSelectionMenu)
}

type Keys struct {
	Esc, Quit,
	Up, Down key.Binding
}

func DefaultKeys() Keys {
	return Keys{
		Esc: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "escape/back"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q/ctrl+c", "quit"),
		),
		Up: key.NewBinding(
			key.WithKeys("k"),
		),
		Down: key.NewBinding(
			key.WithKeys("j"),
		),
	}
}

type Styles struct{}

func DefaultStyles() Styles {
	return Styles{}
}
