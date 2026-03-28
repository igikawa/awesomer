package daemon

import (
	rootConfig "awesomeProject/internal/config"
	daemonConfig "awesomeProject/internal/daemon/config"
	"awesomeProject/internal/daemon/info"
	parser2 "awesomeProject/pkg/parser"
	"bytes"
	"context"
	"errors"
	"log"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

type stubParser struct {
	allProcesses []parser2.Info
	tree         []int32
	treeMap      map[int32][]int32
	allErr       error
	treeErr      error
}

func (s *stubParser) AllProcesses() ([]parser2.Info, error) {
	return s.allProcesses, s.allErr
}

func (s *stubParser) Processes() ([]parser2.Info, error) {
	return s.allProcesses, s.allErr
}

func (s *stubParser) ProcessInfo(pid int32) (parser2.Info, error) {
	return parser2.Info{}, nil
}

func (s *stubParser) ProcessTree(pid int32) ([]int32, map[int32][]int32, error) {
	return s.tree, s.treeMap, s.treeErr
}

func (s *stubParser) HardObjectParse(pid int32) (parser2.Info, error) {
	return parser2.Info{}, nil
}

func TestNewInitializesDependencies(t *testing.T) {
	var buf bytes.Buffer
	d := New(&daemonConfig.Config{}, log.New(&buf, "", 0), info.NewAPI(), &stubParser{})

	if d.cfg == nil || d.l == nil || d.snapshots == nil || d.mu == nil || d.api == nil {
		t.Fatal("New() returned daemon with nil dependency")
	}
}

func TestRunReturnsCreateGroupError(t *testing.T) {
	origCreate := createProcessGroupFn
	defer func() { createProcessGroupFn = origCreate }()

	createProcessGroupFn = func(groupName string) error {
		return errors.New("create failed")
	}

	var buf bytes.Buffer
	d := &Daemon{
		cfg:       &daemonConfig.Config{},
		l:         log.New(&buf, "", 0),
		snapshots: &stubParser{},
		mu:        &sync.Mutex{},
		api:       info.NewAPI(),
	}

	err := d.Run(context.Background())
	if err == nil {
		t.Fatal("Run() error = nil, want non-nil")
	}
}

func TestRunJailsProcessAfterThreeViolationsAndCleansUp(t *testing.T) {
	origCreate := createProcessGroupFn
	origSetRow := setGroupRowFn
	origDelete := deleteProcessGroupFn
	origAdd := addProcessToGroupFn
	origMove := moveToRootGroupFn
	origSleep := sleepFn
	origRead := readDaemonConfigFn
	defer func() {
		createProcessGroupFn = origCreate
		setGroupRowFn = origSetRow
		deleteProcessGroupFn = origDelete
		addProcessToGroupFn = origAdd
		moveToRootGroupFn = origMove
		sleepFn = origSleep
		readDaemonConfigFn = origRead
	}()

	createProcessGroupFn = func(groupName string) error { return nil }
	setGroupRowFn = func(groupName, row, val string) error { return nil }
	deleteProcessGroupFn = func(groupName string) error { return nil }
	readDaemonConfigFn = func() (daemonConfig.Config, error) {
		return daemonConfig.Config{Run: true, Tick: 1, CPULimit: 10, RAMLimit: 10, CPUQuota: 20, RAMQuota: "1G", Whitelist: []string{"systemd", "sshd"}}, nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	added := make([]int, 0, 2)
	var moved []int
	addProcessToGroupFn = func(pid int, groupName string) error {
		added = append(added, pid)
		if len(added) >= 2 {
			cancel()
		}
		return nil
	}

	moveToRootGroupFn = func(pid int) error {
		moved = append(moved, pid)
		return nil
	}
	sleepFn = func(d time.Duration) {}

	api := info.NewAPI()
	d := &Daemon{
		cfg: &daemonConfig.Config{
			Run:      true,
			Tick:     1,
			CPULimit: 10,
			RAMLimit: 10,
			CPUQuota: 20,
			RAMQuota: "1G",
			Whitelist: []string{
				"systemd",
				"sshd",
			},
		},
		l: log.New(&bytes.Buffer{}, "", 0),
		snapshots: &stubParser{
			allProcesses: []parser2.Info{{PID: 200, Name: "hog", CPUPercent: 99}},
			tree:         []int32{200, 201},
			treeMap:      map[int32][]int32{200: {201}},
		},
		mu:  &sync.Mutex{},
		api: api,
	}

	if err := d.Run(ctx); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if len(added) != 2 {
		t.Fatalf("added pids = %v, want [200 201]", added)
	}
	if len(moved) != 2 {
		t.Fatalf("moved pids = %v, want [200 201]", moved)
	}
	if api.InJail(200) || api.InJail(201) {
		t.Fatal("expected tree members to be removed from jail after stop")
	}
}

func TestRealTimeReadConfigReloadsConfigFile(t *testing.T) {
	dir := t.TempDir()
	prevWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	defer func() { _ = os.Chdir(prevWD) }()

	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}

	configYAML := []byte("daemon:\n  run: true\n  tick: 9\n  cpu_limit: 50\n  ram_quota: 3G\n  whitelist:\n    - sshd\n    - dockerd\n")
	if err := os.WriteFile(filepath.Join(dir, rootConfig.FileName), configYAML, 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	origRead := readDaemonConfigFn
	defer func() { readDaemonConfigFn = origRead }()
	readDaemonConfigFn = func() (daemonConfig.Config, error) {
		var cfg daemonConfig.Config
		cfg.Run = true
		cfg.Tick = 9
		cfg.CPULimit = 50
		cfg.RAMQuota = "3G"
		cfg.Whitelist = []string{"sshd", "dockerd"}
		return cfg, nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	d := &Daemon{
		cfg:       &daemonConfig.Config{},
		l:         log.New(&bytes.Buffer{}, "", 0),
		snapshots: &stubParser{},
		mu:        &sync.Mutex{},
		api:       info.NewAPI(),
	}

	done := make(chan struct{})
	go func() {
		d.realTimeReadConfig(ctx)
		close(done)
	}()

	time.Sleep(1100 * time.Millisecond)
	cancel()
	<-done

	if !d.cfg.Run || d.cfg.Tick != 9 || d.cfg.CPULimit != 50 || d.cfg.RAMQuota != "3G" {
		t.Fatalf("cfg not reloaded: %+v", *d.cfg)
	}
	if len(d.cfg.Whitelist) != 2 || d.cfg.Whitelist[1] != "dockerd" {
		t.Fatalf("cfg whitelist not reloaded: %+v", d.cfg.Whitelist)
	}
}

func TestRunStopsWhenConfigTurnsDaemonOff(t *testing.T) {
	origCreate := createProcessGroupFn
	origSetRow := setGroupRowFn
	origDelete := deleteProcessGroupFn
	origMove := moveToRootGroupFn
	origSleep := sleepFn
	origRead := readDaemonConfigFn
	defer func() {
		createProcessGroupFn = origCreate
		setGroupRowFn = origSetRow
		deleteProcessGroupFn = origDelete
		moveToRootGroupFn = origMove
		sleepFn = origSleep
		readDaemonConfigFn = origRead
	}()

	createProcessGroupFn = func(groupName string) error { return nil }
	setGroupRowFn = func(groupName, row, val string) error { return nil }

	deleteCalls := 0
	deleteProcessGroupFn = func(groupName string) error {
		deleteCalls++
		return nil
	}

	readCalls := 0
	readDaemonConfigFn = func() (daemonConfig.Config, error) {
		readCalls++
		return daemonConfig.Config{Run: false, Tick: 1}, nil
	}

	var moved []int
	moveToRootGroupFn = func(pid int) error {
		moved = append(moved, pid)
		return nil
	}
	sleepFn = func(d time.Duration) {}

	d := &Daemon{
		cfg: &daemonConfig.Config{
			Run:       true,
			Tick:      1,
			CPULimit:  10,
			RAMLimit:  10,
			CPUQuota:  20,
			RAMQuota:  "1G",
			Whitelist: []string{"systemd", "sshd"},
		},
		l:         log.New(&bytes.Buffer{}, "", 0),
		snapshots: &stubParser{},
		mu:        &sync.Mutex{},
		api:       info.NewAPI(),
	}
	d.api.SetJail(101)
	d.api.SetJail(202)

	if err := d.Run(context.Background()); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if deleteCalls != 1 {
		t.Fatalf("deleteProcessGroupFn call count = %d, want 1", deleteCalls)
	}
	if readCalls != 1 {
		t.Fatalf("readDaemonConfigFn call count = %d, want 1", readCalls)
	}
	if len(moved) != 2 {
		t.Fatalf("moveToRootGroupFn call count = %d, want 2", len(moved))
	}
	if d.cfg.Run {
		t.Fatal("Daemon config remained enabled after reload")
	}
}

func TestApplyLimitsSkipsWhitelistedProcesses(t *testing.T) {
	d := &Daemon{
		cfg: &daemonConfig.Config{
			Run:       true,
			Tick:      1,
			CPULimit:  10,
			RAMLimit:  10,
			CPUQuota:  20,
			RAMQuota:  "1G",
			Whitelist: []string{"systemd", "dockerd"},
		},
		l:         log.New(&bytes.Buffer{}, "", 0),
		snapshots: &stubParser{},
		mu:        &sync.Mutex{},
		api:       info.NewAPI(),
	}

	violations := make(map[int]int)
	procs := []parser2.Info{
		{PID: 200, Name: "dockerd", CPUPercent: 99, MemPercent: 90},
		{PID: 201, Name: "worker", CPUPercent: 99, MemPercent: 90},
	}

	active := d.applyLimits(procs, *d.cfg, violations)

	if !active[200] || !active[201] {
		t.Fatalf("active PIDs = %v, want both processes tracked", active)
	}
	if violations[200] != 0 {
		t.Fatalf("violations for whitelisted process = %d, want 0", violations[200])
	}
	if violations[201] != 1 {
		t.Fatalf("violations for regular process = %d, want 1", violations[201])
	}
}

func TestApplyLimitsSkipsProcessesAlreadyInJail(t *testing.T) {
	api := info.NewAPI()
	api.SetJail(250)

	d := &Daemon{
		cfg: &daemonConfig.Config{
			Run:       true,
			Tick:      1,
			CPULimit:  10,
			RAMLimit:  10,
			CPUQuota:  20,
			RAMQuota:  "1G",
			Whitelist: []string{"systemd", "sshd"},
		},
		l:         log.New(&bytes.Buffer{}, "", 0),
		snapshots: &stubParser{},
		mu:        &sync.Mutex{},
		api:       api,
	}

	violations := make(map[int]int)
	d.applyLimits([]parser2.Info{{PID: 250, Name: "hog", CPUPercent: 99, MemPercent: 90}}, *d.cfg, violations)

	if violations[250] != 0 {
		t.Fatalf("violations for jailed process = %d, want 0", violations[250])
	}
}
