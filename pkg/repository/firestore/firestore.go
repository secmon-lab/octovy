package firestore

import (
	"context"

	"cloud.google.com/go/firestore"
	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/octovy/pkg/domain/interfaces"
)

// New creates a new Firestore-based repository
func New(ctx context.Context, projectID, databaseID string) (interfaces.ScanRepository, error) {
	var client *firestore.Client
	var err error

	if databaseID != "" {
		client, err = firestore.NewClientWithDatabase(ctx, projectID, databaseID)
	} else {
		client, err = firestore.NewClient(ctx, projectID)
	}

	if err != nil {
		return nil, goerr.Wrap(err, "failed to create Firestore client",
			goerr.V("projectID", projectID),
			goerr.V("databaseID", databaseID),
		)
	}

	return &scanRepository{
		client: client,
	}, nil
}
