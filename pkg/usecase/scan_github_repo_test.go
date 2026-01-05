package usecase_test

import (
	"archive/zip"
	"context"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"cloud.google.com/go/bigquery"

	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/octovy/pkg/domain/interfaces"
	"github.com/m-mizutani/octovy/pkg/domain/mock"
	"github.com/m-mizutani/octovy/pkg/domain/model"
	"github.com/m-mizutani/octovy/pkg/domain/types"
	"github.com/m-mizutani/octovy/pkg/infra"
	"github.com/m-mizutani/octovy/pkg/infra/ghapp"
	"github.com/m-mizutani/octovy/pkg/usecase"
	"github.com/m-mizutani/octovy/pkg/utils/testutil"
)

//go:embed testdata/octovy-test-code-main.zip
var testCodeZip []byte

//go:embed testdata/trivy-result.json
var testTrivyResult []byte

const (
	defaultTestOwner    = "m-mizutani"
	defaultTestRepo     = "octovy"
	defaultTestCommitID = "f7c8851da7c7fcc46212fccfb6c9c4bda520f1ca"
	defaultTestBranch   = "main"
)

func TestScanGitHubRepo(t *testing.T) {
	mockGH := &mock.GitHubAppMock{}
	mockHTTP := &httpMock{}
	mockTrivy := &trivyMock{}
	mockBQ := &mock.BigQueryMock{}

	uc := usecase.New(infra.New(
		infra.WithGitHubApp(mockGH),
		infra.WithHTTPClient(mockHTTP),
		infra.WithTrivy(mockTrivy),
		infra.WithBigQuery(mockBQ),
	))

	ctx := context.Background()

	mockGH.GetArchiveURLFunc = func(ctx context.Context, input *interfaces.GetArchiveURLInput) (*url.URL, error) {
		gt.V(t, input.Owner).Equal("m-mizutani")
		gt.V(t, input.Repo).Equal("octovy")
		gt.V(t, input.CommitID).Equal("f7c8851da7c7fcc46212fccfb6c9c4bda520f1ca")
		gt.V(t, input.InstallID).Equal(12345)

		resp := gt.R1(url.Parse("https://example.com/some/url.zip")).NoError(t)
		return resp, nil
	}
	mockGH.HTTPClientFunc = func(installID types.GitHubAppInstallID) (*http.Client, error) {
		return &http.Client{Transport: &mockTransport{mockHTTP: mockHTTP}}, nil
	}

	mockHTTP.mockDo = func(req *http.Request) (*http.Response, error) {
		gt.V(t, req.URL.String()).Equal("https://example.com/some/url.zip")

		resp := &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader(testCodeZip)),
		}
		return resp, nil
	}

	mockTrivy.mockRun = func(ctx context.Context, args []string) error {
		// Verify args contain expected values
		hasFormatJson := false
		hasListAllPkgs := false
		for i := 0; i < len(args)-1; i++ {
			if args[i] == "--format" && args[i+1] == "json" {
				hasFormatJson = true
			}
			if args[i] == "--list-all-pkgs" {
				hasListAllPkgs = true
			}
		}
		gt.V(t, hasFormatJson).Equal(true)
		gt.V(t, hasListAllPkgs).Equal(true)

		for i := range args {
			if args[i] == "--output" {
				fd := gt.R1(os.Create(args[i+1])).NoError(t)
				gt.R1(fd.Write(testTrivyResult)).NoError(t)
				gt.NoError(t, fd.Close())
				return nil
			}
		}

		t.Error("no --output option")
		return nil
	}

	var calledBQCreateTable int
	mockBQ.CreateTableFunc = func(ctx context.Context, md *bigquery.TableMetadata) error {
		calledBQCreateTable++
		return nil
	}

	mockBQ.GetMetadataFunc = func(ctx context.Context) (*bigquery.TableMetadata, error) {
		return nil, nil
	}

	var calledBQInsert int
	mockBQ.InsertFunc = func(ctx context.Context, schema bigquery.Schema, data any, opts ...interfaces.BigQueryInsertOption) error {
		calledBQInsert++
		return nil
	}

	gt.NoError(t, uc.ScanGitHubRepo(ctx, &model.ScanGitHubRepoInput{
		GitHubMetadata: model.GitHubMetadata{
			GitHubCommit: model.GitHubCommit{
				GitHubRepo: model.GitHubRepo{
					RepoID:   12345,
					Owner:    "m-mizutani",
					RepoName: "octovy",
				},
				CommitID: "f7c8851da7c7fcc46212fccfb6c9c4bda520f1ca",
				Branch:   "main",
			},
			InstallationID: 12345,
		},
		InstallID: 12345,
	}))
	gt.Equal(t, calledBQCreateTable, 1)
	gt.Equal(t, calledBQInsert, 1)
}

