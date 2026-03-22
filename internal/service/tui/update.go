package tui

import (
	"fmt"
	"strconv"

	tea "charm.land/bubbletea/v2"
)

const HeightSpace = 2

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		tableWidth := msg.Width - activeStyle.GetHorizontalFrameSize() - 55
		m.table.SetWidth(tableWidth)
		m.table.SetHeight(msg.Height - HeightSpace)

		m.info.SetWidth(55 - activeStyle.GetHorizontalFrameSize())
		m.info.SetHeight(msg.Height - HeightSpace - activeStyle.GetVerticalFrameSize())

	// update service list
	case tickMsg:
		if m.Tick == 0 {
			break
		}
		return m, tea.Batch(
			m.tick(),
			func() tea.Msg {
				rows, err := m.Service.GetProcesses()
				if err != nil {
					m.Logger.Println(err)
					return nil
				}
				return dataMsg{rows: rows}
			},
		)

	// set new service list
	case dataMsg:
		m.table.SetRows(msg.rows)

	// keyboard shortcuts
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			if m.table.Focused() {
				m.table.Blur()
			} else {
				m.table.Focus()
			}
		case "q", "ctrl+c":
			m.Logger.Println("Initial graceful shutdown...")
			m.DaemonCancel()
			m.Logger.Println("Daemon is now stopped")
			return m, tea.Quit
		case "tab":
			m.focusTable = !m.focusTable
			return m, nil
		case "enter":
			pid, err := strconv.Atoi(m.table.SelectedRow()[0])
			if err != nil {
				m.Logger.Println(err)
				return m, nil
			}

			m.info.SetContent(m.formatedInfo(int32(pid)))

			return m, nil
		case "h":
			pid, err := strconv.Atoi(m.table.SelectedRow()[0])
			if err != nil {
				m.Logger.Println(err)
				return m, nil
			}

			m.info.SetContent(m.formatedBigInfo(int32(pid)))

			return m, nil

		// service manipulation
		case "k":
			pid, err := strconv.Atoi(m.table.SelectedRow()[0])
			if err != nil {
				m.Logger.Println(err)
			}
			err = m.Service.KillProcess(pid)
			if err != nil {
				m.Logger.Println(err)
			}
			m.info.SetContent(fmt.Sprintf("Killed service:\n\nPID: %d\n\n", pid))
			return m, nil
		case "d":
			pid, err := strconv.Atoi(m.table.SelectedRow()[0])
			if err != nil {
				m.Logger.Println(err)
			}
			err = m.Service.KillProcessTree(pid)
			if err != nil {
				m.Logger.Println(err)
			}
			m.info.SetContent(fmt.Sprintf("Killed tree of service:\n\nPID: %d\n\n", pid))
			return m, nil
		case "s":
			pid, err := strconv.Atoi(m.table.SelectedRow()[0])
			if err != nil {
				m.Logger.Println(err)
			}
			err = m.Service.StopProcess(pid)
			if err != nil {
				m.Logger.Println(err)
			}
			m.info.SetContent(fmt.Sprintf("Stopped service:\n\nPID: %d\n\n", pid))
			return m, nil
		case "r":
			pid, err := strconv.Atoi(m.table.SelectedRow()[0])
			if err != nil {
				m.Logger.Println(err)
			}
			err = m.Service.ResumeProcess(pid)
			if err != nil {
				m.Logger.Println(err)
			}
			m.info.SetContent(fmt.Sprintf("Resumed service:\n\nPID: %d\n\n", pid))
			return m, nil

		// sort mode manipulation
		case "n":
			m.Service.SetSortProcMod("-n")
		case "c":
			m.Service.SetSortProcMod("-c")
		case "m":
			m.Service.SetSortProcMod("-m")
		case "t":
			m.Service.SetSortProcMod("-t")
		case "u":
			m.Service.SetSortProcMod("-u")
		case "p":
			m.Service.SetSortProcMod("-p")
		}
	}

	if m.focusTable {
		m.table, cmd = m.table.Update(msg)
	} else {
		m.info, cmd = m.info.Update(msg)
	}
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m model) formatedInfo(pid int32) string {
	var info string

	proc, err := m.Parser.ProcessInfo(pid)
	if err != nil {
		info = fmt.Sprintf("Error parsing service info: %s", err)
		return info
	}

	info = fmt.Sprintf("Selected service:\n\n"+
		"PID: %d\n\n"+
		"Name: %s\n\n"+
		"CMD: %s\n\n"+
		"Nice: %d\n\n"+
		"User: %s\n\n",
		proc.PID, proc.Name, proc.Cmd, proc.Nice, proc.User)

	return info
}

func (m model) formatedBigInfo(pid int32) string {
	var info string

	proc, err := m.Parser.HardObjectParse(pid)
	if err != nil {
		info = fmt.Sprintf("Error parsing service info: %s", err)
		return info
	}

	switch len(proc.Connections) {
	case 0:
		info += "\nConnections: nothing\n\n"
	default:
		info += "\nConnections:\n"
		for _, conn := range proc.Connections {
			info += fmt.Sprintf("\tLocal address: %s\n", conn.LocalAddr)
			info += fmt.Sprintf("\tRemote address: %s\n", conn.RemoteAddr)
			info += fmt.Sprintf("\tStatus: %s\n\n", conn.Status)
		}
	}

	switch len(proc.OpenFiles) {
	case 0:
		info += "\nOpened files: nothing\n\n"
	default:
		info += "\nOpened files:\n"
		for _, file := range proc.OpenFiles {
			info += fmt.Sprintf("\t%s\n", file)
		}
	}

	switch len(proc.Children) {
	case 0:
		info += "\nChild service: nothing\n\n"
	default:
		info += "\nChild service:\n"
		_, tree, err := m.Parser.ProcessTree(pid)
		if err != nil {
			info += fmt.Sprintf("\nError parsing service tree: %s\n", err)
		}
		s, err := m.Service.GetTuiTree(pid, tree)
		info += fmt.Sprintf("\n%s\n", s)
	}

	return info
}
