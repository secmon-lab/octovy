package model

import (
	"time"

	"github.com/m-mizutani/octovy/pkg/domain/types"
)

// Repository represents a GitHub repository
type Repository struct {
	ID             types.GitHubRepoID
	Owner          string
	Name           string
	DefaultBranch  types.BranchName
	InstallationID int64
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
