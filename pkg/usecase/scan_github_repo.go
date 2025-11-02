package usecase

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/go-github/v53/github"
	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/octovy/pkg/domain/interfaces"
	"github.com/m-mizutani/octovy/pkg/domain/model"
	"github.com/m-mizutani/octovy/pkg/domain/model/trivy"
	"github.com/m-mizutani/octovy/pkg/domain/types"
	"github.com/m-mizutani/octovy/pkg/infra"
	"github.com/m-mizutani/octovy/pkg/utils"
)

// ScanGitHubRepo is a usecase to download a source code from GitHub and scan it with Trivy. Using GitHub App credentials to download a private repository, then the app should be installed to the repository and have read access.
// After scanning, the result is stored to the database. The temporary files are removed after the scan.
func (x *UseCase) ScanGitHubRepo(ctx context.Context, input *model.ScanGitHubRepoInput) error {
	if err := input.Validate(); err != nil {
		return err
	}

	// Create and finalize GitHub check
	conclusion := "cancelled"
	checkID, err := x.clients.GitHubApp().CreateCheckRun(ctx, input.InstallID, &input.GitHubRepo, input.CommitID)
	if err != nil {
		return err
	}
	defer func() {
		opt := &github.UpdateCheckRunOptions{
			Status:     github.String("completed"),
			Conclusion: &conclusion,
		}
		if err := x.clients.GitHubApp().UpdateCheckRun(ctx, input.InstallID, &input.GitHubRepo, checkID, opt); err != nil {
			utils.CtxLogger(ctx).Error("Failed to update check run", "err", err)
		}
	}()

	// Extract zip file to local temp directory
	tmpDir, err := os.MkdirTemp("", fmt.Sprintf("octovy.%s.%s.%s.*", input.Owner, input.RepoName, input.CommitID))
	if err != nil {
		return goerr.Wrap(err, "failed to create temp directory for zip file")
	}
	defer utils.SafeRemoveAll(tmpDir)

	if err := x.downloadGitHubRepo(ctx, input, tmpDir); err != nil {
		return err
	}

	cfg, err := model.LoadConfigsFromDir(filepath.Join(tmpDir, ".octovy"))
	if err != nil {
		return err
	}

	report, err := x.scanGitHubRepo(ctx, tmpDir)
	if err != nil {
		return err
	}
	utils.CtxLogger(ctx).Info("scan finished", "input", input, "report", report)

	if err := x.InsertScanResult(ctx, input.GitHubMetadata, *report, *cfg); err != nil {
		return err
	}

	if nil != x.clients.Storage() && nil != input.GitHubMetadata.PullRequest {
		if err := x.CommentGitHubPR(ctx, input, report, cfg); err != nil {
			return err
		}
	}

	conclusion = "success"

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
	defer utils.SafeRemove(tmpZip.Name())

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

func (x *UseCase) scanGitHubRepo(ctx context.Context, codeDir string) (*trivy.Report, error) {
	// Scan local directory
	tmpResult, err := os.CreateTemp("", "octovy_result.*.json")
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create temp file for scan result")
	}
	defer utils.SafeRemove(tmpResult.Name())

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

	utils.CtxLogger(ctx).Info("Scan result", slog.Any("report", tmpResult.Name()))

	return &report, nil
}

func unmarshalFile(path string, v any) error {
	fd, err := os.Open(filepath.Clean(path))
	if err != nil {
		return goerr.Wrap(err, "failed to open file", goerr.V("path", path))
	}
	defer utils.SafeClose(fd)

	if err := json.NewDecoder(fd).Decode(v); err != nil {
		return goerr.Wrap(err, "failed to decode json", goerr.V("path", path))
	}

	return nil
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
	defer utils.SafeClose(zipFile)

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

	target := stepDownDirectory(f.Name)
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
	defer utils.SafeClose(out)

	rc, err := f.Open()
	if err != nil {
		return goerr.Wrap(err, "failed to open zip entry")
	}
	defer utils.SafeClose(rc)

	// #nosec
	_, err = io.Copy(out, rc)
	if err != nil {
		return goerr.Wrap(err, "failed to copy file content")
	}

	return nil
}

func stepDownDirectory(fpath string) string {
	if len(fpath) > 0 && fpath[0] == filepath.Separator {
		fpath = fpath[1:]
	}

	p := fpath
	var arr []string
	for {
		d, f := filepath.Split(p)
		if d == "" {
			break
		}
		arr = append([]string{f}, arr...)
		p = filepath.Clean(d)
	}

	return filepath.Join(arr...)
}
