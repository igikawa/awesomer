package cgroups

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

var (
	cgroupRootPath   = "/sys/fs/cgroup"
	process1CommPath = "/proc/1/comm"
	systemdRunDir    = "/run/systemd/system"
	lookPathFn       = exec.LookPath
	runCommandFn     = runCommand
)

type resourceBackend int

const (
	backendCgroup resourceBackend = iota
	backendSystemd
)

func groupPath(groupName string) string {
	return strings.TrimRight(cgroupRootPath, "/") + "/" + groupName
}

func rootGroupPath() string {
	return strings.TrimRight(cgroupRootPath, "/") + "/cgroup.procs"
}

// CreateProcessGroup creates new cgroup
func CreateProcessGroup(groupName string) error {
	if currentBackend() == backendSystemd {
		return createSystemdUnit(groupName)
	}

	err := os.Mkdir(groupPath(groupName), 0755)
	if err != nil && !os.IsExist(err) {
		return fmt.Errorf("failed to create cgroup directory: %v", err)
	}
	return nil
}

// DeleteProcessGroup deletes group of process
func DeleteProcessGroup(groupName string) error {
	if currentBackend() == backendSystemd {
		return deleteSystemdUnit(groupName)
	}

	err := os.Remove(groupPath(groupName))
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove cgroup directory: %v", err)
	}
	return nil
}

// AddProcessToGroup adds service on group
func AddProcessToGroup(pid int, groupName string) error {
	if err := ensureGroupExists(groupName); err != nil {
		return err
	}
	groupProcsPath, err := processGroupProcsPath(groupName)
	if err != nil {
		return err
	}

	err = os.WriteFile(groupProcsPath, []byte(strconv.Itoa(pid)), 0644)
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
	if currentBackend() == backendSystemd {
		return setSystemdProperty(groupName, row, val)
	}

	if err := ensureGroupExists(groupName); err != nil {
		return err
	}
	err := os.WriteFile(groupRowPath(groupName, row), []byte(val), 0644)
	if err != nil {
		return fmt.Errorf("cgroups, SetGroupRow: %v", err)
	}
	return nil
}

func checkExistGroup(groupName string) bool {
	path, err := processGroupDirPath(groupName)
	if err != nil {
		return false
	}

	_, err = os.Stat(path)
	return err == nil
}

func UseSystemd() bool {
	return currentBackend() == backendSystemd
}

// currentBackend decides whether resource management should talk to systemd or
// write to cgroup v2 directly, so higher-level code stays backend-agnostic.
func currentBackend() resourceBackend {
	if _, err := os.Stat(systemdRunDir); err != nil {
		return backendCgroup
	}

	initName, err := os.ReadFile(process1CommPath)
	if err != nil {
		return backendCgroup
	}
	if strings.TrimSpace(string(initName)) != "systemd" {
		return backendCgroup
	}

	if _, err := lookPathFn("systemctl"); err != nil {
		return backendCgroup
	}
	if _, err := lookPathFn("systemd-run"); err != nil {
		return backendCgroup
	}

	return backendSystemd
}

// processGroupDirPath resolves the actual directory for a resource group.
// With systemd this means translating a unit into its ControlGroup path first.
func processGroupDirPath(groupName string) (string, error) {
	if currentBackend() == backendCgroup {
		return groupPath(groupName), nil
	}

	controlGroup, err := systemdControlGroup(groupName)
	if err != nil {
		return "", err
	}

	return filepath.Join(strings.TrimRight(cgroupRootPath, "/"), strings.TrimPrefix(controlGroup, "/")), nil
}

func processGroupProcsPath(groupName string) (string, error) {
	groupDir, err := processGroupDirPath(groupName)
	if err != nil {
		return "", err
	}

	return filepath.Join(groupDir, "cgroup.procs"), nil
}

func groupRowPath(groupName, row string) string {
	return filepath.Join(groupPath(groupName), row)
}

func ensureGroupExists(groupName string) error {
	if checkExistGroup(groupName) {
		return nil
	}

	return CreateProcessGroup(groupName)
}

// createSystemdUnit keeps a long-lived transient unit around so processes can
// be attached to its cgroup and limited through systemd properties.
func createSystemdUnit(groupName string) error {
	if checkExistGroup(groupName) {
		return nil
	}

	sleepPath, err := lookPathFn("sleep")
	if err != nil {
		return fmt.Errorf("failed to find sleep binary: %v", err)
	}

	_, err = runCommandFn("systemd-run", systemdRunArgs(groupName, sleepPath)...)
	if err != nil {
		return fmt.Errorf("failed to create systemd unit: %v", err)
	}

	return nil
}

func deleteSystemdUnit(groupName string) error {
	if !checkExistGroup(groupName) {
		return nil
	}

	_, err := runCommandFn("systemctl", "stop", systemdUnitName(groupName))
	if err != nil {
		return fmt.Errorf("failed to stop systemd unit: %v", err)
	}

	return nil
}

func setSystemdProperty(groupName, row, val string) error {
	if !checkExistGroup(groupName) {
		if err := createSystemdUnit(groupName); err != nil {
			return err
		}
	}

	propertyName, propertyValue, err := systemdProperty(row, val)
	if err != nil {
		return err
	}

	_, err = runCommandFn("systemctl", "set-property", systemdUnitName(groupName), propertyName+"="+propertyValue)
	if err != nil {
		return fmt.Errorf("failed to set systemd property: %v", err)
	}

	return nil
}

func systemdControlGroup(groupName string) (string, error) {
	out, err := runCommandFn("systemctl", "show", "--property", "ControlGroup", "--value", systemdUnitName(groupName))
	if err != nil {
		return "", fmt.Errorf("failed to get systemd control group: %v", err)
	}
	if strings.TrimSpace(out) == "" {
		return "", fmt.Errorf("empty systemd control group for %s", groupName)
	}

	return strings.TrimSpace(out), nil
}

// systemdProperty maps raw cgroup-style rows to the closest systemd unit
// properties used by the same resource-limiting flow.
func systemdProperty(row, val string) (string, string, error) {
	switch row {
	case "cpu.max":
		parts := strings.Fields(val)
		if len(parts) != 2 {
			return "", "", fmt.Errorf("invalid cpu.max value: %s", val)
		}
		if parts[0] == "max" {
			return "CPUQuota", "infinity", nil
		}

		quota, err := strconv.ParseFloat(parts[0], 64)
		if err != nil {
			return "", "", fmt.Errorf("invalid cpu.max quota: %v", err)
		}
		period, err := strconv.ParseFloat(parts[1], 64)
		if err != nil {
			return "", "", fmt.Errorf("invalid cpu.max period: %v", err)
		}
		if period == 0 {
			return "", "", fmt.Errorf("invalid cpu.max period: must be greater than 0")
		}

		return "CPUQuota", fmt.Sprintf("%.0f%%", quota/period*100), nil
	case "memory.max":
		return "MemoryMax", val, nil
	default:
		return "", "", fmt.Errorf("unsupported systemd property mapping for %s", row)
	}
}

func systemdUnitName(groupName string) string {
	return groupName + ".service"
}

func systemdRunArgs(groupName, sleepPath string) []string {
	return []string{
		"--quiet",
		"--unit", systemdUnitName(groupName),
		"--service-type=exec",
		"--property=CPUAccounting=yes",
		"--property=MemoryAccounting=yes",
		sleepPath,
		"infinity",
	}
}

func runCommand(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			return "", err
		}
		return "", fmt.Errorf("%w: %s", err, msg)
	}

	return strings.TrimSpace(string(out)), nil
}
