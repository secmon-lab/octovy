package server_test

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/octovy/pkg/controller/server"
	"github.com/m-mizutani/octovy/pkg/domain/mock"
	"github.com/m-mizutani/octovy/pkg/domain/model"
	"github.com/m-mizutani/octovy/pkg/domain/types"
)

//go:embed testdata/github/pull_request.opened.json
var testGitHubPullRequestOpened []byte

//go:embed testdata/github/pull_request.synchronize.json
var testGitHubPullRequestSynchronize []byte

//go:embed testdata/github/pull_request.synchronize-draft.json
var testGitHubPullRequestSynchronizeDraft []byte

//go:embed testdata/github/push.json
var testGitHubPush []byte

//go:embed testdata/github/push.default.json
var testGitHubPushDefault []byte

func TestGitHubPullRequestSync(t *testing.T) {
	const secret = "dummy"

	type testCase struct {
		input *model.ScanGitHubRepoInput
		event string
		body  []byte
	}

	runTest := func(tc testCase) func(t *testing.T) {
		return func(t *testing.T) {
			mock := &mock.UseCaseMock{
				ScanGitHubRepoFunc: func(ctx context.Context, input *model.ScanGitHubRepoInput) error {
					gt.V(t, input).Equal(tc.input)
					return nil
				},
			}

			serv := server.New(mock, server.WithGitHubSecret(secret))
			req := newGitHubWebhookRequest(t, tc.event, tc.body, secret)
			w := httptest.NewRecorder()
			serv.Mux().ServeHTTP(w, req)
			gt.V(t, w.Code).Equal(http.StatusOK)
			if tc.input != nil {
				gt.A(t, mock.ScanGitHubRepoCalls()).Length(1)
			} else {
				gt.A(t, mock.ScanGitHubRepoCalls()).Length(0)
			}
		}
	}

	t.Run("pull_request.opened", runTest(testCase{
		event: "pull_request",
		body:  testGitHubPullRequestOpened,
		input: &model.ScanGitHubRepoInput{
			GitHubMetadata: model.GitHubMetadata{
				GitHubCommit: model.GitHubCommit{
					GitHubRepo: model.GitHubRepo{
						RepoID:   581995051,
						Owner:    "m-mizutani",
						RepoName: "masq",
					},
					Ref:      "update/packages/20230918",
					Branch:   "update/packages/20230918",
					CommitID: "aa0378cad00d375c1897c1b5b5a4dd125984b511",
					Committer: model.GitHubUser{
						ID:    605953,
						Login: "m-mizutani",
					},
				},
				DefaultBranch: "main",
				PullRequest: &model.GitHubPullRequest{
					ID:           1518635674,
					Number:       13,
					BaseBranch:   "main",
					BaseCommitID: "8acdc26c9f12b9cc88e5f0b23f082f648d9e5645",
					User: model.GitHubUser{
						ID:    605953,
						Login: "m-mizutani",
					},
				},
			},
			InstallID: 41633205,
		},
	}))

	t.Run("pull_request.synchronize", runTest(testCase{
		event: "pull_request",
		body:  testGitHubPullRequestSynchronize,
		input: &model.ScanGitHubRepoInput{
			GitHubMetadata: model.GitHubMetadata{
				GitHubCommit: model.GitHubCommit{
					GitHubRepo: model.GitHubRepo{
						RepoID:   359010704,
						Owner:    "m-mizutani",
						RepoName: "octovy",
					},
					Ref:      "release/v0.2.0",
					Branch:   "release/v0.2.0",
					CommitID: "69454c171c2f0f2dbc9ccb0c9ef9b72fd769f046",
					Committer: model.GitHubUser{
						ID:    605953,
						Login: "m-mizutani",
					},
				},
				DefaultBranch: "main",
				PullRequest: &model.GitHubPullRequest{
					ID:           1473604329,
					Number:       89,
					BaseCommitID: "08fb7816c6d0a485239ca5f342342186f972a6e7",
					BaseBranch:   "main",
					User: model.GitHubUser{
						ID:    605953,
						Login: "m-mizutani",
					},
				},
			},
			InstallID: 41633205,
		},
	}))

	t.Run("pull_request.synchronize: draft", runTest(testCase{
		event: "pull_request",
		body:  testGitHubPullRequestSynchronizeDraft,
		input: nil,
	}))

	t.Run("push", runTest(testCase{
		event: "push",
		body:  testGitHubPush,
		input: &model.ScanGitHubRepoInput{
			GitHubMetadata: model.GitHubMetadata{
				GitHubCommit: model.GitHubCommit{
					GitHubRepo: model.GitHubRepo{
						RepoID:   581995051,
						Owner:    "m-mizutani",
						RepoName: "masq",
					},
					CommitID: "aa0378cad00d375c1897c1b5b5a4dd125984b511",
					Ref:      "refs/heads/update/packages/20230918",
					Branch:   "update/packages/20230918",
					Committer: model.GitHubUser{
						Login: "m-mizutani",
						Email: "mizutani@hey.com",
					},
				},
				DefaultBranch: "main",
			},
			InstallID: 41633205,
		},
	}))

	t.Run("push: to default", runTest(testCase{
		event: "push",
		body:  testGitHubPushDefault,
		input: &model.ScanGitHubRepoInput{
			GitHubMetadata: model.GitHubMetadata{
				GitHubCommit: model.GitHubCommit{
					GitHubRepo: model.GitHubRepo{
						RepoID:   281879096,
						Owner:    "m-mizutani",
						RepoName: "ops",
					},
					CommitID: "f58ae7668c3dfc193a1d2c0372cc52847613cde4",
					Ref:      "refs/heads/master",
					Branch:   "master",
					Committer: model.GitHubUser{
						Login: "m-mizutani",
						Email: "mizutani@hey.com",
					},
				},
				DefaultBranch: "master",
			},
			InstallID: 41633205,
		},
	}))
}

