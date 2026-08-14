// Package config loads and validates process configuration from environment variables.
package config

import (
	"fmt"
	"os"
	"strings"
)

const environmentKey = "SEATFLOW_ENV"

// Config contains the environment and validated service-specific values.
type Config struct {
	Environment string
}

// Load reads SEATFLOW_ENV and the supplied required environment variables.
func Load(required ...string) (Config, error) {
	names := append([]string{environmentKey}, required...)
	values := make(map[string]string, len(names))
	seen := make(map[string]struct{}, len(names))
	missing := make([]string, 0, len(names))

	for _, name := range names {
		if name == "" {
			return Config{}, fmt.Errorf("required environment variable name must not be empty")
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}

		value, ok := os.LookupEnv(name)
		value = strings.TrimSpace(value)
		if !ok || value == "" {
			missing = append(missing, name)
			continue
		}
		values[name] = value
	}

	if len(missing) > 0 {
		return Config{}, fmt.Errorf("required environment variables are not set: %s", strings.Join(missing, ", "))
	}

	return Config{Environment: values[environmentKey]}, nil
}
