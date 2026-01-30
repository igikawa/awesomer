package main

import (
	"awesomeProject/internal/tui"
	"awesomeProject/pkg/logger"

	"os"
)

func main() {
	_, err := os.Open(".env")
	if err != nil {
		_, err = os.Create(".env")
		if err != nil {
			logger.Logger.Println(err)
		}
	}

	err = tui.Run()
	if err != nil {
		logger.Logger.Println("Error running program:", err)
	}
}
