package daemon

import (
	"awesomeProject/internal/daemon/config"
	"awesomeProject/internal/process/parser"
	"awesomeProject/pkg/cgroups"
	"awesomeProject/pkg/logger"
	"fmt"

	"time"
)

const CgroupName = "processJail"

func Run(cfg config.Config) error {
	jail := make(map[int]int)

	err := cgroups.CreateProcessGroup(CgroupName)
	if err != nil {
		logger.DaemonLogger.Println("Failed to create process group:", err)
		return err
	}

	err = cgroups.SetGroupRow(CgroupName, "cpu.max", fmt.Sprintf("%d 100000", int(cfg.CPUQuota)*1000))
	if err != nil {
		logger.DaemonLogger.Println("Failed to set CPU quota:", err)
		return err
	}
	err = cgroups.SetGroupRow(CgroupName, "memory.max", fmt.Sprintf("%d", cfg.RAMQuota))
	if err != nil {
		logger.DaemonLogger.Println("Failed to set memory quota:", err)
		return err
	}

	fmt.Println("Daemon is running")

	for {
		procs, err := parser.Object.AllProcessess()
		if err != nil {
			logger.DaemonLogger.Println(err.Error())
		}
		activePIDs := make(map[int]bool)
		for _, p := range procs {
			activePIDs[int(p.PID)] = true
			if p.PID < 100 || p.Name == "systemd" || p.Name == "sshd" {
				continue
			}
			if p.CPUPercent > cfg.CPULimit || p.MemPercent > cfg.RAMLimit {
				jail[int(p.PID)]++
				if jail[int(p.PID)] >= 3 {
					tree, _, _ := parser.Object.ProcessTree(p.PID)
					for _, memberPid := range tree {
						cgroups.AddProcessToGroup(int(memberPid), CgroupName)
					}
				}
			}
		}
		for pid := range jail {
			if !activePIDs[pid] {
				delete(jail, pid)
			}
		}

		time.Sleep(time.Duration(cfg.Tick) * time.Second)
	}
}
