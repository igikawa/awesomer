package main

import (
	"awesomeProject/internal/config"
	"awesomeProject/internal/daemon"
	"awesomeProject/internal/daemon/info"
	"awesomeProject/internal/tui"
	"awesomeProject/pkg/logger"
	"log"

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
	apiInstance := info.NewAPI()

	if cfg.Daemon.Run {
		go func() {
			l, err := logger.NewDaemonLogger(&cfg.LoggerConfig)
			if err != nil {
				panic(err)
			}
			d := daemon.New(&cfg.Daemon, l, apiInstance)

			if err := d.Run(); err != nil {
				log.Fatal(err)
			}
		}()
	}

	err = tui.Run(cfg, apiInstance)
	if err != nil {
		log.Fatal(err)
	}
}
