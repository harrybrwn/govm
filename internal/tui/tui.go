// Package tui holds the tui models.
package tui

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/harrybrwn/govm"
	"github.com/harrybrwn/govm/internal/nerdfont"
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
	Up, Down,
	ToggleHelp key.Binding
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
			key.WithKeys("k", "up"),
			key.WithHelp("k/"+nerdfont.WeatherDirectionUp, "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("j", "down"),
			key.WithHelp("j/"+nerdfont.WeatherDirectionDown, "down"),
		),
		ToggleHelp: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "toggle help"),
		),
	}
}

type Styles struct{}

func DefaultStyles() Styles {
	return Styles{}
}

func (k *Keys) ShortHelp() []key.Binding {
	return []key.Binding{k.Esc, k.Quit, k.Up, k.Down, k.ToggleHelp}
}

func (k *Keys) FullHelp() [][]key.Binding {
	return [][]key.Binding{}
}
