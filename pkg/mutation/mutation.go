package mutation

import "golang.org/x/sys/unix"

func SetPRlimit(pid int, limit int, cur, max uint64) error {
	err := unix.Prlimit(
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

	err := unix.SchedSetaffinity(pid, &cpuset)
	if err != nil {
		return err
	}

	return nil
}

func SetCPUattr(pid int, attr *unix.SchedAttr) error {
	err := unix.SchedSetAttr(pid, attr, 0)
	if err != nil {
		return err
	}

	return nil
}
