package processes

import (
	"fmt"
	"os"
	"slices"
	"sort"
	"strings"
	"syscall"

	"github.com/charmbracelet/bubbles/table"
)

var SortMode string

func GetProcesses(sortMod string) ([]table.Row, error) {
	proc, err := ParserObj.GetAllProcessess()
	if err != nil {
		return nil, fmt.Errorf("pkg process, GetProcesses: %w", err)
	}

	switch sortMod {
	case "-n":
		sortByName(proc)
	case "-c":
		sortByCPU(proc)
	case "-m":
		sortByMem(proc)
	case "-t":
		sortByThreads(proc)
	}

	var info []table.Row
	for _, p := range proc {
		info = append(info, table.Row{
			fmt.Sprintf("%d", p.PID),
			fmt.Sprintf("%s", p.Name),
			fmt.Sprintf("%.2f %%", p.CPUPercent),
			fmt.Sprintf("%.2f %%", p.MemPercent),
			fmt.Sprintf("%d", p.Threads),
		})
	}
	return info, nil
}

func sortByCPU(proc []ProcessInfo) {
	slices.SortFunc(proc, func(a, b ProcessInfo) int {
		if a.CPUPercent > b.CPUPercent {
			return -1
		} else if a.CPUPercent < b.CPUPercent {
			return 1
		}
		return 0
	})
}

func sortByMem(proc []ProcessInfo) {
	slices.SortFunc(proc, func(a, b ProcessInfo) int {
		if a.MemPercent > b.MemPercent {
			return -1
		} else if a.MemPercent < b.MemPercent {
			return 1
		}
		return 0
	})
}

func sortByThreads(proc []ProcessInfo) {
	slices.SortFunc(proc, func(a, b ProcessInfo) int {
		if a.Threads > b.Threads {
			return -1
		} else if a.Threads < b.Threads {
			return 1
		}
		return 0
	})
}

func sortByName(proc []ProcessInfo) {
	sort.Slice(proc, func(i, j int) bool {
		iName := proc[i].Name
		jName := proc[j].Name
		return iName < jName
	})
}

func StopProcess(pid int) error {
	stop, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("pkg process, StopProcesses: %w", err)
	}
	err = stop.Signal(syscall.SIGSTOP)
	if err != nil {
		return fmt.Errorf("pkg process, StopProcesses: %w", err)
	}
	return nil
}

func ResumeProcess(pid int) error {
	resume, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("pkg process, ResumeProcesses: %w", err)
	}
	err = resume.Signal(syscall.SIGCONT)
	if err != nil {
		return fmt.Errorf("pkg process, ResumeProcesses: %w", err)
	}
	return nil
}

func KillProcess(pid int) error {
	kill, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("pkg process, CompleteProcesses: %w", err)
	}
	err = kill.Signal(syscall.SIGKILL)
	if err != nil {
		return fmt.Errorf("pkg process, CompleteProcesses: %w", err)
	}
	return nil
}

func KillProcessTree(pid int) error {
	tree, _, err := ParserObj.GetProcessTree(int32(pid))
	if err != nil {
		return fmt.Errorf("pkg process, KillProcessTree: %w", err)
	}

	for i := range tree {
		kill, err := os.FindProcess(int(tree[i]))
		if err != nil {
			return fmt.Errorf("pkg process, CompleteProcesses: %w", err)
		}
		err = kill.Signal(syscall.SIGKILL)
		if err != nil {
			return fmt.Errorf("pkg process, CompleteProcesses: %w", err)
		}
	}

	return nil
}

func GetTuiTree(root int32, tree map[int32][]int32) (string, error) {
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
