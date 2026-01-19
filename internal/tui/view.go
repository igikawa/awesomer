package tui

import (
	"github.com/charmbracelet/lipgloss"
)

func (m model) View() string {
	return lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		Render(m.table.View())
}
