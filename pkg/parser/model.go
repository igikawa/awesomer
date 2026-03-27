package parser

type ChildInfo struct {
	PID  int32
	Name string
}

type NetworkInfo struct {
	LocalAddr  string
	RemoteAddr string
	Status     string
}

type Info struct {
	PPID        int32
	PID         int32
	Name        string
	CPUPercent  float64
	MemPercent  float64
	Threads     int32
	Cmd         string
	Nice        int32
	User        string
	CPUAffinity []int
	NoFileSoft  uint64
	NoFileHard  uint64

	// big rows it is parsing a HardObjectParse func:
	Connections []NetworkInfo
	OpenFiles   []string
	Children    []ChildInfo
}