func TestScanGitHubRepoCleansTempAndPersistsMetadata(t *testing.T) {
	fx := newScanTestFixture(t, nil)
	ctx := context.Background()

	var inserted *model.ScanRawRecord
	fx.mockBQ.InsertFunc = func(ctx context.Context, schema bigquery.Schema, data any, opts ...interfaces.BigQueryInsertOption) error {
		var ok bool
		inserted, ok = data.(*model.ScanRawRecord)
		gt.True(t, ok)
		return nil
	}

	input := &model.ScanGitHubRepoInput{
		GitHubMetadata: model.GitHubMetadata{
			GitHubCommit: model.GitHubCommit{
				GitHubRepo: model.GitHubRepo{
					RepoID:   12345,
					Owner:    defaultTestOwner,
					RepoName: defaultTestRepo,
				},
				CommitID: defaultTestCommitID,
				Branch:   defaultTestBranch,
			},
			InstallationID: 12345,
		},
		InstallID: 12345,
	}

	gt.NoError(t, fx.uc.ScanGitHubRepo(ctx, input))
	gt.V(t, inserted).NotEqual((*model.ScanRawRecord)(nil))
	gt.V(t, inserted.Scan.GitHub.GitHubCommit.CommitID).Equal(defaultTestCommitID)
	gt.V(t, inserted.Scan.GitHub.GitHubRepo.Owner).Equal(defaultTestOwner)

	tempPattern := filepath.Join(os.TempDir(), fmt.Sprintf("octovy.%s.%s.%s*", defaultTestOwner, defaultTestRepo, defaultTestCommitID))
	matches, err := filepath.Glob(tempPattern)
	gt.NoError(t, err)
	gt.V(t, len(matches)).Equal(0)
}

func TestScanGitHubRepoRejectsPathTraversal(t *testing.T) {
	zipData := buildZipArchive(t, map[string]string{
		defaultTestRepo + "/../evil/file.txt": "malicious",
	})
	fx := newScanTestFixture(t, zipData)
	ctx := context.Background()

	fx.mockBQ.InsertFunc = func(ctx context.Context, schema bigquery.Schema, data any, opts ...interfaces.BigQueryInsertOption) error {
		t.Fatalf("Insert should not be called when extraction fails")
		return nil
	}

	input := &model.ScanGitHubRepoInput{
		GitHubMetadata: model.GitHubMetadata{
			GitHubCommit: model.GitHubCommit{
				GitHubRepo: model.GitHubRepo{
					RepoID:   12345,
					Owner:    defaultTestOwner,
					RepoName: defaultTestRepo,
				},
				CommitID: defaultTestCommitID,
				Branch:   defaultTestBranch,
			},
			InstallationID: 12345,
		},
		InstallID: 12345,
	}

	err := fx.uc.ScanGitHubRepo(ctx, input)
	gt.Error(t, err)
	gt.True(t, strings.Contains(err.Error(), "illegal file path of zip"))
}

func TestScanGitHubRepoTrivyError(t *testing.T) {
	fx := newScanTestFixture(t, nil)
	ctx := context.Background()

	fx.mockTrivy.mockRun = func(ctx context.Context, args []string) error {
		return errors.New("trivy failed")
	}
	fx.mockBQ.InsertFunc = func(ctx context.Context, schema bigquery.Schema, data any, opts ...interfaces.BigQueryInsertOption) error {
		t.Fatalf("Insert should not be called when Trivy fails")
		return nil
	}

	input := &model.ScanGitHubRepoInput{
		GitHubMetadata: model.GitHubMetadata{
			GitHubCommit: model.GitHubCommit{
				GitHubRepo: model.GitHubRepo{
					RepoID:   12345,
					Owner:    defaultTestOwner,
					RepoName: defaultTestRepo,
				},
				CommitID: defaultTestCommitID,
				Branch:   defaultTestBranch,
			},
			InstallationID: 12345,
		},
		InstallID: 12345,
	}

	err := fx.uc.ScanGitHubRepo(ctx, input)
	gt.Error(t, err)
	gt.True(t, strings.Contains(err.Error(), "failed to scan local directory"))
}

type scanTestFixture struct {
	uc        *usecase.UseCase
	mockGH    *mock.GitHubAppMock
	mockHTTP  *httpMock
	mockTrivy *trivyMock
	mockBQ    *mock.BigQueryMock
}

