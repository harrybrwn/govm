package tui

import (
	"fmt"
	"log/slog"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type NextChainedModelMsg struct{}

func NextChainedModel() tea.Msg {
	slog.Info("sending next model")
	return NextChainedModelMsg{}
}

type PrevChainedModelMsg struct{}

func PrevChainedModel() tea.Msg {
	slog.Info("sending prev model")
	return PrevChainedModelMsg{}
}

type CheckChainLengthMsg struct{}

func CheckChainLength() tea.Msg { return CheckChainLengthMsg{} }

type ChainModel interface {
	ChainView() tea.View
}

type Chained struct {
	Models         []tea.Model
	StopCmd        tea.Cmd
	IgnoreProgress bool
	current        int
	progress       []string
}

func (m *Chained) Init() tea.Cmd {
	inits := make([]tea.Cmd, 0)
	for i := range m.Models {
		cmd := m.Models[i].Init()
		if cmd != nil {
			inits = append(inits, cmd)
		}
	}
	return tea.Batch(inits...)
}

func (m *Chained) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	logger := slog.With("model", fmt.Sprintf("%T", m))
	switch msg.(type) {
	case CheckChainLengthMsg:
		if m.current >= len(m.Models) {
			logger.Info("stopping model sequence")
			if m.StopCmd == nil {
				return m, tea.Quit
			}
			return m, m.StopCmd
		}
		return m, nil
	case NextChainedModelMsg:
		if !m.IgnoreProgress {
			sub := m.Models[m.current]
			if submodel, ok := sub.(ChainModel); ok {
				sv := submodel.ChainView()
				m.progress = append(m.progress, sv.Content)
			}
		}
		m.current++
		return m, CheckChainLength
	case PrevChainedModelMsg:
		m.current = max(m.current-1, 0)
		if len(m.progress) > 0 {
			m.progress = m.progress[:len(m.progress)-1]
		}
		return m, CheckChainLength
	}
	model, cmd := m.Models[m.current].Update(msg)
	m.Models[m.current] = model
	return m, tea.Batch(cmd, CheckChainLength)
}

func (m *Chained) View() tea.View {
	if m.current >= len(m.Models) {
		return tea.View{}
	}
	v := m.Models[m.current].View()
	if len(m.progress) == 0 {
		return v
	}
	v.SetContent(lipgloss.JoinVertical(lipgloss.Top,
		strings.Join(m.progress, "\n"),
		v.Content))
	return v
}
