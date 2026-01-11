package server

import (
	"net/http"

	"log/slog"

	"github.com/go-chi/chi/v5"
	"github.com/m-mizutani/octovy/pkg/domain/interfaces"
	"github.com/m-mizutani/octovy/pkg/domain/types"
	"github.com/m-mizutani/octovy/pkg/utils/errutil"
	"github.com/m-mizutani/octovy/pkg/utils/logging"
)

type Server struct {
	mux *chi.Mux
}

func safeWrite(w http.ResponseWriter, code int, body []byte) {
	w.WriteHeader(code)

	// nosemgrep: go.lang.security.audit.xss.no-direct-write-to-responsewriter.no-direct-write-to-responsewriter
	// Why: The response data is not from user input
	if _, err := w.Write(body); err != nil {
		logging.Default().Error("fail to write response", slog.Any("error", err))
	}
}

type config struct {
	ghSecret types.GitHubAppSecret
}

type Option func(*config)

func WithGitHubSecret(secret types.GitHubAppSecret) Option {
	return func(cfg *config) {
		cfg.ghSecret = secret
	}
}

func New(uc interfaces.UseCase, options ...Option) *Server {
	cfg := &config{}
	for _, opt := range options {
		opt(cfg)
	}

	r := chi.NewRouter()
	r.Use(preProcess)
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		safeWrite(w, http.StatusOK, []byte("ok"))
	})
	r.Route("/webhook", func(r chi.Router) {
		r.Route("/github", func(r chi.Router) {
			r.Post("/app", func(w http.ResponseWriter, r *http.Request) {
				// Validate and parse the webhook event synchronously
				result, err := validateGitHubAppEvent(r, cfg.ghSecret)
				if err != nil {
					errutil.HandleError(r.Context(), "fail to validate GitHub App event", err)
					safeWrite(w, http.StatusInternalServerError, []byte(err.Error()))
					return
				}

				// If no scan is required, return immediately
				if result.ScanInput == nil {
					safeWrite(w, http.StatusOK, []byte(`{"status":"ok","message":"no scan required"}`))
					return
				}

				// Create a detached context for background processing
				// The original request context will be cancelled when the HTTP response is sent
				bgCtx := DetachContext(r.Context())

				// Start background scan
				go runGitHubRepoScan(bgCtx, uc, result.ScanInput)

				// Return immediately with 202 Accepted
				safeWrite(w, http.StatusAccepted, []byte(`{"status":"accepted","message":"scan enqueued"}`))
			})
			r.Post("/action", func(w http.ResponseWriter, r *http.Request) {
				if err := handleGitHubActionEvent(uc, r); err != nil {
					errutil.HandleError(r.Context(), "fail to handle GitHub action event", err)
					safeWrite(w, http.StatusInternalServerError, []byte(err.Error()))
					return
				}

				safeWrite(w, http.StatusOK, []byte("ok"))
			})
		})
	})

	return &Server{
		mux: r,
	}
}

func (x *Server) Mux() *chi.Mux {
	return x.mux
}
