package mutation

import "golang.org/x/sys/unix"

var (
	prlimitFn          = unix.Prlimit
	schedSetaffinityFn = unix.SchedSetaffinity
	schedGetaffinityFn = unix.SchedGetaffinity
	schedSetAttrFn     = unix.SchedSetAttr
)

func SetPRlimit(pid int, limit int, cur, max uint64) error {
	err := prlimitFn(
		pid,
		limit,
		&unix.Rlimit{
			Cur: cur,
			Max: max,
		},
		nil,
	)
	if err != nil {
		return err
	}
	return nil
}

func SetCPUaffinity(pid int, mask []int) error {
	var cpuset unix.CPUSet

	for _, core := range mask {
		cpuset.Set(core)
	}

	err := schedSetaffinityFn(pid, &cpuset)
	if err != nil {
		return err
	}

	return nil
}

func GetPRlimit(pid int, limit int) (uint64, uint64, error) {
	var current unix.Rlimit
	err := prlimitFn(pid, limit, nil, &current)
	if err != nil {
		return 0, 0, err
	}

	return current.Cur, current.Max, nil
}

func GetCPUaffinity(pid int) ([]int, error) {
	var cpuset unix.CPUSet
	err := schedGetaffinityFn(pid, &cpuset)
	if err != nil {
		return nil, err
	}

	cores := make([]int, 0, 64)
	for core := 0; core < 1024; core++ {
		if cpuset.IsSet(core) {
			cores = append(cores, core)
		}
	}

	return cores, nil
}

func SetCPUattr(pid int, attr *unix.SchedAttr) error {
	err := schedSetAttrFn(pid, attr, 0)
	if err != nil {
		return err
	}

	return nil
}
