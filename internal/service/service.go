package service

import (
	"awesomeProject/internal/collector"
	daemonConfig "awesomeProject/internal/daemon/config"
	daemonInfo "awesomeProject/internal/daemon/info"
	"awesomeProject/pkg/cgroups"
	"awesomeProject/pkg/mutation"
	"awesomeProject/pkg/parser"
	"fmt"
	"os"
	"slices"
	"sort"
	"strings"
	"sync"
	"syscall"

	"charm.land/bubbles/v2/table"
	"golang.org/x/sys/unix"
)

const processJailGroup = "processJail"

var (
	setCPUAffinityFn    = mutation.SetCPUaffinity
	setPRLimitFn        = mutation.SetPRlimit
	addProcessToGroupFn = cgroups.AddProcessToGroup
	moveToRootGroupFn   = cgroups.MoveProcessToRootGroup
	setGroupRowFn       = cgroups.SetGroupRow
)

type Service struct {
	snapshots   collector.Provider
	mu          *sync.RWMutex
	daemon      daemonInfo.JailState
	controller  JailController
	daemonCfg   *daemonConfig.Config
	sortProcMod string
	lastSortMod string
	lastStates  []rowState
	lastRows    []table.Row
}

type JailController interface {
	ToggleProcessJail(pid int) (bool, error)
}

type rowState struct {
	PID     int32
	Name    string
	CPU     float64
	Mem     float64
	Threads int32
	User    string
	InJail  bool
}

func New(d daemonInfo.JailState, daemonCfg *daemonConfig.Config, snapshots collector.Provider) *Service {
	return NewWithController(d, daemonCfg, snapshots, newLocalJailController(d, daemonCfg, snapshots))
}

func NewWithController(d daemonInfo.JailState, daemonCfg *daemonConfig.Config, snapshots collector.Provider, controller JailController) *Service {
	if snapshots == nil {
		snapshots = collector.New()
	}
	if d == nil {
		d = daemonInfo.NewAPI()
	}
	if daemonCfg == nil {
		cfg := daemonConfig.DefaultConfig()
		daemonCfg = &cfg
	}
	if controller == nil {
		controller = unavailableJailController{}
	}

	return &Service{
		snapshots:   snapshots,
		mu:          &sync.RWMutex{},
		daemon:      d,
		controller:  controller,
		daemonCfg:   daemonCfg,
		sortProcMod: "-p",
	}
}

func (s *Service) GetProcesses() ([]table.Row, bool, error) {
	proc, err := s.snapshots.Processes()
	if err != nil {
		return nil, false, fmt.Errorf("pkg service, GetProcesses: %w", err)
	}

	sortMode := s.currentSortMode()
	switch sortMode {
	case "-p":
		s.sortByPID(proc)
	case "-n":
		s.sortByName(proc)
	case "-c":
		s.sortByCPU(proc)
	case "-m":
		s.sortByMem(proc)
	case "-t":
		s.sortByThreads(proc)
	case "-u":
		s.sortByUser(proc)
	}

	states := make([]rowState, 0, len(proc))
	for _, p := range proc {
		states = append(states, rowState{
			PID:     p.PID,
			Name:    p.Name,
			CPU:     p.CPUPercent,
			Mem:     p.MemPercent,
			Threads: p.Threads,
			User:    p.User,
			InJail:  s.daemon.InJail(int(p.PID)),
		})
	}

	if rows, ok := s.cachedRows(sortMode, states); ok {
		return rows, false, nil
	}

	rows := make([]table.Row, 0, len(states))
	for _, state := range states {
		out := ""
		if state.InJail {
			out = "*"
		}
		rows = append(rows, table.Row{
			fmt.Sprintf("%d", state.PID),
			state.Name,
			fmt.Sprintf("%.2f %%", state.CPU),
			fmt.Sprintf("%.2f %%", state.Mem),
			fmt.Sprintf("%d", state.Threads),
			state.User,
			out,
		})
	}

	s.storeRows(sortMode, states, rows)
	return cloneRows(rows), true, nil
}

