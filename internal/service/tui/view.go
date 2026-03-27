package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

var activeStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(lipgloss.Color("62")).
	Padding(0, 1)
var idleStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(lipgloss.Color("102")).
	Padding(0, 1)

func (m model) View() tea.View {
	tStyle := idleStyle
	iStyle := idleStyle
	if m.focusTable {
		tStyle = activeStyle
	} else {
		iStyle = activeStyle
	}

	targetHeight := m.height - heightSpace

	// Рендерим левую часть (Таблица)
	// Явно задаем Width и Height для стиля, чтобы рамка была фиксированной
	tableView := tStyle.
		Width(m.width - infoWidth - spacing).
		Height(targetHeight).
		MaxHeight(targetHeight). // Жёсткое ограничение
		Render(m.table.View())

	// Рендерим правую часть (Инфо/Viewport)
	infoView := iStyle.
		Width(infoWidth).
		Height(targetHeight).
		MaxHeight(targetHeight). // Жёсткое ограничение
		Render(m.info.View())

	// Собираем всё вместе
	finalView := lipgloss.JoinHorizontal(
		lipgloss.Top,
		tableView,
		strings.Repeat(" ", spacing),
		infoView,
	)

	return tea.View{
		Content:   finalView,
		AltScreen: true,
	}
}
