package usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/m-mizutani/gt"
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

	// Create test repositories
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
		LastCommitSHA: "abc123def456",
		Status:        types.ScanStatusSuccess,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	gt.NoError(t, repo.CreateOrUpdateBranch(ctx, validRepo.ID, branch))

	// Test that ListRepositoriesByOwner correctly filters by owner
	repos, err := repo.ListRepositoriesByOwner(ctx, "test-owner")
	gt.NoError(t, err)
	gt.V(t, len(repos)).Equal(3) // Should return 3 repos for "test-owner"

	// Verify filtering logic: count repos with both DefaultBranch and InstallationID
	var validCount int
	for _, r := range repos {
		if r.DefaultBranch != "" && r.InstallationID != 0 {
			validCount++
		}
	}
	gt.V(t, validCount).Equal(1) // Only validRepo should pass the filter

	// Verify "other-owner" repo is not included
	otherRepos, err := repo.ListRepositoriesByOwner(ctx, "other-owner")
	gt.NoError(t, err)
	gt.V(t, len(otherRepos)).Equal(1)
	gt.V(t, otherRepos[0].Name).Equal("repo")
}
