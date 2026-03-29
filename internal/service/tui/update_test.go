package tui

import (
	"awesomeProject/internal/config"
	daemonConfig "awesomeProject/internal/daemon/config"
	"awesomeProject/internal/daemon/info"
	"awesomeProject/internal/service"
	"awesomeProject/pkg/parser"
	"bytes"
	"log"
	"os"
	"strings"
	"testing"

	"charm.land/bubbles/v2/table"
)

func TestParseCPUCores(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []int
		wantErr string
	}{
		{
			name:  "normal values are parsed and sorted",
			input: "3,1,2",
			want:  []int{1, 2, 3},
		},
		{
			name:  "duplicates are removed",
			input: "2,2,1",
			want:  []int{1, 2},
		},
		{
			name:    "negative values are rejected",
			input:   "0,-1",
			wantErr: "must be >=",
		},
		{
			name:    "non numeric values are rejected",
			input:   "0,a",
			wantErr: "invalid CPU core",
		},
		{
			name:    "empty input is rejected",
			input:   " , ",
			wantErr: "no CPU cores",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseCPUCores(tt.input)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("parseCPUCores() error = nil, want %q", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("parseCPUCores() error = %q, want substring %q", err.Error(), tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Fatalf("parseCPUCores() error = %v", err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("parseCPUCores() len = %d, want %d", len(got), len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("parseCPUCores()[%d] = %d, want %d", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestIntsToCSV(t *testing.T) {
	got := intsToCSV([]int{0, 2, 7})
	if got != "0,2,7" {
		t.Fatalf("intsToCSV() = %q, want %q", got, "0,2,7")
	}
}

func TestIntsToCSVEmptyUsesUnknown(t *testing.T) {
	if got := intsToCSV(nil); got != "unknown" {
		t.Fatalf("intsToCSV(nil) = %q, want unknown", got)
	}
}

func TestMaxMinInt(t *testing.T) {
	if got := maxInt(2, 7); got != 7 {
		t.Fatalf("maxInt() = %d, want 7", got)
	}
	if got := minInt(2, 7); got != 2 {
		t.Fatalf("minInt() = %d, want 2", got)
	}
}

func TestSelectedPID(t *testing.T) {
	m := model{table: NewTable()}
	m.table.SetRows([]table.Row{{"123", "proc"}})
	m.table.SetCursor(0)

	pid, err := m.selectedPID()
	if err != nil {
		t.Fatalf("selectedPID() error = %v", err)
	}
	if pid != 123 {
		t.Fatalf("selectedPID() = %d, want 123", pid)
	}
}

func TestSelectedPIDWithoutSelectionFails(t *testing.T) {
	m := model{table: NewTable()}

	if _, err := m.selectedPID(); err == nil {
		t.Fatal("selectedPID() error = nil, want non-nil")
	}
}

func TestStartAndClearInput(t *testing.T) {
	m := model{
		table:      NewTable(),
		info:       NewInfo(),
		focusTable: true,
	}
	m.info.SetWidth(40)
	m.info.SetHeight(10)

	m.startInput(inputModeAffinity, 55)
	if m.inputMode != inputModeAffinity || m.inputPID != 55 {
		t.Fatalf("startInput() state = mode %v pid %d", m.inputMode, m.inputPID)
	}
	if m.focusTable {
		t.Fatal("focusTable = true, want false after startInput")
	}
	if !strings.Contains(m.info.View(), "CPU affinity") {
		t.Fatalf("info view = %q, want CPU affinity prompt", m.info.View())
	}

	m.clearInput()
	if m.inputMode != inputModeNone || m.inputPID != 0 || m.inputValue != "" {
		t.Fatal("clearInput() did not reset input state")
	}
}

func TestSyncLayoutAssignsDimensions(t *testing.T) {
	ui := mergeUIConfig(config.UIConfig{})
	m := model{
		table:  NewTable(),
		info:   NewInfo(),
		UI:     ui,
		width:  120,
		height: 40,
		styles: buildUIStyles(ui),
	}

	m.syncLayout()

	if m.tableWidth <= 0 || m.infoWidth <= 0 || m.panelH <= 0 || m.infoBodyH <= 0 {
		t.Fatalf("syncLayout() produced invalid sizes: %+v", m)
	}
}

func TestSyncLayoutRespectsConfiguredPanelWidths(t *testing.T) {
	ui := mergeUIConfig(config.UIConfig{TableWidth: 50, InfoWidth: 40})
	m := model{
		table:  NewTable(ui),
		info:   NewInfo(),
		UI:     ui,
		width:  120,
		height: 40,
		styles: buildUIStyles(ui),
	}

	m.syncLayout()

	if m.tableWidth != 50 {
		t.Fatalf("tableWidth = %d, want 50", m.tableWidth)
	}
	if m.infoWidth != 40 {
		t.Fatalf("infoWidth = %d, want 40", m.infoWidth)
	}
}

func TestInfoHeaderAndFooterViews(t *testing.T) {
	ui := mergeUIConfig(config.UIConfig{})
	m := model{
		table:      NewTable(),
		info:       NewInfo(),
		focusTable: false,
		UI:         ui,
		styles:     buildUIStyles(ui),
	}
	m.info.SetContent("line1\nline2\nline3")
	m.info.SetHeight(2)

	if got := m.infoHeaderView(20); !strings.Contains(got, "Details [scroll]") {
		t.Fatalf("infoHeaderView() = %q", got)
	}
	if got := m.infoFooterView(40); !strings.Contains(got, "Lines") {
		t.Fatalf("infoFooterView() = %q", got)
	}
}

func TestApplySortHeaderTitlesMarksActiveColumn(t *testing.T) {
	tbl := NewTable()

	applySortHeaderTitles(&tbl, "-c")

	columns := tbl.Columns()
	if !strings.Contains(columns[2].Title, "CPU") {
		t.Fatalf("CPU column title = %q, want CPU fragment", columns[2].Title)
	}
	if columns[2].Title == "CPU" {
		t.Fatalf("CPU column title = %q, want styled active header", columns[2].Title)
	}
	if columns[0].Title != "PID" {
		t.Fatalf("PID column title = %q, want plain PID", columns[0].Title)
	}
}

func TestMergeUIConfigFillsDefaults(t *testing.T) {
	ui := mergeUIConfig(config.UIConfig{TableWidth: 45})
	defaults := config.DefaultUIConfig()

	if ui.TableWidth != 45 {
		t.Fatalf("TableWidth = %d, want 45", ui.TableWidth)
	}
	if ui.InfoWidth != defaults.InfoWidth {
		t.Fatalf("InfoWidth = %d, want %d", ui.InfoWidth, defaults.InfoWidth)
	}
	if ui.ActiveBorderColor != defaults.ActiveBorderColor {
		t.Fatalf("ActiveBorderColor = %q, want %q", ui.ActiveBorderColor, defaults.ActiveBorderColor)
	}
}

func TestFormatedInfoIncludesCurrentProcess(t *testing.T) {
	m := model{
		table:  NewTable(),
		info:   NewInfo(),
		Logger: log.New(&bytes.Buffer{}, "", 0),
		Parser: parser.NewParser(),
		Service: service.New(
			info.NewAPI(),
			&daemonConfig.Config{},
			nil,
		),
	}

	content := m.formatedInfo(int32(os.Getpid()))
	for _, fragment := range []string{"Selected service", "PID:", "Name:", "Command"} {
		if !strings.Contains(content, fragment) {
			t.Fatalf("formatedInfo() output %q does not contain %q", content, fragment)
		}
	}
}

func TestFormatedBigInfoIncludesSections(t *testing.T) {
	m := model{
		table:  NewTable(),
		info:   NewInfo(),
		Logger: log.New(&bytes.Buffer{}, "", 0),
		Parser: parser.NewParser(),
		Service: service.New(
			info.NewAPI(),
			&daemonConfig.Config{},
			nil,
		),
	}

	content := m.formatedBigInfo(int32(os.Getpid()))
	for _, fragment := range []string{"Extended service details", "Connections", "Opened files", "Child service"} {
		if !strings.Contains(content, fragment) {
			t.Fatalf("formatedBigInfo() output %q does not contain %q", content, fragment)
		}
	}
}
