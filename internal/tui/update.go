package tui

import (
	"awesomeProject/internal/processes"
	"awesomeProject/pkg/logger"
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
)

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tickMsg:
		switch m.Tick {
		case 0:
			break
		default:
			rows, err := processes.GetProcesses(processes.SortMode)
			if err != nil {
				logger.Logger.Println(err)
			}
			m.table.SetRows(rows)
			return m, m.tick()
		}
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			if m.table.Focused() {
				m.table.Blur()
			} else {
				m.table.Focus()
			}
		case "q", "ctrl+c":
			return m, tea.Quit
		case "enter": // TODO: print process info
			return m, tea.Batch(
				tea.Printf("Let's go to %s!", m.table.SelectedRow()[1]),
			)
		case "d":
			name := m.table.SelectedRow()[1]
			pid, err := strconv.Atoi(m.table.SelectedRow()[0])
			if err != nil {
				logger.Logger.Println(err)
			}
			err = processes.KillProcess(pid)
			if err != nil {
				logger.Logger.Println(err)
			}
			return m, tea.Batch(
				tea.Printf("Kill: %s!", name),
			)
		case "s":
			pid, err := strconv.Atoi(m.table.SelectedRow()[0])
			if err != nil {
				logger.Logger.Println(err)
			}
			err = processes.StopProcess(pid)
			if err != nil {
				logger.Logger.Println(err)
			}
			return m, tea.Batch(
				tea.Printf("Stop: %s!", m.table.SelectedRow()[0]),
			)
		case "r":
			pid, err := strconv.Atoi(m.table.SelectedRow()[0])
			if err != nil {
				logger.Logger.Println(err)
			}
			err = processes.ResumeProcess(pid)
			if err != nil {
				logger.Logger.Println(err)
			}
			return m, tea.Batch(
				tea.Printf("Resume: %s!", m.table.SelectedRow()[0]),
			)
		}
	}
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}
