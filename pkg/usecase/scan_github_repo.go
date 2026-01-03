package usecase

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/octovy/pkg/domain/interfaces"
	"github.com/m-mizutani/octovy/pkg/domain/model"
	"github.com/m-mizutani/octovy/pkg/domain/model/trivy"
	"github.com/m-mizutani/octovy/pkg/domain/types"
	"github.com/m-mizutani/octovy/pkg/infra"
	"github.com/m-mizutani/octovy/pkg/utils/logging"
	"github.com/m-mizutani/octovy/pkg/utils/safe"
)

// ScanGitHubRepo is a usecase to download a source code from GitHub and scan it with Trivy. Using GitHub App credentials to download a private repository, then the app should be installed to the repository and have read access.
// After scanning, the result is stored to the database. The temporary files are removed after the scan.
func (x *UseCase) ScanGitHubRepo(ctx context.Context, input *model.ScanGitHubRepoInput) error {
	if err := input.Validate(); err != nil {
		return err
	}

	// Extract zip file to local temp directory
	tmpDir, err := os.MkdirTemp("", fmt.Sprintf("octovy.%s.%s.%s.*", input.Owner, input.RepoName, input.CommitID))
	if err != nil {
		return goerr.Wrap(err, "failed to create temp directory for zip file")
	}
	defer safe.RemoveAll(tmpDir)

	if err := x.downloadGitHubRepo(ctx, input, tmpDir); err != nil {
		return err
	}

	return x.ScanAndInsert(ctx, tmpDir, input.GitHubMetadata)
}

// ScanAndInsert scans a directory with Trivy and inserts the result to BigQuery
func (x *UseCase) ScanAndInsert(ctx context.Context, dir string, meta model.GitHubMetadata) error {
	report, err := x.scanDirectory(ctx, dir)
	if err != nil {
		return err
	}
	logging.From(ctx).Info("scan finished", "owner", meta.Owner, "repo", meta.RepoName, "commit", meta.CommitID)

	if err := x.InsertScanResult(ctx, meta, *report); err != nil {
		return err
	}

	return nil
}

func (x *UseCase) downloadGitHubRepo(ctx context.Context, input *model.ScanGitHubRepoInput, dstDir string) error {
	zipURL, err := x.clients.GitHubApp().GetArchiveURL(ctx, &interfaces.GetArchiveURLInput{
		Owner:     input.Owner,
		Repo:      input.RepoName,
		CommitID:  input.CommitID,
		InstallID: input.InstallID,
	})
	if err != nil {
		return err
	}

	// Download zip file
	tmpZip, err := os.CreateTemp("", fmt.Sprintf("octovy_code.%s.%s.%s.*.zip",
		input.Owner, input.RepoName, input.CommitID,
	))
	if err != nil {
		return goerr.Wrap(err, "failed to create temp file for zip file")
	}
	defer safe.Remove(tmpZip.Name())

	if err := downloadZipFile(ctx, x.clients.HTTPClient(), zipURL, tmpZip); err != nil {
		return err
	}
	if err := tmpZip.Close(); err != nil {
		return goerr.Wrap(err, "failed to close temp file for zip file")
	}

	if err := extractZipFile(ctx, tmpZip.Name(), dstDir); err != nil {
		return err
	}

	return nil
}

// scanDirectory scans a directory with Trivy and returns the report
func (x *UseCase) scanDirectory(ctx context.Context, codeDir string) (*trivy.Report, error) {
	// Scan local directory
	tmpResult, err := os.CreateTemp("", "octovy_result.*.json")
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create temp file for scan result")
	}
	defer safe.Remove(tmpResult.Name())

	if err := tmpResult.Close(); err != nil {
		return nil, goerr.Wrap(err, "failed to close temp file for scan result")
	}

	if err := x.clients.Trivy().Run(ctx, []string{
		"fs",
		"--exit-code", "0",
		"--no-progress",
		"--format", "json",
		"--output", tmpResult.Name(),
		"--list-all-pkgs",
		codeDir,
	}); err != nil {
		return nil, goerr.Wrap(err, "failed to scan local directory")
	}

	var report trivy.Report
	if err := unmarshalFile(tmpResult.Name(), &report); err != nil {
		return nil, err
	}

	logging.From(ctx).Debug("Scan result saved", "result_file", tmpResult.Name())

	return &report, nil
}

