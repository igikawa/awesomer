package process

import (
	"awesomeProject/pkg/logger"

	"fmt"

	"github.com/shirou/gopsutil/v4/process"
)

func init() {
	ParserObj = NewParser()
}

var (
	ParserObj ParserAbstractionLayer
)

type ParserAbstractionLayer interface {
	AllProcessess() ([]Info, error)
	ProcessInfo(pid int32) (Info, error)
	ProcessTree(pid int32) ([]int32, map[int32][]int32, error)
}

type ChildInfo struct {
	PID  int32
	Name string
}

type Info struct {
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

func (p *Parser) ProcessTree(pid int32) ([]int32, map[int32][]int32, error) {
	proc, err := p.AllProcessess()
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

	return Info{
		PPID:       ppid,
		PID:        pid,
		Name:       name,
		CPUPercent: cpu,
		MemPercent: mem,
		Threads:    threads,
		Cmd:        cmd,
		OpenFiles:  formatedOpenFiles,
		Children:   formattedChildren,
	}, nil
}

func (p *Parser) AllProcessess() ([]Info, error) {
	proc, err := process.Processes()
	if err != nil {
		return nil, fmt.Errorf("pkg process, GetProcesses: %w", err)
	}

	var info []Info

	for _, processObj := range proc {
		proc, _ := p.ProcessInfo(processObj.Pid)
		info = append(info, proc)
	}

	return info, nil
}
