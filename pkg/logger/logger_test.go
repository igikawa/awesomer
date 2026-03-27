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
