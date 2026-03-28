package tui

import (
	"awesomeProject/internal/collector"
	"awesomeProject/internal/config"
	daemonConfig "awesomeProject/internal/daemon/config"
	"awesomeProject/internal/daemon/info"
	parserpkg "awesomeProject/pkg/parser"
	"bytes"
	"errors"
	"log"
	"strings"
	"testing"
	"time"

	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
)

type stubProgram struct {
	run func() (tea.Model, error)
}

func (s stubProgram) Run() (tea.Model, error) {
	return s.run()
}

type stubService struct {
	rows          []table.Row
	changed       bool
	getErr        error
	affinityErr   error
	noFileErr     error
	jailState     bool
	toggleErr     error
	sortModes     []string
	affinityPID   int
	affinityCores []int
	noFilePID     int
	noFileLimit   uint64
	togglePID     int
	treeText      string
	treeErr       error
}

func (s *stubService) GetProcesses() ([]table.Row, bool, error) { return s.rows, s.changed, s.getErr }
func (s *stubService) SetSortProcMod(mode string)               { s.sortModes = append(s.sortModes, mode) }
func (s *stubService) GetTuiTree(root int32, tree map[int32][]int32) (string, error) {
	return s.treeText, s.treeErr
}
func (s *stubService) StopProcess(pid int) error     { return nil }
func (s *stubService) ResumeProcess(pid int) error   { return nil }
func (s *stubService) KillProcess(pid int) error     { return nil }
func (s *stubService) KillProcessTree(pid int) error { return nil }
func (s *stubService) SetCPUAffinity(pid int, cores []int) error {
	s.affinityPID = pid
	s.affinityCores = append([]int(nil), cores...)
	return s.affinityErr
}
func (s *stubService) SetNoFileLimit(pid int, limit uint64) error {
	s.noFilePID = pid
	s.noFileLimit = limit
	return s.noFileErr
}
func (s *stubService) ToggleProcessJail(pid int) (bool, error) {
	s.togglePID = pid
	return s.jailState, s.toggleErr
}

type stubParser struct {
	info        parserpkg.Info
	infoErr     error
	hardInfo    parserpkg.Info
	hardInfoErr error
	tree        []int32
	treeMap     map[int32][]int32
	treeErr     error
}

func (s *stubParser) ProcessInfo(pid int32) (parserpkg.Info, error) { return s.info, s.infoErr }
func (s *stubParser) ProcessTree(pid int32) ([]int32, map[int32][]int32, error) {
	return s.tree, s.treeMap, s.treeErr
}
func (s *stubParser) HardObjectParse(pid int32) (parserpkg.Info, error) {
	return s.hardInfo, s.hardInfoErr
}

func newStubModel() model {
	ui := mergeUIConfig(config.UIConfig{})
	tbl := NewTable()
	tbl.SetRows([]table.Row{{"123", "proc"}})
	tbl.SetCursor(0)

	infoView := NewInfo()
	infoView.SetWidth(60)
	infoView.SetHeight(12)

	return model{
		table:        tbl,
		info:         infoView,
		focusTable:   true,
		Tick:         1,
		UI:           ui,
		width:        120,
		height:       40,
		tableWidth:   40,
		infoWidth:    60,
		panelH:       20,
		infoBodyH:    16,
		styles:       buildUIStyles(ui),
		DaemonCancel: func() {},
		Service:      &stubService{rows: []table.Row{{"123", "proc"}}, changed: true},
		Logger:       log.New(&bytes.Buffer{}, "", 0),
		Parser: &stubParser{
			info: parserpkg.Info{PID: 123, Name: "proc", User: "user", Cmd: "cmd", CPUAffinity: []int{0, 1}, NoFileSoft: 1024, NoFileHard: 2048},
			hardInfo: parserpkg.Info{
				Connections: []parserpkg.NetworkInfo{{LocalAddr: "127.0.0.1:1", RemoteAddr: "127.0.0.1:2", Status: "ESTABLISHED"}},
				OpenFiles:   []string{"/tmp/file"},
				Children:    []parserpkg.ChildInfo{{PID: 124, Name: "child"}},
			},
			treeMap: map[int32][]int32{123: {124}},
		},
	}
}

