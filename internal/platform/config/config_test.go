package config

import (
	"strings"
	"testing"
)

func TestLoad(t *testing.T) {
	t.Setenv("SEATFLOW_ENV", " test ")
	t.Setenv("HTTP_ADDR", " :8080 ")

	cfg, err := Load("HTTP_ADDR")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Environment != "test" {
		t.Fatalf("Environment = %q, want %q", cfg.Environment, "test")
	}
}

func TestLoadReportsMissingVariables(t *testing.T) {
	t.Setenv("SEATFLOW_ENV", "")
	t.Setenv("GRPC_ADDR", "")

	_, err := Load("GRPC_ADDR")
	if err == nil {
		t.Fatal("Load() error = nil, want missing variables error")
	}
	for _, name := range []string{"SEATFLOW_ENV", "GRPC_ADDR"} {
		if !strings.Contains(err.Error(), name) {
			t.Errorf("Load() error = %q, want it to contain %q", err, name)
		}
	}
}

func TestLoadRejectsEmptyVariableName(t *testing.T) {
	t.Setenv("SEATFLOW_ENV", "test")

	_, err := Load("")
	if err == nil || !strings.Contains(err.Error(), "must not be empty") {
		t.Fatalf("Load() error = %v, want empty name error", err)
	}
}
