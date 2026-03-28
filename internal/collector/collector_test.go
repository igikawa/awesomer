package collector

import (
	"awesomeProject/pkg/parser"
	"errors"
	"slices"
	"testing"
	"time"
)

type stubParser struct {
	processes []parser.Info
	allErr    error
	allCalls  int
}

func (s *stubParser) AllProcesses() ([]parser.Info, error) {
	s.allCalls++
	if s.allErr != nil {
		return nil, s.allErr
	}
	return cloneProcesses(s.processes), nil
}

func (s *stubParser) ProcessInfo(pid int32) (parser.Info, error) {
	return parser.Info{}, nil
}

func (s *stubParser) ProcessTree(pid int32) ([]int32, map[int32][]int32, error) {
	return nil, nil, nil
}

func (s *stubParser) HardObjectParse(pid int32) (parser.Info, error) {
	return parser.Info{}, nil
}

func newTestCollector(p *stubParser) *Collector {
	return &Collector{
		parse: p,
		ttl:   time.Minute,
		trees: make(map[int32]treeSnapshot),
	}
}

func TestProcessesUsesSharedSnapshot(t *testing.T) {
	p := &stubParser{processes: []parser.Info{{PID: 1, Name: "init"}}}
	c := newTestCollector(p)

	first, err := c.Processes()
	if err != nil {
		t.Fatalf("Processes() error = %v", err)
	}
	second, err := c.Processes()
	if err != nil {
		t.Fatalf("Processes() second error = %v", err)
	}

	if p.allCalls != 1 {
		t.Fatalf("AllProcesses() calls = %d, want 1", p.allCalls)
	}

	first[0].Name = "mutated"
	if second[0].Name != "init" {
		t.Fatalf("cached process name = %q, want init", second[0].Name)
	}
}

func TestProcessTreeReusesCachedProcesses(t *testing.T) {
	p := &stubParser{
		processes: []parser.Info{
			{PID: 10, PPID: 1},
			{PID: 20, PPID: 10},
			{PID: 30, PPID: 10},
		},
	}
	c := newTestCollector(p)

	order, tree, err := c.ProcessTree(10)
	if err != nil {
		t.Fatalf("ProcessTree() error = %v", err)
	}
	if p.allCalls != 1 {
		t.Fatalf("AllProcesses() calls = %d, want 1", p.allCalls)
	}
	if !slices.Equal(order, []int32{20, 30, 10}) {
		t.Fatalf("order = %v, want [20 30 10]", order)
	}
	if !slices.Equal(tree[10], []int32{20, 30}) {
		t.Fatalf("tree[10] = %v, want [20 30]", tree[10])
	}
}

func TestProcessesPropagatesParserError(t *testing.T) {
	c := newTestCollector(&stubParser{allErr: errors.New("boom")})

	if _, err := c.Processes(); err == nil {
		t.Fatal("Processes() error = nil, want non-nil")
	}
}
