package tui

import "github.com/charmbracelet/lipgloss"

func (m model) View() string {
	tableView := tableStyle.
		Width(m.width - 55 - 2 - 4).
		Render(m.table.View())
	infoView := baseStyle.Render(m.info)

	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		tableView,
		"  ",
		infoView,
	)
}
