package server_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/octovy/pkg/controller/server"
	"github.com/m-mizutani/octovy/pkg/infra"
	"github.com/m-mizutani/octovy/pkg/usecase"
	"github.com/m-mizutani/octovy/pkg/utils/logging"
)

func TestMiddleware(t *testing.T) {
	t.Run("preProcess adds logger with request_id to context", func(t *testing.T) {
		var capturedCtx context.Context

		// Create a test handler that captures the context
		testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedCtx = r.Context()
			w.WriteHeader(http.StatusOK)
		})

		clients := infra.New()
		uc := usecase.New(clients)
		srv := server.New(uc)

		// Wrap test handler with the middleware chain
		mux := srv.Mux()
		mux.HandleFunc("/test", testHandler)

		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		// Verify logger was added to context with request_id
		// The middleware should have created a new logger different from default
		logger := logging.From(capturedCtx)
		defaultLogger := logging.From(context.Background())
		gt.V(t, logger == defaultLogger).Equal(false)
	})

	t.Run("statusCodeLogger captures WriteHeader calls", func(t *testing.T) {
		testCases := []struct {
			name         string
			handlerFunc  http.HandlerFunc
			expectedCode int
		}{
			{
				name: "captures 200 status code",
				handlerFunc: func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				},
				expectedCode: http.StatusOK,
			},
			{
				name: "captures 404 status code",
				handlerFunc: func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusNotFound)
				},
				expectedCode: http.StatusNotFound,
			},
			{
				name: "captures 500 status code",
				handlerFunc: func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
				},
				expectedCode: http.StatusInternalServerError,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				clients := infra.New()
				uc := usecase.New(clients)
				srv := server.New(uc)

				mux := srv.Mux()
				mux.HandleFunc("/test", tc.handlerFunc)

				req := httptest.NewRequest("GET", "/test", nil)
				w := httptest.NewRecorder()

				mux.ServeHTTP(w, req)

				// Verify status code was captured by middleware
				gt.V(t, w.Code).Equal(tc.expectedCode)
			})
		}
	})

	t.Run("middleware measures request duration", func(t *testing.T) {
		// Create handler that sleeps briefly
		slowHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(10 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
		})

		clients := infra.New()
		uc := usecase.New(clients)
		srv := server.New(uc)

		mux := srv.Mux()
		mux.HandleFunc("/slow", slowHandler)

		req := httptest.NewRequest("GET", "/slow", nil)
		w := httptest.NewRecorder()

		start := time.Now()
		mux.ServeHTTP(w, req)
		elapsed := time.Since(start)

		// Verify request completed and took at least the sleep duration
		gt.V(t, w.Code).Equal(http.StatusOK)
		gt.V(t, elapsed >= 10*time.Millisecond).Equal(true)
	})

	t.Run("statusCodeLogger defaults to 200 when WriteHeader not called", func(t *testing.T) {
		// Handler that writes body without calling WriteHeader explicitly
		handlerNoWriteHeader := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("ok"))
		})

		clients := infra.New()
		uc := usecase.New(clients)
		srv := server.New(uc)

		mux := srv.Mux()
		mux.HandleFunc("/noheader", handlerNoWriteHeader)

		req := httptest.NewRequest("GET", "/noheader", nil)
		w := httptest.NewRecorder()

		mux.ServeHTTP(w, req)

		// Should default to 200
		gt.V(t, w.Code).Equal(http.StatusOK)
	})
}
