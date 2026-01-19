package main

import (
	"awesomeProject/internal/processes"
	"awesomeProject/internal/tui"
	"awesomeProject/pkg/logger"

	"os"
)

func main() {

	switch len(os.Args) {
	case 1:
		processes.SortMode = "-m"
	default:
		processes.SortMode = os.Args[1]
	}

	logger.Logger.Println("SortMode:", processes.SortMode)

	_, err := os.Create(".env")
	if err != nil {
		logger.Logger.Println(err)
	}

	err = tui.Run()
	if err != nil {
		logger.Logger.Println("Error running program:", err)
	}
}
