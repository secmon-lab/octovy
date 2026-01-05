package usecase_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/octovy/pkg/usecase"
)

func TestLoadTrivyReport(t *testing.T) {
	ctx := context.Background()

	t.Run("valid trivy report JSON", func(t *testing.T) {
		validJSON := `{
  "SchemaVersion": 2,
  "ArtifactName": ".",
  "ArtifactType": "filesystem",
  "Results": [
    {
      "Target": "go.mod",
      "Class": "lang-pkgs",
      "Type": "gomod"
    }
  ]
}`
		reader := strings.NewReader(validJSON)
		report, err := usecase.LoadTrivyReport(ctx, reader)

		gt.NoError(t, err)
		gt.V(t, report).NotEqual(nil)
		gt.V(t, report.SchemaVersion).Equal(2)
		gt.V(t, report.ArtifactName).Equal(".")
		gt.V(t, report.ArtifactType).Equal("filesystem")
	})

	t.Run("invalid JSON format", func(t *testing.T) {
		invalidJSON := `{invalid json`
		reader := strings.NewReader(invalidJSON)
		_, err := usecase.LoadTrivyReport(ctx, reader)

		gt.Error(t, err)
	})

	t.Run("missing SchemaVersion", func(t *testing.T) {
		noSchemaJSON := `{
  "ArtifactName": ".",
  "ArtifactType": "filesystem"
}`
		reader := strings.NewReader(noSchemaJSON)
		_, err := usecase.LoadTrivyReport(ctx, reader)

		gt.Error(t, err)
	})

	t.Run("result with empty target", func(t *testing.T) {
		emptyTargetJSON := `{
  "SchemaVersion": 2,
  "ArtifactName": ".",
  "ArtifactType": "filesystem",
  "Results": [
    {
      "Target": "",
      "Type": "gomod"
    }
  ]
}`
		reader := strings.NewReader(emptyTargetJSON)
		_, err := usecase.LoadTrivyReport(ctx, reader)

		gt.Error(t, err)
		gt.True(t, strings.Contains(err.Error(), "result target is empty"))
	})

	t.Run("multiple results with one empty target", func(t *testing.T) {
		mixedJSON := `{
  "SchemaVersion": 2,
  "ArtifactName": ".",
  "ArtifactType": "filesystem",
  "Results": [
    {
      "Target": "go.mod",
      "Type": "gomod"
    },
    {
      "Target": "",
      "Type": "npm"
    }
  ]
}`
		reader := strings.NewReader(mixedJSON)
		_, err := usecase.LoadTrivyReport(ctx, reader)

		gt.Error(t, err)
		gt.True(t, strings.Contains(err.Error(), "result target is empty"))
	})
}

func TestLoadTrivyReportFromFile(t *testing.T) {
	ctx := context.Background()

	t.Run("valid trivy report file", func(t *testing.T) {
		validJSON := `{
  "SchemaVersion": 2,
  "ArtifactName": "test-artifact",
  "ArtifactType": "filesystem",
  "Results": []
}`
		tmpFile, err := os.CreateTemp("", "trivy_test_*.json")
		gt.NoError(t, err)
		defer os.Remove(tmpFile.Name())

		_, err = tmpFile.WriteString(validJSON)
		gt.NoError(t, err)
		gt.NoError(t, tmpFile.Close())

		report, err := usecase.LoadTrivyReportFromFile(ctx, tmpFile.Name())

		gt.NoError(t, err)
		gt.V(t, report).NotEqual(nil)
		gt.V(t, report.SchemaVersion).Equal(2)
		gt.V(t, report.ArtifactName).Equal("test-artifact")
	})

	t.Run("file does not exist", func(t *testing.T) {
		nonExistentPath := filepath.Join(t.TempDir(), "nonexistent.json")
		_, err := usecase.LoadTrivyReportFromFile(ctx, nonExistentPath)

		gt.Error(t, err)
	})

	t.Run("invalid JSON in file", func(t *testing.T) {
		invalidJSON := `{invalid`
		tmpFile, err := os.CreateTemp("", "trivy_test_invalid_*.json")
		gt.NoError(t, err)
		defer os.Remove(tmpFile.Name())

		_, err = tmpFile.WriteString(invalidJSON)
		gt.NoError(t, err)
		gt.NoError(t, tmpFile.Close())

		_, err = usecase.LoadTrivyReportFromFile(ctx, tmpFile.Name())

		gt.Error(t, err)
	})
}
