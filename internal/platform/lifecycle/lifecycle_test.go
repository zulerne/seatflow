package lifecycle

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
)

func TestRunReturnsAfterCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var output bytes.Buffer
	Run(ctx, slog.New(slog.NewTextHandler(&output, nil)))
	for _, message := range []string{"service started", "shutdown started", "shutdown completed"} {
		if !strings.Contains(output.String(), message) {
			t.Errorf("logs = %q, want message %q", output.String(), message)
		}
	}
}
