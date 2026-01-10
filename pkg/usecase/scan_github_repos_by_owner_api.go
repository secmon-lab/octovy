package usecase

import (
	"context"
	"log/slog"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/octovy/pkg/domain/model"
	"github.com/m-mizutani/octovy/pkg/domain/types"
	"github.com/m-mizutani/octovy/pkg/utils/logging"
)

// scanFailure represents a failed repository scan with its error details.
type scanFailure struct {
	Owner string
	Repo  string
	Error string
}

// ScanGitHubReposByOwnerFromAPI scans all repositories owned by the specified owner
// using GitHub App API to fetch the repository list (instead of Firestore).
// This is triggered by the --all flag in scan remote command.
func (x *UseCase) ScanGitHubReposByOwnerFromAPI(ctx context.Context, input *model.ScanGitHubReposByOwnerFromAPIInput) error {
	logger := logging.From(ctx)

	// Validate GitHub App is configured
	if x.clients.GitHubApp() == nil {
		return goerr.Wrap(types.ErrInvalidOption, "GitHub App is required for --all mode")
	}

	// Get installation ID if not provided
	installID := input.InstallID
	if installID == 0 {
		id, err := x.clients.GitHubApp().GetInstallationIDForOwner(ctx, input.Owner)
		if err != nil {
			return goerr.Wrap(err, "failed to get installation ID for owner",
				goerr.V("owner", input.Owner),
			)
		}
		installID = id
	}

	logger.Info("Starting scan with --all mode (GitHub API)",
		slog.String("owner", input.Owner),
		slog.Any("installID", installID),
	)

	// Get all repositories from GitHub API
	repos, err := x.clients.GitHubApp().ListInstallationRepos(ctx, installID)
	if err != nil {
		return goerr.Wrap(err, "failed to list installation repos",
			goerr.V("owner", input.Owner),
			goerr.V("installID", installID),
		)
	}

	logger.Info("Retrieved repositories from GitHub API",
		slog.String("owner", input.Owner),
		slog.Int("total_repos", len(repos)),
	)

	// Filter repositories by owner and exclude archived/disabled
	var validRepos []*model.GitHubAPIRepository
	for _, repo := range repos {
		// Filter by owner
		if repo.Owner != input.Owner {
			logger.Debug("Skipping repository due to different owner",
				slog.String("repo_owner", repo.Owner),
				slog.String("repo_name", repo.Name),
				slog.String("expected_owner", input.Owner),
			)
			continue
		}

		// Exclude archived and disabled repositories
		if repo.Archived {
			logger.Debug("Skipping archived repository",
				slog.String("owner", repo.Owner),
				slog.String("repo", repo.Name),
			)
			continue
		}
		if repo.Disabled {
			logger.Debug("Skipping disabled repository",
				slog.String("owner", repo.Owner),
				slog.String("repo", repo.Name),
			)
			continue
		}

		// Ensure default branch is set
		if repo.DefaultBranch == "" {
			logger.Debug("Skipping repository due to missing default branch",
				slog.String("owner", repo.Owner),
				slog.String("repo", repo.Name),
			)
			continue
		}

		validRepos = append(validRepos, repo)
	}

	logger.Info("Filtered repositories for scanning",
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
	var successCount int
	var failures []scanFailure

	for i, repo := range validRepos {
		logger.Info("Scanning repository",
			slog.Int("progress", i+1),
			slog.Int("total", len(validRepos)),
			slog.String("owner", repo.Owner),
			slog.String("repo", repo.Name),
			slog.String("default_branch", repo.DefaultBranch),
		)

		// Prepare scan input for this repository
		scanInput := &model.ScanGitHubRepoRemoteInput{
			Owner:     repo.Owner,
			Repo:      repo.Name,
			Branch:    repo.DefaultBranch,
			InstallID: installID,
		}

		// Scan the repository
		if err := x.ScanGitHubRepoRemote(ctx, scanInput); err != nil {
			failures = append(failures, scanFailure{
				Owner: repo.Owner,
				Repo:  repo.Name,
				Error: err.Error(),
			})
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

	// Log summary with failure details
	logger.Info("Completed scan with --all mode",
		slog.String("owner", input.Owner),
		slog.Int("total_repos", len(validRepos)),
		slog.Int("success", successCount),
		slog.Int("failure", len(failures)),
	)

	// Log detailed failure information
	for _, f := range failures {
		logger.Error("Repository scan failure details",
			slog.String("owner", f.Owner),
			slog.String("repo", f.Repo),
			slog.String("error", f.Error),
		)
	}

	if len(failures) > 0 {
		// Build failure summary for error message
		failedRepos := make([]string, len(failures))
		for i, f := range failures {
			failedRepos[i] = f.Owner + "/" + f.Repo
		}

		return goerr.New("some repositories failed to scan",
			goerr.V("owner", input.Owner),
			goerr.V("success_count", successCount),
			goerr.V("failure_count", len(failures)),
			goerr.V("failed_repos", failedRepos),
		)
	}

	return nil
}
