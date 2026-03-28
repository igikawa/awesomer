package collector

import (
	"awesomeProject/pkg/parser"
	"slices"
	"sync"
	"time"
)

const snapshotTTL = 500 * time.Millisecond

type Provider interface {
	Processes() ([]parser.Info, error)
	ProcessTree(pid int32) ([]int32, map[int32][]int32, error)
}

type processSnapshot struct {
	fetchedAt time.Time
	processes []parser.Info
}

type treeSnapshot struct {
	fetchedAt time.Time
	order     []int32
	tree      map[int32][]int32
}

// Collector keeps a short-lived shared snapshot so the TUI and daemon can
// reuse the same /proc scan instead of rebuilding identical state in parallel.
type Collector struct {
	parse     parser.AbstractionLayer
	ttl       time.Duration
	mu        sync.RWMutex
	processes processSnapshot
	trees     map[int32]treeSnapshot
}

func New() *Collector {
	return &Collector{
		parse: parser.NewParser(),
		ttl:   snapshotTTL,
		trees: make(map[int32]treeSnapshot),
	}
}

func (c *Collector) Processes() ([]parser.Info, error) {
	if cached, ok := c.loadProcesses(); ok {
		return cached, nil
	}

	processes, err := c.parse.AllProcesses()
	if err != nil {
		return nil, err
	}

	c.storeProcesses(processes)
	return cloneProcesses(processes), nil
}

func (c *Collector) ProcessTree(pid int32) ([]int32, map[int32][]int32, error) {
	if order, tree, ok := c.loadTree(pid); ok {
		return order, tree, nil
	}

	processes, err := c.Processes()
	if err != nil {
		return nil, nil, err
	}

	tree := make(map[int32][]int32)
	for _, proc := range processes {
		tree[proc.PPID] = append(tree[proc.PPID], proc.PID)
	}

	order := walkTree(pid, tree, nil)
	c.storeTree(pid, order, tree)

	return slices.Clone(order), cloneTree(tree), nil
}

func walkTree(pid int32, tree map[int32][]int32, order []int32) []int32 {
	for _, child := range tree[pid] {
		order = walkTree(child, tree, order)
	}
	return append(order, pid)
}

func (c *Collector) loadProcesses() ([]parser.Info, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.expired(c.processes.fetchedAt) || len(c.processes.processes) == 0 {
		return nil, false
	}

	return cloneProcesses(c.processes.processes), true
}

func (c *Collector) storeProcesses(processes []parser.Info) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.processes = processSnapshot{
		fetchedAt: time.Now(),
		processes: cloneProcesses(processes),
	}
	c.trees = make(map[int32]treeSnapshot)
}

func (c *Collector) loadTree(pid int32) ([]int32, map[int32][]int32, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	tree, ok := c.trees[pid]
	if !ok || c.expired(tree.fetchedAt) {
		return nil, nil, false
	}

	return slices.Clone(tree.order), cloneTree(tree.tree), true
}

func (c *Collector) storeTree(pid int32, order []int32, tree map[int32][]int32) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.trees[pid] = treeSnapshot{
		fetchedAt: time.Now(),
		order:     slices.Clone(order),
		tree:      cloneTree(tree),
	}
}

func (c *Collector) expired(fetchedAt time.Time) bool {
	return fetchedAt.IsZero() || time.Since(fetchedAt) > c.ttl
}

func cloneProcesses(processes []parser.Info) []parser.Info {
	cloned := make([]parser.Info, len(processes))
	copy(cloned, processes)
	return cloned
}

func cloneTree(tree map[int32][]int32) map[int32][]int32 {
	if tree == nil {
		return nil
	}

	cloned := make(map[int32][]int32, len(tree))
	for pid, children := range tree {
		cloned[pid] = slices.Clone(children)
	}

	return cloned
}
