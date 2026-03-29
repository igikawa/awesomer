package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewConfigReturnsZeroValuesWhenConfigFileIsMissing(t *testing.T) {
	origUserConfigDir := userConfigDirFn
	origEUID := currentEUIDFn
	defer func() {
		userConfigDirFn = origUserConfigDir
		currentEUIDFn = origEUID
	}()

	userConfigDirFn = func() (string, error) {
		return filepath.Join(t.TempDir(), ".config"), nil
	}
	currentEUIDFn = func() int { return 1000 }

	cfg := NewConfig()

	if cfg.Tick != 0 {
		t.Fatalf("Tick = %d, want 0", cfg.Tick)
	}
	if cfg.LoggerConfig.LogPath != "" {
		t.Fatalf("LogPath = %q, want empty", cfg.LoggerConfig.LogPath)
	}
	if cfg.Daemon.Tick != 0 {
		t.Fatalf("Daemon.Tick = %d, want 0", cfg.Daemon.Tick)
	}
	if cfg.Daemon.Run {
		t.Fatal("Daemon.Run = true, want false by default")
	}
	if cfg.UI.InfoWidth != 0 {
		t.Fatalf("UI.InfoWidth = %d, want 0", cfg.UI.InfoWidth)
	}
}

func TestNewConfigReadsYAMLFile(t *testing.T) {
	dir := t.TempDir()
	origUserConfigDir := userConfigDirFn
	origEUID := currentEUIDFn
	defer func() {
		userConfigDirFn = origUserConfigDir
		currentEUIDFn = origEUID
	}()

	userConfigDirFn = func() (string, error) {
		return dir, nil
	}
	currentEUIDFn = func() int { return 1000 }

	configYAML := []byte("tick: 9\nlogger:\n  log_path: /tmp/app.log\ndaemon:\n  run: true\n  tick: 7\n  ram_quota: 2G\n  whitelist:\n    - systemd\n    - dockerd\nui:\n  table_width: 48\n  border_color: \"240\"\n")
	if err := os.MkdirAll(filepath.Join(dir, AppName), 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, AppName, FileName), configYAML, 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	cfg := NewConfig()

	if cfg.Tick != 9 {
		t.Fatalf("Tick = %d, want 9", cfg.Tick)
	}
	if cfg.LoggerConfig.LogPath != "/tmp/app.log" {
		t.Fatalf("LogPath = %q, want /tmp/app.log", cfg.LoggerConfig.LogPath)
	}
	if !cfg.Daemon.Run {
		t.Fatal("Daemon.Run = false, want true")
	}
	if cfg.Daemon.Tick != 7 {
		t.Fatalf("Daemon.Tick = %d, want 7", cfg.Daemon.Tick)
	}
	if cfg.Daemon.RAMQuota != "2G" {
		t.Fatalf("Daemon.RAMQuota = %q, want 2G", cfg.Daemon.RAMQuota)
	}
	if len(cfg.Daemon.Whitelist) != 2 || cfg.Daemon.Whitelist[1] != "dockerd" {
		t.Fatalf("Daemon.Whitelist = %v, want [systemd dockerd]", cfg.Daemon.Whitelist)
	}
	if cfg.UI.TableWidth != 48 {
		t.Fatalf("UI.TableWidth = %d, want 48", cfg.UI.TableWidth)
	}
	if cfg.UI.BorderColor != "240" {
		t.Fatalf("UI.BorderColor = %q, want 240", cfg.UI.BorderColor)
	}
}

func TestResolveConfigPathPrefersUserConfig(t *testing.T) {
	dir := t.TempDir()
	origUserConfigDir := userConfigDirFn
	origEUID := currentEUIDFn
	defer func() {
		userConfigDirFn = origUserConfigDir
		currentEUIDFn = origEUID
	}()

	userConfigDirFn = func() (string, error) {
		return dir, nil
	}
	currentEUIDFn = func() int { return 1000 }

	userPath := filepath.Join(dir, AppName, FileName)
	if err := os.MkdirAll(filepath.Dir(userPath), 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(userPath, []byte("tick: 1\n"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	path, err := ResolveConfigPath()
	if err != nil {
		t.Fatalf("ResolveConfigPath() error = %v", err)
	}
	if path != userPath {
		t.Fatalf("ResolveConfigPath() = %q, want %q", path, userPath)
	}
}

func TestResolveConfigPathReturnsUserPathWhenNothingExists(t *testing.T) {
	dir := t.TempDir()
	origUserConfigDir := userConfigDirFn
	origEUID := currentEUIDFn
	defer func() {
		userConfigDirFn = origUserConfigDir
		currentEUIDFn = origEUID
	}()

	userConfigDirFn = func() (string, error) {
		return dir, nil
	}
	currentEUIDFn = func() int { return 1000 }

	path, err := ResolveConfigPath()
	if err != nil {
		t.Fatalf("ResolveConfigPath() error = %v", err)
	}

	want := filepath.Join(dir, AppName, FileName)
	if path != want {
		t.Fatalf("ResolveConfigPath() = %q, want %q", path, want)
	}
}

func TestResolveConfigPathUsesSystemPathForRoot(t *testing.T) {
	origEUID := currentEUIDFn
	defer func() { currentEUIDFn = origEUID }()

	currentEUIDFn = func() int { return 0 }

	path, err := ResolveConfigPath()
	if err != nil {
		t.Fatalf("ResolveConfigPath() error = %v", err)
	}
	if path != SystemConfigPath {
		t.Fatalf("ResolveConfigPath() = %q, want %q", path, SystemConfigPath)
	}
}
