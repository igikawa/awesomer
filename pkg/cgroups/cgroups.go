package cgroups

import (
	"fmt"
	"os"
	"strconv"
)

// CreateProcessGroup creates new cgroup
func CreateProcessGroup(groupName string) error {
	err := os.Mkdir(fmt.Sprintf("/sys/fs/cgroup/%s", groupName), 0755)
	if err != nil && !os.IsExist(err) {
		return fmt.Errorf("failed to create cgroup directory: %v", err)
	}
	return nil
}

// AddProcessToGroup adds service on group
func AddProcessToGroup(pid int, groupName string) error {
	if !checkExistGroup(groupName) {
		err := CreateProcessGroup(groupName)
		if err != nil {
			return err
		}
	}
	err := os.WriteFile(
		fmt.Sprintf("/sys/fs/cgroup/%s/cgroup.procs", groupName),
		[]byte(strconv.Itoa(pid)),
		0644,
	)
	if err != nil {
		return fmt.Errorf("failed to add process to cgroup: %v", err)
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
		return fmt.Errorf("cgroups, SetGroupRow: %v", err)
	}
	return nil
}

func checkExistGroup(groupName string) bool {
	_, err := os.Stat(fmt.Sprintf("/sys/fs/cgroup/%s", groupName))
	return err == nil
}
