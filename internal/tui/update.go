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

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		m.table.SetWidth(msg.Width)

		tableHeight := msg.Height
		if tableHeight < 1 {
			tableHeight = 1
		}
		m.table.SetHeight(tableHeight)

	case tickMsg:
		if m.Tick == 0 {
			break
		}
		return m, tea.Batch(
			m.tick(),
			func() tea.Msg {
				rows, err := processes.GetProcesses(processes.SortMode)
				if err != nil {
					logger.Logger.Println(err)
					return nil
				}
				return dataMsg{rows: rows}
			},
		)

	case dataMsg:
		m.table.SetRows(msg.rows)

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
		// processes manipulation
		case "d":
			pid, err := strconv.Atoi(m.table.SelectedRow()[0])
			if err != nil {
				logger.Logger.Println(err)
			}
			err = processes.KillProcess(pid)
			if err != nil {
				logger.Logger.Println(err)
			}
		case "s":
			pid, err := strconv.Atoi(m.table.SelectedRow()[0])
			if err != nil {
				logger.Logger.Println(err)
			}
			err = processes.StopProcess(pid)
			if err != nil {
				logger.Logger.Println(err)
			}
		case "r":
			pid, err := strconv.Atoi(m.table.SelectedRow()[0])
			if err != nil {
				logger.Logger.Println(err)
			}
			err = processes.ResumeProcess(pid)
			if err != nil {
				logger.Logger.Println(err)
			}
		// sort mode manipulation
		case "n":
			processes.SortMode = "-n"
		case "c":
			processes.SortMode = "-c"
		case "m":
			processes.SortMode = "-m"
		case "t":
			processes.SortMode = "-t"
		case "p":
			processes.SortMode = "empty"
		}
	}

	m.table, cmd = m.table.Update(msg)
	return m, cmd
}
