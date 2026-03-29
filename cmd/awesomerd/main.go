package main

import (
	"awesomeProject/internal/collector"
	"awesomeProject/internal/config"
	"awesomeProject/internal/daemon"
	"awesomeProject/internal/daemon/info"
	"awesomeProject/internal/daemon/ipc"
	"awesomeProject/pkg/logger"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
)

var (
	openConfigFileFn    = func(name string) (*os.File, error) { return os.Open(name) }
	createConfigFileFn  = func(name string) (*os.File, error) { return os.Create(name) }
	mkdirAllFn          = os.MkdirAll
	resolveConfigPathFn = config.ResolveConfigPath
	loadConfigFn        = config.NewConfig
	currentEUIDFn       = os.Geteuid
	daemonSocketPathFn  = func() string { return ipc.SocketPath }
	startIPCServerFn    = ipc.Start
	newInfoAPIFn        = info.NewAPI
	newCollectorFn      = func() collector.Provider { return collector.New() }
	newDaemonLoggerFn   = logger.NewDaemonLogger
	newDaemonFn         = daemon.New
	logFatalFn          = log.Fatal
)

type daemonControl interface {
	Run(context.Context) error
	ToggleProcessJail(pid int) (bool, error)
}

func main() {
	if err := runApp(); err != nil {
		logFatalFn(err)
	}
}

func runApp() error {
	if currentEUIDFn() != 0 {
		return errors.New("awesomerd must run as root")
	}

	ensureConfigFileExists()

	cfg := loadConfigFn()
	if !cfg.Daemon.Run {
		return errors.New("daemon.run must be true for awesomerd")
	}

	api := newInfoAPIFn()
	snapshots := newCollectorFn()

	dl, err := newDaemonLoggerFn(&cfg.LoggerConfig)
	if err != nil {
		return err
	}

	d := newDaemonFn(&cfg.Daemon, dl, api, snapshots)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	server, err := startIPCServerFn(ctx, daemonSocketPathFn(), api, d)
	if err != nil {
		return err
	}
	defer func() { _ = server.Close() }()

	return d.Run(ctx)
}

func ensureConfigFileExists() {
	path, err := resolveConfigPathFn()
	if err != nil {
		fmt.Println(err)
		return
	}

	file, err := openConfigFileFn(path)
	if err != nil {
		if err := mkdirAllFn(filepath.Dir(path), 0755); err != nil {
			fmt.Println(err)
			return
		}
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
