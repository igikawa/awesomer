package daemon

import (
	"awesomeProject/internal/daemon/config"
	"awesomeProject/internal/daemon/info"
	"awesomeProject/internal/process/parser"
	"awesomeProject/pkg/cgroups"
	"awesomeProject/pkg/logger"

	"fmt"
	"sync"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

const CgroupName = "processJail"

func Run(cfg config.Config) error {
	jail := make(map[int]int)
	mu := &sync.RWMutex{}

	go realTimeReadConfig(&cfg, mu)

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
		mu.RLock()
		cpuLim := cfg.CPULimit
		memLim := cfg.RAMLimit
		tick := cfg.Tick
		isRunning := cfg.Run
		mu.RUnlock()

		if !isRunning {
			logger.DaemonLogger.Println("Daemon is stopped")
			break
		}

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
			if p.CPUPercent > cpuLim || p.MemPercent > memLim {
				jail[int(p.PID)]++
				if jail[int(p.PID)] >= 3 {
					tree, _, _ := parser.Object.ProcessTree(p.PID)
					for _, memberPid := range tree {
						err := cgroups.AddProcessToGroup(int(memberPid), CgroupName)
						info.SetJail(int(memberPid))
						logger.DaemonLogger.Printf("Added process to Jail: %d, err: %s\n", int(memberPid), err)
					}
				}
			}
		}
		for pid := range jail {
			if !activePIDs[pid] {
				info.DeleteFromJail(pid)
				delete(jail, pid)
			}
		}

		time.Sleep(time.Duration(tick) * time.Second)
	}
	return nil
}

func realTimeReadConfig(conf *config.Config, mu *sync.RWMutex) {
	var readConf = func() config.Config {
		var cfg config.Config
		cleanenv.ReadConfig(".env", &cfg)
		return cfg
	}
	for {
		mu.Lock()
		*conf = readConf()
		mu.Unlock()
		time.Sleep(time.Second)
	}
}
