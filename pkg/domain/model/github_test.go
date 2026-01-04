package model_test

import (
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/octovy/pkg/domain/model"
)

func TestGitHubRepoValidate(t *testing.T) {
	t.Run("valid repo passes validation", func(t *testing.T) {
		repo := &model.GitHubRepo{
			RepoID:   123,
			Owner:    "test-owner",
			RepoName: "test-repo",
		}
		gt.NoError(t, repo.Validate())
	})

	t.Run("missing repo ID fails validation", func(t *testing.T) {
		repo := &model.GitHubRepo{
			Owner:    "test-owner",
			RepoName: "test-repo",
		}
		gt.Error(t, repo.Validate())
	})

	t.Run("missing owner fails validation", func(t *testing.T) {
		repo := &model.GitHubRepo{
			RepoID:   123,
			RepoName: "test-repo",
		}
		gt.Error(t, repo.Validate())
	})

	t.Run("missing repo name fails validation", func(t *testing.T) {
		repo := &model.GitHubRepo{
			RepoID: 123,
			Owner:  "test-owner",
		}
		gt.Error(t, repo.Validate())
	})
}

func TestGitHubCommitValidate(t *testing.T) {
	validCommitID := "a" + "1234567890123456789012345678901234567890"[:39]

	t.Run("valid commit passes validation", func(t *testing.T) {
		commit := &model.GitHubCommit{
			GitHubRepo: model.GitHubRepo{
				RepoID:   123,
				Owner:    "test-owner",
				RepoName: "test-repo",
			},
			CommitID: validCommitID,
		}
		gt.NoError(t, commit.Validate())
	})

	t.Run("invalid commit hash format fails validation", func(t *testing.T) {
		commit := &model.GitHubCommit{
			GitHubRepo: model.GitHubRepo{
				RepoID:   123,
				Owner:    "test-owner",
				RepoName: "test-repo",
			},
			CommitID: "invalid-commit-id",
		}
		gt.Error(t, commit.Validate())
	})

	t.Run("short commit hash fails validation", func(t *testing.T) {
		commit := &model.GitHubCommit{
			GitHubRepo: model.GitHubRepo{
				RepoID:   123,
				Owner:    "test-owner",
				RepoName: "test-repo",
			},
			CommitID: "abc123",
		}
		gt.Error(t, commit.Validate())
	})

	t.Run("uppercase commit hash fails validation", func(t *testing.T) {
		commit := &model.GitHubCommit{
			GitHubRepo: model.GitHubRepo{
				RepoID:   123,
				Owner:    "test-owner",
				RepoName: "test-repo",
			},
			CommitID: "A234567890123456789012345678901234567890",
		}
		gt.Error(t, commit.Validate())
	})

	t.Run("commit with invalid repo fails validation", func(t *testing.T) {
		commit := &model.GitHubCommit{
			GitHubRepo: model.GitHubRepo{
				RepoID: 0, // Invalid
				Owner:  "test-owner",
			},
			CommitID: validCommitID,
		}
		gt.Error(t, commit.Validate())
	})
}

func TestGitHubMetadata(t *testing.T) {
	t.Run("metadata with pull request is valid", func(t *testing.T) {
		validCommitID := "a" + "1234567890123456789012345678901234567890"[:39]

		meta := &model.GitHubMetadata{
			GitHubCommit: model.GitHubCommit{
				GitHubRepo: model.GitHubRepo{
					RepoID:   123,
					Owner:    "test-owner",
					RepoName: "test-repo",
				},
				CommitID: validCommitID,
			},
			PullRequest: &model.GitHubPullRequest{
				ID:     456,
				Number: 789,
			},
			DefaultBranch: "main",
		}

		// Metadata doesn't have validation, but check structure is correct
		gt.V(t, meta.PullRequest.ID).Equal(int64(456))
		gt.V(t, meta.PullRequest.Number).Equal(789)
		gt.V(t, meta.DefaultBranch).Equal("main")
	})

	t.Run("metadata without pull request is valid", func(t *testing.T) {
		validCommitID := "a" + "1234567890123456789012345678901234567890"[:39]

		meta := &model.GitHubMetadata{
			GitHubCommit: model.GitHubCommit{
				GitHubRepo: model.GitHubRepo{
					RepoID:   123,
					Owner:    "test-owner",
					RepoName: "test-repo",
				},
				CommitID: validCommitID,
			},
			DefaultBranch: "main",
		}

		gt.V(t, meta.PullRequest).Equal((*model.GitHubPullRequest)(nil))
	})
}

