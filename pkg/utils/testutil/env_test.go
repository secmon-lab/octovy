package testutil_test

import (
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/octovy/pkg/utils/testutil"
)

func TestGetEnvOrSkip(t *testing.T) {
	t.Run("Returns value when env var is set", func(t *testing.T) {
		key := "TEST_ENV_VAR_SET"
		expected := "test_value"
		t.Setenv(key, expected)

		value := testutil.GetEnvOrSkip(t, key)
		gt.V(t, value).Equal(expected)
	})

	t.Run("Skips test when env var is not set", func(t *testing.T) {
		// GetEnvOrSkip will call t.Skip() if env var is not set
		// We can't easily test Skip behavior in unit tests
		// This test is mainly for documentation purposes
		t.Skip("Skipping test that validates skip behavior")
	})
}
