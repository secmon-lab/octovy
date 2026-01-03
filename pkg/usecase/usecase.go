package usecase

import (
	"github.com/m-mizutani/octovy/pkg/domain/types"
	"github.com/m-mizutani/octovy/pkg/infra"
)

type UseCase struct {
	tableID types.BQTableID
	clients *infra.Clients
}

func New(clients *infra.Clients, options ...Option) *UseCase {
	uc := &UseCase{
		tableID: "scans",
		clients: clients,
	}

	for _, opt := range options {
		opt(uc)
	}

	return uc
}

type Option func(*UseCase)

func WithBigQueryTableID(tableID types.BQTableID) Option {
	return func(x *UseCase) {
		x.tableID = tableID
	}
}