func TestGitHubMetadataValidateBasic(t *testing.T) {
	t.Run("all required fields present", func(t *testing.T) {
		meta := &model.GitHubMetadata{
			GitHubCommit: model.GitHubCommit{
				GitHubRepo: model.GitHubRepo{
					Owner:    "secmon-lab",
					RepoName: "octovy",
				},
				CommitID: "abc123def456",
			},
		}
		gt.NoError(t, meta.ValidateBasic())
	})

	t.Run("owner missing", func(t *testing.T) {
		meta := &model.GitHubMetadata{
			GitHubCommit: model.GitHubCommit{
				GitHubRepo: model.GitHubRepo{
					Owner:    "",
					RepoName: "octovy",
				},
				CommitID: "abc123def456",
			},
		}
		gt.Error(t, meta.ValidateBasic())
	})

	t.Run("repo missing", func(t *testing.T) {
		meta := &model.GitHubMetadata{
			GitHubCommit: model.GitHubCommit{
				GitHubRepo: model.GitHubRepo{
					Owner:    "secmon-lab",
					RepoName: "",
				},
				CommitID: "abc123def456",
			},
		}
		gt.Error(t, meta.ValidateBasic())
	})

	t.Run("commitID missing", func(t *testing.T) {
		meta := &model.GitHubMetadata{
			GitHubCommit: model.GitHubCommit{
				GitHubRepo: model.GitHubRepo{
					Owner:    "secmon-lab",
					RepoName: "octovy",
				},
				CommitID: "",
			},
		}
		gt.Error(t, meta.ValidateBasic())
	})

	t.Run("all fields missing", func(t *testing.T) {
		meta := &model.GitHubMetadata{
			GitHubCommit: model.GitHubCommit{
				GitHubRepo: model.GitHubRepo{
					Owner:    "",
					RepoName: "",
				},
				CommitID: "",
			},
		}
		gt.Error(t, meta.ValidateBasic())
	})
}

func TestScanGitHubRepoInputValidate(t *testing.T) {
	validCommitID := "a" + "1234567890123456789012345678901234567890"[:39]

	t.Run("valid input passes validation", func(t *testing.T) {
		input := &model.ScanGitHubRepoInput{
			GitHubMetadata: model.GitHubMetadata{
				GitHubCommit: model.GitHubCommit{
					GitHubRepo: model.GitHubRepo{
						RepoID:   123,
						Owner:    "test-owner",
						RepoName: "test-repo",
					},
					CommitID: validCommitID,
				},
			},
			InstallID: 456,
		}
		gt.NoError(t, input.Validate())
	})

	t.Run("missing install ID fails validation", func(t *testing.T) {
		input := &model.ScanGitHubRepoInput{
			GitHubMetadata: model.GitHubMetadata{
				GitHubCommit: model.GitHubCommit{
					GitHubRepo: model.GitHubRepo{
						RepoID:   123,
						Owner:    "test-owner",
						RepoName: "test-repo",
					},
					CommitID: validCommitID,
				},
			},
			InstallID: 0,
		}
		gt.Error(t, input.Validate())
	})

	t.Run("invalid metadata fails validation", func(t *testing.T) {
		input := &model.ScanGitHubRepoInput{
			GitHubMetadata: model.GitHubMetadata{
				GitHubCommit: model.GitHubCommit{
					GitHubRepo: model.GitHubRepo{
						RepoID: 0, // Invalid
						Owner:  "test-owner",
					},
					CommitID: validCommitID,
				},
			},
			InstallID: 456,
		}
		gt.Error(t, input.Validate())
	})

	t.Run("invalid commit ID fails validation", func(t *testing.T) {
		input := &model.ScanGitHubRepoInput{
			GitHubMetadata: model.GitHubMetadata{
				GitHubCommit: model.GitHubCommit{
					GitHubRepo: model.GitHubRepo{
						RepoID:   123,
						Owner:    "test-owner",
						RepoName: "test-repo",
					},
					CommitID: "invalid",
				},
			},
			InstallID: 456,
		}
		gt.Error(t, input.Validate())
	})
}
