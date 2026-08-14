package logging

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestNewAddsServiceFields(t *testing.T) {
	var output bytes.Buffer
	logger := New(&output, "gateway", "test")
	logger.Info("ready")

	var record map[string]any
	if err := json.Unmarshal(output.Bytes(), &record); err != nil {
		t.Fatalf("decode log record: %v", err)
	}
	for key, want := range map[string]string{
		"service":     "gateway",
		"environment": "test",
		"msg":         "ready",
	} {
		if got := record[key]; got != want {
			t.Errorf("record[%q] = %v, want %q", key, got, want)
		}
	}
}
