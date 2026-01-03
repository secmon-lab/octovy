package usecase_test

import (
	"testing"

	"github.com/m-mizutani/octovy/pkg/infra"
	"github.com/m-mizutani/octovy/pkg/usecase"
)

func TestNew(t *testing.T) {
	t.Run("create new usecase with all clients", func(t *testing.T) {
		// This test verifies that the usecase can be created with proper clients
		// The actual behavior is tested in individual method tests
		clients := infra.New()
		uc := usecase.New(clients)

		// Test that methods are accessible (compile-time check)
		// Actual behavior tests should be in specific test functions
		_ = uc.ScanGitHubRepo
		_ = uc.InsertScanResult
	})
}

// TestWithBigQueryTableID tests are covered in insert_scan_result_test.go
// with proper mock BigQuery client to verify actual table ID usage
