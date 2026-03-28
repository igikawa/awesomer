package service

import (
	daemonConfig "awesomeProject/internal/daemon/config"
	daemonAPI "awesomeProject/internal/daemon/info"
	parser2 "awesomeProject/pkg/parser"
	"errors"
	"slices"
	"strings"
	"sync"
	"testing"

	"golang.org/x/sys/unix"
)

type stubParser struct {
	processes    []parser2.Info
	processTree  []int32
	processMap   map[int32][]int32
	allProcErr   error
	processErr   error
	processInfo  parser2.Info
	hardInfo     parser2.Info
	hardInfoErr  error
	processIDHit int32
}

func (s *stubParser) AllProcesses() ([]parser2.Info, error) {
	if s.allProcErr != nil {
		return nil, s.allProcErr
	}
	return slices.Clone(s.processes), nil
}

func (s *stubParser) ProcessInfo(pid int32) (parser2.Info, error) {
	s.processIDHit = pid
	if s.processErr != nil {
		return parser2.Info{}, s.processErr
	}
	return s.processInfo, nil
}

func (s *stubParser) ProcessTree(pid int32) ([]int32, map[int32][]int32, error) {
	s.processIDHit = pid
	if s.processErr != nil {
		return nil, nil, s.processErr
	}
	return slices.Clone(s.processTree), mapsClone(s.processMap), nil
}

func (s *stubParser) HardObjectParse(pid int32) (parser2.Info, error) {
	s.processIDHit = pid
	if s.hardInfoErr != nil {
		return parser2.Info{}, s.hardInfoErr
	}
	return s.hardInfo, nil
}

func mapsClone(in map[int32][]int32) map[int32][]int32 {
	if in == nil {
		return nil
	}

	out := make(map[int32][]int32, len(in))
	for key, values := range in {
		out[key] = slices.Clone(values)
	}
	return out
}

func newTestService(p parser2.AbstractionLayer) *Service {
	return &Service{
		p:         p,
		mu:        nil,
		daemon:    daemonAPI.NewAPI(),
		daemonCfg: &daemonConfig.Config{CPUQuota: 25, RAMQuota: "512M"},
	}
}

func TestNewInitializesServiceDefaults(t *testing.T) {
	svc := New(daemonAPI.NewAPI(), &daemonConfig.Config{})

	if svc.p == nil {
		t.Fatal("New() parser is nil")
	}
	if svc.mu == nil {
		t.Fatal("New() mutex is nil")
	}
	if svc.daemon == nil {
		t.Fatal("New() daemon API is nil")
	}
	if svc.daemonCfg == nil {
		t.Fatal("New() daemon config is nil")
	}
	if svc.sortProcMod != "empty" {
		t.Fatalf("sortProcMod = %q, want empty", svc.sortProcMod)
	}
}

func TestGetProcessesMarksJailedProcessAndSortsByCPU(t *testing.T) {
	parser := &stubParser{
		processes: []parser2.Info{
			{PID: 10, Name: "slow", CPUPercent: 1, MemPercent: 5, Threads: 2, User: "alice"},
			{PID: 20, Name: "fast", CPUPercent: 9, MemPercent: 1, Threads: 1, User: "bob"},
		},
	}
	svc := newTestService(parser)
	svc.daemon.SetJail(20)
	svc.sortProcMod = "-c"

	rows, err := svc.GetProcesses()
	if err != nil {
		t.Fatalf("GetProcesses() error = %v", err)
	}

	if len(rows) != 2 {
		t.Fatalf("GetProcesses() len = %d, want 2", len(rows))
	}
	if got := rows[0][0]; got != "20" {
		t.Fatalf("rows[0] PID = %s, want 20", got)
	}
	if got := rows[0][6]; got != "*" {
		t.Fatalf("rows[0] jail marker = %q, want *", got)
	}
}

