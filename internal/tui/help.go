package tui

import (
	"charm.land/bubbles/v2/help"
	"charm.land/lipgloss/v2"
)

type Help struct {
	model  help.Model
	keys   help.KeyMap
	cached string
	height int
	width  int
}

func NewHelp(keys help.KeyMap) *Help {
	model := help.New()
	cached := model.View(keys)
	return &Help{
		model:  model,
		keys:   keys,
		cached: cached,
		height: lipgloss.Height(cached),
		width:  lipgloss.Width(cached),
	}
}

func (h *Help) Height() int {
	return h.height
}
func (h *Help) Width() int { return h.width }

func (h *Help) Toggle() {
	h.Set(!h.model.ShowAll)
}

func (h *Help) All() bool { return h.model.ShowAll }

func (h *Help) Set(on bool) {
	h.model.ShowAll = on
	h.cached = h.model.View(h.keys)
	h.height = lipgloss.Height(h.cached)
	h.width = lipgloss.Width(h.cached)
}

func (h *Help) View() string {
	return h.cached
}
