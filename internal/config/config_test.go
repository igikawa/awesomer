package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewConfigReturnsZeroValuesWhenConfigFileIsMissing(t *testing.T) {
	dir := t.TempDir()
	prevWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	defer func() { _ = os.Chdir(prevWD) }()

	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}

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
	prevWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	defer func() { _ = os.Chdir(prevWD) }()

	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}

	configYAML := []byte("tick: 9\nlogger:\n  log_path: /tmp/app.log\ndaemon:\n  run: true\n  tick: 7\n  ram_quota: 2G\n  whitelist:\n    - systemd\n    - dockerd\nui:\n  table_width: 48\n  border_color: \"240\"\n")
	if err := os.WriteFile(filepath.Join(dir, FileName), configYAML, 0644); err != nil {
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
