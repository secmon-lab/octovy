package infra_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/octovy/pkg/domain/mock"
	"github.com/m-mizutani/octovy/pkg/infra"
	"github.com/m-mizutani/octovy/pkg/infra/trivy"
)

func TestNew(t *testing.T) {
	t.Run("create new clients without options", func(t *testing.T) {
		clients := infra.New()
		// HTTPClient should return the default http.DefaultClient
		gt.V(t, clients.HTTPClient()).Equal(http.DefaultClient)
		// Trivy should return a default trivy client (implementation detail)
		trivyClient := clients.Trivy()
		// Verify it's the same instance when called again
		gt.V(t, clients.Trivy()).Equal(trivyClient)
		// GitHub and BigQuery should be nil without configuration
		gt.V(t, clients.GitHubApp()).Equal(nil)
		gt.V(t, clients.BigQuery()).Equal(nil)
	})

	t.Run("WithGitHubApp option sets GitHub App client", func(t *testing.T) {
		mockGH := &mock.GitHubAppMock{}
		clients := infra.New(infra.WithGitHubApp(mockGH))
		gt.V(t, clients.GitHubApp()).Equal(mockGH)
	})

	t.Run("WithHTTPClient option sets HTTP client", func(t *testing.T) {
		mockHTTP := &mockHTTPClient{}
		clients := infra.New(infra.WithHTTPClient(mockHTTP))
		gt.V(t, clients.HTTPClient()).Equal(mockHTTP)
	})

	t.Run("WithTrivy option sets Trivy client", func(t *testing.T) {
		mockTrivy := &mockTrivyClient{}
		clients := infra.New(infra.WithTrivy(mockTrivy))
		gt.V(t, clients.Trivy()).Equal(mockTrivy)
	})

	t.Run("WithBigQuery option sets BigQuery client", func(t *testing.T) {
		mockBQ := &mock.BigQueryMock{}
		clients := infra.New(infra.WithBigQuery(mockBQ))
		gt.V(t, clients.BigQuery()).Equal(mockBQ)
	})

	t.Run("multiple options can be combined", func(t *testing.T) {
		mockGH := &mock.GitHubAppMock{}
		mockBQ := &mock.BigQueryMock{}
		mockHTTP := &mockHTTPClient{}

		clients := infra.New(
			infra.WithGitHubApp(mockGH),
			infra.WithBigQuery(mockBQ),
			infra.WithHTTPClient(mockHTTP),
		)

		gt.V(t, clients.GitHubApp()).Equal(mockGH)
		gt.V(t, clients.BigQuery()).Equal(mockBQ)
		gt.V(t, clients.HTTPClient()).Equal(mockHTTP)
	})
}

type mockHTTPClient struct{}

func (m *mockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return nil, nil
}

type mockTrivyClient struct{}

func (m *mockTrivyClient) Run(ctx context.Context, args []string) error {
	return nil
}

var _ trivy.Client = (*mockTrivyClient)(nil)
