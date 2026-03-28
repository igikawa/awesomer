package main

import (
	"awesomeProject/internal/collector"
	"awesomeProject/internal/config"
	"awesomeProject/internal/daemon"
	daemonConfig "awesomeProject/internal/daemon/config"
	"awesomeProject/internal/daemon/info"
	"awesomeProject/internal/service/tui"
	"awesomeProject/pkg/logger"
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	//"os/signal"
	//"syscall"
)

var (
	openConfigFileFn   = func(name string) (*os.File, error) { return os.Open(name) }
	createConfigFileFn = func(name string) (*os.File, error) { return os.Create(name) }
	loadConfigFn       = config.NewConfig
	newInfoAPIFn       = info.NewAPI
	newDaemonLoggerFn  = logger.NewDaemonLogger
	newCollectorFn     = func() collector.Provider { return collector.New() }
	newDaemonFn        = func(cfg *daemonConfig.Config, l *log.Logger, api *info.API, snapshots collector.Provider) daemonRunner {
		return daemon.New(cfg, l, api, snapshots)
	}
	newLoggerFn = logger.NewLogger
	runTUIFn    = tui.Run
	logFatalFn  = log.Fatal
)

type daemonRunner interface {
	Run(context.Context) error
}

func main() {
	if err := runApp(); err != nil {
		logFatalFn(err)
	}
}

func runApp() error {
	ensureConfigFileExists(config.FileName)

	cfg := loadConfigFn()
	apiInstance := newInfoAPIFn()
	snapshots := newCollectorFn()

	dl, err := newDaemonLoggerFn(&cfg.LoggerConfig)
	if err != nil {
		return err
	}
	d := newDaemonFn(&cfg.Daemon, dl, apiInstance, snapshots)

	ctx, cancel := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}
	startDaemonIfEnabled(ctx, cfg, d, wg)

	l, err := newLoggerFn(&cfg.LoggerConfig)
	if err != nil {
		return err
	}

	if err = runTUIFn(cancel, cfg, l, apiInstance, snapshots, tui.NewTable(cfg.UI), tui.NewInfo()); err != nil {
		return err
	}

	wg.Wait()
	return nil
}

func ensureConfigFileExists(path string) {
	file, err := openConfigFileFn(path)
	if err != nil {
		file, err = createConfigFileFn(path)
		if err != nil {
			fmt.Println(err)
			return
		}
	}
	if file != nil {
		_ = file.Close()
	}
}

func startDaemonIfEnabled(ctx context.Context, cfg *config.Config, d daemonRunner, wg *sync.WaitGroup) {
	if !cfg.Daemon.Run {
		return
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := d.Run(ctx); err != nil {
			logFatalFn(err)
		}
	}()
}
