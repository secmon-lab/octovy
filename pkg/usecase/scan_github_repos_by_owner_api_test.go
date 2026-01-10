package usecase_test

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"cloud.google.com/go/bigquery"
	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/octovy/pkg/domain/interfaces"
	"github.com/m-mizutani/octovy/pkg/domain/mock"
	"github.com/m-mizutani/octovy/pkg/domain/model"
	"github.com/m-mizutani/octovy/pkg/domain/types"
	"github.com/m-mizutani/octovy/pkg/infra"
	"github.com/m-mizutani/octovy/pkg/usecase"
)

func TestScanGitHubReposByOwnerFromAPI_NoGitHubApp(t *testing.T) {
	ctx := context.Background()

	// Create usecase without GitHub App
	clients := infra.New()
	uc := usecase.New(clients)

	input := &model.ScanGitHubReposByOwnerFromAPIInput{
		Owner: "test-owner",
	}

	err := uc.ScanGitHubReposByOwnerFromAPI(ctx, input)
	gt.Error(t, err)
	gt.S(t, err.Error()).Contains("GitHub App is required for --all mode")
}

func TestScanGitHubReposByOwnerFromAPI_GetInstallationID(t *testing.T) {
	ctx := context.Background()

	mockGH := &mock.GitHubAppMock{}
	var capturedOwner string

	mockGH.GetInstallationIDForOwnerFunc = func(ctx context.Context, owner string) (types.GitHubAppInstallID, error) {
		capturedOwner = owner
		return types.GitHubAppInstallID(12345), nil
	}

	mockGH.ListInstallationReposFunc = func(ctx context.Context, installID types.GitHubAppInstallID) ([]*model.GitHubAPIRepository, error) {
		// Return empty list to complete successfully
		return []*model.GitHubAPIRepository{}, nil
	}

	clients := infra.New(
		infra.WithGitHubApp(mockGH),
	)
	uc := usecase.New(clients)

	input := &model.ScanGitHubReposByOwnerFromAPIInput{
		Owner: "test-owner",
		// InstallID not provided, should be fetched
	}

	err := uc.ScanGitHubReposByOwnerFromAPI(ctx, input)
	gt.NoError(t, err)
	gt.V(t, capturedOwner).Equal("test-owner")
}

