package daemon

import (
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
	d := New(&daemonConfig.Config{}, log.New(&buf, "", 0), info.NewAPI())

	if d.cfg == nil || d.l == nil || d.parse == nil || d.mu == nil || d.api == nil {
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
		cfg:   &daemonConfig.Config{},
		l:     log.New(&buf, "", 0),
		parse: &stubParser{},
		mu:    &sync.Mutex{},
		api:   info.NewAPI(),
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
	origSleep := sleepFn
	origRead := readDaemonConfigFn
	defer func() {
		createProcessGroupFn = origCreate
		setGroupRowFn = origSetRow
		deleteProcessGroupFn = origDelete
		addProcessToGroupFn = origAdd
		sleepFn = origSleep
		readDaemonConfigFn = origRead
	}()

	createProcessGroupFn = func(groupName string) error { return nil }
	setGroupRowFn = func(groupName, row, val string) error { return nil }
	deleteProcessGroupFn = func(groupName string) error { return nil }
	readDaemonConfigFn = func() (daemonConfig.Config, error) {
		return daemonConfig.Config{Run: true, Tick: 1, CPULimit: 10, RAMLimit: 10, CPUQuota: 20, RAMQuota: "1G"}, nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	added := make([]int, 0, 2)
	addProcessToGroupFn = func(pid int, groupName string) error {
		added = append(added, pid)
		if len(added) >= 2 {
			cancel()
		}
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
		},
		l: log.New(&bytes.Buffer{}, "", 0),
		parse: &stubParser{
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
	if !api.InJail(200) || !api.InJail(201) {
		t.Fatal("expected tree members to be marked jailed")
	}
}

func TestRealTimeReadConfigReloadsEnvFile(t *testing.T) {
	dir := t.TempDir()
	prevWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	defer func() { _ = os.Chdir(prevWD) }()

	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}

	env := []byte("DAEMON=true\nDAEMON_TICK=9\nDAEMON_CPU_LIMIT=50\nDAEMON_RAM_QUOTA=3G\n")
	if err := os.WriteFile(filepath.Join(dir, ".env"), env, 0644); err != nil {
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
		return cfg, nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	d := &Daemon{
		cfg:   &daemonConfig.Config{},
		l:     log.New(&bytes.Buffer{}, "", 0),
		parse: &stubParser{},
		mu:    &sync.Mutex{},
		api:   info.NewAPI(),
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
}
