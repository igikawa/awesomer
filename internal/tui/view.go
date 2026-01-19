package tui

func (m model) View() string {
	return baseStyle.Render(m.table.View()) + "\n"
}
