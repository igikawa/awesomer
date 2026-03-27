package tui

import (
	"fmt"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
)

const (
	infoWidth   = 72
	heightSpace = 2
	spacing     = 1
	minTableW   = 44
	minInfoW    = 32
	infoHeaderH = 1
	infoFooterH = 1
)

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (m *model) syncFocus() {
	if m.focusTable {
		m.table.Focus()
		return
	}
	m.table.Blur()
}

func (m *model) syncLayout() {
	if m.width <= 0 || m.height <= 0 {
		return
	}

	vFrame := idleStyle.GetVerticalFrameSize()
	hFrame := idleStyle.GetHorizontalFrameSize()

	m.panelH = maxInt(m.height-heightSpace, vFrame)
	totalWidth := maxInt(m.width, 1)
	m.infoWidth = minInt(infoWidth, maxInt(totalWidth-spacing, 0))
	m.tableWidth = maxInt(totalWidth-m.infoWidth-spacing, 0)

	if m.tableWidth < minTableW {
		maxInfoShrink := maxInt(m.infoWidth-minInfoW, 0)
		requiredShrink := minTableW - m.tableWidth
		shrink := minInt(requiredShrink, maxInfoShrink)
		m.infoWidth -= shrink
		m.tableWidth = maxInt(totalWidth-m.infoWidth-spacing, 0)
	}

	internalTableWidth := maxInt(m.tableWidth-hFrame, 0)
	internalTableHeight := maxInt(m.panelH-vFrame, 0)
	m.table.SetWidth(internalTableWidth)
	m.table.SetHeight(internalTableHeight)

	bodyHeight := maxInt(m.panelH-vFrame-infoHeaderH-infoFooterH, 1)
	m.infoBodyH = bodyHeight
	m.info.SetWidth(maxInt(m.infoWidth-hFrame, 0))
	m.info.SetHeight(bodyHeight)
}

func (m *model) setInfoContent(content string) {
	m.info.SetContent(strings.TrimRight(content, "\n"))
	m.info.GotoTop()
}

func (m model) selectedPID() (int, error) {
	row := m.table.SelectedRow()
	if len(row) == 0 {
		return 0, fmt.Errorf("no service selected")
	}
	return strconv.Atoi(row[0])
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.syncLayout()
		m.syncFocus()

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
			m.focusTable = true
			m.syncFocus()
			return m, nil
		case "q", "ctrl+c":
			m.Logger.Println("Initial graceful shutdown...")
			m.DaemonCancel()
			m.Logger.Println("Daemon is now stopped")
			return m, tea.Quit
		case "tab":
			m.focusTable = !m.focusTable
			m.syncFocus()
			return m, nil
		case "enter":
			pid, err := m.selectedPID()
			if err != nil {
				m.Logger.Println(err)
				return m, nil
			}

			m.setInfoContent(m.formatedInfo(int32(pid)))
			m.focusTable = false
			m.syncFocus()

			return m, nil
		case "h":
			pid, err := m.selectedPID()
			if err != nil {
				m.Logger.Println(err)
				return m, nil
			}

			m.setInfoContent(m.formatedBigInfo(int32(pid)))
			m.focusTable = false
			m.syncFocus()

			return m, nil

		// service manipulation
		case "k":
			pid, err := m.selectedPID()
			if err != nil {
				m.Logger.Println(err)
				return m, nil
			}
			err = m.Service.KillProcess(pid)
			if err != nil {
				m.Logger.Println(err)
			}
			m.setInfoContent(fmt.Sprintf("Killed service\n\nPID: %d", pid))
			return m, nil
		case "d":
			pid, err := m.selectedPID()
			if err != nil {
				m.Logger.Println(err)
				return m, nil
			}
			err = m.Service.KillProcessTree(pid)
			if err != nil {
				m.Logger.Println(err)
			}
			m.setInfoContent(fmt.Sprintf("Killed service tree\n\nPID: %d", pid))
			return m, nil
		case "s":
			pid, err := m.selectedPID()
			if err != nil {
				m.Logger.Println(err)
				return m, nil
			}
			err = m.Service.StopProcess(pid)
			if err != nil {
				m.Logger.Println(err)
			}
			m.setInfoContent(fmt.Sprintf("Stopped service\n\nPID: %d", pid))
			return m, nil
		case "r":
			pid, err := m.selectedPID()
			if err != nil {
				m.Logger.Println(err)
				return m, nil
			}
			err = m.Service.ResumeProcess(pid)
			if err != nil {
				m.Logger.Println(err)
			}
			m.setInfoContent(fmt.Sprintf("Resumed service\n\nPID: %d", pid))
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
	proc, err := m.Parser.ProcessInfo(pid)
	if err != nil {
		return fmt.Sprintf("Error parsing service info: %s", err)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Selected service\n\n")
	fmt.Fprintf(&b, "PID: %d\n", proc.PID)
	fmt.Fprintf(&b, "Name: %s\n", proc.Name)
	fmt.Fprintf(&b, "User: %s\n", proc.User)
	fmt.Fprintf(&b, "Nice: %d\n\n", proc.Nice)
	fmt.Fprintf(&b, "Command\n%s\n", proc.Cmd)

	return b.String()
}

func (m model) formatedBigInfo(pid int32) string {
	proc, err := m.Parser.HardObjectParse(pid)
	if err != nil {
		return fmt.Sprintf("Error parsing service info: %s", err)
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Extended service details\n\n")

	switch len(proc.Connections) {
	case 0:
		b.WriteString("Connections\nnone\n\n")
	default:
		b.WriteString("Connections\n")
		for _, conn := range proc.Connections {
			fmt.Fprintf(&b, "local:  %s\n", conn.LocalAddr)
			fmt.Fprintf(&b, "remote: %s\n", conn.RemoteAddr)
			fmt.Fprintf(&b, "state:  %s\n\n", conn.Status)
		}
	}

	switch len(proc.OpenFiles) {
	case 0:
		b.WriteString("Opened files\nnone\n\n")
	default:
		b.WriteString("Opened files\n")
		for _, file := range proc.OpenFiles {
			fmt.Fprintf(&b, "%s\n", file)
		}
		b.WriteString("\n")
	}

	switch len(proc.Children) {
	case 0:
		b.WriteString("Child service\nnone\n")
	default:
		b.WriteString("Child service\n")
		_, tree, err := m.Parser.ProcessTree(pid)
		if err != nil {
			fmt.Fprintf(&b, "Error parsing service tree: %s\n", err)
			break
		}
		s, err := m.Service.GetTuiTree(pid, tree)
		if err != nil {
			fmt.Fprintf(&b, "Error building service tree: %s\n", err)
			break
		}
		fmt.Fprintf(&b, "%s\n", s)
	}

	return b.String()
}
