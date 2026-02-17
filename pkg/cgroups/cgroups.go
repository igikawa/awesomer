package cgroups

import (
	"awesomeProject/pkg/logger"
	"fmt"
	"os"
	"strconv"
)

// TODO implement cgroups pkg

// AddProcessToGroup create a new directory(group) for every process
func AddProcessToGroup(pid int, name string) error {
	err := os.Mkdir(fmt.Sprintf("/sys/fs/cgroup/%s", name), 0755)
	if err != nil && !os.IsExist(err) {
		logger.Logger.Fatalf("Failed to create cgroup directory: %v", err)
		return err
	}
	err = os.WriteFile(fmt.Sprintf("/sys/fs/cgroup/%s/cgroup.procs", name), []byte(strconv.Itoa(pid)), 0666)
	if err != nil {
		logger.Logger.Fatalf("Failed to add process to cgroup: %v", err)
		return err
	}
	return nil
}

func SetCPUWeight(pid int, groupName, weight string) error {
	err := AddProcessToGroup(pid, groupName)
	if err != nil {
		logger.Logger.Fatalf("cgroups, SetCPUWeight: %v", err)
		return err
	}
	err = os.WriteFile(fmt.Sprintf("/sys/fs/cgroup/%s/cpu.weight", groupName), []byte(weight), 0777)
	if err != nil {
		logger.Logger.Fatalf("cgroups, SetCPUWeight: %v", err)
		return err
	}
	return nil
}

func SetCPUMax(pid int, groupName, quota string) error {
	err := AddProcessToGroup(pid, groupName)
	if err != nil {
		logger.Logger.Fatalf("cgroups, SetCPUMax: %v", err)
		return err
	}
	err = os.WriteFile(fmt.Sprintf("/sys/fs/cgroup/%s/cpu.max", groupName), []byte(quota), 0777)
	if err != nil {
		logger.Logger.Fatalf("cgroups, SetCPUQuota: %v", err)
		return err
	}
	return nil
}

func SetCPUNice(pid int, groupName, core string) error {
	err := AddProcessToGroup(pid, groupName)
	if err != nil {
		logger.Logger.Fatalf("cgroups, SetCPUNice: %v", err)
		return err
	}
	err = os.WriteFile(fmt.Sprintf("/sys/fs/cgroup/%s/cpu.weight.nice", groupName), []byte(core), 0777)
	if err != nil {
		logger.Logger.Fatalf("cgroups, SetCPUNice: %v", err)
		return err
	}
	return nil
}

func SetMemoryMax(pid int, groupName, memory string) error {
	err := AddProcessToGroup(pid, groupName)
	if err != nil {
		logger.Logger.Fatalf("cgroups, SetCPUMax: %v", err)
		return err
	}
	err = os.WriteFile(fmt.Sprintf("/sys/fs/cgroup/%s/memory.max", groupName), []byte(memory), 0777)
	if err != nil {
		logger.Logger.Fatalf("cgroups, SetMemoryLimit: %v", err)
		return err
	}
	return nil
}
