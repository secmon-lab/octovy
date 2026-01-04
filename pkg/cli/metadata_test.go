package cli_test

import (
	"context"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/octovy/pkg/cli"
	"github.com/m-mizutani/octovy/pkg/domain/model"
)

func TestAutoDetectGitMetadata(t *testing.T) {
	ctx := context.Background()

	t.Run("auto-detect from current git repository", func(t *testing.T) {
		meta := model.GitHubMetadata{}
		err := cli.AutoDetectGitMetadata(ctx, &meta)

		if err != nil {
			t.Skipf("Not in a git repository: %v", err)
		}

		gt.V(t, meta.Owner).NotEqual("")
		gt.V(t, meta.RepoName).NotEqual("")
		gt.V(t, meta.CommitID).NotEqual("")
	})

	t.Run("preserve existing metadata", func(t *testing.T) {
		meta := model.GitHubMetadata{
			GitHubCommit: model.GitHubCommit{
				GitHubRepo: model.GitHubRepo{
					Owner:    "custom-owner",
					RepoName: "custom-repo",
				},
				CommitID: "custom-commit",
			},
		}
		err := cli.AutoDetectGitMetadata(ctx, &meta)

		if err != nil {
			t.Skipf("Not in a git repository: %v", err)
		}

		gt.V(t, meta.Owner).Equal("custom-owner")
		gt.V(t, meta.RepoName).Equal("custom-repo")
		gt.V(t, meta.CommitID).Equal("custom-commit")
	})
}
