package tui

import (
	"awesomeProject/internal/collector"
	"awesomeProject/internal/config"
	daemonInfo "awesomeProject/internal/daemon/info"
	"awesomeProject/internal/service"
	parserpkg "awesomeProject/pkg/parser"
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
	"p/n/c/m/t/u - sort by PID, name, CPU, memory, threads, user\n\n" +
	"Enter - show service info\n\n" +
	"h - show big service info\n\n" +
	"A - set CPU affinity\n\n" +
	"L - set RLIMIT_NOFILE\n\n" +
	"J - toggle process jail\n\n" +
	"s - stop service\n\n" +
	"r - resume service\n\n" +
	"k - kill service\n\n" +
	"d - kill service tree\n\n" +
	"q - exit\n\n"

type tickMsg time.Time

type dataMsg struct {
	rows    []table.Row
	changed bool
}

type inputMode int

const (
	inputModeNone inputMode = iota
	inputModeAffinity
	inputModeNoFile
)

type model struct {
	table      table.Model
	info       viewport.Model
	focusTable bool
	Tick       int
	UI         config.UIConfig
	width      int
	height     int
	tableWidth int
	infoWidth  int
	panelH     int
	infoBodyH  int
	inputMode  inputMode
	inputPID   int
	inputValue string
	styles     uiStyles

	DaemonCancel context.CancelFunc
	Service      serviceAPI
	Logger       *log.Logger
	Parser       parserAPI
}

type uiStyles struct {
	activePanel lipgloss.Style
	idlePanel   lipgloss.Style
}

type serviceAPI interface {
	GetProcesses() ([]table.Row, bool, error)
	SetSortProcMod(string)
	CurrentSortProcMod() string
	GetTuiTree(int32, map[int32][]int32) (string, error)
	StopProcess(int) error
	ResumeProcess(int) error
	KillProcess(int) error
	KillProcessTree(int) error
	SetCPUAffinity(int, []int) error
	SetNoFileLimit(int, uint64) error
	ToggleProcessJail(int) (bool, error)
}

type parserAPI interface {
	ProcessInfo(int32) (parserpkg.Info, error)
	ProcessTree(int32) ([]int32, map[int32][]int32, error)
	HardObjectParse(int32) (parserpkg.Info, error)
}

type program interface {
	Run() (tea.Model, error)
}

var (
	newServiceFn = func(api daemonInfo.JailState, cfg *config.Config, snapshots collector.Provider) serviceAPI {
		if controller, ok := api.(service.JailController); ok {
			return service.NewWithController(api, &cfg.Daemon, snapshots, controller)
		}
		return service.New(api, &cfg.Daemon, snapshots)
	}
	newParserFn = func() parserAPI {
		return parserpkg.NewParser()
	}
	newProgramFn = func(m model) program {
		return tea.NewProgram(m)
	}
	exitFn = os.Exit
)

func (m model) tick() tea.Cmd {
	if m.Tick <= 0 {
		return nil
	}

	s := time.Duration(m.Tick) * time.Second
	return tea.Tick(s, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func NewTable(uiCfg ...config.UIConfig) table.Model {
	ui := config.DefaultUIConfig()
	if len(uiCfg) > 0 {
		ui = mergeUIConfig(uiCfg[0])
	}

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
		Foreground(lipgloss.Color(ui.SelectionTextColor)).
		Background(lipgloss.Color(ui.SelectionBackgroundColor)).
		Bold(false)
	t.SetStyles(s)
	applySortHeaderTitles(&t, "-p")

	return t
}

func NewInfo() viewport.Model {
	m := viewport.New()
	m.SoftWrap = true
	m.FillHeight = true
	m.LeftGutterFunc = func(info viewport.GutterContext) string {
		if info.Soft {
			return "> "
		}
		return "  "
	}
	m.SetContent(INFO)

	return m
}

func Run(daemonCancel context.CancelFunc, cfg *config.Config, l *log.Logger, api daemonInfo.JailState, snapshots collector.Provider, t table.Model, i viewport.Model) error {
	ui := mergeUIConfig(cfg.UI)
	m := model{
		table:      t,
		info:       i,
		focusTable: true,
		Tick:       cfg.Tick,
		UI:         ui,
		styles:     buildUIStyles(ui),

		DaemonCancel: daemonCancel,
		Service:      newServiceFn(api, cfg, snapshots),
		Logger:       l,
		Parser:       newParserFn(),
	}
	applySortHeaderTitles(&m.table, m.Service.CurrentSortProcMod())

	app := newProgramFn(m)
	if _, err := app.Run(); err != nil {
		defer exitFn(1)
		return err
	}

	return nil
}

var activeHeaderTitleStyle = lipgloss.NewStyle().Bold(true)

func applySortHeaderTitles(t *table.Model, sortMode string) {
	columns := t.Columns()
	for i := range columns {
		columns[i].Title = headerTitleForColumn(sortMode, i)
	}
	t.SetColumns(columns)
}

func headerTitleForColumn(sortMode string, index int) string {
	title := plainHeaderTitle(index)
	if sortMode == sortModeForColumn(index) && title != "" {
		return activeHeaderTitleStyle.Render(title)
	}
	return title
}

func plainHeaderTitle(index int) string {
	switch index {
	case 0:
		return "PID"
	case 1:
		return "Name"
	case 2:
		return "CPU"
	case 3:
		return "Mem"
	case 4:
		return "Threads"
	case 5:
		return "User"
	default:
		return ""
	}
}

func sortModeForColumn(index int) string {
	switch index {
	case 0:
		return "-p"
	case 1:
		return "-n"
	case 2:
		return "-c"
	case 3:
		return "-m"
	case 4:
		return "-t"
	case 5:
		return "-u"
	default:
		return ""
	}
}

func mergeUIConfig(ui config.UIConfig) config.UIConfig {
	defaults := config.DefaultUIConfig()

	if ui.TableWidth <= 0 {
		ui.TableWidth = defaults.TableWidth
	}
	if ui.InfoWidth <= 0 {
		ui.InfoWidth = defaults.InfoWidth
	}
	if ui.BorderColor == "" {
		ui.BorderColor = defaults.BorderColor
	}
	if ui.ActiveBorderColor == "" {
		ui.ActiveBorderColor = defaults.ActiveBorderColor
	}
	if ui.SelectionTextColor == "" {
		ui.SelectionTextColor = defaults.SelectionTextColor
	}
	if ui.SelectionBackgroundColor == "" {
		ui.SelectionBackgroundColor = defaults.SelectionBackgroundColor
	}

	return ui
}

func buildUIStyles(ui config.UIConfig) uiStyles {
	return uiStyles{
		activePanel: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(ui.ActiveBorderColor)).
			Padding(0, 1),
		idlePanel: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(ui.BorderColor)).
			Padding(0, 1),
	}
}