func newScanTestFixture(t *testing.T, zipData []byte) *scanTestFixture {
	t.Helper()
	if zipData == nil {
		zipData = testCodeZip
	}

	mockGH := &mock.GitHubAppMock{}
	mockHTTP := &httpMock{}
	mockTrivy := &trivyMock{}
	mockBQ := &mock.BigQueryMock{}

	clients := infra.New(
		infra.WithGitHubApp(mockGH),
		infra.WithHTTPClient(mockHTTP),
		infra.WithTrivy(mockTrivy),
		infra.WithBigQuery(mockBQ),
	)
	fx := &scanTestFixture{
		uc:        usecase.New(clients),
		mockGH:    mockGH,
		mockHTTP:  mockHTTP,
		mockTrivy: mockTrivy,
		mockBQ:    mockBQ,
	}

	mockGH.GetArchiveURLFunc = func(ctx context.Context, input *interfaces.GetArchiveURLInput) (*url.URL, error) {
		gt.V(t, input.Owner).Equal(defaultTestOwner)
		gt.V(t, input.Repo).Equal(defaultTestRepo)
		gt.V(t, input.CommitID).Equal(defaultTestCommitID)
		gt.V(t, input.InstallID).Equal(types.GitHubAppInstallID(12345))
		return gt.R1(url.Parse("https://example.com/archive.zip")).NoError(t), nil
	}
	mockGH.HTTPClientFunc = func(installID types.GitHubAppInstallID) (*http.Client, error) {
		return &http.Client{Transport: &mockTransport{mockHTTP: mockHTTP}}, nil
	}
	mockHTTP.mockDo = func(req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader(zipData)),
		}, nil
	}

	mockTrivy.mockRun = func(ctx context.Context, args []string) error {
		return writeTrivyOutput(t, args)
	}

	mockBQ.GetMetadataFunc = func(ctx context.Context) (*bigquery.TableMetadata, error) {
		return nil, nil
	}
	mockBQ.CreateTableFunc = func(ctx context.Context, md *bigquery.TableMetadata) error {
		return nil
	}
	mockBQ.InsertFunc = func(ctx context.Context, schema bigquery.Schema, data any, opts ...interfaces.BigQueryInsertOption) error {
		return nil
	}

	return fx
}

func writeTrivyOutput(t *testing.T, args []string) error {
	t.Helper()
	for i := range args {
		if args[i] == "--output" && i+1 < len(args) {
			fd := gt.R1(os.Create(args[i+1])).NoError(t)
			defer func() {
				gt.NoError(t, fd.Close())
			}()
			_, err := fd.Write(testTrivyResult)
			gt.NoError(t, err)
			return nil
		}
	}
	t.Fatalf("no --output option supplied to trivy")
	return nil
}

func buildZipArchive(t *testing.T, files map[string]string) []byte {
	t.Helper()
	buf := &bytes.Buffer{}
	zw := zip.NewWriter(buf)
	for name, content := range files {
		fw, err := zw.Create(name)
		gt.NoError(t, err)
		_, err = fw.Write([]byte(content))
		gt.NoError(t, err)
	}
	gt.NoError(t, zw.Close())
	return buf.Bytes()
}

type trivyMock struct {
	mockRun func(ctx context.Context, args []string) error
}

func (x *trivyMock) Run(ctx context.Context, args []string) error {
	return x.mockRun(ctx, args)
}

type httpMock struct {
	mockDo func(req *http.Request) (*http.Response, error)
}

func (x *httpMock) Do(req *http.Request) (*http.Response, error) {
	return x.mockDo(req)
}

type mockTransport struct {
	mockHTTP *httpMock
}

func (t *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return t.mockHTTP.Do(req)
}

func TestScanGitHubRepoWithData(t *testing.T) {
	if _, ok := os.LookupEnv("TEST_SCAN_GITHUB_REPO"); !ok {
		t.Skip("TEST_SCAN_GITHUB_REPO is not set")
	}

	// Setting up GitHub App
	strAppID := testutil.GetEnvOrSkip(t, "TEST_OCTOVY_GITHUB_APP_ID")
	privateKey := testutil.GetEnvOrSkip(t, "TEST_OCTOVY_GITHUB_APP_PRIVATE_KEY")

	appID := gt.R1(strconv.ParseInt(strAppID, 10, 64)).NoError(t)
	ghApp := gt.R1(ghapp.New(types.GitHubAppID(appID), types.GitHubAppPrivateKey(privateKey))).NoError(t)

	uc := usecase.New(infra.New(
		infra.WithGitHubApp(ghApp),
	))

	ctx := context.Background()

	gt.NoError(t, uc.ScanGitHubRepo(ctx, &model.ScanGitHubRepoInput{
		GitHubMetadata: model.GitHubMetadata{
			GitHubCommit: model.GitHubCommit{
				GitHubRepo: model.GitHubRepo{
					RepoID:   41633205,
					Owner:    "m-mizutani",
					RepoName: "octovy",
				},
				CommitID: "6581604ef668e77a178e18dbc56e898f5fd87014",
			},
			InstallationID: 41633205,
		},
		InstallID: 41633205,
	}))
}

func TestScanGitHubRepoValidation(t *testing.T) {
	t.Run("invalid install ID fails validation", func(t *testing.T) {
		uc := usecase.New(infra.New())
		ctx := context.Background()

		input := &model.ScanGitHubRepoInput{
			GitHubMetadata: model.GitHubMetadata{
				GitHubCommit: model.GitHubCommit{
					GitHubRepo: model.GitHubRepo{
						RepoID:   123,
						Owner:    "test",
						RepoName: "repo",
					},
					CommitID: "a234567890123456789012345678901234567890",
				},
				InstallationID: 0,
			},
			InstallID: 0, // Invalid
		}

		err := uc.ScanGitHubRepo(ctx, input)
		gt.Error(t, err)
	})

	t.Run("invalid commit ID fails validation", func(t *testing.T) {
		uc := usecase.New(infra.New())
		ctx := context.Background()

		input := &model.ScanGitHubRepoInput{
			GitHubMetadata: model.GitHubMetadata{
				GitHubCommit: model.GitHubCommit{
					GitHubRepo: model.GitHubRepo{
						RepoID:   123,
						Owner:    "test",
						RepoName: "repo",
					},
					CommitID: "", // Invalid
				},
				InstallationID: 12345,
			},
			InstallID: 12345,
		}

		err := uc.ScanGitHubRepo(ctx, input)
		gt.Error(t, err)
	})
}

