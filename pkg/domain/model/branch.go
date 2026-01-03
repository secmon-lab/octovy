package model

import (
	"time"

	"github.com/m-mizutani/octovy/pkg/domain/types"
)

// Branch represents a branch in a GitHub repository
type Branch struct {
	Name          types.BranchName
	LastScanID    types.ScanID
	LastScanAt    time.Time
	LastCommitSHA types.CommitSHA
	Status        types.ScanStatus
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
