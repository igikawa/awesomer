package tui

import (
	"awesomeProject/internal/processes"
	"awesomeProject/pkg/logger"
	"fmt"

	"strconv"

	tea "github.com/charmbracelet/bubbletea"
)

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		tableWidth := msg.Width - baseStyle.GetWidth() - 4
		m.table.SetWidth(tableWidth)
		m.table.SetHeight(msg.Height - 4)

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
			pid, err := strconv.Atoi(m.table.SelectedRow()[0])
			if err != nil {
				logger.Logger.Println(err)
			}

			p, _ := processes.GetProcessInfo(pid)
			name, _ := p.Name()
			cmd, _ := p.Cmdline()

			m.info = fmt.Sprintf("Selected process:\n\n"+
				"PID: %d\n\n"+
				"Name: %s\n\n"+
				"CMD: %s\n\n", p.Pid, name, cmd)

			files, _ := p.OpenFiles()
			switch len(files) {
			case 0:
				m.info += "Opened files: nothing\n\n"
			default:
				m.info += "Opened files:\n"
				for _, file := range files {
					m.info += fmt.Sprintf("\t%s\n", file.Path)
				}
			}

			children, _ := p.Children()
			switch len(children) {
			case 0:
				m.info += "\nChild processes: nothing\n"
			default:
				m.info += "Child processes:\n"
				for _, child := range children {
					name, _ := child.Name()
					m.info += fmt.Sprintf("\tPID: %d, Name: %s\n", child.Pid, name)
				}
			}

			return m, nil

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
			m.info = fmt.Sprintf("Killed process:\n\nPID: %d\n\n", pid)
			return m, nil
		case "s":
			pid, err := strconv.Atoi(m.table.SelectedRow()[0])
			if err != nil {
				logger.Logger.Println(err)
			}
			err = processes.StopProcess(pid)
			if err != nil {
				logger.Logger.Println(err)
			}
			m.info = fmt.Sprintf("Stopped process:\n\nPID: %d\n\n", pid)
			return m, nil
		case "r":
			pid, err := strconv.Atoi(m.table.SelectedRow()[0])
			if err != nil {
				logger.Logger.Println(err)
			}
			err = processes.ResumeProcess(pid)
			if err != nil {
				logger.Logger.Println(err)
			}
			m.info = fmt.Sprintf("Resumed process:\n\nPID: %d\n\n", pid)
			return m, nil
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