func TestDownloadZipFile(t *testing.T) {
	ctx := context.Background()

	t.Run("successfully download zip file with 200 response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("zip content"))
		}))
		defer server.Close()

		zipURL, err := url.Parse(server.URL)
		gt.NoError(t, err)

		var buf bytes.Buffer
		httpClient := &http.Client{}
		err = usecase.DownloadZipFileForTest(ctx, httpClient, zipURL, &buf)
		gt.NoError(t, err)
		gt.V(t, buf.String()).Equal("zip content")
	})

	t.Run("404 response wraps ErrInvalidGitHubData", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("not found"))
		}))
		defer server.Close()

		zipURL, err := url.Parse(server.URL)
		gt.NoError(t, err)

		var buf bytes.Buffer
		httpClient := &http.Client{}
		err = usecase.DownloadZipFileForTest(ctx, httpClient, zipURL, &buf)
		gt.Error(t, err)
		gt.True(t, errors.Is(err, types.ErrInvalidGitHubData))
	})

	t.Run("500 response wraps ErrInvalidGitHubData", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("server error"))
		}))
		defer server.Close()

		zipURL, err := url.Parse(server.URL)
		gt.NoError(t, err)

		var buf bytes.Buffer
		httpClient := &http.Client{}
		err = usecase.DownloadZipFileForTest(ctx, httpClient, zipURL, &buf)
		gt.Error(t, err)
		gt.True(t, errors.Is(err, types.ErrInvalidGitHubData))
	})
}

func TestStepDownDirectory(t *testing.T) {
	t.Run("remove first directory from path", func(t *testing.T) {
		result, err := usecase.StepDownDirectoryForTest("root/subdir/file.txt")
		gt.NoError(t, err)
		gt.V(t, result).Equal("subdir/file.txt")
	})

	t.Run("absolute path becomes relative", func(t *testing.T) {
		result, err := usecase.StepDownDirectoryForTest("/root/subdir/file.txt")
		gt.NoError(t, err)
		gt.V(t, result).Equal("subdir/file.txt")
	})

	t.Run("single directory returns empty", func(t *testing.T) {
		result, err := usecase.StepDownDirectoryForTest("root")
		gt.NoError(t, err)
		gt.V(t, result).Equal("")
	})

	t.Run("empty string returns empty", func(t *testing.T) {
		result, err := usecase.StepDownDirectoryForTest("")
		gt.NoError(t, err)
		gt.V(t, result).Equal("")
	})

	t.Run("path traversal returns error", func(t *testing.T) {
		_, err := usecase.StepDownDirectoryForTest("root/../evil/file.txt")
		gt.Error(t, err)
		gt.True(t, strings.Contains(err.Error(), "illegal file path of zip"))
	})
}

func TestExtractCode(t *testing.T) {
	ctx := context.Background()

	t.Run("extract file successfully", func(t *testing.T) {
		tmpDir := gt.R1(os.MkdirTemp("", "extract-test-*")).NoError(t)
		defer os.RemoveAll(tmpDir)

		// Create a test zip file
		zipPath := filepath.Join(tmpDir, "test.zip")
		zipFile, err := os.Create(zipPath)
		gt.NoError(t, err)

		zipWriter := zip.NewWriter(zipFile)
		fileWriter, err := zipWriter.Create("root/subdir/test.txt")
		gt.NoError(t, err)
		_, err = fileWriter.Write([]byte("test content"))
		gt.NoError(t, err)
		gt.NoError(t, zipWriter.Close())
		gt.NoError(t, zipFile.Close())

		// Open and extract
		reader, err := zip.OpenReader(zipPath)
		gt.NoError(t, err)
		defer reader.Close()

		extractDir := filepath.Join(tmpDir, "extracted")
		gt.NoError(t, os.MkdirAll(extractDir, 0755))

		for _, f := range reader.File {
			err = usecase.ExtractCodeForTest(ctx, f, extractDir)
			gt.NoError(t, err)
		}

		// Verify extracted file
		content, err := os.ReadFile(filepath.Join(extractDir, "subdir/test.txt"))
		gt.NoError(t, err)
		gt.V(t, string(content)).Equal("test content")
	})

	t.Run("skip files in root directory after stepDown", func(t *testing.T) {
		tmpDir := gt.R1(os.MkdirTemp("", "extract-test-*")).NoError(t)
		defer os.RemoveAll(tmpDir)

		// Create a zip file with only root-level file (stepDownDirectory returns empty)
		zipPath := filepath.Join(tmpDir, "test.zip")
		zipFile, err := os.Create(zipPath)
		gt.NoError(t, err)

		zipWriter := zip.NewWriter(zipFile)
		fileWriter, err := zipWriter.Create("rootfile.txt") // No parent dir
		gt.NoError(t, err)
		_, err = fileWriter.Write([]byte("content"))
		gt.NoError(t, err)
		gt.NoError(t, zipWriter.Close())
		gt.NoError(t, zipFile.Close())

		// Try to extract - should skip because stepDownDirectory returns empty
		reader, err := zip.OpenReader(zipPath)
		gt.NoError(t, err)
		defer reader.Close()

		extractDir := filepath.Join(tmpDir, "extracted")
		gt.NoError(t, os.MkdirAll(extractDir, 0755))

		for _, f := range reader.File {
			err = usecase.ExtractCodeForTest(ctx, f, extractDir)
			gt.NoError(t, err) // Should not error, just skip
		}

		// Verify no file was extracted
		files, err := os.ReadDir(extractDir)
		gt.NoError(t, err)
		gt.V(t, len(files)).Equal(0)
	})
}

