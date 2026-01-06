package usecase

import (
	"context"
	"log/slog"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/octovy/pkg/domain/model"
	"github.com/m-mizutani/octovy/pkg/domain/types"
	"github.com/m-mizutani/octovy/pkg/utils/logging"
)

// ScanGitHubReposByOwner scans all repositories owned by the specified owner.
// It retrieves repositories from Firestore and scans only those that have both
// DefaultBranch and InstallationID configured.
func (x *UseCase) ScanGitHubReposByOwner(ctx context.Context, input *model.ScanGitHubReposByOwnerInput) error {
	// Validate Firestore is configured
	if x.clients.ScanRepository() == nil {
		return goerr.Wrap(types.ErrInvalidOption,
			"owner-only mode requires Firestore. Please configure Firestore or specify both owner and repo")
	}

	logger := logging.From(ctx)
	logger.Info("Starting owner-only scan mode",
		slog.String("owner", input.Owner),
	)

	// Get all repositories for the owner
	repos, err := x.clients.ScanRepository().ListRepositoriesByOwner(ctx, input.Owner)
	if err != nil {
		return goerr.Wrap(err, "failed to list repositories by owner",
			goerr.V("owner", input.Owner),
		)
	}

	logger.Info("Retrieved repositories from Firestore",
		slog.String("owner", input.Owner),
		slog.Int("total_repos", len(repos)),
	)

	// Filter repositories that have both DefaultBranch and InstallationID
	var validRepos []*model.Repository
	for _, repo := range repos {
		if repo.DefaultBranch != "" && repo.InstallationID != 0 {
			validRepos = append(validRepos, repo)
		} else {
			logger.Debug("Skipping repository due to missing metadata",
				slog.String("owner", repo.Owner),
				slog.String("repo", repo.Name),
				slog.String("default_branch", string(repo.DefaultBranch)),
				slog.Int64("installation_id", repo.InstallationID),
			)
		}
	}

	logger.Info("Filtered repositories with required metadata",
		slog.String("owner", input.Owner),
		slog.Int("valid_repos", len(validRepos)),
		slog.Int("skipped_repos", len(repos)-len(validRepos)),
	)

	if len(validRepos) == 0 {
		logger.Warn("No repositories to scan",
			slog.String("owner", input.Owner),
		)
		return nil
	}

	// Scan each repository
	var successCount, failureCount int
	for i, repo := range validRepos {
		logger.Info("Scanning repository",
			slog.Int("progress", i+1),
			slog.Int("total", len(validRepos)),
			slog.String("owner", repo.Owner),
			slog.String("repo", repo.Name),
			slog.String("default_branch", string(repo.DefaultBranch)),
		)

		// Prepare scan input for this repository
		scanInput := &model.ScanGitHubRepoRemoteInput{
			Owner:     repo.Owner,
			Repo:      repo.Name,
			Branch:    string(repo.DefaultBranch),
			InstallID: 0, // Will be retrieved from DB completion mode
		}

		// Scan the repository
		if err := x.ScanGitHubRepoRemote(ctx, scanInput); err != nil {
			failureCount++
			logger.Warn("Failed to scan repository",
				slog.String("owner", repo.Owner),
				slog.String("repo", repo.Name),
				slog.String("error", err.Error()),
			)
			continue
		}

		successCount++
		logger.Info("Successfully scanned repository",
			slog.String("owner", repo.Owner),
			slog.String("repo", repo.Name),
		)
	}

	// Log summary
	logger.Info("Completed owner-only scan mode",
		slog.String("owner", input.Owner),
		slog.Int("total_repos", len(validRepos)),
		slog.Int("success", successCount),
		slog.Int("failure", failureCount),
	)

	if failureCount > 0 {
		return goerr.New("some repositories failed to scan",
			goerr.V("owner", input.Owner),
			goerr.V("success_count", successCount),
			goerr.V("failure_count", failureCount),
		)
	}

	return nil
}
