package usecase

import (
	"github.com/m-mizutani/octovy/pkg/infra"
)

type UseCase struct {
	clients *infra.Clients
}

func New(clients *infra.Clients) *UseCase {
	return &UseCase{
		clients: clients,
	}
}