func TestGetProcessesSupportsAllSortModes(t *testing.T) {
	tests := []struct {
		name    string
		mode    string
		wantPID string
	}{
		{name: "sort by name", mode: "-n", wantPID: "20"},
		{name: "sort by mem", mode: "-m", wantPID: "30"},
		{name: "sort by threads", mode: "-t", wantPID: "30"},
		{name: "sort by user", mode: "-u", wantPID: "20"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := &stubParser{
				processes: []parser2.Info{
					{PID: 10, Name: "bravo", CPUPercent: 1, MemPercent: 5, Threads: 2, User: "zoe"},
					{PID: 20, Name: "alpha", CPUPercent: 9, MemPercent: 1, Threads: 1, User: "adam"},
					{PID: 30, Name: "charlie", CPUPercent: 3, MemPercent: 8, Threads: 5, User: "mike"},
				},
			}
			svc := newTestService(parser)
			svc.mu = &sync.RWMutex{}
			svc.SetSortProcMod(tt.mode)

			rows, err := svc.GetProcesses()
			if err != nil {
				t.Fatalf("GetProcesses() error = %v", err)
			}
			if got := rows[0][0]; got != tt.wantPID {
				t.Fatalf("first PID = %s, want %s", got, tt.wantPID)
			}
		})
	}
}

func TestGetProcessesPropagatesParserError(t *testing.T) {
	svc := newTestService(&stubParser{allProcErr: errors.New("boom")})

	if _, err := svc.GetProcesses(); err == nil {
		t.Fatal("GetProcesses() error = nil, want non-nil")
	}
}

func TestGetTuiTreeFormatsTree(t *testing.T) {
	svc := newTestService(&stubParser{})

	out, err := svc.GetTuiTree(10, map[int32][]int32{
		10: {20, 30},
		20: {40},
	})
	if err != nil {
		t.Fatalf("GetTuiTree() error = %v", err)
	}

	for _, fragment := range []string{"10", "20", "30", "40", "├──", "└──"} {
		if !strings.Contains(out, fragment) {
			t.Fatalf("tree output %q does not contain %q", out, fragment)
		}
	}
}

func TestSetCPUAffinityUsesMutationHook(t *testing.T) {
	parser := &stubParser{}
	svc := newTestService(parser)

	orig := setCPUAffinityFn
	defer func() { setCPUAffinityFn = orig }()

	var (
		gotPID   int
		gotCores []int
	)
	setCPUAffinityFn = func(pid int, cores []int) error {
		gotPID = pid
		gotCores = slices.Clone(cores)
		return nil
	}

	err := svc.SetCPUAffinity(42, []int{0, 2, 4})
	if err != nil {
		t.Fatalf("SetCPUAffinity() error = %v", err)
	}
	if gotPID != 42 {
		t.Fatalf("SetCPUAffinity() pid = %d, want 42", gotPID)
	}
	if !slices.Equal(gotCores, []int{0, 2, 4}) {
		t.Fatalf("SetCPUAffinity() cores = %v, want [0 2 4]", gotCores)
	}
}

func TestSetCPUAffinityRejectsEmptyInput(t *testing.T) {
	svc := newTestService(&stubParser{})

	if err := svc.SetCPUAffinity(42, nil); err == nil {
		t.Fatal("SetCPUAffinity() error = nil, want non-nil")
	}
}

func TestSetCPUAffinityPropagatesMutationError(t *testing.T) {
	svc := newTestService(&stubParser{})

	orig := setCPUAffinityFn
	defer func() { setCPUAffinityFn = orig }()
	setCPUAffinityFn = func(pid int, cores []int) error { return errors.New("boom") }

	if err := svc.SetCPUAffinity(42, []int{1}); err == nil {
		t.Fatal("SetCPUAffinity() error = nil, want non-nil")
	}
}

func TestSetNoFileLimitUsesRlimitNoFile(t *testing.T) {
	parser := &stubParser{}
	svc := newTestService(parser)

	orig := setPRLimitFn
	defer func() { setPRLimitFn = orig }()

	var (
		gotPID   int
		gotLimit int
		gotCur   uint64
		gotMax   uint64
	)
	setPRLimitFn = func(pid int, limit int, cur, max uint64) error {
		gotPID = pid
		gotLimit = limit
		gotCur = cur
		gotMax = max
		return nil
	}

	err := svc.SetNoFileLimit(73, 4096)
	if err != nil {
		t.Fatalf("SetNoFileLimit() error = %v", err)
	}
	if gotPID != 73 || gotLimit != unix.RLIMIT_NOFILE || gotCur != 4096 || gotMax != 4096 {
		t.Fatalf("SetNoFileLimit() got pid=%d limit=%d cur=%d max=%d", gotPID, gotLimit, gotCur, gotMax)
	}
}

func TestSetNoFileLimitRejectsZero(t *testing.T) {
	svc := newTestService(&stubParser{})

	if err := svc.SetNoFileLimit(73, 0); err == nil {
		t.Fatal("SetNoFileLimit() error = nil, want non-nil")
	}
}

