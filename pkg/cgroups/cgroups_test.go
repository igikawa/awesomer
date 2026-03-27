package cgroups

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCreateAndDeleteProcessGroup(t *testing.T) {
	tmp := t.TempDir()
	origRoot := cgroupRootPath
	cgroupRootPath = tmp
	defer func() { cgroupRootPath = origRoot }()

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
	cgroupRootPath = tmp
	defer func() { cgroupRootPath = origRoot }()

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
	cgroupRootPath = tmp
	defer func() { cgroupRootPath = origRoot }()

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
	cgroupRootPath = tmp
	defer func() { cgroupRootPath = origRoot }()

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
	cgroupRootPath = tmp
	defer func() { cgroupRootPath = origRoot }()

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