func TestExtractZipFile(t *testing.T) {
	ctx := context.Background()

	t.Run("extract zip file with nested directories", func(t *testing.T) {
		tmpDir := gt.R1(os.MkdirTemp("", "zip-test-*")).NoError(t)
		defer os.RemoveAll(tmpDir)

		// Create test zip
		zipPath := filepath.Join(tmpDir, "test.zip")
		zipFile, err := os.Create(zipPath)
		gt.NoError(t, err)

		zipWriter := zip.NewWriter(zipFile)

		// Add multiple files
		files := map[string]string{
			"root/file1.txt":             "content1",
			"root/subdir/file2.txt":      "content2",
			"root/subdir/deep/file3.txt": "content3",
		}

		for name, content := range files {
			fileWriter, err := zipWriter.Create(name)
			gt.NoError(t, err)
			_, err = fileWriter.Write([]byte(content))
			gt.NoError(t, err)
		}

		gt.NoError(t, zipWriter.Close())
		gt.NoError(t, zipFile.Close())

		// Extract
		extractDir := filepath.Join(tmpDir, "extracted")
		err = usecase.ExtractZipFileForTest(ctx, zipPath, extractDir)
		gt.NoError(t, err)

		// Verify all files extracted (with root directory removed)
		content1, err := os.ReadFile(filepath.Join(extractDir, "file1.txt"))
		gt.NoError(t, err)
		gt.V(t, string(content1)).Equal("content1")

		content2, err := os.ReadFile(filepath.Join(extractDir, "subdir/file2.txt"))
		gt.NoError(t, err)
		gt.V(t, string(content2)).Equal("content2")

		content3, err := os.ReadFile(filepath.Join(extractDir, "subdir/deep/file3.txt"))
		gt.NoError(t, err)
		gt.V(t, string(content3)).Equal("content3")
	})
}

// mockTrivyClient for testing scanDirectory
type mockTrivyClient struct {
	runFunc  func(ctx context.Context, args []string) error
	lastArgs []string
}

func (m *mockTrivyClient) Run(ctx context.Context, args []string) error {
	m.lastArgs = args
	if m.runFunc != nil {
		return m.runFunc(ctx, args)
	}
	return nil
}

func TestScanDirectory(t *testing.T) {
	t.Run("trivy arguments contain required flags", func(t *testing.T) {
		tmpDir := gt.R1(os.MkdirTemp("", "scan-test-*")).NoError(t)
		defer os.RemoveAll(tmpDir)

		mockTrivy := &mockTrivyClient{}
		mockTrivy.runFunc = func(ctx context.Context, args []string) error {
			// Find output file and write test data
			for i, arg := range args {
				if arg == "--output" && i+1 < len(args) {
					outputFile := args[i+1]
					// Write JSON with explicit Results field
					testJSON := `{"SchemaVersion":2,"ArtifactName":"test","Results":[]}`
					os.WriteFile(outputFile, []byte(testJSON), 0644)
					break
				}
			}
			return nil
		}

		clients := infra.New(infra.WithTrivy(mockTrivy))
		uc := usecase.New(clients)

		ctx := context.Background()
		codeDir := tmpDir

		report, err := uc.ScanDirectoryForTest(ctx, codeDir)
		gt.NoError(t, err)
		gt.V(t, report.SchemaVersion).Equal(2)
		gt.V(t, report.ArtifactName).Equal("test")
		gt.V(t, report.ArtifactType).Equal("")

		// Verify trivy was called with correct arguments
		hasFs := false
		hasFormat := false
		hasJson := false
		hasOutput := false
		hasListAllPkgs := false
		hasCodeDir := false
		for _, arg := range mockTrivy.lastArgs {
			if arg == "fs" {
				hasFs = true
			}
			if arg == "--format" {
				hasFormat = true
			}
			if arg == "json" {
				hasJson = true
			}
			if arg == "--output" {
				hasOutput = true
			}
			if arg == "--list-all-pkgs" {
				hasListAllPkgs = true
			}
			if arg == codeDir {
				hasCodeDir = true
			}
		}
		gt.V(t, hasFs).Equal(true)
		gt.V(t, hasFormat).Equal(true)
		gt.V(t, hasJson).Equal(true)
		gt.V(t, hasOutput).Equal(true)
		gt.V(t, hasListAllPkgs).Equal(true)
		gt.V(t, hasCodeDir).Equal(true)
	})

	t.Run("trivy error is wrapped", func(t *testing.T) {
		tmpDir := gt.R1(os.MkdirTemp("", "scan-test-*")).NoError(t)
		defer os.RemoveAll(tmpDir)

		mockTrivy := &mockTrivyClient{}
		mockTrivy.runFunc = func(ctx context.Context, args []string) error {
			return os.ErrPermission
		}

		clients := infra.New(infra.WithTrivy(mockTrivy))
		uc := usecase.New(clients)

		ctx := context.Background()
		report, err := uc.ScanDirectoryForTest(ctx, tmpDir)
		gt.Error(t, err)
		gt.V(t, report).Equal(nil)
	})
}

