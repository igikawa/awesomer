package tui

import (
	"awesomeProject/internal/config"
	"awesomeProject/internal/processes"
	"awesomeProject/pkg/logger"

	"os"
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type tickMsg time.Time

type dataMsg struct {
	rows []table.Row
}

type model struct {
	table  table.Model
	Tick   int
	width  int
	height int
}

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
		{Title: "CPU", Width: 15},
		{Title: "Mem", Width: 15},
		{Title: "Threads", Width: 7},
	}

	rows, err := processes.GetProcesses(processes.SortMode)
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

func Run() error {

	t := newTable()

	m := model{
		t,
		config.NewConfig().Tick,
		70,
		20,
	}
	if _, err := tea.NewProgram(m).Run(); err != nil {
		defer os.Exit(1)
		return err
	}
	return nil
}