func (s *Service) SetSortProcMod(sortMod string) {
	s.mu.Lock()
	s.sortProcMod = sortMod
	s.mu.Unlock()
}

func (s *Service) currentSortMode() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.sortProcMod
}

func (s *Service) CurrentSortProcMod() string {
	return s.currentSortMode()
}

func (s *Service) cachedRows(sortMode string, states []rowState) ([]table.Row, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if sortMode != s.lastSortMod || !slices.Equal(states, s.lastStates) {
		return nil, false
	}

	return cloneRows(s.lastRows), true
}

func (s *Service) storeRows(sortMode string, states []rowState, rows []table.Row) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.lastSortMod = sortMode
	s.lastStates = slices.Clone(states)
	s.lastRows = cloneRows(rows)
}

func cloneRows(rows []table.Row) []table.Row {
	cloned := make([]table.Row, len(rows))
	for i := range rows {
		cloned[i] = slices.Clone(rows[i])
	}
	return cloned
}

func (s *Service) sortByCPU(proc []parser.Info) {
	slices.SortFunc(proc, func(a, b parser.Info) int {
		if a.CPUPercent > b.CPUPercent {
			return -1
		} else if a.CPUPercent < b.CPUPercent {
			return 1
		}
		return 0
	})
}

func (s *Service) sortByPID(proc []parser.Info) {
	slices.SortFunc(proc, func(a, b parser.Info) int {
		if a.PID < b.PID {
			return -1
		} else if a.PID > b.PID {
			return 1
		}
		return 0
	})
}

func (s *Service) sortByMem(proc []parser.Info) {
	slices.SortFunc(proc, func(a, b parser.Info) int {
		if a.MemPercent > b.MemPercent {
			return -1
		} else if a.MemPercent < b.MemPercent {
			return 1
		}
		return 0
	})
}

func (s *Service) sortByThreads(proc []parser.Info) {
	slices.SortFunc(proc, func(a, b parser.Info) int {
		if a.Threads > b.Threads {
			return -1
		} else if a.Threads < b.Threads {
			return 1
		}
		return 0
	})
}

func (s *Service) sortByName(proc []parser.Info) {
	sort.Slice(proc, func(i, j int) bool {
		iName := proc[i].Name
		jName := proc[j].Name
		return iName < jName
	})
}

func (s *Service) sortByUser(proc []parser.Info) {
	sort.Slice(proc, func(i, j int) bool {
		iUser := proc[i].User
		jUser := proc[j].User
		return iUser < jUser
	})
}

// GetTuiTree renders a lightweight ASCII tree so the details panel can show
// parent-child process relationships without extra UI widgets.
func (s *Service) GetTuiTree(root int32, tree map[int32][]int32) (string, error) {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("%d\n", root))

	var walk func(int32, string)
	walk = func(pid int32, prefix string) {
		children, ok := tree[pid]
		if !ok || len(children) == 0 {
			return
		}

		sort.Slice(children, func(i, j int) bool { return children[i] < children[j] })

		for i, child := range children {
			isLast := i == len(children)-1

			connector := "├── "
			nextPrefix := "│   "
			if isLast {
				connector = "└── "
				nextPrefix = "    "
			}

			sb.WriteString(prefix)
			sb.WriteString(connector)
			sb.WriteString(fmt.Sprintf("%d\n", child))

			walk(child, prefix+nextPrefix)
		}
	}

	walk(root, "")

	return sb.String(), nil
}

func (s *Service) StopProcess(pid int) error {
	stop, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("pkg service, StopProcesses: %w", err)
	}
	err = stop.Signal(syscall.SIGSTOP)
	if err != nil {
		return fmt.Errorf("pkg service, StopProcesses: %w", err)
	}
	return nil
}

