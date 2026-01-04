package model

import (
	"regexp"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/octovy/pkg/domain/types"
)

type GitHubRepo struct {
	RepoID   int64  `json:"repo_id" bigquery:"repo_id"`
	Owner    string `json:"owner" bigquery:"owner"`
	RepoName string `json:"repo_name" bigquery:"repo_name"`
}

func (x *GitHubRepo) Validate() error {
	if x.RepoID == 0 {
		return goerr.Wrap(types.ErrValidationFailed, "repo ID is empty")
	}
	if x.Owner == "" {
		return goerr.Wrap(types.ErrValidationFailed, "owner is empty")
	}
	if x.RepoName == "" {
		return goerr.Wrap(types.ErrValidationFailed, "repo name is empty")
	}

	return nil
}

type GitHubCommit struct {
	GitHubRepo
	Committer GitHubUser `json:"committer" bigquery:"committer"`
	CommitID  string     `json:"commit_id" bigquery:"commit_id"`
	Branch    string     `json:"branch" bigquery:"branch"`
	Ref       string     `json:"ref" bigquery:"ref"`
}

type GitHubMetadata struct {
	GitHubCommit
	PullRequest    *GitHubPullRequest `json:"pull_request"`
	DefaultBranch  string             `json:"default_branch"`
	InstallationID int64              `json:"installation_id"`
}

// ValidateBasic validates that required GitHub metadata fields are not empty.
// Unlike GitHubCommit.Validate(), this does not validate CommitID format.
func (x *GitHubMetadata) ValidateBasic() error {
	if x.Owner == "" || x.RepoName == "" || x.CommitID == "" {
		return goerr.New("required GitHub metadata is missing",
			goerr.V("owner", x.Owner),
			goerr.V("repo", x.RepoName),
			goerr.V("commitID", x.CommitID),
		)
	}
	return nil
}

type GitHubPullRequest struct {
	ID           int64      `json:"id"`
	Number       int        `json:"number"`
	BaseBranch   string     `json:"base_branch"`
	BaseCommitID string     `json:"base_commit_id"`
	User         GitHubUser `json:"user"`
}

type GitHubUser struct {
	ID    int64  `json:"id"`
	Login string `json:"login"`
	Email string `json:"email"`
}

var (
	ptnValidCommitID = regexp.MustCompile("^[0-9a-f]{40}$")
)

func (x *GitHubCommit) Validate() error {
	if err := x.GitHubRepo.Validate(); err != nil {
		return err
	}

	if !ptnValidCommitID.MatchString(x.CommitID) {
		return goerr.Wrap(types.ErrValidationFailed, "invalid commit ID")
	}

	return nil
}

type GitHubIssueComment struct {
	ID          string
	Login       string
	Body        string
	IsMinimized bool
}
