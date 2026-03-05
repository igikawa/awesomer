package tui

import (
	"awesomeProject/internal/process"
	"awesomeProject/pkg/logger"

	"os"
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const INFO = "Info\n\n" +
	"↑↓ - select process\n\n" +
	"Enter - show process info\n\n" +
	"h - show big process info\n\n" +
	"s - stop process\n\n" +
	"r - resume process\n\n" +
	"k - kill process\n\n" +
	"d - kill process tree\n\n" +
	"q - exit\n\n"

type tickMsg time.Time

type dataMsg struct {
	rows []table.Row
}

type model struct {
	table  table.Model
	info   string
	Tick   int
	width  int
	height int
}

var baseStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(lipgloss.Color("62")).
	Padding(1, 2).
	Width(55)

var tableStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(lipgloss.Color("62")).
	Padding(0, 1)

func (m model) tick() tea.Cmd {
	s := time.Duration(m.Tick) * time.Second
	return tea.Tick(s, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func newTable() table.Model {
	columns := []table.Column{
		{Title: "PID", Width: 10},
		{Title: "Name", Width: 20},
		{Title: "CPU", Width: 7},
		{Title: "Mem", Width: 7},
		{Title: "Threads", Width: 7},
		{Title: "User", Width: 10},
		{Title: "", Width: 7}, // is controlling with daemon
	}

	rows, err := process.GetProcesses(process.SortMode)
	if err != nil {
		logger.Logger.Println(err)
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(20),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(s)

	return t
}

func Run(tick int) error {

	t := newTable()

	m := model{
		table:  t,
		info:   INFO,
		Tick:   tick,
		width:  80,
		height: 20,
	}
	if _, err := tea.NewProgram(m).Run(); err != nil {
		defer os.Exit(1)
		return err
	}
	return nil
}
