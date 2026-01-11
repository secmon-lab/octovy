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

// ScanGitHubRepoRemote scans a GitHub repository with parameter validation and completion.
// It supports two modes:
// 1. Full specification mode: all parameters (owner, repo, commit/branch, installID) are provided
// 2. DB completion mode: fetch missing parameters from repository (requires ScanRepository)
func (x *UseCase) ScanGitHubRepoRemote(ctx context.Context, input *model.ScanGitHubRepoRemoteInput) error {
	// Validate mutually exclusive parameters
	if input.Commit != "" && input.Branch != "" {
		return goerr.Wrap(types.ErrInvalidOption, "commit and branch cannot be specified at the same time")
	}

	// Determine operation mode
	isFullSpecMode := input.InstallID != 0 && (input.Commit != "" || input.Branch != "")

	if isFullSpecMode {
		scanInput, err := x.prepareScanInputFullSpec(ctx, input)
		if err != nil {
			return err
		}
		return x.ScanGitHubRepo(ctx, scanInput)
	}

	// DB completion mode
	scanInput, err := x.prepareScanInputDBCompletion(ctx, input)
	if err != nil {
		return err
	}
	return x.ScanGitHubRepo(ctx, scanInput)
}

// prepareScanInputFullSpec prepares ScanGitHubRepoInput for full specification mode
func (x *UseCase) prepareScanInputFullSpec(ctx context.Context, input *model.ScanGitHubRepoRemoteInput) (*model.ScanGitHubRepoInput, error) {
	logging.From(ctx).Info("Preparing scan input in full specification mode",
		"owner", input.Owner,
		"repo", input.Repo,
		"commit", input.Commit,
		"branch", input.Branch,
		"install_id", input.InstallID,
	)

	commitID := input.Commit
	branchName := input.Branch

	// Resolve commit ID from branch if needed
	if input.Branch != "" && input.Commit == "" {
		if x.clients.GitHubApp() == nil {
			return nil, goerr.Wrap(types.ErrInvalidOption, "GitHub App client is required to resolve branch to commit")
		}

		resolvedCommit, err := x.resolveBranchToCommit(ctx, input.Owner, input.Repo, input.Branch, input.InstallID)
		if err != nil {
			return nil, err
		}
		commitID = resolvedCommit
	}

	return &model.ScanGitHubRepoInput{
		GitHubMetadata: model.GitHubMetadata{
			GitHubCommit: model.GitHubCommit{
				GitHubRepo: model.GitHubRepo{
					RepoID:   0, // RepoID is optional for CLI-initiated scans
					Owner:    input.Owner,
					RepoName: input.Repo,
				},
				CommitID: commitID,
				Branch:   branchName,
			},
			InstallationID: int64(input.InstallID),
		},
		InstallID: input.InstallID,
	}, nil
}

// prepareScanInputDBCompletion prepares ScanGitHubRepoInput for DB completion mode
func (x *UseCase) prepareScanInputDBCompletion(ctx context.Context, input *model.ScanGitHubRepoRemoteInput) (*model.ScanGitHubRepoInput, error) {
	if x.clients.ScanRepository() == nil {
		return nil, goerr.Wrap(types.ErrInvalidOption,
			"DB completion mode requires ScanRepository. Please specify installation ID and commit/branch, or configure Firestore")
	}

	logging.From(ctx).Info("Preparing scan input in DB completion mode",
		"owner", input.Owner,
		"repo", input.Repo,
		"branch", input.Branch,
	)

	// Get repository information from database
	repoID := types.GitHubRepoID(fmt.Sprintf("%s/%s", input.Owner, input.Repo))
	repoInfo, err := x.clients.ScanRepository().GetRepository(ctx, repoID)
	if err != nil {
		return nil, goerr.Wrap(err, "repository not found in database",
			goerr.V("owner", input.Owner),
			goerr.V("repo", input.Repo),
		)
	}

	// Determine which branch to use
	branchName := types.BranchName(input.Branch)
	if branchName == "" {
		branchName = repoInfo.DefaultBranch
	}

	// Get branch information from database
	branchInfo, err := x.clients.ScanRepository().GetBranch(ctx, repoID, branchName)
	if err != nil {
		return nil, goerr.Wrap(err, "branch not found in database",
			goerr.V("owner", input.Owner),
			goerr.V("repo", input.Repo),
			goerr.V("branch", branchName),
		)
	}

	logging.From(ctx).Info("Retrieved repository information from database",
		"owner", input.Owner,
		"repo", input.Repo,
		"branch", branchName,
		"commit", branchInfo.LastCommitSHA,
		"installation_id", repoInfo.InstallationID,
	)

	return &model.ScanGitHubRepoInput{
		GitHubMetadata: model.GitHubMetadata{
			GitHubCommit: model.GitHubCommit{
				GitHubRepo: model.GitHubRepo{
					RepoID:   0, // RepoID is optional for CLI-initiated scans
					Owner:    input.Owner,
					RepoName: input.Repo,
				},
				CommitID: string(branchInfo.LastCommitSHA),
				Branch:   string(branchName),
			},
			InstallationID: repoInfo.InstallationID,
		},
		InstallID: types.GitHubAppInstallID(repoInfo.InstallationID),
	}, nil
}

// resolveBranchToCommit resolves a branch name to a commit SHA using GitHub API
func (x *UseCase) resolveBranchToCommit(ctx context.Context, owner, repo, branch string, installID types.GitHubAppInstallID) (string, error) {
	httpClient, err := x.clients.GitHubApp().HTTPClient(installID)
	if err != nil {
		return "", goerr.Wrap(err, "failed to create GitHub HTTP client")
	}

	// Call GitHub API to get branch information
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/branches/%s", owner, repo, branch)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", goerr.Wrap(err, "failed to create request for branch information")
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", goerr.Wrap(err, "failed to get branch information",
			goerr.V("owner", owner),
			goerr.V("repo", repo),
			goerr.V("branch", branch),
		)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", goerr.Wrap(types.ErrInvalidGitHubData, "failed to get branch information",
			goerr.V("owner", owner),
			goerr.V("repo", repo),
			goerr.V("branch", branch),
			goerr.V("status", resp.StatusCode),
			goerr.V("body", string(body)),
		)
	}

	// Parse response
	var branchInfo struct {
		Commit struct {
			SHA string `json:"sha"`
		} `json:"commit"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&branchInfo); err != nil {
		return "", goerr.Wrap(err, "failed to parse branch information")
	}

	logging.From(ctx).Info("Resolved commit ID from branch",
		"branch", branch,
		"commit_id", branchInfo.Commit.SHA,
	)

	return branchInfo.Commit.SHA, nil
}

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

	scanID, err := x.InsertScanResult(ctx, meta, *report)
	if err != nil {
		return err
	}
	logging.From(ctx).Info("scan result inserted", "scan_id", scanID)

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

	report, err := LoadTrivyReportFromFile(ctx, tmpResult.Name())
	if err != nil {
		return nil, err
	}

	logging.From(ctx).Debug("Scan result saved", "result_file", tmpResult.Name())

	return report, nil
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