func TestScanGitHubRepoCleanup(t *testing.T) {
	t.Run("temporary directories are cleaned up", func(t *testing.T) {
		// Create a test directory to monitor
		baseDir := gt.R1(os.MkdirTemp("", "cleanup-test-*")).NoError(t)
		defer os.RemoveAll(baseDir)

		// Mock clients that do minimal work
		mockTrivy := &mockTrivyClient{}
		mockTrivy.runFunc = func(ctx context.Context, args []string) error {
			// Write valid JSON to output file
			for i, arg := range args {
				if arg == "--output" && i+1 < len(args) {
					testJSON := `{"SchemaVersion":2,"ArtifactName":"test","Results":[]}`
					os.WriteFile(args[i+1], []byte(testJSON), 0644)
					break
				}
			}
			return nil
		}

		clients := infra.New(infra.WithTrivy(mockTrivy))
		uc := usecase.New(clients)

		// Test the cleanup behavior
		ctx := context.Background()
		testDir := filepath.Join(baseDir, "test-repo")
		os.MkdirAll(testDir, 0755)

		_, err := uc.ScanDirectoryForTest(ctx, testDir)
		gt.NoError(t, err)

		// Verify temp files are cleaned up
		// scanDirectory creates temp files with pattern octovy_result.*.json
		// and should clean them up via defer
		entries, err := os.ReadDir(os.TempDir())
		gt.NoError(t, err)
		for _, entry := range entries {
			// No octovy_result.*.json files should remain
			if strings.Contains(entry.Name(), "octovy_result") {
				gt.False(t, strings.Contains(entry.Name(), "octovy_result"))
			}
		}
	})
}

