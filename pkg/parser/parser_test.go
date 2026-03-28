package parser

import (
	"os"
	"strings"
	"testing"
)

func TestWalkingOnAirReturnsPostOrderTree(t *testing.T) {
	p := NewParser()
	tree := map[int32][]int32{
		1: {2, 3},
		2: {4},
	}

	got := p.walkingOnAir(1, tree, nil)
	want := []int32{4, 2, 3, 1}

	if len(got) != len(want) {
		t.Fatalf("walkingOnAir() len = %d, want %d", len(got), len(want))
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("walkingOnAir()[%d] = %d, want %d", i, got[i], want[i])
		}
	}
}

func TestProcessInfoReturnsCurrentProcessData(t *testing.T) {
	p := NewParser()
	pid := int32(os.Getpid())

	info, err := p.ProcessInfo(pid)
	if err != nil {
		t.Fatalf("ProcessInfo() error = %v", err)
	}
	if info.PID != pid {
		t.Fatalf("PID = %d, want %d", info.PID, pid)
	}
	if info.Name == "" {
		t.Fatal("Name is empty")
	}
}

func TestProcessInfoInvalidPidReturnsError(t *testing.T) {
	p := NewParser()

	if _, err := p.ProcessInfo(-1); err == nil {
		t.Fatal("ProcessInfo(-1) error = nil, want non-nil")
	}
}

func TestAllProcessesIncludesCurrentProcess(t *testing.T) {
	p := NewParser()
	currentPID := int32(os.Getpid())

	procs, err := p.AllProcesses()
	if err != nil {
		t.Fatalf("AllProcesses() error = %v", err)
	}
	if len(procs) == 0 {
		t.Fatal("AllProcesses() returned no processes")
	}

	found := false
	for _, proc := range procs {
		if proc.PID == currentPID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("current pid %d not found in process list", currentPID)
	}
}

func TestProcessTreeContainsRoot(t *testing.T) {
	p := NewParser()
	currentPID := int32(os.Getpid())

	tree, rels, err := p.ProcessTree(currentPID)
	if err != nil {
		t.Fatalf("ProcessTree() error = %v", err)
	}
	if len(tree) == 0 {
		t.Fatal("ProcessTree() returned empty traversal")
	}
	if tree[len(tree)-1] != currentPID {
		t.Fatalf("root pid at tail = %d, want %d", tree[len(tree)-1], currentPID)
	}
	if rels == nil {
		t.Fatal("relations map is nil")
	}
}

func TestHardObjectParseReturnsStructuredData(t *testing.T) {
	p := NewParser()
	info, err := p.HardObjectParse(int32(os.Getpid()))
	if err != nil {
		t.Fatalf("HardObjectParse() error = %v", err)
	}
	if info.Connections == nil && info.OpenFiles == nil && info.Children == nil {
		t.Fatal("HardObjectParse() returned no structured data fields")
	}
}

func TestHardObjectParseInvalidPidReturnsError(t *testing.T) {
	p := NewParser()

	_, err := p.HardObjectParse(-1)
	if err == nil {
		t.Fatal("HardObjectParse(-1) error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), "no such file") {
		t.Fatalf("HardObjectParse(-1) error = %q", err.Error())
	}
}
