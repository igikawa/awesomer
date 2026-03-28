package logger

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewLoggerWritesToConfiguredFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.log")

	l, err := NewLogger(&Config{LogPath: path})
	if err != nil {
		t.Fatalf("NewLogger() error = %v", err)
	}

	l.Print("hello")

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if len(data) == 0 {
		t.Fatal("log file is empty")
	}
}

func TestNewDaemonLoggerWritesToConfiguredFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "daemon.log")

	l, err := NewDaemonLogger(&Config{DaemonLogPath: path})
	if err != nil {
		t.Fatalf("NewDaemonLogger() error = %v", err)
	}

	l.Print("daemon")

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if len(data) == 0 {
		t.Fatal("daemon log file is empty")
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.LogPath != "./awesome.log" {
		t.Fatalf("LogPath = %q, want ./awesome.log", cfg.LogPath)
	}
	if cfg.DaemonLogPath != "./awesome.daemon.log" {
		t.Fatalf("DaemonLogPath = %q, want ./awesome.daemon.log", cfg.DaemonLogPath)
	}
}

func TestNewLoggerReturnsErrorForInvalidPath(t *testing.T) {
	_, err := NewLogger(&Config{LogPath: filepath.Join("/missing", "app.log")})
	if err == nil {
		t.Fatal("NewLogger() error = nil, want non-nil")
	}
}

func TestNewDaemonLoggerReturnsErrorForInvalidPath(t *testing.T) {
	_, err := NewDaemonLogger(&Config{DaemonLogPath: filepath.Join("/missing", "daemon.log")})
	if err == nil {
		t.Fatal("NewDaemonLogger() error = nil, want non-nil")
	}
}