func TestScanGitHubRepoRemote(t *testing.T) {
	t.Run("commit and branch are mutually exclusive", func(t *testing.T) {
		uc := usecase.New(infra.New())
		ctx := context.Background()

		input := &model.ScanGitHubRepoRemoteInput{
			Owner:     "test-owner",
			Repo:      "test-repo",
			Commit:    "abc123",
			Branch:    "main",
			InstallID: 12345,
		}

		err := uc.ScanGitHubRepoRemote(ctx, input)
		gt.Error(t, err)
		gt.True(t, strings.Contains(err.Error(), "commit and branch cannot be specified at the same time"))
	})

	t.Run("full specification mode with commit", func(t *testing.T) {
		fx := newScanTestFixture(t, nil)
		ctx := context.Background()

		var scanCalled bool
		fx.mockGH.GetArchiveURLFunc = func(ctx context.Context, input *interfaces.GetArchiveURLInput) (*url.URL, error) {
			scanCalled = true
			gt.V(t, input.Owner).Equal("test-owner")
			gt.V(t, input.Repo).Equal("test-repo")
			gt.V(t, input.CommitID).Equal("abc1234567890123456789012345678901234567")
			gt.V(t, input.InstallID).Equal(types.GitHubAppInstallID(12345))
			return gt.R1(url.Parse("https://example.com/archive.zip")).NoError(t), nil
		}

		input := &model.ScanGitHubRepoRemoteInput{
			Owner:     "test-owner",
			Repo:      "test-repo",
			Commit:    "abc1234567890123456789012345678901234567",
			InstallID: 12345,
		}

		err := fx.uc.ScanGitHubRepoRemote(ctx, input)
		gt.NoError(t, err)
		gt.V(t, scanCalled).Equal(true)
	})

	t.Run("full specification mode with branch resolution", func(t *testing.T) {
		fx := newScanTestFixture(t, nil)
		ctx := context.Background()

		var branchResolvedCommit string
		fx.mockHTTP.mockDo = func(req *http.Request) (*http.Response, error) {
			// GitHub API branch endpoint
			if strings.Contains(req.URL.Path, "/branches/") {
				responseJSON := `{"commit":{"sha":"1234567890123456789012345678901234567890"}}`
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader([]byte(responseJSON))),
				}, nil
			}
			// Archive download
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader(testCodeZip)),
			}, nil
		}

		fx.mockGH.GetArchiveURLFunc = func(ctx context.Context, input *interfaces.GetArchiveURLInput) (*url.URL, error) {
			branchResolvedCommit = input.CommitID
			return gt.R1(url.Parse("https://example.com/archive.zip")).NoError(t), nil
		}

		input := &model.ScanGitHubRepoRemoteInput{
			Owner:     "test-owner",
			Repo:      "test-repo",
			Branch:    "main",
			InstallID: 12345,
		}

		err := fx.uc.ScanGitHubRepoRemote(ctx, input)
		gt.NoError(t, err)
		gt.V(t, branchResolvedCommit).Equal("1234567890123456789012345678901234567890")
	})

	t.Run("DB completion mode requires Firestore", func(t *testing.T) {
		// No Firestore configured
		uc := usecase.New(infra.New())
		ctx := context.Background()

		input := &model.ScanGitHubRepoRemoteInput{
			Owner: "test-owner",
			Repo:  "test-repo",
		}

		err := uc.ScanGitHubRepoRemote(ctx, input)
		gt.Error(t, err)
		gt.True(t, strings.Contains(err.Error(), "DB completion mode requires ScanRepository"))
	})

	t.Run("DB completion mode fetches from repository", func(t *testing.T) {
		fx := newScanTestFixture(t, nil)
		ctx := context.Background()

		mockRepo := &mock.ScanRepositoryMock{}
		mockRepo.GetRepositoryFunc = func(ctx context.Context, repoID types.GitHubRepoID) (*model.Repository, error) {
			gt.V(t, repoID).Equal(types.GitHubRepoID("test-owner/test-repo"))
			return &model.Repository{
				ID:             repoID,
				Owner:          "test-owner",
				Name:           "test-repo",
				DefaultBranch:  "main",
				InstallationID: 67890,
			}, nil
		}
		mockRepo.GetBranchFunc = func(ctx context.Context, repoID types.GitHubRepoID, branchName types.BranchName) (*model.Branch, error) {
			gt.V(t, repoID).Equal(types.GitHubRepoID("test-owner/test-repo"))
			gt.V(t, branchName).Equal(types.BranchName("main"))
			return &model.Branch{
				Name:          "main",
				LastCommitSHA: "abcdef1234567890123456789012345678901234",
			}, nil
		}
		mockRepo.CreateOrUpdateRepositoryFunc = func(ctx context.Context, repo *model.Repository) error {
			return nil
		}
		mockRepo.CreateOrUpdateBranchFunc = func(ctx context.Context, repoID types.GitHubRepoID, branch *model.Branch) error {
			return nil
		}
		mockRepo.CreateOrUpdateTargetFunc = func(ctx context.Context, repoID types.GitHubRepoID, branchName types.BranchName, target *model.Target) error {
			return nil
		}
		mockRepo.ListVulnerabilitiesFunc = func(ctx context.Context, repoID types.GitHubRepoID, branchName types.BranchName, targetID types.TargetID) ([]*model.Vulnerability, error) {
			return nil, nil
		}
		mockRepo.BatchCreateVulnerabilitiesFunc = func(ctx context.Context, repoID types.GitHubRepoID, branchName types.BranchName, targetID types.TargetID, vulns []*model.Vulnerability) error {
			return nil
		}

		// Create usecase with Firestore repository
		clients := infra.New(
			infra.WithGitHubApp(fx.mockGH),
			infra.WithHTTPClient(fx.mockHTTP),
			infra.WithTrivy(fx.mockTrivy),
			infra.WithBigQuery(fx.mockBQ),
			infra.WithScanRepository(mockRepo),
		)
		uc := usecase.New(clients)

		var scanCalledWithCommit string
		fx.mockGH.GetArchiveURLFunc = func(ctx context.Context, input *interfaces.GetArchiveURLInput) (*url.URL, error) {
			scanCalledWithCommit = input.CommitID
			gt.V(t, input.InstallID).Equal(types.GitHubAppInstallID(67890))
			return gt.R1(url.Parse("https://example.com/archive.zip")).NoError(t), nil
		}

		input := &model.ScanGitHubRepoRemoteInput{
			Owner: "test-owner",
			Repo:  "test-repo",
		}

		err := uc.ScanGitHubRepoRemote(ctx, input)
		gt.NoError(t, err)
		gt.V(t, scanCalledWithCommit).Equal("abcdef1234567890123456789012345678901234")
	})

	t.Run("DB completion mode with custom branch", func(t *testing.T) {
		fx := newScanTestFixture(t, nil)
		ctx := context.Background()

		mockRepo := &mock.ScanRepositoryMock{}
		mockRepo.GetRepositoryFunc = func(ctx context.Context, repoID types.GitHubRepoID) (*model.Repository, error) {
			return &model.Repository{
				ID:             repoID,
				Owner:          "test-owner",
				Name:           "test-repo",
				DefaultBranch:  "main",
				InstallationID: 67890,
			}, nil
		}
		mockRepo.GetBranchFunc = func(ctx context.Context, repoID types.GitHubRepoID, branchName types.BranchName) (*model.Branch, error) {
			gt.V(t, branchName).Equal(types.BranchName("feature-branch"))
			return &model.Branch{
				Name:          "feature-branch",
				LastCommitSHA: "fedcba0987654321098765432109876543210987",
			}, nil
		}
		mockRepo.CreateOrUpdateRepositoryFunc = func(ctx context.Context, repo *model.Repository) error {
			return nil
		}
		mockRepo.CreateOrUpdateBranchFunc = func(ctx context.Context, repoID types.GitHubRepoID, branch *model.Branch) error {
			return nil
		}
		mockRepo.CreateOrUpdateTargetFunc = func(ctx context.Context, repoID types.GitHubRepoID, branchName types.BranchName, target *model.Target) error {
			return nil
		}
		mockRepo.ListVulnerabilitiesFunc = func(ctx context.Context, repoID types.GitHubRepoID, branchName types.BranchName, targetID types.TargetID) ([]*model.Vulnerability, error) {
			return nil, nil
		}
		mockRepo.BatchCreateVulnerabilitiesFunc = func(ctx context.Context, repoID types.GitHubRepoID, branchName types.BranchName, targetID types.TargetID, vulns []*model.Vulnerability) error {
			return nil
		}

		clients := infra.New(
			infra.WithGitHubApp(fx.mockGH),
			infra.WithHTTPClient(fx.mockHTTP),
			infra.WithTrivy(fx.mockTrivy),
			infra.WithBigQuery(fx.mockBQ),
			infra.WithScanRepository(mockRepo),
		)
		uc := usecase.New(clients)

		var scanCalledWithCommit string
		fx.mockGH.GetArchiveURLFunc = func(ctx context.Context, input *interfaces.GetArchiveURLInput) (*url.URL, error) {
			scanCalledWithCommit = input.CommitID
			return gt.R1(url.Parse("https://example.com/archive.zip")).NoError(t), nil
		}

		input := &model.ScanGitHubRepoRemoteInput{
			Owner:  "test-owner",
			Repo:   "test-repo",
			Branch: "feature-branch",
		}

		err := uc.ScanGitHubRepoRemote(ctx, input)
		gt.NoError(t, err)
		gt.V(t, scanCalledWithCommit).Equal("fedcba0987654321098765432109876543210987")
	})

	t.Run("DB completion mode repository not found", func(t *testing.T) {
		ctx := context.Background()

		mockRepo := &mock.ScanRepositoryMock{}
		mockRepo.GetRepositoryFunc = func(ctx context.Context, repoID types.GitHubRepoID) (*model.Repository, error) {
			return nil, errors.New("repository not found")
		}

		clients := infra.New(
			infra.WithScanRepository(mockRepo),
		)
		uc := usecase.New(clients)

		input := &model.ScanGitHubRepoRemoteInput{
			Owner: "test-owner",
			Repo:  "test-repo",
		}

		err := uc.ScanGitHubRepoRemote(ctx, input)
		gt.Error(t, err)
		gt.True(t, strings.Contains(err.Error(), "repository not found"))
	})

	t.Run("DB completion mode branch not found", func(t *testing.T) {
		ctx := context.Background()

		mockRepo := &mock.ScanRepositoryMock{}
		mockRepo.GetRepositoryFunc = func(ctx context.Context, repoID types.GitHubRepoID) (*model.Repository, error) {
			return &model.Repository{
				ID:             repoID,
				Owner:          "test-owner",
				Name:           "test-repo",
				DefaultBranch:  "main",
				InstallationID: 67890,
			}, nil
		}
		mockRepo.GetBranchFunc = func(ctx context.Context, repoID types.GitHubRepoID, branchName types.BranchName) (*model.Branch, error) {
			return nil, errors.New("branch not found")
		}

		clients := infra.New(
			infra.WithScanRepository(mockRepo),
		)
		uc := usecase.New(clients)

		input := &model.ScanGitHubRepoRemoteInput{
			Owner: "test-owner",
			Repo:  "test-repo",
		}

		err := uc.ScanGitHubRepoRemote(ctx, input)
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("branch not found")
	})

	t.Run("branch resolution fails with GitHub API error", func(t *testing.T) {
		fx := newScanTestFixture(t, nil)
		ctx := context.Background()

		fx.mockHTTP.mockDo = func(req *http.Request) (*http.Response, error) {
			if strings.Contains(req.URL.Path, "/branches/") {
				return &http.Response{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(bytes.NewReader([]byte("branch not found"))),
				}, nil
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewReader(testCodeZip)),
			}, nil
		}

		input := &model.ScanGitHubRepoRemoteInput{
			Owner:     "test-owner",
			Repo:      "test-repo",
			Branch:    "nonexistent",
			InstallID: 12345,
		}

		err := fx.uc.ScanGitHubRepoRemote(ctx, input)
		gt.Error(t, err)
		gt.S(t, err.Error()).Contains("failed to get branch information")
	})
}
