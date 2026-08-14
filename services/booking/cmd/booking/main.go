package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/zulerne/seatflow/internal/platform/config"
	"github.com/zulerne/seatflow/internal/platform/lifecycle"
	"github.com/zulerne/seatflow/internal/platform/logging"
)

const serviceName = "booking"

func main() {
	os.Exit(run())
}

func run() int {
	startupLogger := logging.New(os.Stderr, serviceName, "unknown")
	cfg, err := config.Load("GRPC_ADDR")
	if err != nil {
		startupLogger.Error("startup failed", "error", err)
		return 1
	}

	logger := logging.New(os.Stdout, serviceName, cfg.Environment)
	rootCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	lifecycle.Run(rootCtx, logger)
	return 0
}