func TestScanGitHubReposByOwnerFromAPI_FilterRepositories(t *testing.T) {
	ctx := context.Background()

	mockGH := &mock.GitHubAppMock{}
	mockHTTP := &httpMock{}
	mockTrivy := &trivyMock{}
	mockBQ := &mock.BigQueryMock{}

	var scannedRepos []string

	mockGH.ListInstallationReposFunc = func(ctx context.Context, installID types.GitHubAppInstallID) ([]*model.GitHubAPIRepository, error) {
		return []*model.GitHubAPIRepository{
			// Valid repository (should be scanned)
			{
				Owner:         "test-owner",
				Name:          "valid-repo",
				DefaultBranch: "main",
				Archived:      false,
				Disabled:      false,
			},
			// Archived repository (should be skipped)
			{
				Owner:         "test-owner",
				Name:          "archived-repo",
				DefaultBranch: "main",
				Archived:      true,
				Disabled:      false,
			},
			// Disabled repository (should be skipped)
			{
				Owner:         "test-owner",
				Name:          "disabled-repo",
				DefaultBranch: "main",
				Archived:      false,
				Disabled:      true,
			},
			// Repository without default branch (should be skipped)
			{
				Owner:         "test-owner",
				Name:          "no-default-branch",
				DefaultBranch: "",
				Archived:      false,
				Disabled:      false,
			},
			// Repository with different owner (should be skipped)
			{
				Owner:         "other-owner",
				Name:          "other-repo",
				DefaultBranch: "main",
				Archived:      false,
				Disabled:      false,
			},
			// Another valid repository (should be scanned)
			{
				Owner:         "test-owner",
				Name:          "another-valid-repo",
				DefaultBranch: "develop",
				Archived:      false,
				Disabled:      false,
			},
		}, nil
	}

	mockGH.GetArchiveURLFunc = func(ctx context.Context, input *interfaces.GetArchiveURLInput) (*url.URL, error) {
		scannedRepos = append(scannedRepos, input.Repo)
		// Return error to stop early without actually downloading/scanning
		return nil, io.EOF
	}

	mockGH.HTTPClientFunc = func(installID types.GitHubAppInstallID) (*http.Client, error) {
		return &http.Client{Transport: &mockTransport{mockHTTP: mockHTTP}}, nil
	}

	mockHTTP.mockDo = func(req *http.Request) (*http.Response, error) {
		if strings.Contains(req.URL.Path, "/branches/") {
			branchResponse := `{"commit":{"sha":"abc123def456789012345678901234567890abcd"}}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(branchResponse)),
			}, nil
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(&emptyZipReader{}),
		}, nil
	}

	mockTrivy.mockRun = func(ctx context.Context, args []string) error {
		return nil
	}

	mockBQ.InsertFunc = func(ctx context.Context, schema bigquery.Schema, data any, opts ...interfaces.BigQueryInsertOption) error {
		return nil
	}

	clients := infra.New(
		infra.WithGitHubApp(mockGH),
		infra.WithHTTPClient(mockHTTP),
		infra.WithTrivy(mockTrivy),
		infra.WithBigQuery(mockBQ),
	)
	uc := usecase.New(clients)

	input := &model.ScanGitHubReposByOwnerFromAPIInput{
		Owner:     "test-owner",
		InstallID: types.GitHubAppInstallID(12345),
	}

	// Execute (will fail because GetArchiveURL returns error, but filtering is tested)
	err := uc.ScanGitHubReposByOwnerFromAPI(ctx, input)
	gt.Error(t, err) // Expected to fail due to mock returning io.EOF

	// Verify only valid repositories were attempted
	gt.V(t, len(scannedRepos)).Equal(2)
	gt.V(t, scannedRepos[0]).Equal("valid-repo")
	gt.V(t, scannedRepos[1]).Equal("another-valid-repo")
}

func TestScanGitHubReposByOwnerFromAPI_NoRepositories(t *testing.T) {
	ctx := context.Background()

	mockGH := &mock.GitHubAppMock{}

	mockGH.ListInstallationReposFunc = func(ctx context.Context, installID types.GitHubAppInstallID) ([]*model.GitHubAPIRepository, error) {
		return []*model.GitHubAPIRepository{}, nil
	}

	clients := infra.New(
		infra.WithGitHubApp(mockGH),
	)
	uc := usecase.New(clients)

	input := &model.ScanGitHubReposByOwnerFromAPIInput{
		Owner:     "test-owner",
		InstallID: types.GitHubAppInstallID(12345),
	}

	err := uc.ScanGitHubReposByOwnerFromAPI(ctx, input)
	gt.NoError(t, err) // Should complete successfully with no repos
}

func TestScanGitHubReposByOwnerFromAPI_PartialFailure(t *testing.T) {
	ctx := context.Background()

	mockGH := &mock.GitHubAppMock{}
	mockHTTP := &httpMock{}
	mockTrivy := &trivyMock{}
	mockBQ := &mock.BigQueryMock{}

	mockGH.ListInstallationReposFunc = func(ctx context.Context, installID types.GitHubAppInstallID) ([]*model.GitHubAPIRepository, error) {
		return []*model.GitHubAPIRepository{
			{
				Owner:         "test-owner",
				Name:          "repo-1",
				DefaultBranch: "main",
				Archived:      false,
				Disabled:      false,
			},
			{
				Owner:         "test-owner",
				Name:          "repo-2",
				DefaultBranch: "main",
				Archived:      false,
				Disabled:      false,
			},
		}, nil
	}

	callCount := 0
	mockGH.GetArchiveURLFunc = func(ctx context.Context, input *interfaces.GetArchiveURLInput) (*url.URL, error) {
		callCount++
		// First repo fails, second succeeds (but we return error for both to simplify test)
		return nil, io.EOF
	}

	mockGH.HTTPClientFunc = func(installID types.GitHubAppInstallID) (*http.Client, error) {
		return &http.Client{Transport: &mockTransport{mockHTTP: mockHTTP}}, nil
	}

	mockHTTP.mockDo = func(req *http.Request) (*http.Response, error) {
		if strings.Contains(req.URL.Path, "/branches/") {
			branchResponse := `{"commit":{"sha":"abc123def456789012345678901234567890abcd"}}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(branchResponse)),
			}, nil
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(&emptyZipReader{}),
		}, nil
	}

	mockTrivy.mockRun = func(ctx context.Context, args []string) error {
		return nil
	}

	mockBQ.InsertFunc = func(ctx context.Context, schema bigquery.Schema, data any, opts ...interfaces.BigQueryInsertOption) error {
		return nil
	}

	clients := infra.New(
		infra.WithGitHubApp(mockGH),
		infra.WithHTTPClient(mockHTTP),
		infra.WithTrivy(mockTrivy),
		infra.WithBigQuery(mockBQ),
	)
	uc := usecase.New(clients)

	input := &model.ScanGitHubReposByOwnerFromAPIInput{
		Owner:     "test-owner",
		InstallID: types.GitHubAppInstallID(12345),
	}

	err := uc.ScanGitHubReposByOwnerFromAPI(ctx, input)
	gt.Error(t, err)
	gt.S(t, err.Error()).Contains("some repositories failed to scan")

	// Both repos should have been attempted
	gt.V(t, callCount).Equal(2)
}

// Mock types (httpMock, mockTransport, trivyMock) are defined in scan_github_repo_test.go
