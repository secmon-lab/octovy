package config

import (
	"context"
	"log/slog"

	"github.com/m-mizutani/octovy/pkg/domain/interfaces"
	"github.com/m-mizutani/octovy/pkg/repository/firestore"
	"github.com/urfave/cli/v3"
)

type Firestore struct {
	projectID  string
	databaseID string
}

func (x *Firestore) Flags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{
			Name:        "firestore-project-id",
			Usage:       "Firestore project ID (optional)",
			Sources:     cli.EnvVars("OCTOVY_FIRESTORE_PROJECT_ID"),
			Destination: &x.projectID,
		},
		&cli.StringFlag{
			Name:        "firestore-database-id",
			Usage:       "Firestore database ID",
			Sources:     cli.EnvVars("OCTOVY_FIRESTORE_DATABASE_ID"),
			Value:       "(default)",
			Destination: &x.databaseID,
		},
	}
}

func (x *Firestore) Enabled() bool {
	return x.projectID != ""
}

func (x *Firestore) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Any("projectID", x.projectID),
		slog.Any("databaseID", x.databaseID),
	)
}

func (x *Firestore) NewRepository(ctx context.Context) (interfaces.ScanRepository, error) {
	return firestore.New(ctx, x.projectID, x.databaseID)
}
