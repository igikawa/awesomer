package tui

import (
	"awesomeProject/internal/config"
	"fmt"
	"maps"
	"slices"
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

	vFrame := m.styles.idlePanel.GetVerticalFrameSize()
	hFrame := m.styles.idlePanel.GetHorizontalFrameSize()

	m.panelH = maxInt(m.height-heightSpace, vFrame)
	totalWidth := maxInt(m.width, 1)
	m.tableWidth, m.infoWidth = resolvePanelWidths(totalWidth, m.UI)

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

// resolvePanelWidths applies preferred widths from config while still clamping
// both panels to the available terminal width.
func resolvePanelWidths(totalWidth int, ui config.UIConfig) (int, int) {
	usableWidth := maxInt(totalWidth-spacing, 0)

	switch {
	case ui.TableWidth > 0 && ui.InfoWidth > 0:
		tableWidth := ui.TableWidth
		infoWidth := ui.InfoWidth
		if tableWidth+infoWidth <= usableWidth {
			return tableWidth, infoWidth
		}
		return maxInt(usableWidth-infoWidth, 0), minInt(infoWidth, usableWidth)
	case ui.TableWidth > 0:
		tableWidth := minInt(ui.TableWidth, usableWidth)
		return tableWidth, maxInt(usableWidth-tableWidth, 0)
	default:
		infoWidth := minInt(ui.InfoWidth, usableWidth)
		return maxInt(usableWidth-infoWidth, 0), infoWidth
	}
}

func (m *model) setInfoContent(content string) {
	m.info.SetContent(strings.TrimRight(content, "\n"))
	m.info.GotoTop()
}

func (m model) refreshRowsCmd() tea.Cmd {
	return func() tea.Msg {
		rows, changed, err := m.Service.GetProcesses()
		if err != nil {
			m.Logger.Println(err)
			return nil
		}
		return dataMsg{rows: rows, changed: changed}
	}
}

func (m model) selectedPID() (int, error) {
	row := m.table.SelectedRow()
	if len(row) == 0 {
		return 0, fmt.Errorf("no service selected")
	}
	return strconv.Atoi(row[0])
}

func (m *model) startInput(mode inputMode, pid int) {
	m.inputMode = mode
	m.inputPID = pid
	m.inputValue = ""
	m.focusTable = false
	m.syncFocus()
	m.renderInputPrompt("")
}

func (m *model) clearInput() {
	m.inputMode = inputModeNone
	m.inputPID = 0
	m.inputValue = ""
}

func (m *model) renderInputPrompt(message string) {
	var b strings.Builder

	switch m.inputMode {
	case inputModeAffinity:
		fmt.Fprintf(&b, "CPU affinity\n\n")
		fmt.Fprintf(&b, "PID: %d\n", m.inputPID)
		b.WriteString("Format: comma-separated core list, example: 0,1,3\n")
		b.WriteString("Press Enter to apply or Esc to cancel.\n\n")
	case inputModeNoFile:
		fmt.Fprintf(&b, "RLIMIT_NOFILE\n\n")
		fmt.Fprintf(&b, "PID: %d\n", m.inputPID)
		b.WriteString("Format: integer value, example: 4096\n")
		b.WriteString("Press Enter to apply or Esc to cancel.\n\n")
	default:
	}

	if message != "" {
		fmt.Fprintf(&b, "%s\n\n", message)
	}
	fmt.Fprintf(&b, "> %s", m.inputValue)

	m.setInfoContent(b.String())
}

func parseCPUCores(raw string) ([]int, error) {
	parts := strings.Split(raw, ",")
	uniq := make(map[int]struct{}, len(parts))

	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value == "" {
			continue
		}

		core, err := strconv.Atoi(value)
		if err != nil {
			return nil, fmt.Errorf("invalid CPU core: %q", value)
		}
		if core < 0 {
			return nil, fmt.Errorf("CPU core must be >= 0: %d", core)
		}

		uniq[core] = struct{}{}
	}

	if len(uniq) == 0 {
		return nil, fmt.Errorf("no CPU cores provided")
	}

	cores := slices.Collect(maps.Keys(uniq))
	slices.Sort(cores)

	return cores, nil
}

func (m *model) submitInput() tea.Cmd {
	switch m.inputMode {
	case inputModeAffinity:
		return m.submitAffinityInput()
	case inputModeNoFile:
		return m.submitNoFileInput()
	default:
		return nil
	}
}

