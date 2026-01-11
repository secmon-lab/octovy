package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/google/go-github/v53/github"
	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/octovy/pkg/domain/interfaces"
	"github.com/m-mizutani/octovy/pkg/domain/model"
	"github.com/m-mizutani/octovy/pkg/domain/types"
	"github.com/m-mizutani/octovy/pkg/utils/logging"
)

// handleGitHubAppEventResult represents the result of validating and parsing a GitHub App event.
// If ScanInput is nil, no scan is required (either no scan needed or validation failed).
type handleGitHubAppEventResult struct {
	ScanInput *model.ScanGitHubRepoInput
}

// validateGitHubAppEvent validates and parses a GitHub App webhook event.
// It returns the scan input if a scan is required, or nil if no scan is needed.
// This function is synchronous and should be called before starting background processing.
func validateGitHubAppEvent(r *http.Request, key types.GitHubAppSecret) (*handleGitHubAppEventResult, error) {
	ctx := r.Context()
	payload, err := github.ValidatePayload(r, []byte(key))
	if err != nil {
		return nil, goerr.Wrap(err, "validating payload")
	}

	event, err := github.ParseWebHook(github.WebHookType(r), payload)
	if err != nil {
		return nil, goerr.Wrap(err, "parsing webhook")
	}

	logging.From(ctx).With(slog.Any("event", event)).Info("Received GitHub App event")

	scanInput := githubEventToScanInput(event)
	return &handleGitHubAppEventResult{ScanInput: scanInput}, nil
}

// runGitHubRepoScan executes the GitHub repository scan in the provided context.
// This function is designed to be called from a background goroutine.
func runGitHubRepoScan(ctx context.Context, uc interfaces.UseCase, scanInput *model.ScanGitHubRepoInput) {
	logger := logging.From(ctx).With(slog.Any("input", scanInput))
	logger.Info("Starting GitHub repository scan")

	if err := uc.ScanGitHubRepo(ctx, scanInput); err != nil {
		logger.Error("Background scan failed", slog.Any("error", err))
	} else {
		logger.Info("GitHub repository scan completed successfully")
	}
}

func refToBranch(v string) string {
	if ref := strings.SplitN(v, "/", 3); len(ref) == 3 && ref[0] == "refs" && ref[1] == "heads" {
		return ref[2]
	}
	return v
}

func githubEventToScanInput(event interface{}) *model.ScanGitHubRepoInput {
	switch ev := event.(type) {
	case *github.PushEvent:
		if ev.HeadCommit == nil || ev.HeadCommit.ID == nil {
			logging.Default().Warn("ignore push event without head commit", slog.Any("event", ev))
			return nil
		}

		return &model.ScanGitHubRepoInput{
			GitHubMetadata: model.GitHubMetadata{
				GitHubCommit: model.GitHubCommit{
					GitHubRepo: model.GitHubRepo{
						RepoID:   ev.GetRepo().GetID(),
						Owner:    ev.GetRepo().GetOwner().GetLogin(),
						RepoName: ev.GetRepo().GetName(),
					},
					CommitID: ev.GetHeadCommit().GetID(),
					Branch:   refToBranch(ev.GetRef()),
					Ref:      ev.GetRef(),
					Committer: model.GitHubUser{
						Login: ev.GetHeadCommit().GetCommitter().GetLogin(),
						Email: ev.GetHeadCommit().GetCommitter().GetEmail(),
					},
				},
				DefaultBranch:  ev.GetRepo().GetDefaultBranch(),
				InstallationID: ev.GetInstallation().GetID(),
			},
			InstallID: types.GitHubAppInstallID(ev.GetInstallation().GetID()),
		}

	case *github.PullRequestEvent:
		if ev.GetAction() != "opened" && ev.GetAction() != "synchronize" {
			logging.Default().Debug("ignore PR event", slog.String("action", ev.GetAction()))
			return nil
		}
		if ev.GetPullRequest().GetDraft() {
			logging.Default().Debug("ignore draft PR", slog.String("action", ev.GetAction()))
			return nil
		}

		pr := ev.GetPullRequest()

		input := &model.ScanGitHubRepoInput{
			GitHubMetadata: model.GitHubMetadata{
				GitHubCommit: model.GitHubCommit{
					GitHubRepo: model.GitHubRepo{
						RepoID:   ev.GetRepo().GetID(),
						Owner:    ev.GetRepo().GetOwner().GetLogin(),
						RepoName: ev.GetRepo().GetName(),
					},
					CommitID: pr.GetHead().GetSHA(),
					Ref:      pr.GetHead().GetRef(),
					Branch:   pr.GetHead().GetRef(),
					Committer: model.GitHubUser{
						ID:    pr.GetHead().GetUser().GetID(),
						Login: pr.GetHead().GetUser().GetLogin(),
						Email: pr.GetHead().GetUser().GetEmail(),
					},
				},
				DefaultBranch:  ev.GetRepo().GetDefaultBranch(),
				InstallationID: ev.GetInstallation().GetID(),
				PullRequest: &model.GitHubPullRequest{
					ID:           pr.GetID(),
					Number:       pr.GetNumber(),
					BaseBranch:   pr.GetBase().GetRef(),
					BaseCommitID: pr.GetBase().GetSHA(),
					User: model.GitHubUser{
						ID:    pr.GetBase().GetUser().GetID(),
						Login: pr.GetBase().GetUser().GetLogin(),
						Email: pr.GetBase().GetUser().GetEmail(),
					},
				},
			},
			InstallID: types.GitHubAppInstallID(ev.GetInstallation().GetID()),
		}

		return input

	case *github.InstallationEvent, *github.InstallationRepositoriesEvent:
		return nil // ignore

	default:
		logging.Default().Warn("unsupported event", slog.Any("event", fmt.Sprintf("%T", event)))
		return nil
	}
}

func handleGitHubActionEvent(_ interfaces.UseCase, _ *http.Request) error {
	return nil
}

// Test helpers - exported for testing
func RefToBranchForTest(v string) string {
	return refToBranch(v)
}

func GithubEventToScanInputForTest(event interface{}) *model.ScanGitHubRepoInput {
	return githubEventToScanInput(event)
}
