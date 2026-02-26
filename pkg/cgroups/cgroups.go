package cgroups

import (
	"awesomeProject/pkg/logger"
	"fmt"
	"os"
	"strconv"
)

// CreateProcessGroup creates new cgroup
func CreateProcessGroup(groupName string) error {
	err := os.Mkdir(fmt.Sprintf("/sys/fs/cgroup/%s", groupName), 0755)
	if err != nil && !os.IsExist(err) {
		logger.Logger.Printf("Failed to create cgroup directory: %v", err)
		return err
	}
	return nil
}

// AddProcessToGroup adds process on group
func AddProcessToGroup(pid int, groupName string) error {
	if !checkExistGroup(groupName) {
		err := CreateProcessGroup(groupName)
		if err != nil {
			return err
		}
	}
	err := os.WriteFile(fmt.Sprintf("/sys/fs/cgroup/%s/cgroup.procs", groupName), []byte(strconv.Itoa(pid)), 0644)
	if err != nil {
		logger.Logger.Printf("Failed to add process to cgroup: %v", err)
		return err
	}
	return nil
}

// SetGroupRow adds row on cgroup. For example row:
// cpu.weight,
// cpu.max,
// cpu.weight.nice,
// memory.max,
func SetGroupRow(groupName, row, val string) error {
	if !checkExistGroup(groupName) {
		err := CreateProcessGroup(groupName)
		if err != nil {
			return err
		}
	}
	err := os.WriteFile(fmt.Sprintf("/sys/fs/cgroup/%s/%s", groupName, row), []byte(val), 0644)
	if err != nil {
		logger.Logger.Printf("cgroups, SetGroupRow: %v", err)
		return err
	}
	return nil
}

func checkExistGroup(groupName string) bool {
	_, err := os.Stat(fmt.Sprintf("/sys/fs/cgroup/%s", groupName))
	return err == nil
}
