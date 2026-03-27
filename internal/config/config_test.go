package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewConfigReturnsZeroValuesWhenEnvIsMissing(t *testing.T) {
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
}

func TestNewConfigReadsEnvFile(t *testing.T) {
	dir := t.TempDir()
	prevWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	defer func() { _ = os.Chdir(prevWD) }()

	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}

	env := []byte("TICK=9\nLOG_PATH=/tmp/app.log\nDAEMON=true\nDAEMON_TICK=7\nDAEMON_RAM_QUOTA=2G\n")
	if err := os.WriteFile(filepath.Join(dir, ".env"), env, 0644); err != nil {
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
}
