package main

import (
	"awesomeProject/internal/config"
	"awesomeProject/internal/daemon"
	"awesomeProject/internal/daemon/info"
	"awesomeProject/internal/tui"
	"awesomeProject/pkg/logger"
	"context"
	"fmt"
	"log"
	"os"
	"sync"
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

	dl, err := logger.NewDaemonLogger(&cfg.LoggerConfig)
	if err != nil {
		panic(err)
	}
	d := daemon.New(&cfg.Daemon, dl, apiInstance)

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)

	wg := &sync.WaitGroup{}

	if cfg.Daemon.Run {
		wg.Add(1)
		go func() {
			if err := d.Run(ctx); err != nil {
				log.Fatal(err)
			}
			wg.Done()
		}()
	}

	l, err := logger.NewLogger(&cfg.LoggerConfig)

	if err = tui.Run(cancel, cfg, l, apiInstance, tui.NewTable()); err != nil {
		log.Fatal(err)
	}

	wg.Wait()
}
