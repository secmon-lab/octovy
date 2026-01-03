package model

import (
	"time"

	"github.com/m-mizutani/octovy/pkg/domain/types"
)

// Target represents a scan target (Trivy's Result)
type Target struct {
	ID        types.TargetID
	Target    string
	Class     string
	Type      string
	CreatedAt time.Time
	UpdatedAt time.Time
}
