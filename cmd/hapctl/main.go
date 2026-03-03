package main

import (
	"os"

	"github.com/eliasmeireles/hapctl/internal/cmd"
	"github.com/eliasmeireles/hapctl/internal/logger"
)

func main() {
	if err := logger.Init(""); err != nil {
		logger.Fatal("Failed to initialize logger: %v", err)
	}

	if err := cmd.Execute(); err != nil {
		logger.Fatal("Command execution failed: %v", err)
		os.Exit(1)
	}
}
