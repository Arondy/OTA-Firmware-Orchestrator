package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/Arondy/OTA-Firmware-Orchestrator/internal/core"
	"github.com/Arondy/OTA-Firmware-Orchestrator/internal/core/config"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	logger := config.GetLogger()
	defer config.CloseLoggerFiles()
	defer logger.Sync()

	cfg := config.LoadConfig()

	if err := core.Run(ctx, cfg, logger); err != nil {
		logger.Named("Main").Errorf("Server error: %s", err)
		os.Exit(1)
	}
}
