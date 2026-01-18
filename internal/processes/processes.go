package processes

import (
	"fmt"
	"os"
	"runtime"
	"slices"
	"syscall"

	"github.com/charmbracelet/bubbles/table"
	"github.com/shirou/gopsutil/v4/process"
)

func GetProcesses() ([]table.Row, error) {
	proc, err := process.Processes()
	if err != nil {
		return nil, fmt.Errorf("pkg process, GetProcesses: %w", err)
	}
	slices.SortFunc(proc, func(a, b *process.Process) int {
		aPercent, _ := a.CPUPercent()
		bPercent, _ := b.CPUPercent()

		if aPercent > bPercent {
			return -1
		} else if aPercent < bPercent {
			return 1
		}

		return 0
	})
	var info []table.Row
	for _, p := range proc {
		name, err := p.Name()
		if err != nil {
			return nil, fmt.Errorf("pkg process, GetProcesses: %w", err)
		}
		cpu, err := p.CPUPercent()
		if err != nil {
			return nil, fmt.Errorf("pkg process, GetProcesses: %w", err)
		}
		mem, err := p.MemoryPercent()
		if err != nil {
			return nil, fmt.Errorf("pkg process, GetProcesses: %w", err)
		}
		threads, err := p.NumThreads()
		if err != nil {
			return nil, fmt.Errorf("pkg process, GetProcesses: %w", err)
		}

		info = append(info, table.Row{
			fmt.Sprintf("%d", p.Pid),
			fmt.Sprintf("%s", name),
			fmt.Sprintf("%d %%", int(cpu)/runtime.NumCPU()),
			fmt.Sprintf("%.2f %%", mem),
			fmt.Sprintf("%d", threads),
		})
	}
	return info, nil
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
