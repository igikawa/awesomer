package processes

import (
	"awesomeProject/pkg/logger"

	"fmt"
	"runtime"

	"github.com/shirou/gopsutil/v4/process"
)

func init() {
	ParserObj = NewParser()
	NumCPU = float64(runtime.NumCPU())
}

var (
	ParserObj ParserAbstractionLayer
	NumCPU    float64
)

type ParserAbstractionLayer interface {
	GetAllProcessess() ([]ProcessInfo, error)
	GetProcessInfo(pid int32) (ProcessInfo, error)
	GetProcessTree(pid int32) ([]int32, map[int32][]int32, error)
}

type ChildInfo struct {
	PID  int32
	Name string
}

type ProcessInfo struct {
	PPID       int32
	PID        int32
	Name       string
	CPUPercent float64
	MemPercent float32
	Threads    int32
	Cmd        string
	OpenFiles  []string
	Children   []ChildInfo
}

type Parser struct{}

func NewParser() *Parser {
	return &Parser{}
}

func (p *Parser) GetProcessTree(pid int32) ([]int32, map[int32][]int32, error) {
	proc, err := p.GetAllProcessess()
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

func (p *Parser) GetAllProcessess() ([]ProcessInfo, error) {
	proc, err := process.Processes()
	if err != nil {
		return nil, fmt.Errorf("pkg process, GetProcesses: %w", err)
	}

	var info []ProcessInfo

	for _, p := range proc {
		ppid, err := p.Ppid()
		if err != nil {
			logger.Logger.Println(err)
		}

		name, err := p.Name()
		if err != nil {
			logger.Logger.Println(err)
		}

		cpu, err := p.CPUPercent()
		if err != nil {
			logger.Logger.Println(err)
		}

		mem, err := p.MemoryPercent()
		if err != nil {
			logger.Logger.Println(err)
		}

		threads, err := p.NumThreads()
		if err != nil {
			logger.Logger.Println(err)
		}

		children, err := p.Children()
		if err != nil {
			logger.Logger.Println(err)
		}
		if err != nil {
			logger.Logger.Println(err)
		}
		var formattedChildren []ChildInfo
		for _, c := range children {
			name, _ := c.Name()
			formattedChildren = append(
				formattedChildren,
				ChildInfo{PID: c.Pid, Name: name},
			)
		}

		info = append(info, ProcessInfo{
			PPID:       ppid,
			PID:        p.Pid,
			Name:       name,
			CPUPercent: cpu / NumCPU,
			MemPercent: mem,
			Threads:    threads,
			Children:   formattedChildren,
		})
	}
	return info, nil
}

func (p *Parser) GetProcessInfo(pid int32) (ProcessInfo, error) {
	proc := process.Process{Pid: pid}

	ppid, err := proc.Ppid()
	if err != nil {
		logger.Logger.Println(err)
	}

	name, err := proc.Name()
	if err != nil {
		logger.Logger.Println(err)
	}

	cpu, err := proc.CPUPercent()
	if err != nil {
		logger.Logger.Println(err)
	}

	mem, err := proc.MemoryPercent()
	if err != nil {
		logger.Logger.Println(err)
	}

	threads, err := proc.NumThreads()
	if err != nil {
		logger.Logger.Println(err)
	}

	cmd, err := proc.Cmdline()

	openFiles, err := proc.OpenFiles()
	if err != nil {
		logger.Logger.Println(err)
	}
	var formatedOpenFiles []string
	for _, f := range openFiles {
		formatedOpenFiles = append(
			formatedOpenFiles,
			fmt.Sprintf("%s", f.Path))
	}

	children, err := proc.Children()
	if err != nil {
		logger.Logger.Println(err)
	}
	var formattedChildren []ChildInfo
	for _, c := range children {
		name, _ := c.Name()
		formattedChildren = append(
			formattedChildren,
			ChildInfo{PID: c.Pid, Name: name},
		)
	}

	return ProcessInfo{
		PPID:       ppid,
		PID:        pid,
		Name:       name,
		CPUPercent: cpu / NumCPU,
		MemPercent: mem,
		Threads:    threads,
		Cmd:        cmd,
		OpenFiles:  formatedOpenFiles,
		Children:   formattedChildren,
	}, nil
}
