package main

import (
	"awesomeProject/internal/collector"
	"awesomeProject/internal/config"
	daemonInfo "awesomeProject/internal/daemon/info"
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

type remoteStateStub struct{}

func (remoteStateStub) InJail(pid int) bool                     { return pid == 42 }
func (remoteStateStub) SetJail(pid int)                         {}
func (remoteStateStub) DeleteFromJail(pid int)                  {}
func (remoteStateStub) PIDs() []int                             { return []int{42} }
func (remoteStateStub) ToggleProcessJail(pid int) (bool, error) { return true, nil }

func TestRunAppStartsTUIAndCreatesConfig(t *testing.T) {
	dir := t.TempDir()
	origOpen := openConfigFileFn
	origCreate := createConfigFileFn
	origMkdirAll := mkdirAllFn
	origResolve := resolveConfigPathFn
	origLoad := loadConfigFn
	origSocketPath := daemonSocketPathFn
	origRemoteState := newRemoteStateFn
	origIPCAvailable := ipcAvailableFn
	origInfo := newInfoAPIFn
	origCollector := newCollectorFn
	origLogger := newLoggerFn
	origRunTUI := runTUIFn
	defer func() {
		openConfigFileFn = origOpen
		createConfigFileFn = origCreate
		mkdirAllFn = origMkdirAll
		resolveConfigPathFn = origResolve
		loadConfigFn = origLoad
		daemonSocketPathFn = origSocketPath
		newRemoteStateFn = origRemoteState
		ipcAvailableFn = origIPCAvailable
		newInfoAPIFn = origInfo
		newCollectorFn = origCollector
		newLoggerFn = origLogger
		runTUIFn = origRunTUI
	}()

	configPath := filepath.Join(dir, ".config", "awesomer", "config.yaml")
	resolveConfigPathFn = func() (string, error) { return configPath, nil }
	daemonSocketPathFn = func() string { return filepath.Join(dir, "awesomer.sock") }
	ipcAvailableFn = func(path string) bool { return false }
	openConfigFileFn = func(name string) (*os.File, error) { return nil, os.ErrNotExist }

	mkdirPath := ""
	mkdirAllFn = func(path string, perm os.FileMode) error {
		mkdirPath = path
		return os.MkdirAll(path, perm)
	}

	createPath := ""
	createConfigFileFn = func(name string) (*os.File, error) {
		createPath = name
		return os.Create(name)
	}

	loadConfigFn = func() *config.Config {
		return &config.Config{
			LoggerConfig: logger.Config{LogPath: filepath.Join(dir, "app.log")},
		}
	}
	newInfoAPIFn = func() *daemonInfo.API { return daemonInfo.NewAPI() }
	newCollectorFn = func() collector.Provider { return nil }
	newLoggerFn = func(cfg *logger.Config) (*log.Logger, error) { return log.New(&bytes.Buffer{}, "", 0), nil }

	tuiCalls := 0
	runTUIFn = func(cancel context.CancelFunc, cfg *config.Config, l *log.Logger, api daemonInfo.JailState, snapshots collector.Provider, tbl table.Model, v viewport.Model) error {
		tuiCalls++
		return nil
	}

	if err := runApp(nil); err != nil {
		t.Fatalf("runApp() error = %v", err)
	}
	if mkdirPath != filepath.Dir(configPath) {
		t.Fatalf("mkdir path = %q, want %q", mkdirPath, filepath.Dir(configPath))
	}
	if createPath != configPath {
		t.Fatalf("create path = %q, want %q", createPath, configPath)
	}
	if tuiCalls != 1 {
		t.Fatalf("tui calls = %d, want 1", tuiCalls)
	}
}

func TestRunAppUsesRemoteDaemonStateWhenSocketIsAvailable(t *testing.T) {
	dir := t.TempDir()
	origOpen := openConfigFileFn
	origResolve := resolveConfigPathFn
	origLoad := loadConfigFn
	origSocketPath := daemonSocketPathFn
	origRemoteState := newRemoteStateFn
	origIPCAvailable := ipcAvailableFn
	origCollector := newCollectorFn
	origLogger := newLoggerFn
	origRunTUI := runTUIFn
	defer func() {
		openConfigFileFn = origOpen
		resolveConfigPathFn = origResolve
		loadConfigFn = origLoad
		daemonSocketPathFn = origSocketPath
		newRemoteStateFn = origRemoteState
		ipcAvailableFn = origIPCAvailable
		newCollectorFn = origCollector
		newLoggerFn = origLogger
		runTUIFn = origRunTUI
	}()

	file, err := os.CreateTemp(dir, "config-*.yaml")
	if err != nil {
		t.Fatalf("CreateTemp() error = %v", err)
	}
	resolveConfigPathFn = func() (string, error) { return file.Name(), nil }
	openConfigFileFn = func(name string) (*os.File, error) { return file, nil }
	loadConfigFn = func() *config.Config {
		return &config.Config{LoggerConfig: logger.Config{LogPath: file.Name()}}
	}
	daemonSocketPathFn = func() string { return filepath.Join(dir, "awesomer.sock") }
	ipcAvailableFn = func(path string) bool { return true }
	newRemoteStateFn = func(path string) daemonInfo.JailState { return remoteStateStub{} }
	newCollectorFn = func() collector.Provider { return nil }
	newLoggerFn = func(cfg *logger.Config) (*log.Logger, error) { return log.New(&bytes.Buffer{}, "", 0), nil }

	runTUIFn = func(cancel context.CancelFunc, cfg *config.Config, l *log.Logger, api daemonInfo.JailState, snapshots collector.Provider, tbl table.Model, v viewport.Model) error {
		if !api.InJail(42) {
			t.Fatal("expected remote daemon state to be passed into TUI")
		}
		return nil
	}

	if err := runApp(nil); err != nil {
		t.Fatalf("runApp() error = %v", err)
	}
}

func TestRunAppPropagatesTUIError(t *testing.T) {
	origOpen := openConfigFileFn
	origResolve := resolveConfigPathFn
	origLoad := loadConfigFn
	origSocketPath := daemonSocketPathFn
	origIPCAvailable := ipcAvailableFn
	origCollector := newCollectorFn
	origLogger := newLoggerFn
	origRunTUI := runTUIFn
	defer func() {
		openConfigFileFn = origOpen
		resolveConfigPathFn = origResolve
		loadConfigFn = origLoad
		daemonSocketPathFn = origSocketPath
		ipcAvailableFn = origIPCAvailable
		newCollectorFn = origCollector
		newLoggerFn = origLogger
		runTUIFn = origRunTUI
	}()

	file, err := os.CreateTemp(t.TempDir(), "config-*.yaml")
	if err != nil {
		t.Fatalf("CreateTemp() error = %v", err)
	}
	resolveConfigPathFn = func() (string, error) { return file.Name(), nil }
	daemonSocketPathFn = func() string { return filepath.Join(t.TempDir(), "awesomer.sock") }
	ipcAvailableFn = func(path string) bool { return false }
	openConfigFileFn = func(name string) (*os.File, error) { return file, nil }
	loadConfigFn = func() *config.Config {
		return &config.Config{LoggerConfig: logger.Config{LogPath: file.Name()}}
	}
	newCollectorFn = func() collector.Provider { return nil }
	newLoggerFn = func(cfg *logger.Config) (*log.Logger, error) { return log.New(&bytes.Buffer{}, "", 0), nil }
	runTUIFn = func(cancel context.CancelFunc, cfg *config.Config, l *log.Logger, api daemonInfo.JailState, snapshots collector.Provider, tbl table.Model, v viewport.Model) error {
		return errors.New("tui failed")
	}

	if err := runApp(nil); err == nil {
		t.Fatal("runApp() error = nil, want non-nil")
	}
}

func TestValidateArgsRejectsDaemonOnlyFlag(t *testing.T) {
	if err := validateArgs([]string{"--daemon-only"}); err == nil {
		t.Fatal("validateArgs() error = nil, want non-nil")
	}
}
