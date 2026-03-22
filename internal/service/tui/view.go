package tui

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

var activeStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(lipgloss.Color("62")).
	Padding(1, 2)
var idleStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(lipgloss.Color("102")).
	Padding(1, 2)

func (m model) View() tea.View {
	tStyle := idleStyle
	iStyle := idleStyle
	if m.focusTable {
		tStyle = activeStyle
	} else {
		iStyle = activeStyle
	}

	tableView := tStyle.Width(m.width - 55 - 2 - 4).Render(m.table.View())
	infoView := iStyle.Height(m.height - 4).Render(m.info.View())

	return tea.View{
		Content: lipgloss.JoinHorizontal(
			lipgloss.Top,
			tableView,
			"  ",
			infoView,
		),
	}
}
