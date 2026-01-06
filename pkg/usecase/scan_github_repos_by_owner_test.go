package usecase_test

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/bigquery"
	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/octovy/pkg/domain/interfaces"
	"github.com/m-mizutani/octovy/pkg/domain/mock"
	"github.com/m-mizutani/octovy/pkg/domain/model"
	"github.com/m-mizutani/octovy/pkg/domain/types"
	"github.com/m-mizutani/octovy/pkg/infra"
	"github.com/m-mizutani/octovy/pkg/repository/memory"
	"github.com/m-mizutani/octovy/pkg/usecase"
)

func TestScanGitHubReposByOwner_NoFirestore(t *testing.T) {
	ctx := context.Background()

	// Create usecase without Firestore
	clients := infra.New()
	uc := usecase.New(clients)

	input := &model.ScanGitHubReposByOwnerInput{
		Owner: "test-owner",
	}

	err := uc.ScanGitHubReposByOwner(ctx, input)
	gt.Error(t, err)
	gt.S(t, err.Error()).Contains("owner-only mode requires Firestore")
}

func TestScanGitHubReposByOwner_NoRepositories(t *testing.T) {
	ctx := context.Background()

	// Create usecase with Firestore
	repo := memory.New()
	clients := infra.New(infra.WithScanRepository(repo))
	uc := usecase.New(clients)

	input := &model.ScanGitHubReposByOwnerInput{
		Owner: "test-owner",
	}

	err := uc.ScanGitHubReposByOwner(ctx, input)
	gt.NoError(t, err)
}

func TestScanGitHubReposByOwner_FilterRepositories(t *testing.T) {
	ctx := context.Background()

	// Create test repositories in memory
	repo := memory.New()
	now := time.Now()

	// Repository with both DefaultBranch and InstallationID (should be scanned)
	validRepo := &model.Repository{
		ID:             "test-owner/valid-repo",
		Owner:          "test-owner",
		Name:           "valid-repo",
		DefaultBranch:  "main",
		InstallationID: 12345,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	gt.NoError(t, repo.CreateOrUpdateRepository(ctx, validRepo))

	// Repository without DefaultBranch (should be skipped)
	noDefaultBranch := &model.Repository{
		ID:             "test-owner/no-default-branch",
		Owner:          "test-owner",
		Name:           "no-default-branch",
		DefaultBranch:  "",
		InstallationID: 12345,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	gt.NoError(t, repo.CreateOrUpdateRepository(ctx, noDefaultBranch))

	// Repository without InstallationID (should be skipped)
	noInstallID := &model.Repository{
		ID:             "test-owner/no-install-id",
		Owner:          "test-owner",
		Name:           "no-install-id",
		DefaultBranch:  "main",
		InstallationID: 0,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	gt.NoError(t, repo.CreateOrUpdateRepository(ctx, noInstallID))

	// Repository with different owner (should not be retrieved)
	differentOwner := &model.Repository{
		ID:             "other-owner/repo",
		Owner:          "other-owner",
		Name:           "repo",
		DefaultBranch:  "main",
		InstallationID: 12345,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	gt.NoError(t, repo.CreateOrUpdateRepository(ctx, differentOwner))

	// Create branch for valid repository to enable DB completion mode
	branch := &model.Branch{
		Name:          "main",
		LastScanID:    "scan-123",
		LastScanAt:    now,
		LastCommitSHA: "abc123def456789012345678901234567890abcd",
		Status:        types.ScanStatusSuccess,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	gt.NoError(t, repo.CreateOrUpdateBranch(ctx, validRepo.ID, branch))

	// Setup mocks to track which repositories are scanned
	mockGH := &mock.GitHubAppMock{}
	mockHTTP := &httpMock{}
	mockTrivy := &trivyMock{}
	mockBQ := &mock.BigQueryMock{}

	var scannedRepos []string

	mockGH.GetArchiveURLFunc = func(ctx context.Context, input *interfaces.GetArchiveURLInput) (*url.URL, error) {
		// Track which repo was scanned (this is what we're testing)
		scannedRepos = append(scannedRepos, input.Repo)
		// Return error to stop early without actually downloading/scanning
		return nil, io.EOF
	}

	mockGH.HTTPClientFunc = func(installID types.GitHubAppInstallID) (*http.Client, error) {
		return &http.Client{Transport: &mockTransport{mockHTTP: mockHTTP}}, nil
	}

	mockHTTP.mockDo = func(req *http.Request) (*http.Response, error) {
		// Mock GitHub API branch endpoint for resolving branch to commit
		if strings.Contains(req.URL.Path, "/branches/") {
			branchResponse := `{"commit":{"sha":"abc123def456789012345678901234567890abcd"}}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(branchResponse)),
			}, nil
		}
		// Mock archive download
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

	// Create usecase with mocks
	clients := infra.New(
		infra.WithScanRepository(repo),
		infra.WithGitHubApp(mockGH),
		infra.WithHTTPClient(mockHTTP),
		infra.WithTrivy(mockTrivy),
		infra.WithBigQuery(mockBQ),
	)
	uc := usecase.New(clients)

	input := &model.ScanGitHubReposByOwnerInput{
		Owner: "test-owner",
	}

	// Execute the usecase (will fail due to mockGH.GetArchiveURLFunc returning error, but that's fine)
	err := uc.ScanGitHubReposByOwner(ctx, input)
	gt.Error(t, err) // Expected to fail, but we can verify filtering worked

	// Verify only valid-repo was attempted to be scanned
	// This proves the filtering logic worked correctly:
	// - Only repos with DefaultBranch and InstallationID were processed
	// - Repos without DefaultBranch or InstallationID were skipped
	// - Repos with different owner were not retrieved
	gt.V(t, len(scannedRepos)).Equal(1)
	gt.V(t, scannedRepos[0]).Equal("valid-repo")
}

// emptyZipReader returns empty data to avoid actual zip extraction
type emptyZipReader struct{}

func (r *emptyZipReader) Read(p []byte) (int, error) {
	return 0, io.EOF
}
