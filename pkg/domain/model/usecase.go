package model

import (
	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/octovy/pkg/domain/types"
)

type ScanGitHubRepoInput struct {
	GitHubMetadata
	InstallID types.GitHubAppInstallID
}

func (x *ScanGitHubRepoInput) Validate() error {
	// Use ValidateBasic instead of Validate to allow RepoID to be 0
	// RepoID is optional for CLI-initiated scans and will be 0 in those cases
	if err := x.GitHubMetadata.ValidateBasic(); err != nil {
		return err
	}
	// Validate commit ID format
	if !ptnValidCommitID.MatchString(x.CommitID) {
		return goerr.Wrap(types.ErrValidationFailed, "invalid commit ID")
	}
	if x.InstallID == 0 {
		return goerr.Wrap(types.ErrInvalidOption, "install ID is empty")
	}

	return nil
}

type ScanGitHubRepoRemoteInput struct {
	Owner     string
	Repo      string
	Commit    string
	Branch    string
	InstallID types.GitHubAppInstallID
}
