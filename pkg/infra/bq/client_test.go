package bq_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	"cloud.google.com/go/bigquery"
	"github.com/m-mizutani/bqs"
	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/octovy/pkg/domain/interfaces"
	"github.com/m-mizutani/octovy/pkg/domain/model"
	"github.com/m-mizutani/octovy/pkg/domain/types"
	"github.com/m-mizutani/octovy/pkg/infra/bq"
	"github.com/m-mizutani/octovy/pkg/utils/testutil"
	"google.golang.org/api/impersonate"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestClient(t *testing.T) {
	projectID := testutil.GetEnvOrSkip(t, "TEST_BIGQUERY_PROJECT_ID")
	datasetID := testutil.GetEnvOrSkip(t, "TEST_BIGQUERY_DATASET_ID")

	ctx := context.Background()

	tblName := types.BQTableID(time.Now().Format("insert_test_20060102_150405"))
	client, err := bq.New(ctx, types.GoogleProjectID(projectID), types.BQDatasetID(datasetID), tblName)
	gt.NoError(t, err)

	var baseSchema bigquery.Schema

	t.Run("Create base table at first", func(t *testing.T) {
		var scan model.Scan
		baseSchema = gt.R1(bqs.Infer(scan)).NoError(t)
		gt.NoError(t, err)

		gt.NoError(t, client.CreateTable(ctx, &bigquery.TableMetadata{
			Name:   tblName.String(),
			Schema: baseSchema,
		}))
	})

	t.Run("Insert record", func(t *testing.T) {
		var scan model.Scan
		// Load test data directly
		data := gt.R1(os.ReadFile("testdata/data.json")).NoError(t)
		gt.NoError(t, json.Unmarshal(data, &scan.Report))
		dataSchema := gt.R1(bqs.Infer(scan)).NoError(t)
		mergedSchema := gt.R1(bqs.Merge(baseSchema, dataSchema)).NoError(t)

		md := gt.R1(client.GetMetadata(ctx)).NoError(t)
		gt.False(t, bqs.Equal(mergedSchema, baseSchema))
		gt.NoError(t, client.UpdateTable(ctx, bigquery.TableMetadataToUpdate{
			Schema: mergedSchema,
		}, md.ETag))

		record := model.ScanRawRecord{
			Scan:      scan,
			Timestamp: scan.Timestamp.UnixMicro(),
		}
		gt.NoError(t, client.Insert(ctx, mergedSchema, record))
	})
}

func TestImpersonation(t *testing.T) {
	projectID := testutil.GetEnvOrSkip(t, "TEST_BIGQUERY_PROJECT_ID")
	datasetID := testutil.GetEnvOrSkip(t, "TEST_BIGQUERY_DATASET_ID")
	serviceAccount := testutil.GetEnvOrSkip(t, "TEST_BIGQUERY_IMPERSONATE_SERVICE_ACCOUNT")

	ctx := context.Background()

	ts, err := impersonate.CredentialsTokenSource(ctx, impersonate.CredentialsConfig{
		TargetPrincipal: serviceAccount,
		Scopes: []string{
			"https://www.googleapis.com/auth/bigquery",
			"https://www.googleapis.com/auth/cloud-platform",
		},
	})
	gt.NoError(t, err)

	tblName := types.BQTableID(time.Now().Format("impersonation_test_20060102_150405"))
	client, err := bq.New(ctx, types.GoogleProjectID(projectID), types.BQDatasetID(datasetID), tblName, option.WithTokenSource(ts))
	gt.NoError(t, err)

	msg := struct {
		Msg string
	}{
		Msg: "Hello, BigQuery: " + time.Now().String(),
	}

	schema := gt.R1(bqs.Infer(msg)).NoError(t)

	gt.NoError(t, client.CreateTable(ctx, &bigquery.TableMetadata{
		Name:   tblName.String(),
		Schema: schema,
	}))

	gt.NoError(t, client.Insert(ctx, schema, msg))
}

