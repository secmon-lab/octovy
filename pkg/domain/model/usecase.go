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
	if err := x.GitHubMetadata.Validate(); err != nil {
		return err
	}
	if x.InstallID == 0 {
		return goerr.Wrap(types.ErrInvalidOption, "install ID is empty")
	}

	return nil
}
