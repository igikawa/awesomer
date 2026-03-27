package tui

import (
	"fmt"
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

var panelTitleStyle = lipgloss.NewStyle().
	Bold(true)

var panelMetaStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("244"))

func (m model) infoHeaderView(width int) string {
	title := "Details"
	if !m.focusTable {
		title = "Details [scroll]"
	}
	return panelTitleStyle.Width(width).Render(title)
}

func (m model) infoFooterView(width int) string {
	total := m.info.TotalLineCount()
	if total == 0 {
		return panelMetaStyle.Width(width).Render("Tab focus | Enter details | h full")
	}

	currentTop := minInt(m.info.YOffset()+1, total)
	currentBottom := minInt(m.info.YOffset()+m.info.VisibleLineCount(), total)
	position := fmt.Sprintf("Lines %d-%d/%d", currentTop, currentBottom, total)

	hint := "Up/Down PgUp/PgDn"
	if m.focusTable {
		hint = "Tab focus panel"
	}

	footer := lipgloss.JoinHorizontal(
		lipgloss.Top,
		panelMetaStyle.Render(position),
		panelMetaStyle.Render(strings.Repeat(" ", maxInt(width-lipgloss.Width(position)-lipgloss.Width(hint), 1))),
		panelMetaStyle.Render(hint),
	)

	return lipgloss.NewStyle().Width(width).Render(footer)
}

func (m model) View() tea.View {
	tStyle := idleStyle
	iStyle := idleStyle
	if m.focusTable {
		tStyle = activeStyle
	} else {
		iStyle = activeStyle
	}

	infoInnerWidth := maxInt(m.infoWidth-iStyle.GetHorizontalFrameSize(), 0)
	infoInnerHeight := maxInt(m.panelH-iStyle.GetVerticalFrameSize(), 0)
	infoBodyHeight := minInt(m.infoBodyH, maxInt(infoInnerHeight-infoHeaderH-infoFooterH, 0))

	tableView := tStyle.
		Width(m.tableWidth).
		Height(m.panelH).
		MaxHeight(m.panelH).
		Render(m.table.View())

	infoBody := lipgloss.NewStyle().
		Width(infoInnerWidth).
		Height(infoBodyHeight).
		MaxHeight(infoBodyHeight).
		Render(m.info.View())

	infoContent := lipgloss.JoinVertical(
		lipgloss.Left,
		m.infoHeaderView(infoInnerWidth),
		infoBody,
		m.infoFooterView(infoInnerWidth),
	)

	infoView := iStyle.
		Width(m.infoWidth).
		Height(m.panelH).
		MaxHeight(m.panelH).
		Render(infoContent)

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
