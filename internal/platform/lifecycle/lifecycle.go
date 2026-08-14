// Package lifecycle coordinates process startup and signal-aware shutdown.
package lifecycle

import (
	"context"
	"log/slog"
)

// Run announces startup and waits for root context cancellation.
func Run(ctx context.Context, logger *slog.Logger) {
	logger.InfoContext(ctx, "service started")
	<-ctx.Done()
	logger.Info("shutdown started")
	logger.Info("shutdown completed")
}