func (s *Service) ResumeProcess(pid int) error {
	resume, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("pkg service, ResumeProcesses: %w", err)
	}
	err = resume.Signal(syscall.SIGCONT)
	if err != nil {
		return fmt.Errorf("pkg service, ResumeProcesses: %w", err)
	}
	return nil
}

func (s *Service) KillProcess(pid int) error {
	kill, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("pkg service, CompleteProcesses: %w", err)
	}
	err = kill.Signal(syscall.SIGKILL)
	if err != nil {
		return fmt.Errorf("pkg service, CompleteProcesses: %w", err)
	}
	return nil
}

func (s *Service) KillProcessTree(pid int) error {
	tree, _, err := s.snapshots.ProcessTree(int32(pid))
	if err != nil {
		return fmt.Errorf("pkg service, KillProcessTree: %w", err)
	}

	for i := range tree {
		kill, err := os.FindProcess(int(tree[i]))
		if err != nil {
			return fmt.Errorf("pkg service, CompleteProcesses: %w", err)
		}
		err = kill.Signal(syscall.SIGKILL)
		if err != nil {
			return fmt.Errorf("pkg service, CompleteProcesses: %w", err)
		}
	}

	return nil
}

func (s *Service) SetCPUAffinity(pid int, cores []int) error {
	if len(cores) == 0 {
		return fmt.Errorf("pkg service, SetCPUAffinity: no CPU cores provided")
	}

	if err := setCPUAffinityFn(pid, cores); err != nil {
		return fmt.Errorf("pkg service, SetCPUAffinity: %w", err)
	}

	return nil
}

func (s *Service) SetNoFileLimit(pid int, limit uint64) error {
	if limit == 0 {
		return fmt.Errorf("pkg service, SetNoFileLimit: limit must be greater than 0")
	}

	if err := setPRLimitFn(pid, unix.RLIMIT_NOFILE, limit, limit); err != nil {
		return fmt.Errorf("pkg service, SetNoFileLimit: %w", err)
	}

	return nil
}

func (s *Service) ToggleProcessJail(pid int) (bool, error) {
	inJail, err := s.controller.ToggleProcessJail(pid)
	if err != nil {
		return false, fmt.Errorf("pkg service, ToggleProcessJail: %w", err)
	}
	return inJail, nil
}

type localJailController struct {
	state     daemonInfo.JailState
	daemonCfg *daemonConfig.Config
	snapshots collector.Provider
}

func newLocalJailController(state daemonInfo.JailState, daemonCfg *daemonConfig.Config, snapshots collector.Provider) JailController {
	if snapshots == nil {
		snapshots = collector.New()
	}
	return localJailController{
		state:     state,
		daemonCfg: daemonCfg,
		snapshots: snapshots,
	}
}

func (c localJailController) ToggleProcessJail(pid int) (bool, error) {
	tree, _, err := c.snapshots.ProcessTree(int32(pid))
	if err != nil {
		return false, err
	}

	inJail := c.state.InJail(pid)
	if !inJail {
		if err = setGroupRowFn(processJailGroup, "cpu.max", fmt.Sprintf("%d 100000", int(c.daemonCfg.CPUQuota)*1000)); err != nil {
			return false, err
		}
		if err = setGroupRowFn(processJailGroup, "memory.max", c.daemonCfg.RAMQuota); err != nil {
			return false, err
		}
	}

	for _, memberPID := range tree {
		targetPID := int(memberPID)
		if inJail {
			if err = moveToRootGroupFn(targetPID); err != nil {
				return false, err
			}
			c.state.DeleteFromJail(targetPID)
			continue
		}
		if err = addProcessToGroupFn(targetPID, processJailGroup); err != nil {
			return false, err
		}
		c.state.SetJail(targetPID)
	}

	return !inJail, nil
}

type unavailableJailController struct{}

func (unavailableJailController) ToggleProcessJail(pid int) (bool, error) {
	return false, fmt.Errorf("daemon control is unavailable")
}