func unmarshalFile(path string, v any) error {
	fd, err := os.Open(filepath.Clean(path))
	if err != nil {
		return goerr.Wrap(err, "failed to open file", goerr.V("path", path))
	}
	defer safe.Close(fd)

	if err := json.NewDecoder(fd).Decode(v); err != nil {
		return goerr.Wrap(err, "failed to decode json", goerr.V("path", path))
	}

	return nil
}

// ScanDirectoryForTest is exported for testing purposes
func (x *UseCase) ScanDirectoryForTest(ctx context.Context, codeDir string) (*trivy.Report, error) {
	return x.scanDirectory(ctx, codeDir)
}

func downloadZipFile(ctx context.Context, httpClient infra.HTTPClient, zipURL *url.URL, w io.Writer) error {
	zipReq, err := http.NewRequestWithContext(ctx, http.MethodGet, zipURL.String(), nil)
	if err != nil {
		return goerr.Wrap(err, "failed to create request for zip file", goerr.V("url", zipURL))
	}

	zipResp, err := httpClient.Do(zipReq)
	if err != nil {
		return goerr.Wrap(err, "failed to download zip file", goerr.V("url", zipURL))
	}
	defer zipResp.Body.Close()

	if zipResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(zipResp.Body)
		return goerr.Wrap(types.ErrInvalidGitHubData, "failed to download zip file",
			goerr.V("url", zipURL),
			goerr.V("resp", zipResp),
			goerr.V("body", body),
		)
	}

	if _, err = io.Copy(w, zipResp.Body); err != nil {
		return goerr.Wrap(err, "failed to write zip file",
			goerr.V("url", zipURL),
			goerr.V("resp", zipResp),
		)
	}

	return nil
}

func extractZipFile(ctx context.Context, src, dst string) error {
	zipFile, err := zip.OpenReader(src)
	if err != nil {
		return goerr.Wrap(err, "failed to open zip file", goerr.V("file", src))
	}
	defer safe.Close(zipFile)

	// Extract a source code zip file
	for _, f := range zipFile.File {
		if err := extractCode(ctx, f, dst); err != nil {
			return err
		}
	}

	return nil
}

func extractCode(_ context.Context, f *zip.File, dst string) error {
	if f.FileInfo().IsDir() {
		return nil
	}

	target, err := stepDownDirectory(f.Name)
	if err != nil {
		return err
	}
	if target == "" {
		return nil
	}

	fpath := filepath.Join(dst, target)
	if !strings.HasPrefix(fpath, filepath.Clean(dst)+string(os.PathSeparator)) {
		return goerr.Wrap(types.ErrInvalidGitHubData, "illegal file path of zip", goerr.V("path", fpath))
	}

	if err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
		return goerr.Wrap(err, "failed to create directory", goerr.V("path", fpath))
	}

	// #nosec
	out, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
	if err != nil {
		return goerr.Wrap(err, "failed to open file", goerr.V("fpath", fpath))
	}
	defer safe.Close(out)

	rc, err := f.Open()
	if err != nil {
		return goerr.Wrap(err, "failed to open zip entry")
	}
	defer safe.Close(rc)

	// #nosec
	_, err = io.Copy(out, rc)
	if err != nil {
		return goerr.Wrap(err, "failed to copy file content")
	}

	return nil
}

func stepDownDirectory(fpath string) (string, error) {
	if fpath == "" {
		return "", nil
	}

	normalized := strings.ReplaceAll(fpath, "\\", "/")
	normalized = strings.TrimLeft(normalized, "/")
	if normalized == "" {
		return "", nil
	}

	parts := strings.Split(normalized, "/")
	if len(parts) <= 1 {
		return "", nil
	}
	parts = parts[1:]

	var safeParts []string
	for _, part := range parts {
		if part == "" || part == "." {
			continue
		}
		if part == ".." {
			return "", goerr.Wrap(types.ErrInvalidGitHubData, "illegal file path of zip", goerr.V("path", fpath))
		}
		safeParts = append(safeParts, part)
	}

	if len(safeParts) == 0 {
		return "", nil
	}

	return filepath.Join(safeParts...), nil
}