func TestClientErrors(t *testing.T) {
	t.Run("GetMetadata on non-existent table returns nil", func(t *testing.T) {
		projectID := testutil.GetEnvOrSkip(t, "TEST_BIGQUERY_PROJECT_ID")
		datasetID := testutil.GetEnvOrSkip(t, "TEST_BIGQUERY_DATASET_ID")

		ctx := context.Background()
		nonExistentTable := types.BQTableID("non_existent_table_999999")
		client, err := bq.New(ctx, types.GoogleProjectID(projectID), types.BQDatasetID(datasetID), nonExistentTable)
		gt.NoError(t, err)

		md, err := client.GetMetadata(ctx)
		gt.NoError(t, err)
		gt.V(t, md).Equal(nil)
	})

	t.Run("Insert with mismatched schema fails", func(t *testing.T) {
		projectID := testutil.GetEnvOrSkip(t, "TEST_BIGQUERY_PROJECT_ID")
		datasetID := testutil.GetEnvOrSkip(t, "TEST_BIGQUERY_DATASET_ID")

		ctx := context.Background()
		tblName := types.BQTableID(time.Now().Format("mismatch_test_20060102_150405"))
		client, err := bq.New(ctx, types.GoogleProjectID(projectID), types.BQDatasetID(datasetID), tblName)
		gt.NoError(t, err)

		// Create table with specific schema
		schema := bigquery.Schema{
			{Name: "field1", Type: bigquery.StringFieldType},
		}
		gt.NoError(t, client.CreateTable(ctx, &bigquery.TableMetadata{
			Name:   tblName.String(),
			Schema: schema,
		}))

		// Try to insert data with different structure
		wrongData := struct {
			WrongField int
		}{
			WrongField: 123,
		}

		err = client.Insert(ctx, schema, wrongData)
		gt.Error(t, err)
	})
}

func TestNewClient(t *testing.T) {
	t.Run("New creates client successfully", func(t *testing.T) {
		projectID := testutil.GetEnvOrSkip(t, "TEST_BIGQUERY_PROJECT_ID")
		datasetID := testutil.GetEnvOrSkip(t, "TEST_BIGQUERY_DATASET_ID")

		ctx := context.Background()
		tblName := types.BQTableID("test_table")
		client, err := bq.New(ctx, types.GoogleProjectID(projectID), types.BQDatasetID(datasetID), tblName)
		gt.NoError(t, err)
		gt.V(t, client).NotEqual(nil)
	})
}

func TestProtoFieldJSONName(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "keeps valid names",
			input: "SchemaVersion",
			want:  "SchemaVersion",
		},
		{
			name:  "renames invalid names",
			input: "ruby-advisory-db",
			want:  "col_cnVieS1hZHZpc29yeS1kYg",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := bq.ProtoFieldJSONName(tc.input); got != tc.want {
				t.Fatalf("unexpected name: want=%s, got=%s", tc.want, got)
			}
		})
	}
}

func TestSanitizeProtoJSON(t *testing.T) {
	t.Parallel()

	raw := []byte(`{"VendorSeverity":{"ruby-advisory-db":3,"nvd":2}}`)
	sanitized := gt.R1(bq.SanitizeProtoJSON(raw)).NoError(t)

	dec := json.NewDecoder(bytes.NewReader(sanitized))
	dec.UseNumber()
	payload := map[string]any{}
	gt.NoError(t, dec.Decode(&payload))

	vs, ok := payload["VendorSeverity"].(map[string]any)
	if !ok {
		t.Fatalf("VendorSeverity not found in %v", payload)
	}

	renamed := bq.ProtoFieldJSONName("ruby-advisory-db")

	if _, ok := vs[renamed]; !ok {
		t.Fatalf("sanitized key %s not found: %+v", renamed, vs)
	}
	if _, ok := vs["ruby-advisory-db"]; ok {
		t.Fatalf("unexpected original key remains: %+v", vs)
	}
}

