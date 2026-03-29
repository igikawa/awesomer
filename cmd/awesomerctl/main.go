package main

import (
	"awesomeProject/internal/collector"
	"awesomeProject/internal/config"
	"awesomeProject/internal/daemon/info"
	"awesomeProject/internal/daemon/ipc"
	"awesomeProject/internal/service/tui"
	"awesomeProject/pkg/logger"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

var (
	openConfigFileFn    = func(name string) (*os.File, error) { return os.Open(name) }
	createConfigFileFn  = func(name string) (*os.File, error) { return os.Create(name) }
	mkdirAllFn          = os.MkdirAll
	resolveConfigPathFn = config.ResolveConfigPath
	loadConfigFn        = config.NewConfig
	daemonSocketPathFn  = func() string { return ipc.SocketPath }
	newRemoteStateFn    = func(path string) info.JailState { return ipc.NewClient(path) }
	ipcAvailableFn      = ipc.IsAvailable
	newInfoAPIFn        = info.NewAPI
	newCollectorFn      = func() collector.Provider { return collector.New() }
	newLoggerFn         = logger.NewLogger
	runTUIFn            = tui.Run
	logFatalFn          = log.Fatal
)

func main() {
	if err := runApp(os.Args[1:]); err != nil {
		logFatalFn(err)
	}
}

func runApp(args []string) error {
	if err := validateArgs(args); err != nil {
		return err
	}

	ensureConfigFileExists()

	cfg := loadConfigFn()
	state := selectJailState()
	snapshots := newCollectorFn()

	l, err := newLoggerFn(&cfg.LoggerConfig)
	if err != nil {
		return err
	}

	return runTUIFn(func() {}, cfg, l, state, snapshots, tui.NewTable(cfg.UI), tui.NewInfo())
}

func validateArgs(args []string) error {
	for _, arg := range args {
		if arg == "--daemon-only" {
			return errors.New("use awesomerd for daemon mode")
		}
	}
	return nil
}

func selectJailState() info.JailState {
	if ipcAvailableFn(daemonSocketPathFn()) {
		return newRemoteStateFn(daemonSocketPathFn())
	}

	return newInfoAPIFn()
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
