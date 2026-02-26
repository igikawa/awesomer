package tui

import (
	"awesomeProject/internal/process"
	"awesomeProject/internal/process/parser"
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
		case "enter":
			pid, err := strconv.Atoi(m.table.SelectedRow()[0])
			if err != nil {
				logger.Logger.Println(err)
				return m, nil
			}

			m.info = formatedInfo(int32(pid))

			return m, nil
		case "h":
			pid, err := strconv.Atoi(m.table.SelectedRow()[0])
			if err != nil {
				logger.Logger.Println(err)
				return m, nil
			}

			m.info = formatedBigInfo(int32(pid))

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

func formatedInfo(pid int32) string {
	var info string

	p, err := parser.Object.ProcessInfo(pid)
	if err != nil {
		logger.Logger.Println(err)
		info = fmt.Sprintf("Error parsing process info: %s", err)
		return info
	}

	info = fmt.Sprintf("Selected process:\n\n"+
		"PID: %d\n\n"+
		"Name: %s\n\n"+
		"CMD: %s\n\n"+
		"Nice: %d\n\n"+
		"User: %s\n\n",
		p.PID, p.Name, p.Cmd, p.Nice, p.User)

	return info
}

func formatedBigInfo(pid int32) string {
	var info string

	p, err := parser.Object.HardObjectParse(pid)
	if err != nil {
		logger.Logger.Println(err)
		info = fmt.Sprintf("Error parsing process info: %s", err)
		return info
	}

	switch len(p.Connections) {
	case 0:
		info += "\nConnections: nothing\n\n"
	default:
		info += "\nConnections:\n"
		for _, conn := range p.Connections {
			info += fmt.Sprintf("\tLocal address: %s\n", conn.LocalAddr)
			info += fmt.Sprintf("\tRemote address: %s\n", conn.RemoteAddr)
			info += fmt.Sprintf("\tStatus: %s\n\n", conn.Status)
		}
	}

	switch len(p.OpenFiles) {
	case 0:
		info += "\nOpened files: nothing\n\n"
	default:
		info += "\nOpened files:\n"
		for _, file := range p.OpenFiles {
			info += fmt.Sprintf("\t%s\n", file)
		}
	}

	switch len(p.Children) {
	case 0:
		info += "\nChild process: nothing\n\n"
	default:
		info += "\nChild process:\n"
		_, tree, err := parser.Object.ProcessTree(pid)
		if err != nil {
			logger.Logger.Println(err)
		}
		s, err := process.GetTuiTree(pid, tree)
		info += fmt.Sprintf("\n%s\n", s)
	}

	return info
}
