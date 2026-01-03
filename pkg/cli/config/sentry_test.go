package config_test

import (
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/octovy/pkg/cli/config"
)

func TestSentryFlags(t *testing.T) {
	sentryConfig := &config.Sentry{}
	flags := sentryConfig.Flags()

	gt.V(t, len(flags)).Equal(2)

	// Verify flag names
	flagNames := make(map[string]bool)
	for _, flag := range flags {
		flagNames[flag.Names()[0]] = true
	}

	gt.True(t, flagNames["sentry-dsn"])
	gt.True(t, flagNames["sentry-env"])
}
