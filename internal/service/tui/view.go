package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

var panelTitleStyle = lipgloss.NewStyle().
	Bold(true)

var panelMetaStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("244"))

func (m model) infoHeaderView(width int) string {
	title := "Details"
	if m.inputMode != inputModeNone {
		title = "Details [input]"
	} else if !m.focusTable {
		title = "Details [scroll]"
	}
	return panelTitleStyle.Width(width).Render(title)
}

func (m model) infoFooterView(width int) string {
	total := m.info.TotalLineCount()
	if total == 0 {
		return panelMetaStyle.Width(width).Render("Enter details | h full | A affinity | L limits | J jail")
	}

	currentTop := minInt(m.info.YOffset()+1, total)
	currentBottom := minInt(m.info.YOffset()+m.info.VisibleLineCount(), total)
	position := fmt.Sprintf("Lines %d-%d/%d", currentTop, currentBottom, total)

	hint := "Up/Down PgUp/PgDn"
	if m.inputMode != inputModeNone {
		hint = "Enter apply | Esc cancel"
	} else if m.focusTable {
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
	tStyle := m.styles.idlePanel
	iStyle := m.styles.idlePanel
	if m.focusTable {
		tStyle = m.styles.activePanel
	} else {
		iStyle = m.styles.activePanel
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
