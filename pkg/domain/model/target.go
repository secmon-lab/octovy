package model

import (
	"crypto/sha256"
	"encoding/hex"
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

// ToTargetID converts a target string to a TargetID by SHA256 hashing
// This ensures safe Firestore document IDs even when target contains special characters
// (e.g., "alpine:3.14" -> SHA256 hash)
func ToTargetID(target string) types.TargetID {
	hash := sha256.Sum256([]byte(target))
	return types.TargetID(hex.EncodeToString(hash[:]))
}
