package main

import (
	"awesomeProject/internal/collector"
	"awesomeProject/internal/config"
	daemonConfig "awesomeProject/internal/daemon/config"
	"awesomeProject/internal/daemon/info"
	"awesomeProject/pkg/logger"
	"bytes"
	"context"
	"errors"
	"log"
	"os"
	"path/filepath"
	"testing"

	"charm.land/bubbles/v2/table"
	"charm.land/bubbles/v2/viewport"
)

type stubDaemon struct {
	runCalls int
	runErr   error
}

func (d *stubDaemon) Run(ctx context.Context) error {
	d.runCalls++
	return d.runErr
}

func TestRunAppStartsDaemonAndTUI(t *testing.T) {
	dir := t.TempDir()
	prevWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	defer func() { _ = os.Chdir(prevWD) }()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}

	origOpen := openConfigFileFn
	origCreate := createConfigFileFn
	origLoad := loadConfigFn
	origInfo := newInfoAPIFn
	origCollector := newCollectorFn
	origDL := newDaemonLoggerFn
	origDaemon := newDaemonFn
	origLogger := newLoggerFn
	origRunTUI := runTUIFn
	origFatal := logFatalFn
	defer func() {
		openConfigFileFn = origOpen
		createConfigFileFn = origCreate
		loadConfigFn = origLoad
		newInfoAPIFn = origInfo
		newCollectorFn = origCollector
		newDaemonLoggerFn = origDL
		newDaemonFn = origDaemon
		newLoggerFn = origLogger
		runTUIFn = origRunTUI
		logFatalFn = origFatal
	}()

	openConfigFileFn = func(name string) (*os.File, error) { return nil, os.ErrNotExist }
	createPath := ""
	createConfigFileFn = func(name string) (*os.File, error) {
		createPath = name
		return os.Create(filepath.Join(dir, name))
	}
	loadConfigFn = func() *config.Config {
		return &config.Config{
			Tick: 1,
			LoggerConfig: logger.Config{
				LogPath:       filepath.Join(dir, "app.log"),
				DaemonLogPath: filepath.Join(dir, "daemon.log"),
			},
			Daemon: daemonConfig.Config{Run: true},
		}
	}
	newInfoAPIFn = func() *info.API { return info.NewAPI() }
	newCollectorFn = func() collector.Provider { return nil }
	newDaemonLoggerFn = func(cfg *logger.Config) (*log.Logger, error) {
		return log.New(&bytes.Buffer{}, "", 0), nil
	}
	newLoggerFn = func(cfg *logger.Config) (*log.Logger, error) {
		return log.New(&bytes.Buffer{}, "", 0), nil
	}

	daemonStub := &stubDaemon{}
	newDaemonFn = func(cfg *daemonConfig.Config, l *log.Logger, api *info.API, snapshots collector.Provider) daemonRunner {
		return daemonStub
	}

	tuiCalls := 0
	runTUIFn = func(cancel context.CancelFunc, cfg *config.Config, l *log.Logger, api *info.API, snapshots collector.Provider, t table.Model, v viewport.Model) error {
		tuiCalls++
		cancel()
		return nil
	}
	logFatalFn = func(v ...any) {}

	if err := runApp(); err != nil {
		t.Fatalf("runApp() error = %v", err)
	}
	if createPath != config.FileName {
		t.Fatalf("create path = %q, want %q", createPath, config.FileName)
	}
	if daemonStub.runCalls != 1 {
		t.Fatalf("daemon run calls = %d, want 1", daemonStub.runCalls)
	}
	if tuiCalls != 1 {
		t.Fatalf("tui calls = %d, want 1", tuiCalls)
	}
}

func TestRunAppPropagatesTUIError(t *testing.T) {
	origOpen := openConfigFileFn
	origLoad := loadConfigFn
	origInfo := newInfoAPIFn
	origCollector := newCollectorFn
	origDL := newDaemonLoggerFn
	origDaemon := newDaemonFn
	origLogger := newLoggerFn
	origRunTUI := runTUIFn
	defer func() {
		openConfigFileFn = origOpen
		loadConfigFn = origLoad
		newInfoAPIFn = origInfo
		newCollectorFn = origCollector
		newDaemonLoggerFn = origDL
		newDaemonFn = origDaemon
		newLoggerFn = origLogger
		runTUIFn = origRunTUI
	}()

	file, err := os.CreateTemp(t.TempDir(), "config-*.yaml")
	if err != nil {
		t.Fatalf("CreateTemp() error = %v", err)
	}
	openConfigFileFn = func(name string) (*os.File, error) { return file, nil }
	loadConfigFn = func() *config.Config {
		return &config.Config{LoggerConfig: logger.Config{LogPath: file.Name(), DaemonLogPath: file.Name()}}
	}
	newInfoAPIFn = func() *info.API { return info.NewAPI() }
	newCollectorFn = func() collector.Provider { return nil }
	newDaemonLoggerFn = func(cfg *logger.Config) (*log.Logger, error) { return log.New(&bytes.Buffer{}, "", 0), nil }
	newLoggerFn = func(cfg *logger.Config) (*log.Logger, error) { return log.New(&bytes.Buffer{}, "", 0), nil }
	newDaemonFn = func(cfg *daemonConfig.Config, l *log.Logger, api *info.API, snapshots collector.Provider) daemonRunner {
		return &stubDaemon{}
	}
	runTUIFn = func(cancel context.CancelFunc, cfg *config.Config, l *log.Logger, api *info.API, snapshots collector.Provider, t table.Model, v viewport.Model) error {
		return errors.New("tui failed")
	}

	if err := runApp(); err == nil {
		t.Fatal("runApp() error = nil, want non-nil")
	}
}
