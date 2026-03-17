package tui

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

func (m model) View() tea.View {
	tableView := tableStyle.
		Width(m.width - 55 - 2 - 4).
		Render(m.table.View())
	infoView := baseStyle.Render(m.info.View())

	return tea.View{
		Content: lipgloss.JoinHorizontal(
			lipgloss.Top,
			tableView,
			"  ",
			infoView,
		),
	}
}
