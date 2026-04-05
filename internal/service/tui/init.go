package tui

import (
	tea "charm.land/bubbletea/v2"
)

func (m *model) Init() tea.Cmd {
	cmds := []tea.Cmd{
		tea.RequestWindowSize,
		m.refreshRowsCmd(),
	}
	if tick := m.tick(); tick != nil {
		cmds = append(cmds, tick)
	}
	return tea.Batch(cmds...)
}