func newGitHubWebhookRequest(t *testing.T, event string, body []byte, secret types.GitHubAppSecret) *http.Request {
	req := gt.R1(http.NewRequest(http.MethodPost, "/webhook/github/app", bytes.NewReader(body))).NoError(t)

	h := hmac.New(sha256.New, []byte(secret))
	h.Write(body)

	req.Header.Set("X-GitHub-Event", event)
	req.Header.Set("X-Hub-Signature-256", "sha256="+hex.EncodeToString(h.Sum(nil)))
	req.Header.Set("X-GitHub-Delivery", uuid.NewString())
	req.Header.Set("Content-Type", "application/json")

	return req
}

func TestGitHubInvalidSignature(t *testing.T) {
	const secret = "test-secret"

	t.Run("invalid signature returns 500", func(t *testing.T) {
		mockUC := &mock.UseCaseMock{
			ScanGitHubRepoFunc: func(ctx context.Context, input *model.ScanGitHubRepoInput) error {
				return nil
			},
		}

		srv := server.New(mockUC, server.WithGitHubSecret(secret))

		body := []byte(`{"action":"push"}`)
		req := gt.R1(http.NewRequest(http.MethodPost, "/webhook/github/app", bytes.NewReader(body))).NoError(t)

		// Set wrong signature
		req.Header.Set("X-GitHub-Event", "push")
		req.Header.Set("X-Hub-Signature-256", "sha256=wrongsignature")
		req.Header.Set("X-GitHub-Delivery", uuid.NewString())
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		srv.Mux().ServeHTTP(rec, req)

		gt.V(t, rec.Code).Equal(http.StatusInternalServerError)
		gt.A(t, mockUC.ScanGitHubRepoCalls()).Length(0)
	})

	t.Run("malformed payload returns 500", func(t *testing.T) {
		mockUC := &mock.UseCaseMock{
			ScanGitHubRepoFunc: func(ctx context.Context, input *model.ScanGitHubRepoInput) error {
				return nil
			},
		}

		srv := server.New(mockUC, server.WithGitHubSecret(secret))

		body := []byte(`{invalid json}`)
		req := newGitHubWebhookRequest(t, "push", body, secret)

		rec := httptest.NewRecorder()
		srv.Mux().ServeHTTP(rec, req)

		gt.V(t, rec.Code).Equal(http.StatusInternalServerError)
		gt.A(t, mockUC.ScanGitHubRepoCalls()).Length(0)
	})

	t.Run("UseCase error propagates to 500", func(t *testing.T) {
		mockUC := &mock.UseCaseMock{
			ScanGitHubRepoFunc: func(ctx context.Context, input *model.ScanGitHubRepoInput) error {
				return errors.New("intentional error")
			},
		}

		srv := server.New(mockUC, server.WithGitHubSecret(secret))

		req := newGitHubWebhookRequest(t, "push", testGitHubPush, secret)

		rec := httptest.NewRecorder()
		srv.Mux().ServeHTTP(rec, req)

		gt.V(t, rec.Code).Equal(http.StatusInternalServerError)
		gt.A(t, mockUC.ScanGitHubRepoCalls()).Length(1)
	})
}
