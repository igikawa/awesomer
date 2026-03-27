package cgroups

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

var cgroupRootPath = "/sys/fs/cgroup"

func groupPath(groupName string) string {
	return strings.TrimRight(cgroupRootPath, "/") + "/" + groupName
}

func rootGroupPath() string {
	return strings.TrimRight(cgroupRootPath, "/") + "/cgroup.procs"
}

// CreateProcessGroup creates new cgroup
func CreateProcessGroup(groupName string) error {
	err := os.Mkdir(groupPath(groupName), 0755)
	if err != nil && !os.IsExist(err) {
		return fmt.Errorf("failed to create cgroup directory: %v", err)
	}
	return nil
}

// DeleteProcessGroup deletes group of process
func DeleteProcessGroup(groupName string) error {
	err := os.Remove(groupPath(groupName))
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove cgroup directory: %v", err)
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
		groupPath(groupName)+"/cgroup.procs",
		[]byte(strconv.Itoa(pid)),
		0644,
	)
	if err != nil {
		return fmt.Errorf("failed to add process to cgroup: %v", err)
	}
	return nil
}

// MoveProcessToRootGroup moves a process back to the root cgroup.
func MoveProcessToRootGroup(pid int) error {
	err := os.WriteFile(rootGroupPath(), []byte(strconv.Itoa(pid)), 0644)
	if err != nil {
		return fmt.Errorf("failed to move process to root cgroup: %v", err)
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
	err := os.WriteFile(groupPath(groupName)+"/"+row, []byte(val), 0644)
	if err != nil {
		return fmt.Errorf("cgroups, SetGroupRow: %v", err)
	}
	return nil
}

func checkExistGroup(groupName string) bool {
	_, err := os.Stat(groupPath(groupName))
	return err == nil
}