func TestSetNoFileLimitPropagatesMutationError(t *testing.T) {
	svc := newTestService(&stubParser{})

	orig := setPRLimitFn
	defer func() { setPRLimitFn = orig }()
	setPRLimitFn = func(pid int, limit int, cur, max uint64) error { return errors.New("boom") }

	if err := svc.SetNoFileLimit(73, 1024); err == nil {
		t.Fatal("SetNoFileLimit() error = nil, want non-nil")
	}
}

func TestToggleProcessJailAddsWholeTree(t *testing.T) {
	parser := &stubParser{
		processTree: []int32{101, 202, 303},
		processMap: map[int32][]int32{
			101: {202},
			202: {303},
		},
	}
	svc := newTestService(parser)

	origAdd := addProcessToGroupFn
	origSetRow := setGroupRowFn
	defer func() {
		addProcessToGroupFn = origAdd
		setGroupRowFn = origSetRow
	}()

	var added []int
	var rows [][3]string
	addProcessToGroupFn = func(pid int, groupName string) error {
		if groupName != processJailGroup {
			t.Fatalf("groupName = %s, want %s", groupName, processJailGroup)
		}
		added = append(added, pid)
		return nil
	}
	setGroupRowFn = func(groupName, row, val string) error {
		rows = append(rows, [3]string{groupName, row, val})
		return nil
	}

	inJail, err := svc.ToggleProcessJail(101)
	if err != nil {
		t.Fatalf("ToggleProcessJail() error = %v", err)
	}
	if !inJail {
		t.Fatalf("ToggleProcessJail() returned false, want true")
	}
	if !slices.Equal(added, []int{101, 202, 303}) {
		t.Fatalf("added pids = %v, want [101 202 303]", added)
	}
	if len(rows) != 2 {
		t.Fatalf("setGroupRow calls = %d, want 2", len(rows))
	}
	if !svc.daemon.InJail(101) || !svc.daemon.InJail(202) || !svc.daemon.InJail(303) {
		t.Fatalf("expected all tree members to be marked in jail")
	}
}

func TestToggleProcessJailRemovesWholeTree(t *testing.T) {
	parser := &stubParser{
		processTree: []int32{101, 202},
		processMap:  map[int32][]int32{101: {202}},
	}
	svc := newTestService(parser)
	svc.daemon.SetJail(101)
	svc.daemon.SetJail(202)

	origMove := moveToRootGroupFn
	origSetRow := setGroupRowFn
	defer func() {
		moveToRootGroupFn = origMove
		setGroupRowFn = origSetRow
	}()

	var moved []int
	moveToRootGroupFn = func(pid int) error {
		moved = append(moved, pid)
		return nil
	}
	setGroupRowFn = func(groupName, row, val string) error {
		t.Fatalf("setGroupRow should not be called when removing from jail")
		return nil
	}

	inJail, err := svc.ToggleProcessJail(101)
	if err != nil {
		t.Fatalf("ToggleProcessJail() error = %v", err)
	}
	if inJail {
		t.Fatalf("ToggleProcessJail() returned true, want false")
	}
	if !slices.Equal(moved, []int{101, 202}) {
		t.Fatalf("moved pids = %v, want [101 202]", moved)
	}
	if svc.daemon.InJail(101) || svc.daemon.InJail(202) {
		t.Fatalf("expected tree members to be removed from jail")
	}
}

func TestToggleProcessJailPropagatesErrors(t *testing.T) {
	parser := &stubParser{
		processTree: []int32{1},
		processMap:  map[int32][]int32{},
	}
	svc := newTestService(parser)

	origSetRow := setGroupRowFn
	defer func() { setGroupRowFn = origSetRow }()

	setGroupRowFn = func(groupName, row, val string) error {
		return errors.New("boom")
	}

	_, err := svc.ToggleProcessJail(1)
	if err == nil {
		t.Fatal("ToggleProcessJail() error = nil, want non-nil")
	}
}

func TestToggleProcessJailPropagatesProcessTreeError(t *testing.T) {
	svc := newTestService(&stubParser{processErr: errors.New("tree failed")})

	if _, err := svc.ToggleProcessJail(1); err == nil {
		t.Fatal("ToggleProcessJail() error = nil, want non-nil")
	}
}
