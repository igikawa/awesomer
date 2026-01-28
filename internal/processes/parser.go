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
}

type ChildInfo struct {
	PID  int32
	Name string
}

type ProcessInfo struct {
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

func (p *Parser) GetAllProcessess() ([]ProcessInfo, error) {
	proc, err := process.Processes()
	if err != nil {
		return nil, fmt.Errorf("pkg process, GetProcesses: %w", err)
	}

	var info []ProcessInfo

	for _, p := range proc {
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

		info = append(info, ProcessInfo{
			PID:        p.Pid,
			Name:       name,
			CPUPercent: cpu / NumCPU,
			MemPercent: mem,
			Threads:    threads,
		})
	}
	return info, nil
}

func (p *Parser) GetProcessInfo(pid int32) (ProcessInfo, error) {
	proc := process.Process{Pid: pid}
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