func TestRunUsesInjectedProgram(t *testing.T) {
	origService := newServiceFn
	origParser := newParserFn
	origProgram := newProgramFn
	origExit := exitFn
	defer func() {
		newServiceFn = origService
		newParserFn = origParser
		newProgramFn = origProgram
		exitFn = origExit
	}()

	newServiceFn = func(api *info.API, cfg *config.Config, snapshots collector.Provider) serviceAPI {
		return &stubService{}
	}
	newParserFn = func() parserAPI { return &stubParser{} }
	called := false
	newProgramFn = func(m model) program {
		return stubProgram{run: func() (tea.Model, error) {
			called = true
			if m.UI.InfoWidth != config.DefaultUIConfig().InfoWidth {
				t.Fatalf("UI.InfoWidth = %d, want default %d", m.UI.InfoWidth, config.DefaultUIConfig().InfoWidth)
			}
			return m, nil
		}}
	}
	exitFn = func(code int) {}

	cfg := &config.Config{Daemon: daemonConfig.Config{}}
	if err := Run(func() {}, cfg, log.New(&bytes.Buffer{}, "", 0), info.NewAPI(), nil, NewTable(cfg.UI), NewInfo()); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !called {
		t.Fatal("program Run() was not called")
	}
}

func TestRunReturnsProgramError(t *testing.T) {
	origService := newServiceFn
	origParser := newParserFn
	origProgram := newProgramFn
	origExit := exitFn
	defer func() {
		newServiceFn = origService
		newParserFn = origParser
		newProgramFn = origProgram
		exitFn = origExit
	}()

	newServiceFn = func(api *info.API, cfg *config.Config, snapshots collector.Provider) serviceAPI {
		return &stubService{}
	}
	newParserFn = func() parserAPI { return &stubParser{} }
	newProgramFn = func(m model) program {
		return stubProgram{run: func() (tea.Model, error) { return m, errors.New("boom") }}
	}
	exitFn = func(code int) {}

	cfg := &config.Config{}
	if err := Run(func() {}, cfg, log.New(&bytes.Buffer{}, "", 0), info.NewAPI(), nil, NewTable(cfg.UI), NewInfo()); err == nil {
		t.Fatal("Run() error = nil, want non-nil")
	}
}

func TestRunPropagatesCustomUIConfig(t *testing.T) {
	origService := newServiceFn
	origParser := newParserFn
	origProgram := newProgramFn
	origExit := exitFn
	defer func() {
		newServiceFn = origService
		newParserFn = origParser
		newProgramFn = origProgram
		exitFn = origExit
	}()

	newServiceFn = func(api *info.API, cfg *config.Config, snapshots collector.Provider) serviceAPI {
		return &stubService{}
	}
	newParserFn = func() parserAPI { return &stubParser{} }
	newProgramFn = func(m model) program {
		return stubProgram{run: func() (tea.Model, error) {
			if m.UI.TableWidth != 55 {
				t.Fatalf("UI.TableWidth = %d, want 55", m.UI.TableWidth)
			}
			if m.UI.BorderColor != "240" {
				t.Fatalf("UI.BorderColor = %q, want 240", m.UI.BorderColor)
			}
			return m, nil
		}}
	}
	exitFn = func(code int) {}

	cfg := &config.Config{UI: config.UIConfig{TableWidth: 55, BorderColor: "240"}}
	if err := Run(func() {}, cfg, log.New(&bytes.Buffer{}, "", 0), info.NewAPI(), nil, NewTable(cfg.UI), NewInfo()); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
}

func TestTickProducesTickMsg(t *testing.T) {
	m := newStubModel()
	cmd := m.tick()
	msg := cmd()
	if _, ok := msg.(tickMsg); !ok {
		t.Fatalf("tick() returned %T, want tickMsg", msg)
	}
}

func TestTickReturnsNilWhenDisabled(t *testing.T) {
	m := newStubModel()
	m.Tick = 0
	if cmd := m.tick(); cmd != nil {
		t.Fatal("tick() cmd != nil for disabled ticker")
	}
}

func TestInitReturnsTickCommand(t *testing.T) {
	m := newStubModel()
	if cmd := m.Init(); cmd == nil {
		t.Fatal("Init() returned nil cmd")
	}
}

func TestSubmitInputAffinitySuccess(t *testing.T) {
	m := newStubModel()
	svc := m.Service.(*stubService)
	m.inputMode = inputModeAffinity
	m.inputPID = 123
	m.inputValue = "3,1"

	cmd := m.submitInput()
	if svc.affinityPID != 123 || len(svc.affinityCores) != 2 || svc.affinityCores[0] != 1 {
		t.Fatalf("SetCPUAffinity captured pid=%d cores=%v", svc.affinityPID, svc.affinityCores)
	}
	if cmd == nil {
		t.Fatal("submitInput() cmd = nil, want refresh command")
	}
	if !strings.Contains(m.info.View(), "Updated CPU affinity") {
		t.Fatalf("info view = %q", m.info.View())
	}
}

