package memory

import "github.com/m-mizutani/octovy/pkg/domain/interfaces"

// New creates a new in-memory repository
func New() interfaces.ScanRepository {
	return &scanRepository{
		repos:   make(map[string]*repoData),
	}
}
