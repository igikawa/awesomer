package tui

import (
	"awesomeProject/internal/process"
	"awesomeProject/pkg/logger"

	"fmt"
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
)

const HeightSpace = 4

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		tableWidth := msg.Width - baseStyle.GetWidth() - HeightSpace
		m.table.SetWidth(tableWidth)
		m.table.SetHeight(msg.Height - HeightSpace)

	// update process list
	case tickMsg:
		if m.Tick == 0 {
			break
		}
		return m, tea.Batch(
			m.tick(),
			func() tea.Msg {
				rows, err := process.GetProcesses(process.SortMode)
				if err != nil {
					logger.Logger.Println(err)
					return nil
				}
				return dataMsg{rows: rows}
			},
		)

	// set new process list
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
				return m, nil
			}

			p, err := process.ParserObj.ProcessInfo(int32(pid))
			if err != nil {
				logger.Logger.Println(err)
				m.info = fmt.Sprintf("Error parsing process info: %s", err)
				return m, nil
			}

			m.info = fmt.Sprintf("Selected process:\n\n"+
				"PID: %d\n\n"+
				"Name: %s\n\n"+
				"CMD: %s\n\n",
				p.PID, p.Name, p.Cmd)

			switch len(p.OpenFiles) {
			case 0:
				m.info += "Opened files: nothing\n\n"
			default:
				m.info += "Opened files:\n"
				for _, file := range p.OpenFiles {
					m.info += fmt.Sprintf("\t%s\n", file)
				}
			}

			switch len(p.Children) {
			case 0:
				m.info += "\nChild process: nothing\n"
			default:
				m.info += "\nChild process:\n"
				_, tree, err := process.ParserObj.ProcessTree(int32(pid))
				if err != nil {
					logger.Logger.Println(err)
				}
				s, err := process.GetTuiTree(int32(pid), tree)
				m.info += fmt.Sprintf("\n%s\n", s)
			}

			return m, nil

		// process manipulation
		case "k":
			pid, err := strconv.Atoi(m.table.SelectedRow()[0])
			if err != nil {
				logger.Logger.Println(err)
			}
			err = process.KillProcess(pid)
			if err != nil {
				logger.Logger.Println(err)
			}
			m.info = fmt.Sprintf("Killed process:\n\nPID: %d\n\n", pid)
			return m, nil
		case "d":
			pid, err := strconv.Atoi(m.table.SelectedRow()[0])
			if err != nil {
				logger.Logger.Println(err)
			}
			err = process.KillProcessTree(pid)
			if err != nil {
				logger.Logger.Println(err)
			}
			m.info = fmt.Sprintf("Killed tree of process:\n\nPID: %d\n\n", pid)
			return m, nil
		case "s":
			pid, err := strconv.Atoi(m.table.SelectedRow()[0])
			if err != nil {
				logger.Logger.Println(err)
			}
			err = process.StopProcess(pid)
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
			err = process.ResumeProcess(pid)
			if err != nil {
				logger.Logger.Println(err)
			}
			m.info = fmt.Sprintf("Resumed process:\n\nPID: %d\n\n", pid)
			return m, nil
		// sort mode manipulation
		case "n":
			process.SortMode = "-n"
		case "c":
			process.SortMode = "-c"
		case "m":
			process.SortMode = "-m"
		case "t":
			process.SortMode = "-t"
		case "p":
			process.SortMode = "empty"
		}
	}

	m.table, cmd = m.table.Update(msg)
	return m, cmd
}