func TestSubmitInputNoFileSuccess(t *testing.T) {
	m := newStubModel()
	svc := m.Service.(*stubService)
	m.inputMode = inputModeNoFile
	m.inputPID = 123
	m.inputValue = "4096"

	cmd := m.submitInput()
	if svc.noFilePID != 123 || svc.noFileLimit != 4096 {
		t.Fatalf("SetNoFileLimit captured pid=%d limit=%d", svc.noFilePID, svc.noFileLimit)
	}
	if cmd == nil {
		t.Fatal("submitInput() cmd = nil, want refresh command")
	}
}

func TestSubmitInputHandlesErrors(t *testing.T) {
	m := newStubModel()
	svc := m.Service.(*stubService)
	svc.affinityErr = errors.New("bad affinity")
	m.inputMode = inputModeAffinity
	m.inputPID = 123
	m.inputValue = "1"

	if cmd := m.submitInput(); cmd != nil {
		t.Fatal("submitInput() cmd != nil on error")
	}
	if !strings.Contains(m.info.View(), "Error: bad affinity") {
		t.Fatalf("info view = %q", m.info.View())
	}
}

func TestUpdateHandlesTickAndSorting(t *testing.T) {
	m := newStubModel()
	svc := m.Service.(*stubService)
	svc.changed = true

	updated, cmd := m.Update(tickMsg(time.Now()))
	if cmd == nil {
		t.Fatal("Update(tickMsg) cmd = nil")
	}
	msg := cmd()
	if msg == nil {
		t.Fatal("tick batch returned nil")
	}

	m2 := updated.(model)
	updated, _ = m2.Update(tea.KeyPressMsg{Code: 'n', Text: "n"})
	if len(svc.sortModes) == 0 || svc.sortModes[0] != "-n" {
		t.Fatalf("sort modes = %v", svc.sortModes)
	}
	_ = updated
}

func TestUpdateHandlesToggleJailAndQuit(t *testing.T) {
	m := newStubModel()
	svc := m.Service.(*stubService)
	svc.jailState = true
	svc.changed = true
	cancelled := false
	m.DaemonCancel = func() { cancelled = true }

	updated, cmd := m.Update(tea.KeyPressMsg{Code: 'j', Text: "j"})
	if cmd == nil {
		t.Fatal("Update(j) cmd = nil")
	}
	if svc.togglePID != 123 {
		t.Fatalf("toggle pid = %d, want 123", svc.togglePID)
	}
	if !strings.Contains(updated.(model).info.View(), "Moved process tree into processJail") {
		t.Fatalf("info view = %q", updated.(model).info.View())
	}

	_, cmd = updated.(model).Update(tea.KeyPressMsg{Code: 'q', Text: "q"})
	if cmd == nil {
		t.Fatal("Update(q) cmd = nil")
	}
	if !cancelled {
		t.Fatal("DaemonCancel was not called")
	}
}

func TestUpdateSkipsTableRefreshWhenRowsUnchanged(t *testing.T) {
	m := newStubModel()
	svc := m.Service.(*stubService)
	svc.rows = []table.Row{{"123", "proc"}}
	svc.changed = false

	updated, cmd := m.Update(tickMsg(time.Now()))
	if cmd == nil {
		t.Fatal("Update(tickMsg) cmd = nil")
	}

	msg := cmd()
	if msg == nil {
		t.Fatal("tick batch returned nil")
	}

	m2 := updated.(model)
	before := m2.table.Rows()
	updated, _ = m2.Update(msg)
	after := updated.(model).table.Rows()
	if len(before) != len(after) {
		t.Fatalf("row count changed from %d to %d", len(before), len(after))
	}
}

func TestViewRendersPanels(t *testing.T) {
	m := newStubModel()
	m.syncLayout()
	view := m.View()
	if !view.AltScreen {
		t.Fatal("View().AltScreen = false, want true")
	}
	if !strings.Contains(view.Content, "PID") || !strings.Contains(view.Content, "Details") {
		t.Fatalf("View().Content = %q", view.Content)
	}
}
