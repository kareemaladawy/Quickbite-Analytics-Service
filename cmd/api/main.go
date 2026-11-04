package main

import (
	"os"

	"quickbite/analytics-service/lib/boot"
	"quickbite/analytics-service/lib/logger"
)

func main() {
	if err := boot.Run(); err != nil {
		logger.Base().Error("fatal boot error", "error", err.Error())
		os.Exit(1)
	}
}
