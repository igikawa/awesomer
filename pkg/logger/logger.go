package logger

import (
	"log"
	"os"
)

type Config struct {
	LogPath       string `env:"LOG_PATH" env-default:"./awesome.log"`
	DaemonLogPath string `env:"DAEMON_LOG_PATH" env-default:"./awesome.daemon.log"`
}

var Logger *log.Logger
var DaemonLogger *log.Logger

func NewLogger(cfg Config) {
	logFile, err := os.OpenFile(cfg.LogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		panic(err)
	}
	daemonLogFile, err := os.OpenFile(cfg.DaemonLogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		panic(err)
	}

	Logger = log.New(logFile, "", log.Ldate|log.Ltime|log.Lshortfile)
	DaemonLogger = log.New(daemonLogFile, "", log.Ldate|log.Ltime|log.Lshortfile)
}
