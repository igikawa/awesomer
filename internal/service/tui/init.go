package tui

import (
	tea "charm.land/bubbletea/v2"
)

func (m model) Init() tea.Cmd {
	return tea.Batch(
		m.tick(),
	)
}
