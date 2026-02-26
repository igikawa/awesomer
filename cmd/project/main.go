package main

import (
	"awesomeProject/internal/config"
	"awesomeProject/internal/daemon"
	"awesomeProject/internal/tui"
	"awesomeProject/pkg/logger"

	"fmt"
	"os"
	//"os/signal"
	//"syscall"
)

func main() {
	_, err := os.Open(".env")
	if err != nil {
		_, err = os.Create(".env")
		if err != nil {
			fmt.Println(err)
		}
	}

	cfg := config.NewConfig()

	logger.NewLogger(cfg.LoggerConfig)

	if cfg.Daemon.Run {
		go func() {
			err := daemon.Run(cfg.Daemon)
			if err != nil {
				logger.DaemonLogger.Fatal(err)
			}
		}()
	}

	err = tui.Run(cfg.Tick)
	if err != nil {
		logger.Logger.Println("Error running program:", err)
	}
}
