package logger

import (
	"log"
	"os"
)

type Config struct {
	LogPath       string `env:"LOG_PATH" env-default:"./awesome.log"`
	DaemonLogPath string `env:"DAEMON_LOG_PATH" env-default:"./awesome.daemon.log"`
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
