package logger

import (
	"log"
	"os"
)

type Config struct {
	LogPath       string `yaml:"log_path"`
	DaemonLogPath string `yaml:"daemon_log_path"`
}

func DefaultConfig() Config {
	return Config{
		LogPath:       "./awesome.log",
		DaemonLogPath: "./awesome.daemon.log",
	}
}

func NewLogger(cfg *Config) (*log.Logger, error) {
	logFile, err := os.OpenFile(cfg.LogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, err
	}

	Logger := log.New(logFile, "", log.Ldate|log.Ltime|log.Lshortfile)
	return Logger, nil
}

func NewDaemonLogger(cfg *Config) (*log.Logger, error) {
	logFile, err := os.OpenFile(cfg.DaemonLogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, err
	}
	Logger := log.New(logFile, "", log.Ldate|log.Ltime|log.Lshortfile)
	return Logger, nil
}
