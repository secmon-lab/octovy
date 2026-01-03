package infra

import (
	"net/http"

	"github.com/m-mizutani/octovy/pkg/domain/interfaces"
	"github.com/m-mizutani/octovy/pkg/infra/trivy"
)

type Clients struct {
	githubApp      interfaces.GitHubApp
	httpClient     HTTPClient
	trivyClient    trivy.Client
	bqClient       interfaces.BigQuery
	scanRepository interfaces.ScanRepository
}

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type Option func(*Clients)

func New(options ...Option) *Clients {
	client := &Clients{
		httpClient:  http.DefaultClient,
		trivyClient: trivy.New("trivy"),
	}

	for _, opt := range options {
		opt(client)
	}

	return client
}

func (x *Clients) GitHubApp() interfaces.GitHubApp {
	return x.githubApp
}
func (x *Clients) HTTPClient() HTTPClient {
	return x.httpClient
}
func (x *Clients) Trivy() trivy.Client {
	return x.trivyClient
}
func (x *Clients) BigQuery() interfaces.BigQuery {
	return x.bqClient
}
func (x *Clients) ScanRepository() interfaces.ScanRepository {
	return x.scanRepository
}

func WithGitHubApp(client interfaces.GitHubApp) Option {
	return func(x *Clients) {
		x.githubApp = client
	}
}

func WithHTTPClient(client HTTPClient) Option {
	return func(x *Clients) {
		x.httpClient = client
	}
}

func WithTrivy(client trivy.Client) Option {
	return func(x *Clients) {
		x.trivyClient = client
	}
}

func WithBigQuery(client interfaces.BigQuery) Option {
	return func(x *Clients) {
		x.bqClient = client
	}
}

func WithScanRepository(repo interfaces.ScanRepository) Option {
	return func(x *Clients) {
		x.scanRepository = repo
	}
}