// submitAffinityInput and submitNoFileInput keep the two interactive form flows
// separate so validation, mutation, and success rendering stay readable.
func (m *model) submitAffinityInput() tea.Cmd {
	cores, err := parseCPUCores(m.inputValue)
	if err != nil {
		m.renderInputPrompt("Error: " + err.Error())
		return nil
	}

	if err := m.Service.SetCPUAffinity(m.inputPID, cores); err != nil {
		m.renderInputPrompt("Error: " + err.Error())
		return nil
	}

	pid := m.inputPID
	m.clearInput()
	m.setInfoContent("Updated CPU affinity\n\n" + m.formatedInfo(int32(pid)))
	return m.refreshRowsCmd()
}

func (m *model) submitNoFileInput() tea.Cmd {
	limit, err := strconv.ParseUint(strings.TrimSpace(m.inputValue), 10, 64)
	if err != nil {
		m.renderInputPrompt("Error: invalid limit value")
		return nil
	}

	if err := m.Service.SetNoFileLimit(m.inputPID, limit); err != nil {
		m.renderInputPrompt("Error: " + err.Error())
		return nil
	}

	pid := m.inputPID
	m.clearInput()
	m.setInfoContent("Updated RLIMIT_NOFILE\n\n" + m.formatedInfo(int32(pid)))
	return m.refreshRowsCmd()
}

func intsToCSV(values []int) string {
	if len(values) == 0 {
		return "unknown"
	}
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, strconv.Itoa(value))
	}
	return strings.Join(parts, ",")
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
	fmt.Fprintf(&b, "Nice: %d\n", proc.Nice)
	fmt.Fprintf(&b, "CPU affinity: %s\n", intsToCSV(proc.CPUAffinity))
	fmt.Fprintf(&b, "RLIMIT_NOFILE: %d / %d\n\n", proc.NoFileSoft, proc.NoFileHard)
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
	fmt.Fprintf(&b, "CPU affinity: %s\n", intsToCSV(proc.CPUAffinity))
	fmt.Fprintf(&b, "RLIMIT_NOFILE: %d / %d\n\n", proc.NoFileSoft, proc.NoFileHard)

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
				rows, changed, err := m.Service.GetProcesses()
				if err != nil {
					m.Logger.Println(err)
					return nil
				}
				return dataMsg{rows: rows, changed: changed}
			},
		)

	// set new service list
	case dataMsg:
		if msg.changed {
			m.table.SetRows(msg.rows)
		}

	// keyboard shortcuts
	case tea.KeyMsg:
		if m.inputMode != inputModeNone {
			switch msg.String() {
			case "esc":
				m.clearInput()
				m.setInfoContent("Action cancelled")
				m.focusTable = true
				m.syncFocus()
				return m, nil
			case "enter":
				return m, m.submitInput()
			case "backspace":
				runes := []rune(m.inputValue)
				if len(runes) > 0 {
					m.inputValue = string(runes[:len(runes)-1])
				}
				m.renderInputPrompt("")
				return m, nil
			}

			if text := msg.Key().Text; text != "" {
				m.inputValue += text
				m.renderInputPrompt("")
				return m, nil
			}

			return m, nil
		}

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
		case "a":
			pid, err := m.selectedPID()
			if err != nil {
				m.Logger.Println(err)
				return m, nil
			}
			m.startInput(inputModeAffinity, pid)
			return m, nil
		case "l":
			pid, err := m.selectedPID()
			if err != nil {
				m.Logger.Println(err)
				return m, nil
			}
			m.startInput(inputModeNoFile, pid)
			return m, nil
		case "j":
			pid, err := m.selectedPID()
			if err != nil {
				m.Logger.Println(err)
				return m, nil
			}
			inJail, err := m.Service.ToggleProcessJail(pid)
			if err != nil {
				m.Logger.Println(err)
				m.setInfoContent(fmt.Sprintf("Failed to toggle process jail\n\nPID: %d\nError: %s", pid, err))
				return m, nil
			}

			state := "Removed process tree from processJail"
			if inJail {
				state = "Moved process tree into processJail"
			}
			m.setInfoContent(fmt.Sprintf("%s\n\nPID: %d", state, pid))
			return m, m.refreshRowsCmd()

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
			return m, m.refreshRowsCmd()
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
			return m, m.refreshRowsCmd()
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
			return m, m.refreshRowsCmd()
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
			return m, m.refreshRowsCmd()

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
