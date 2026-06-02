package tui

import tea "charm.land/bubbletea/v2"

type SetParentModelMsg struct {
	Parent tea.Model
}

func SetParentModel(model tea.Model) tea.Cmd {
	return func() tea.Msg {
		return SetParentModelMsg{Parent: model}
	}
}
