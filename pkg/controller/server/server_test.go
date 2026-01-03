package server_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/octovy/pkg/controller/server"
	"github.com/m-mizutani/octovy/pkg/domain/mock"
	"github.com/m-mizutani/octovy/pkg/domain/model"
	"github.com/m-mizutani/octovy/pkg/domain/types"
	"github.com/m-mizutani/octovy/pkg/infra"
	"github.com/m-mizutani/octovy/pkg/usecase"
)

func TestServerConfiguration(t *testing.T) {
	t.Run("server accepts GitHub secret configuration", func(t *testing.T) {
		clients := infra.New()
		uc := usecase.New(clients)
		expectedSecret := types.GitHubAppSecret("test-secret-12345")

		// Create server with secret - actual usage is tested in webhook tests
		srv := server.New(uc, server.WithGitHubSecret(expectedSecret))

		// Test that server can handle requests (compile-time check)
		_ = srv.Mux()
	})
}

func TestRouterSmokeTests(t *testing.T) {
	t.Run("GET /health returns 200", func(t *testing.T) {
		clients := infra.New()
		uc := usecase.New(clients)
		srv := server.New(uc)

		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		rec := httptest.NewRecorder()

		srv.Mux().ServeHTTP(rec, req)

		gt.V(t, rec.Code).Equal(http.StatusOK)
		gt.V(t, rec.Body.String()).Equal("ok")
	})

	t.Run("POST /webhook/github/app with mock UseCase", func(t *testing.T) {
		mockUC := &mock.UseCaseMock{
			ScanGitHubRepoFunc: func(ctx context.Context, input *model.ScanGitHubRepoInput) error {
				return nil
			},
		}
		srv := server.New(mockUC, server.WithGitHubSecret(types.GitHubAppSecret("test-secret")))

		body := []byte(`{"action":"push"}`)
		req := httptest.NewRequest(http.MethodPost, "/webhook/github/app", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-GitHub-Event", "push")
		rec := httptest.NewRecorder()

		srv.Mux().ServeHTTP(rec, req)

		// Without proper signature, it should fail
		gt.V(t, rec.Code).Equal(http.StatusInternalServerError)
	})

	t.Run("POST /webhook/github/action", func(t *testing.T) {
		mockUC := &mock.UseCaseMock{
			ScanGitHubRepoFunc: func(ctx context.Context, input *model.ScanGitHubRepoInput) error {
				return nil
			},
		}
		srv := server.New(mockUC)

		body := []byte(`{"action":"test"}`)
		req := httptest.NewRequest(http.MethodPost, "/webhook/github/action", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		srv.Mux().ServeHTTP(rec, req)

		// Action handler should process the request successfully
		gt.V(t, rec.Code).Equal(http.StatusOK)
	})
}

func TestSafeWrite(t *testing.T) {
	// Note: safeWrite is not exported, but we can test it through the /health endpoint
	t.Run("writes response successfully", func(t *testing.T) {
		clients := infra.New()
		uc := usecase.New(clients)
		srv := server.New(uc)

		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		rec := httptest.NewRecorder()

		srv.Mux().ServeHTTP(rec, req)

		gt.V(t, rec.Code).Equal(http.StatusOK)
		gt.V(t, rec.Body.String()).Equal("ok")
	})
}
