package parser

import (
	"fmt"
	"runtime"
	"sync"

	"awesomeProject/pkg/mutation"
	"github.com/shirou/gopsutil/v4/process"
	"golang.org/x/sys/unix"
)

type AbstractionLayer interface {
	AllProcesses() ([]Info, error)
	ProcessInfo(pid int32) (Info, error)
	ProcessTree(pid int32) ([]int32, map[int32][]int32, error)
	HardObjectParse(pid int32) (Info, error)
}

// Parser is implementing AbstractionLayer
type Parser struct{}

func NewParser() *Parser {
	return &Parser{}
}

func (p *Parser) ProcessTree(pid int32) ([]int32, map[int32][]int32, error) {
	proc, err := p.AllProcesses()
	if err != nil {
		return nil, nil, err
	}

	tree := make(map[int32][]int32)

	for _, v := range proc {
		tree[v.PPID] = append(tree[v.PPID], v.PID)
	}

	result := p.walkingOnAir(pid, tree, []int32{})

	return result, tree, nil
}

func (p *Parser) walkingOnAir(pid int32, tree map[int32][]int32, result []int32) []int32 {
	for _, child := range tree[pid] {
		result = p.walkingOnAir(child, tree, result)
	}
	result = append(result, pid)
	return result
}

func (p *Parser) ProcessInfo(pid int32) (Info, error) {
	proc := process.Process{Pid: pid}

	ppid, err := proc.Ppid()
	if err != nil {
		return Info{}, err
	}

	name, _ := proc.Name()
	cpu, _ := proc.CPUPercent()
	mem, _ := proc.MemoryPercent()
	threads, _ := proc.NumThreads()
	cmd, _ := proc.Cmdline()
	nice, _ := proc.Nice()
	user, _ := proc.Username()
	affinity, _ := mutation.GetCPUaffinity(int(pid))
	noFileSoft, noFileHard, _ := mutation.GetPRlimit(int(pid), unix.RLIMIT_NOFILE)

	return Info{
		PPID:        ppid,
		PID:         pid,
		Name:        name,
		CPUPercent:  cpu,
		MemPercent:  float64(mem),
		Threads:     threads,
		Cmd:         cmd,
		Nice:        nice,
		User:        user,
		CPUAffinity: affinity,
		NoFileSoft:  noFileSoft,
		NoFileHard:  noFileHard,
	}, nil
}

func (p *Parser) AllProcesses() ([]Info, error) {
	proc, err := process.Processes()
	if err != nil {
		return nil, fmt.Errorf("pkg service, GetProcesses: %w", err)
	}

	var info []Info

	numWorkers := runtime.NumCPU() * 2
	jobs := make(chan int32, len(proc))
	results := make(chan Info)
	wg := sync.WaitGroup{}

	wg.Add(numWorkers)
	for i := 0; i < numWorkers; i++ {
		go func() {
			defer wg.Done()
			for pid := range jobs {
				procObj, err := p.ProcessInfo(pid)
				if err == nil {
					results <- procObj
				}
			}
		}()
	}

	for _, j := range proc {
		jobs <- j.Pid
	}
	close(jobs)

	go func() {
		wg.Wait()
		close(results)
	}()

	for processObj := range results {
		info = append(info, processObj)
	}

	return info, nil
}

func (p *Parser) HardObjectParse(pid int32) (Info, error) {
	proc := process.Process{Pid: pid}

	connections, err := proc.Connections()
	if err != nil {
		return Info{}, err
	}
	var formatedConnections []NetworkInfo
	for _, connection := range connections {
		formatedConnections = append(formatedConnections, NetworkInfo{
			LocalAddr:  fmt.Sprintf("%s:%d", connection.Laddr.IP, connection.Laddr.Port),
			RemoteAddr: fmt.Sprintf("%s:%d", connection.Raddr.IP, connection.Raddr.Port),
			Status:     connection.Status,
		})
	}

	openFiles, err := proc.OpenFiles()
	if err != nil {
		return Info{}, err
	}
	var formatedOpenFiles []string
	for _, f := range openFiles {
		formatedOpenFiles = append(
			formatedOpenFiles,
			fmt.Sprintf("%s", f.Path))
	}

	children, err := proc.Children()
	if err != nil {
		return Info{}, err
	}
	var formattedChildren []ChildInfo
	for _, c := range children {
		name, _ := c.Name()
		formattedChildren = append(
			formattedChildren,
			ChildInfo{PID: c.Pid, Name: name},
		)
	}

	return Info{
		Connections: formatedConnections,
		OpenFiles:   formatedOpenFiles,
		Children:    formattedChildren,
	}, nil
}
