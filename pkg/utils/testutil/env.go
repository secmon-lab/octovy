package testutil

import (
	"os"
	"testing"
)

// GetEnvOrSkip returns the value of the environment variable. If not set, skip the test.
func GetEnvOrSkip(t *testing.T, key string) string {
	t.Helper()
	value := os.Getenv(key)
	if value == "" {
		t.Skipf("Environment variable %s is not set, skipping test", key)
	}
	return value
}
