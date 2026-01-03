package server_test

import (
	"testing"

	"github.com/google/go-github/v53/github"
	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/octovy/pkg/controller/server"
)

func TestRefToBranch(t *testing.T) {
	t.Run("strips refs/heads/ prefix", func(t *testing.T) {
		result := server.RefToBranchForTest("refs/heads/main")
		gt.V(t, result).Equal("main")
	})

	t.Run("handles nested branch names", func(t *testing.T) {
		result := server.RefToBranchForTest("refs/heads/feature/my-branch")
		gt.V(t, result).Equal("feature/my-branch")
	})

	t.Run("returns original if not refs/heads", func(t *testing.T) {
		result := server.RefToBranchForTest("refs/tags/v1.0.0")
		gt.V(t, result).Equal("refs/tags/v1.0.0")
	})

	t.Run("handles plain branch name", func(t *testing.T) {
		result := server.RefToBranchForTest("main")
		gt.V(t, result).Equal("main")
	})
}

func TestGitHubEventToScanInput(t *testing.T) {
	t.Run("push event without HeadCommit returns nil", func(t *testing.T) {
		event := &github.PushEvent{
			HeadCommit: nil,
		}
		result := server.GithubEventToScanInputForTest(event)
		gt.V(t, result).Equal(nil)
	})

	t.Run("push event with HeadCommit returns ScanInput", func(t *testing.T) {
		commitID := "abc123"
		ref := "refs/heads/main"
		repoID := int64(123)
		owner := "owner"
		repoName := "repo"
		installID := int64(456)

		event := &github.PushEvent{
			HeadCommit: &github.HeadCommit{
				ID: &commitID,
				Committer: &github.CommitAuthor{},
			},
			Ref: &ref,
			Repo: &github.PushEventRepository{
				ID: &repoID,
				Owner: &github.User{
					Login: &owner,
				},
				Name: &repoName,
			},
			Installation: &github.Installation{
				ID: &installID,
			},
		}

		result := server.GithubEventToScanInputForTest(event)
		gt.V(t, result.CommitID).Equal(commitID)
		gt.V(t, result.Branch).Equal("main")
		gt.V(t, result.Owner).Equal(owner)
		gt.V(t, result.RepoName).Equal(repoName)
	})

	t.Run("pull_request event with action other than opened/synchronize returns nil", func(t *testing.T) {
		action := "closed"
		event := &github.PullRequestEvent{
			Action: &action,
		}
		result := server.GithubEventToScanInputForTest(event)
		gt.V(t, result).Equal(nil)
	})

	t.Run("draft pull_request returns nil", func(t *testing.T) {
		action := "opened"
		draft := true
		event := &github.PullRequestEvent{
			Action: &action,
			PullRequest: &github.PullRequest{
				Draft: &draft,
			},
		}
		result := server.GithubEventToScanInputForTest(event)
		gt.V(t, result).Equal(nil)
	})

	t.Run("valid pull_request event returns ScanInput", func(t *testing.T) {
		action := "opened"
		draft := false
		sha := "def456"
		ref := "feature"
		prNumber := 42
		repoID := int64(789)
		owner := "owner"
		repoName := "repo"
		installID := int64(999)

		event := &github.PullRequestEvent{
			Action: &action,
			PullRequest: &github.PullRequest{
				Draft:  &draft,
				Number: &prNumber,
				Head: &github.PullRequestBranch{
					SHA: &sha,
					Ref: &ref,
					User: &github.User{},
				},
				Base: &github.PullRequestBranch{
					Ref: &ref,
					SHA: &sha,
					User: &github.User{},
				},
			},
			Repo: &github.Repository{
				ID: &repoID,
				Owner: &github.User{
					Login: &owner,
				},
				Name: &repoName,
			},
			Installation: &github.Installation{
				ID: &installID,
			},
		}

		result := server.GithubEventToScanInputForTest(event)
		gt.V(t, result.CommitID).Equal(sha)
		gt.V(t, result.Branch).Equal(ref)
		gt.V(t, result.PullRequest.Number).Equal(prNumber)
	})

	t.Run("installation event returns nil", func(t *testing.T) {
		event := &github.InstallationEvent{}
		result := server.GithubEventToScanInputForTest(event)
		gt.V(t, result).Equal(nil)
	})

	t.Run("installation repositories event returns nil", func(t *testing.T) {
		event := &github.InstallationRepositoriesEvent{}
		result := server.GithubEventToScanInputForTest(event)
		gt.V(t, result).Equal(nil)
	})

	t.Run("unsupported event returns nil", func(t *testing.T) {
		event := &github.StarEvent{}
		result := server.GithubEventToScanInputForTest(event)
		gt.V(t, result).Equal(nil)
	})
}