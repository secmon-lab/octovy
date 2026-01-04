package usecase

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/octovy/pkg/domain/model/trivy"
	"github.com/m-mizutani/octovy/pkg/utils/safe"
)

// LoadTrivyReport loads a Trivy report from an io.Reader and validates it
func LoadTrivyReport(ctx context.Context, r io.Reader) (*trivy.Report, error) {
	var report trivy.Report
	if err := json.NewDecoder(r).Decode(&report); err != nil {
		return nil, goerr.Wrap(err, "failed to decode trivy result")
	}

	if err := report.Validate(); err != nil {
		return nil, goerr.Wrap(err, "invalid trivy report")
	}

	return &report, nil
}

// LoadTrivyReportFromFile loads a Trivy report from a file and validates it
func LoadTrivyReportFromFile(ctx context.Context, filePath string) (*trivy.Report, error) {
	fd, err := os.Open(filepath.Clean(filePath))
	if err != nil {
		return nil, goerr.Wrap(err, "failed to open trivy result file", goerr.V("path", filePath))
	}
	defer safe.Close(fd)

	return LoadTrivyReport(ctx, fd)
}
