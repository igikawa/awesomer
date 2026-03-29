package cgroups

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateAndDeleteProcessGroup(t *testing.T) {
	tmp := t.TempDir()
	origRoot := cgroupRootPath
	origProcess1 := process1CommPath
	origRunDir := systemdRunDir
	cgroupRootPath = tmp
	process1CommPath = filepath.Join(tmp, "proc1comm")
	systemdRunDir = filepath.Join(tmp, "run-systemd")
	defer func() {
		cgroupRootPath = origRoot
		process1CommPath = origProcess1
		systemdRunDir = origRunDir
	}()

	if err := CreateProcessGroup("jail"); err != nil {
		t.Fatalf("CreateProcessGroup() error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(tmp, "jail")); err != nil {
		t.Fatalf("Stat(created group) error = %v", err)
	}

	if err := DeleteProcessGroup("jail"); err != nil {
		t.Fatalf("DeleteProcessGroup() error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(tmp, "jail")); !os.IsNotExist(err) {
		t.Fatalf("group still exists after delete, err = %v", err)
	}
}

func TestAddProcessToGroupCreatesGroupAndWritesPid(t *testing.T) {
	tmp := t.TempDir()
	origRoot := cgroupRootPath
	origProcess1 := process1CommPath
	origRunDir := systemdRunDir
	cgroupRootPath = tmp
	process1CommPath = filepath.Join(tmp, "proc1comm")
	systemdRunDir = filepath.Join(tmp, "run-systemd")
	defer func() {
		cgroupRootPath = origRoot
		process1CommPath = origProcess1
		systemdRunDir = origRunDir
	}()

	if err := os.Mkdir(filepath.Join(tmp, "demo"), 0755); err != nil {
		t.Fatalf("Mkdir() error = %v", err)
	}

	if err := AddProcessToGroup(321, "demo"); err != nil {
		t.Fatalf("AddProcessToGroup() error = %v", err)
	}

	data, err := os.ReadFile(filepath.Join(tmp, "demo", "cgroup.procs"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(data) != "321" {
		t.Fatalf("cgroup.procs = %q, want 321", string(data))
	}
}

func TestMoveProcessToRootGroupWritesPid(t *testing.T) {
	tmp := t.TempDir()
	origRoot := cgroupRootPath
	origProcess1 := process1CommPath
	origRunDir := systemdRunDir
	cgroupRootPath = tmp
	process1CommPath = filepath.Join(tmp, "proc1comm")
	systemdRunDir = filepath.Join(tmp, "run-systemd")
	defer func() {
		cgroupRootPath = origRoot
		process1CommPath = origProcess1
		systemdRunDir = origRunDir
	}()

	if err := MoveProcessToRootGroup(111); err != nil {
		t.Fatalf("MoveProcessToRootGroup() error = %v", err)
	}

	data, err := os.ReadFile(filepath.Join(tmp, "cgroup.procs"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(data) != "111" {
		t.Fatalf("root cgroup.procs = %q, want 111", string(data))
	}
}

func TestSetGroupRowWritesConfiguredValue(t *testing.T) {
	tmp := t.TempDir()
	origRoot := cgroupRootPath
	origProcess1 := process1CommPath
	origRunDir := systemdRunDir
	cgroupRootPath = tmp
	process1CommPath = filepath.Join(tmp, "proc1comm")
	systemdRunDir = filepath.Join(tmp, "run-systemd")
	defer func() {
		cgroupRootPath = origRoot
		process1CommPath = origProcess1
		systemdRunDir = origRunDir
	}()

	if err := SetGroupRow("demo", "cpu.max", "10000 100000"); err != nil {
		t.Fatalf("SetGroupRow() error = %v", err)
	}

	data, err := os.ReadFile(filepath.Join(tmp, "demo", "cpu.max"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(data) != "10000 100000" {
		t.Fatalf("cpu.max = %q, want %q", string(data), "10000 100000")
	}
}

func TestCheckExistGroup(t *testing.T) {
	tmp := t.TempDir()
	origRoot := cgroupRootPath
	origProcess1 := process1CommPath
	origRunDir := systemdRunDir
	cgroupRootPath = tmp
	process1CommPath = filepath.Join(tmp, "proc1comm")
	systemdRunDir = filepath.Join(tmp, "run-systemd")
	defer func() {
		cgroupRootPath = origRoot
		process1CommPath = origProcess1
		systemdRunDir = origRunDir
	}()

	if checkExistGroup("missing") {
		t.Fatal("checkExistGroup(missing) = true, want false")
	}

	if err := os.Mkdir(filepath.Join(tmp, "exists"), 0755); err != nil {
		t.Fatalf("Mkdir() error = %v", err)
	}
	if !checkExistGroup("exists") {
		t.Fatal("checkExistGroup(exists) = false, want true")
	}
}

func TestUseSystemdDetectsSystemdEnvironment(t *testing.T) {
	tmp := t.TempDir()
	origProcess1 := process1CommPath
	origRunDir := systemdRunDir
	origLookPath := lookPathFn
	process1CommPath = filepath.Join(tmp, "proc1comm")
	systemdRunDir = filepath.Join(tmp, "run-systemd")
	defer func() {
		process1CommPath = origProcess1
		systemdRunDir = origRunDir
		lookPathFn = origLookPath
	}()

	if err := os.WriteFile(process1CommPath, []byte("systemd\n"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if err := os.MkdirAll(systemdRunDir, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	lookPathFn = func(file string) (string, error) {
		return "/usr/bin/" + file, nil
	}

	if !UseSystemd() {
		t.Fatal("UseSystemd() = false, want true")
	}
}

func TestAddProcessToGroupUsesSystemdUnitControlGroup(t *testing.T) {
	tmp := t.TempDir()
	origRoot := cgroupRootPath
	origProcess1 := process1CommPath
	origRunDir := systemdRunDir
	origLookPath := lookPathFn
	origRunCommand := runCommandFn
	cgroupRootPath = tmp
	process1CommPath = filepath.Join(tmp, "proc1comm")
	systemdRunDir = filepath.Join(tmp, "run-systemd")
	defer func() {
		cgroupRootPath = origRoot
		process1CommPath = origProcess1
		systemdRunDir = origRunDir
		lookPathFn = origLookPath
		runCommandFn = origRunCommand
	}()

	if err := os.WriteFile(process1CommPath, []byte("systemd\n"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if err := os.MkdirAll(systemdRunDir, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.MkdirAll(filepath.Join(tmp, "system.slice", "demo.service"), 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	lookPathFn = func(file string) (string, error) {
		return "/usr/bin/" + file, nil
	}

	runCommandFn = func(name string, args ...string) (string, error) {
		if name == "systemctl" && len(args) >= 5 && args[0] == "show" {
			return "/system.slice/demo.service", nil
		}
		if name == "systemd-run" {
			return "", nil
		}
		return "", nil
	}

	if err := AddProcessToGroup(321, "demo"); err != nil {
		t.Fatalf("AddProcessToGroup() error = %v", err)
	}

	data, err := os.ReadFile(filepath.Join(tmp, "system.slice", "demo.service", "cgroup.procs"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(data) != "321" {
		t.Fatalf("cgroup.procs = %q, want 321", string(data))
	}
}

func TestSetGroupRowUsesSystemdProperties(t *testing.T) {
	tmp := t.TempDir()
	origRoot := cgroupRootPath
	origProcess1 := process1CommPath
	origRunDir := systemdRunDir
	origLookPath := lookPathFn
	origRunCommand := runCommandFn
	cgroupRootPath = tmp
	process1CommPath = filepath.Join(tmp, "proc1comm")
	systemdRunDir = filepath.Join(tmp, "run-systemd")
	defer func() {
		cgroupRootPath = origRoot
		process1CommPath = origProcess1
		systemdRunDir = origRunDir
		lookPathFn = origLookPath
		runCommandFn = origRunCommand
	}()

	if err := os.WriteFile(process1CommPath, []byte("systemd\n"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if err := os.MkdirAll(systemdRunDir, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.MkdirAll(filepath.Join(tmp, "system.slice", "demo.service"), 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	lookPathFn = func(file string) (string, error) {
		return "/usr/bin/" + file, nil
	}

	var calls []string
	runCommandFn = func(name string, args ...string) (string, error) {
		call := name + " " + strings.Join(args, " ")
		calls = append(calls, call)
		if name == "systemctl" && len(args) >= 5 && args[0] == "show" {
			return "/system.slice/demo.service", nil
		}
		return "", nil
	}

	if err := SetGroupRow("demo", "cpu.max", "20000 100000"); err != nil {
		t.Fatalf("SetGroupRow(cpu.max) error = %v", err)
	}
	if err := SetGroupRow("demo", "memory.max", "8G"); err != nil {
		t.Fatalf("SetGroupRow(memory.max) error = %v", err)
	}

	if len(calls) < 4 {
		t.Fatalf("runCommandFn calls = %d, want at least 4", len(calls))
	}
	if !strings.Contains(strings.Join(calls, "\n"), "systemctl set-property demo.service CPUQuota=20%") {
		t.Fatalf("calls = %v, want CPUQuota mapping", calls)
	}
	if !strings.Contains(strings.Join(calls, "\n"), "systemctl set-property demo.service MemoryMax=8G") {
		t.Fatalf("calls = %v, want MemoryMax mapping", calls)
	}
}

func TestCreateProcessGroupUsesSystemdCollectMode(t *testing.T) {
	tmp := t.TempDir()
	origRoot := cgroupRootPath
	origProcess1 := process1CommPath
	origRunDir := systemdRunDir
	origLookPath := lookPathFn
	origRunCommand := runCommandFn
	cgroupRootPath = tmp
	process1CommPath = filepath.Join(tmp, "proc1comm")
	systemdRunDir = filepath.Join(tmp, "run-systemd")
	defer func() {
		cgroupRootPath = origRoot
		process1CommPath = origProcess1
		systemdRunDir = origRunDir
		lookPathFn = origLookPath
		runCommandFn = origRunCommand
	}()

	if err := os.WriteFile(process1CommPath, []byte("systemd\n"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if err := os.MkdirAll(systemdRunDir, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	lookPathFn = func(file string) (string, error) {
		return "/usr/bin/" + file, nil
	}

	var calls []string
	runCommandFn = func(name string, args ...string) (string, error) {
		calls = append(calls, name+" "+strings.Join(args, " "))
		if name == "systemctl" && len(args) >= 5 && args[0] == "show" {
			return "", nil
		}
		return "", nil
	}

	if err := CreateProcessGroup("demo"); err != nil {
		t.Fatalf("CreateProcessGroup() error = %v", err)
	}

	got := strings.Join(calls, "\n")
	if !strings.Contains(got, "systemd-run --quiet --unit demo.service --service-type=exec --property=CollectMode=inactive-or-failed --property=CPUAccounting=yes --property=MemoryAccounting=yes") {
		t.Fatalf("calls = %v, want systemd-run with CollectMode", calls)
	}
}

func TestDeleteProcessGroupResetsSystemdFailedState(t *testing.T) {
	tmp := t.TempDir()
	origRoot := cgroupRootPath
	origProcess1 := process1CommPath
	origRunDir := systemdRunDir
	origLookPath := lookPathFn
	origRunCommand := runCommandFn
	cgroupRootPath = tmp
	process1CommPath = filepath.Join(tmp, "proc1comm")
	systemdRunDir = filepath.Join(tmp, "run-systemd")
	defer func() {
		cgroupRootPath = origRoot
		process1CommPath = origProcess1
		systemdRunDir = origRunDir
		lookPathFn = origLookPath
		runCommandFn = origRunCommand
	}()

	if err := os.WriteFile(process1CommPath, []byte("systemd\n"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if err := os.MkdirAll(systemdRunDir, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.MkdirAll(filepath.Join(tmp, "system.slice", "demo.service"), 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	lookPathFn = func(file string) (string, error) {
		return "/usr/bin/" + file, nil
	}

	var calls []string
	runCommandFn = func(name string, args ...string) (string, error) {
		calls = append(calls, name+" "+strings.Join(args, " "))
		if name == "systemctl" && len(args) >= 5 && args[0] == "show" {
			return "/system.slice/demo.service", nil
		}
		return "", nil
	}

	if err := DeleteProcessGroup("demo"); err != nil {
		t.Fatalf("DeleteProcessGroup() error = %v", err)
	}

	got := strings.Join(calls, "\n")
	if !strings.Contains(got, "systemctl stop demo.service") {
		t.Fatalf("calls = %v, want systemctl stop", calls)
	}
	if !strings.Contains(got, "systemctl reset-failed demo.service") {
		t.Fatalf("calls = %v, want systemctl reset-failed", calls)
	}
}

func TestDeleteProcessGroupIgnoresResetFailedWhenUnitAlreadyUnloaded(t *testing.T) {
	tmp := t.TempDir()
	origRoot := cgroupRootPath
	origProcess1 := process1CommPath
	origRunDir := systemdRunDir
	origLookPath := lookPathFn
	origRunCommand := runCommandFn
	cgroupRootPath = tmp
	process1CommPath = filepath.Join(tmp, "proc1comm")
	systemdRunDir = filepath.Join(tmp, "run-systemd")
	defer func() {
		cgroupRootPath = origRoot
		process1CommPath = origProcess1
		systemdRunDir = origRunDir
		lookPathFn = origLookPath
		runCommandFn = origRunCommand
	}()

	if err := os.WriteFile(process1CommPath, []byte("systemd\n"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if err := os.MkdirAll(systemdRunDir, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.MkdirAll(filepath.Join(tmp, "system.slice", "demo.service"), 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	lookPathFn = func(file string) (string, error) {
		return "/usr/bin/" + file, nil
	}

	runCommandFn = func(name string, args ...string) (string, error) {
		if name == "systemctl" && len(args) >= 5 && args[0] == "show" {
			return "/system.slice/demo.service", nil
		}
		if name == "systemctl" && len(args) >= 2 && args[0] == "reset-failed" {
			return "", fmt.Errorf("exit status 1: Failed to reset failed state of unit demo.service: Unit demo.service not loaded")
		}
		return "", nil
	}

	if err := DeleteProcessGroup("demo"); err != nil {
		t.Fatalf("DeleteProcessGroup() error = %v", err)
	}
}
