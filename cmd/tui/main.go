package main

import (
	"awesomeProject/internal/processes"
	"awesomeProject/internal/tui"
	"awesomeProject/pkg/logger"

	"os"
)

func main() {
	processes.SortMode = os.Args[1]
	logger.Logger.Println("SortMode:", processes.SortMode)

	err := tui.Run()
	if err != nil {
		logger.Logger.Println("Error running program:", err)
	}
}
