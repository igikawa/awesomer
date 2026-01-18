package main

import (
	"awesomeProject/internal/config"
	"awesomeProject/internal/processes"
	"strconv"

	"log"
	"os"
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var baseStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("240"))

var Logger *log.Logger

type tickMsg time.Time

type model struct {
	table table.Model
	Tick  int
}

func (m model) Init() tea.Cmd {
	return m.tick()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tickMsg:
		switch m.Tick {
		case 0:
			break
		default:
			rows, err := processes.GetProcesses()
			if err != nil {
				Logger.Println(err)
			}
			m.table.SetRows(rows)
			return m, m.tick()
		}
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
		case "d":
			name := m.table.SelectedRow()[1]
			pid, err := strconv.Atoi(m.table.SelectedRow()[0])
			if err != nil {
				Logger.Println(err)
			}
			err = processes.KillProcess(pid)
			if err != nil {
				Logger.Println(err)
			}
			return m, tea.Batch(
				tea.Printf("Kill: %s!", name),
			)
		case "s":
			pid, err := strconv.Atoi(m.table.SelectedRow()[0])
			if err != nil {
				Logger.Println(err)
			}
			err = processes.StopProcess(pid)
			if err != nil {
				Logger.Println(err)
			}
			return m, tea.Batch(
				tea.Printf("Stop: %s!", m.table.SelectedRow()[0]),
			)
		case "r":
			pid, err := strconv.Atoi(m.table.SelectedRow()[0])
			if err != nil {
				Logger.Println(err)
			}
			err = processes.ResumeProcess(pid)
			if err != nil {
				Logger.Println(err)
			}
			return m, tea.Batch(
				tea.Printf("Resume: %s!", m.table.SelectedRow()[0]),
			)
		}
	}
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m model) View() string {
	return baseStyle.Render(m.table.View()) + "\n"
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

	rows, err := processes.GetProcesses()
	if err != nil {
		Logger.Println(err)
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(7),
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

func main() {
	logFile, err := os.OpenFile("./awesome.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		panic(err)
	}
	Logger = log.New(logFile, "", log.Ldate|log.Ltime|log.Lshortfile)
	t := newTable()

	m := model{t, config.NewConfig().Tick}
	if _, err := tea.NewProgram(m).Run(); err != nil {
		Logger.Println("Error running program:", err)
		os.Exit(1)
	}
}
