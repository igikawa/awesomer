package tui

import (
	"awesomeProject/internal/config"
	daemonAPI "awesomeProject/internal/daemon/info"
	"awesomeProject/internal/service"
	"awesomeProject/pkg/parser"
	"context"
	"log"
	"os"
	"time"

	"charm.land/bubbles/v2/table"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

const INFO = "Info\n\n" +
	"↑↓ - select service\n\n" +
	"Enter - show service info\n\n" +
	"h - show big service info\n\n" +
	"s - stop service\n\n" +
	"r - resume service\n\n" +
	"k - kill service\n\n" +
	"d - kill service tree\n\n" +
	"q - exit\n\n"

type tickMsg time.Time

type dataMsg struct {
	rows []table.Row
}

type model struct {
	table      table.Model
	info       viewport.Model
	focusTable bool
	Tick       int
	width      int
	height     int

	DaemonCancel context.CancelFunc
	Service      *service.Service
	Logger       *log.Logger
	Parser       *parser.Parser
}

func (m model) tick() tea.Cmd {
	s := time.Duration(m.Tick) * time.Second
	return tea.Tick(s, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func NewTable() table.Model {
	columns := []table.Column{
		{Title: "PID", Width: 10},
		{Title: "Name", Width: 20},
		{Title: "CPU", Width: 7},
		{Title: "Mem", Width: 7},
		{Title: "Threads", Width: 7},
		{Title: "User", Width: 10},
		{Title: "", Width: 2}, // is controlling with daemon
	}

	t := table.New(
		table.WithColumns(columns),
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

func NewInfo() viewport.Model {
	m := viewport.New()
	m.SetContent(INFO)

	return m
}

func Run(daemonCancel context.CancelFunc, cfg *config.Config, l *log.Logger, api *daemonAPI.API, t table.Model, i viewport.Model) error {
	m := model{
		table: t,
		info:  i,
		Tick:  cfg.Tick,

		DaemonCancel: daemonCancel,
		Service:      service.New(api),
		Logger:       l,
		Parser:       parser.NewParser(),
	}

	app := tea.NewProgram(m)
	if _, err := app.Run(); err != nil {
		defer os.Exit(1)
		return err
	}

	return nil
}
