package service

import (
	daemonAPI "awesomeProject/internal/daemon/info"
	parser2 "awesomeProject/pkg/parser"
	"sync"

	"fmt"
	"os"
	"slices"
	"sort"
	"strings"
	"syscall"

	"charm.land/bubbles/v2/table"
)

type Service struct {
	p           *parser2.Parser
	mu          *sync.RWMutex
	daemon      *daemonAPI.API
	sortProcMod string
}

func New(d *daemonAPI.API) *Service {
	return &Service{
		p:           parser2.NewParser(),
		mu:          &sync.RWMutex{},
		daemon:      d,
		sortProcMod: "empty",
	}
}

func (s *Service) GetProcesses() ([]table.Row, error) {
	proc, err := s.p.AllProcesses()
	if err != nil {
		return nil, fmt.Errorf("pkg service, GetProcesses: %w", err)
	}

	switch s.sortProcMod {
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

	var info []table.Row
	for _, p := range proc {
		var out string
		if isControlling := s.daemon.InJail(int(p.PID)); isControlling {
			out = "*"
		}
		info = append(info, table.Row{
			fmt.Sprintf("%d", p.PID),
			fmt.Sprintf("%s", p.Name),
			fmt.Sprintf("%.2f %%", p.CPUPercent),
			fmt.Sprintf("%.2f %%", p.MemPercent),
			fmt.Sprintf("%d", p.Threads),
			fmt.Sprintf("%s", p.User),
			fmt.Sprintf("%s", out),
		})
	}
	return info, nil
}

func (s *Service) SetSortProcMod(sortMod string) {
	s.mu.Lock()
	s.sortProcMod = sortMod
	s.mu.Unlock()
}

func (s *Service) sortByCPU(proc []parser2.Info) {
	slices.SortFunc(proc, func(a, b parser2.Info) int {
		if a.CPUPercent > b.CPUPercent {
			return -1
		} else if a.CPUPercent < b.CPUPercent {
			return 1
		}
		return 0
	})
}

func (s *Service) sortByMem(proc []parser2.Info) {
	slices.SortFunc(proc, func(a, b parser2.Info) int {
		if a.MemPercent > b.MemPercent {
			return -1
		} else if a.MemPercent < b.MemPercent {
			return 1
		}
		return 0
	})
}

func (s *Service) sortByThreads(proc []parser2.Info) {
	slices.SortFunc(proc, func(a, b parser2.Info) int {
		if a.Threads > b.Threads {
			return -1
		} else if a.Threads < b.Threads {
			return 1
		}
		return 0
	})
}

func (s *Service) sortByName(proc []parser2.Info) {
	sort.Slice(proc, func(i, j int) bool {
		iName := proc[i].Name
		jName := proc[j].Name
		return iName < jName
	})
}

func (s *Service) sortByUser(proc []parser2.Info) {
	sort.Slice(proc, func(i, j int) bool {
		iUser := proc[i].User
		jUser := proc[j].User
		return iUser < jUser
	})
}

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
	tree, _, err := s.p.ProcessTree(int32(pid))
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
