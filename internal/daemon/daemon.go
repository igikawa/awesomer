package daemon

import (
	"awesomeProject/internal/daemon/config"
	"awesomeProject/internal/daemon/info"
	"awesomeProject/internal/service/parser"
	"awesomeProject/pkg/cgroups"
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

const CgroupName = "processJail"

type Daemon struct {
	cfg   *config.Config
	l     *log.Logger
	parse *parser.Parser
	mu    *sync.Mutex
	api   *info.API
}

func New(cfg *config.Config, l *log.Logger, api *info.API) *Daemon {
	return &Daemon{
		cfg:   cfg,
		l:     l,
		parse: parser.NewParser(),
		mu:    &sync.Mutex{},
		api:   api,
	}
}

func (d *Daemon) Run(ctx context.Context) error {
	jail := make(map[int]int)
	mu := &sync.RWMutex{}

	go d.realTimeReadConfig(ctx)

	err := cgroups.CreateProcessGroup(CgroupName)
	if err != nil {
		d.l.Println("Failed to create service group:", err)
		return err
	}

	err = cgroups.SetGroupRow(CgroupName, "cpu.max", fmt.Sprintf("%d 100000", int(d.cfg.CPUQuota)*1000))
	if err != nil {
		d.l.Println("Failed to set CPU quota:", err)
		return err
	}
	err = cgroups.SetGroupRow(CgroupName, "memory.max", fmt.Sprintf("%s", d.cfg.RAMQuota))
	if err != nil {
		d.l.Println("Failed to set memory quota:", err)
		return err
	}

	fmt.Println("Daemon is running")

	for {
		select {
		case <-ctx.Done():
			err := cgroups.DeleteProcessGroup(CgroupName)
			if err != nil {
				d.l.Println("Failed to delete service group:", err)
			}
			d.l.Println("Daemon is stopped")
			return nil
		default:
			mu.RLock()
			cpuLim := d.cfg.CPULimit
			memLim := d.cfg.RAMLimit
			tick := d.cfg.Tick
			isRunning := d.cfg.Run
			mu.RUnlock()

			if !isRunning {
				d.l.Println("Daemon is stopped")
				break
			}

			procs, err := d.parse.AllProcesses()

			if err != nil {
				d.l.Println(err.Error())
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
						tree, _, _ := d.parse.ProcessTree(p.PID)
						for _, memberPid := range tree {
							err := cgroups.AddProcessToGroup(int(memberPid), CgroupName)
							if err != nil {
								d.l.Printf("error added process %d in jail: %s", memberPid, err.Error())
								continue
							}
							d.api.SetJail(int(memberPid))
							d.l.Printf("added process %d in jail", memberPid)
						}
					}
				}
			}
			for pid := range jail {
				if !activePIDs[pid] {
					d.api.DeleteFromJail(pid)
					delete(jail, pid)
				}
			}

			time.Sleep(time.Duration(tick) * time.Second)
		}
	}
}

func (d *Daemon) realTimeReadConfig(ctx context.Context) {
	var readConf = func() (config.Config, error) {
		var cfg config.Config
		err := cleanenv.ReadConfig(".env", &cfg)
		if err != nil {
			return cfg, err
		}
		return cfg, nil
	}
	for {
		select {
		case <-ctx.Done():
			return
		default:
			d.mu.Lock()
			c, err := readConf()
			if err != nil {
				d.l.Println("Failed to read config:", err)
				d.mu.Unlock()
				time.Sleep(time.Second)
				continue
			}
			*d.cfg = c
			d.mu.Unlock()

			time.Sleep(time.Second)
		}
	}
}
