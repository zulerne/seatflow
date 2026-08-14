// Package logging creates the structured logger shared by SeatFlow processes.
package logging

import (
	"io"
	"log/slog"
	"strings"
)

// New creates a JSON logger carrying the service and environment on every record.
func New(output io.Writer, service, environment string) *slog.Logger {
	if output == nil {
		output = io.Discard
	}
	if strings.TrimSpace(environment) == "" {
		environment = "unknown"
	}

	handler := slog.NewJSONHandler(output, &slog.HandlerOptions{Level: slog.LevelInfo})
	return slog.New(handler).With(
		slog.String("service", service),
		slog.String("environment", environment),
	)
}