func TestIsSchemaNotFoundError(t *testing.T) {
	t.Run("detects gRPC InvalidArgument with schema mismatch message", func(t *testing.T) {
		err := status.Error(codes.InvalidArgument, "Input schema has more fields than BigQuery schema, extra fields: 'field1,field2'")
		result := bq.IsSchemaNotFoundError(err)
		gt.V(t, result).Equal(true)
	})

	t.Run("detects wrapped gRPC error with goerr", func(t *testing.T) {
		baseErr := status.Error(codes.InvalidArgument, "Input schema has more fields than BigQuery schema, extra fields: 'field1'")
		wrappedErr := goerr.Wrap(baseErr, "failed to insert")
		result := bq.IsSchemaNotFoundError(wrappedErr)
		gt.V(t, result).Equal(true)
	})

	t.Run("returns false for InvalidArgument with different message", func(t *testing.T) {
		err := status.Error(codes.InvalidArgument, "Invalid request parameters")
		result := bq.IsSchemaNotFoundError(err)
		gt.V(t, result).Equal(false)
	})

	t.Run("returns false for different gRPC code", func(t *testing.T) {
		err := status.Error(codes.PermissionDenied, "Input schema has more fields than BigQuery schema, extra fields: 'field1'")
		result := bq.IsSchemaNotFoundError(err)
		gt.V(t, result).Equal(false)
	})

	t.Run("returns false for non-gRPC error", func(t *testing.T) {
		err := errors.New("some other error")
		result := bq.IsSchemaNotFoundError(err)
		gt.V(t, result).Equal(false)
	})

	t.Run("detects schema error in deeply nested goerr wrapping", func(t *testing.T) {
		baseErr := status.Error(codes.InvalidArgument, "Input schema has more fields than BigQuery schema, extra fields: 'a,b,c'")
		wrapped1 := goerr.Wrap(baseErr, "level 1")
		wrapped2 := goerr.Wrap(wrapped1, "level 2")
		wrapped3 := goerr.Wrap(wrapped2, "level 3")
		result := bq.IsSchemaNotFoundError(wrapped3)
		gt.V(t, result).Equal(true)
	})
}

func TestSchemaUpdateAndRetry(t *testing.T) {
	projectID := testutil.GetEnvOrSkip(t, "TEST_BIGQUERY_PROJECT_ID")
	datasetID := testutil.GetEnvOrSkip(t, "TEST_BIGQUERY_DATASET_ID")

	ctx := context.Background()
	tblName := types.BQTableID(time.Now().Format("schema_retry_test_20060102_150405"))
	client, err := bq.New(ctx, types.GoogleProjectID(projectID), types.BQDatasetID(datasetID), tblName)
	gt.NoError(t, err)

	t.Run("schema update and insert with retry", func(t *testing.T) {
		// Step 1: Create table with initial schema
		initialSchema := bigquery.Schema{
			{Name: "id", Type: bigquery.StringFieldType},
			{Name: "value", Type: bigquery.IntegerFieldType},
		}
		gt.NoError(t, client.CreateTable(ctx, &bigquery.TableMetadata{
			Name:   tblName.String(),
			Schema: initialSchema,
		}))

		// Step 2: Update schema to add new field
		md := gt.R1(client.GetMetadata(ctx)).NoError(t)
		gt.V(t, md).NotEqual(nil)

		updatedSchema := bigquery.Schema{
			{Name: "id", Type: bigquery.StringFieldType},
			{Name: "value", Type: bigquery.IntegerFieldType},
			{Name: "new_field", Type: bigquery.StringFieldType},
		}
		gt.NoError(t, client.UpdateTable(ctx, bigquery.TableMetadataToUpdate{
			Schema: updatedSchema,
		}, md.ETag))

		// Step 3: Insert data with schemaUpdated=true in context
		data := struct {
			ID       string `json:"id"`
			Value    int    `json:"value"`
			NewField string `json:"new_field"`
		}{
			ID:       "test-1",
			Value:    42,
			NewField: "test value",
		}

		// Insert with retry option to enable retry logic
		err = client.Insert(ctx, updatedSchema, data, interfaces.WithRetry(true))

		// The insert should succeed (either immediately or after retry)
		gt.NoError(t, err)
	})
}
