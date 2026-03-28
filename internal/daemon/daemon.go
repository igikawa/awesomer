package daemon

import (
	"awesomeProject/internal/collector"
	rootConfig "awesomeProject/internal/config"
	"awesomeProject/internal/daemon/config"
	"awesomeProject/internal/daemon/info"
	"awesomeProject/pkg/cgroups"
	"awesomeProject/pkg/parser"
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
)

const CgroupName = "processJail"
const (
	violationThreshold = 3
	ignoredPIDLimit    = 100
)

var (
	createProcessGroupFn = cgroups.CreateProcessGroup
	setGroupRowFn        = cgroups.SetGroupRow
	deleteProcessGroupFn = cgroups.DeleteProcessGroup
	addProcessToGroupFn  = cgroups.AddProcessToGroup
	moveToRootGroupFn    = cgroups.MoveProcessToRootGroup
	readDaemonConfigFn   = func() (config.Config, error) {
		cfg, err := rootConfig.ReadConfig(rootConfig.FileName)
		if err != nil {
			return config.Config{}, err
		}
		return cfg.Daemon, nil
	}
	sleepFn = time.Sleep
)

type Daemon struct {
	cfg       *config.Config
	l         *log.Logger
	snapshots collector.Provider
	mu        *sync.Mutex
	api       *info.API
}

func New(cfg *config.Config, l *log.Logger, api *info.API, snapshots collector.Provider) *Daemon {
	if snapshots == nil {
		snapshots = collector.New()
	}

	return &Daemon{
		cfg:       cfg,
		l:         l,
		snapshots: snapshots,
		mu:        &sync.Mutex{},
		api:       api,
	}
}

// Run keeps per-process violation counters and only jails a process tree after
// the same PID breaches limits several times in a row.
func (d *Daemon) Run(ctx context.Context) error {
	violations := make(map[int]int)

	watchCtx, stopWatch := context.WithCancel(ctx)
	defer stopWatch()

	go d.realTimeReadConfig(watchCtx)

	if err := d.configureGroup(); err != nil {
		return err
	}

	fmt.Println("Daemon is running")

	for {
		select {
		case <-ctx.Done():
			return d.stop()
		default:
			cfg := d.snapshotConfig()
			if !cfg.Run {
				return d.stop()
			}

			procs, err := d.snapshots.Processes()
			if err != nil {
				d.l.Println(err.Error())
			}

			activePIDs := d.applyLimits(procs, cfg, violations)
			d.cleanupInactive(violations, activePIDs)

			sleepFn(time.Duration(cfg.Tick) * time.Second)
		}
	}
}

// realTimeReadConfig continuously refreshes daemon settings until the context
// is cancelled or the updated config explicitly disables daemon mode.
func (d *Daemon) realTimeReadConfig(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			d.mu.Lock()
			c, err := readDaemonConfigFn()
			if err != nil {
				d.l.Println("Failed to read config:", err)
				d.mu.Unlock()
				sleepFn(time.Second)
				continue
			}
			*d.cfg = c
			d.mu.Unlock()

			if !c.Run {
				return
			}

			sleepFn(time.Second)
		}
	}
}

// stop releases jailed processes before removing processJail so the backing
// cgroup or transient unit can disappear cleanly.
func (d *Daemon) stop() error {
	var stopErr error

	for _, pid := range d.api.PIDs() {
		if err := moveToRootGroupFn(pid); err != nil {
			d.l.Printf("Failed to move process %d to root group: %v", pid, err)
			if stopErr == nil {
				stopErr = err
			}
			continue
		}
		d.api.DeleteFromJail(pid)
	}

	if err := deleteProcessGroupFn(CgroupName); err != nil {
		d.l.Println("Failed to delete service group:", err)
		if stopErr == nil {
			stopErr = err
		}
	}

	d.l.Println("Daemon is stopped")
	return stopErr
}

func (d *Daemon) configureGroup() error {
	if err := createProcessGroupFn(CgroupName); err != nil {
		d.l.Println("Failed to create service group:", err)
		return err
	}

	cfg := d.snapshotConfig()
	if err := setGroupRowFn(CgroupName, "cpu.max", fmt.Sprintf("%d 100000", int(cfg.CPUQuota)*1000)); err != nil {
		d.l.Println("Failed to set CPU quota:", err)
		return err
	}
	if err := setGroupRowFn(CgroupName, "memory.max", cfg.RAMQuota); err != nil {
		d.l.Println("Failed to set memory quota:", err)
		return err
	}

	return nil
}

func (d *Daemon) snapshotConfig() config.Config {
	d.mu.Lock()
	defer d.mu.Unlock()
	return *d.cfg
}

// applyLimits keeps the main loop simple by separating scan, threshold check,
// and the actual tree migration into processJail.
func (d *Daemon) applyLimits(procs []parser.Info, cfg config.Config, violations map[int]int) map[int]bool {
	activePIDs := make(map[int]bool, len(procs))
	whitelist := newWhitelistSet(cfg.Whitelist)

	for _, p := range procs {
		pid := int(p.PID)
		activePIDs[pid] = true
		if shouldSkipProcess(p, whitelist) {
			continue
		}
		if d.api.InJail(pid) {
			continue
		}
		if p.CPUPercent <= cfg.CPULimit && p.MemPercent <= cfg.RAMLimit {
			continue
		}

		violations[pid]++
		if violations[pid] < violationThreshold {
			continue
		}

		d.putTreeInGroup(p.PID)
	}

	return activePIDs
}

func (d *Daemon) putTreeInGroup(rootPID int32) {
	tree, _, err := d.snapshots.ProcessTree(rootPID)
	if err != nil {
		d.l.Printf("error reading process tree for %d: %s", rootPID, err.Error())
		return
	}

	for _, memberPID := range tree {
		if err := addProcessToGroupFn(int(memberPID), CgroupName); err != nil {
			d.l.Printf("error added process %d in jail: %s", memberPID, err.Error())
			continue
		}
		d.api.SetJail(int(memberPID))
		d.l.Printf("added process %d in jail", memberPID)
	}
}

func (d *Daemon) cleanupInactive(violations map[int]int, activePIDs map[int]bool) {
	for pid := range violations {
		if activePIDs[pid] {
			continue
		}
		d.api.DeleteFromJail(pid)
		delete(violations, pid)
	}
}

func newWhitelistSet(items []string) map[string]struct{} {
	whitelist := make(map[string]struct{}, len(items))

	for _, item := range items {
		name := normalizeProcessName(item)
		if name == "" {
			continue
		}
		whitelist[name] = struct{}{}
	}

	return whitelist
}

func shouldSkipProcess(p parser.Info, whitelist map[string]struct{}) bool {
	if p.PID < ignoredPIDLimit {
		return true
	}

	_, found := whitelist[normalizeProcessName(p.Name)]
	return found
}

func normalizeProcessName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}
