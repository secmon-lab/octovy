package logging_test

import (
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/octovy/pkg/utils/logging"
)

func TestConfigure(t *testing.T) {
	t.Run("configure with json format to stdout", func(t *testing.T) {
		err := logging.Configure("json", "info", "stdout")
		gt.NoError(t, err)
		// Successful configuration is validated by no error
		// Actual log format testing requires output interception
	})

	t.Run("configure with text format", func(t *testing.T) {
		err := logging.Configure("text", "debug", "stdout")
		gt.NoError(t, err)
		// Successful configuration is validated by no error
	})

	t.Run("configure with invalid format returns error", func(t *testing.T) {
		err := logging.Configure("invalid", "info", "stdout")
		gt.Error(t, err)
	})

	t.Run("configure with invalid level returns error", func(t *testing.T) {
		err := logging.Configure("json", "invalid", "stdout")
		gt.Error(t, err)
	})
}

func TestDefault(t *testing.T) {
	// Test that Default() returns a functional logger
	logger := logging.Default()
	logger.Info("test message", "key", "value")
	// If this doesn't panic, the logger is functional
}
