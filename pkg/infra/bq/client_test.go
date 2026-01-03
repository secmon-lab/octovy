package bq_test

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"cloud.google.com/go/bigquery"
	"github.com/m-mizutani/bqs"
	"github.com/m-mizutani/gt"
	"github.com/m-mizutani/octovy/pkg/domain/model"
	"github.com/m-mizutani/octovy/pkg/domain/types"
	"github.com/m-mizutani/octovy/pkg/infra/bq"
	"github.com/m-mizutani/octovy/pkg/utils/testutil"
	"google.golang.org/api/impersonate"
	"google.golang.org/api/option"
)

func TestClient(t *testing.T) {
	projectID := testutil.GetEnvOrSkip(t, "TEST_BIGQUERY_PROJECT_ID")
	datasetID := testutil.GetEnvOrSkip(t, "TEST_BIGQUERY_DATASET_ID")

	ctx := context.Background()

	tblName := types.BQTableID(time.Now().Format("insert_test_20060102_150405"))
	client, err := bq.New(ctx, types.GoogleProjectID(projectID), types.BQDatasetID(datasetID))
	gt.NoError(t, err)

	var baseSchema bigquery.Schema

	t.Run("Create base table at first", func(t *testing.T) {
		var scan model.Scan
		baseSchema = gt.R1(bqs.Infer(scan)).NoError(t)
		gt.NoError(t, err)

		gt.NoError(t, client.CreateTable(ctx, tblName, &bigquery.TableMetadata{
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

		md := gt.R1(client.GetMetadata(ctx, tblName)).NoError(t)
		gt.False(t, bqs.Equal(mergedSchema, baseSchema))
		gt.NoError(t, client.UpdateTable(ctx, tblName, bigquery.TableMetadataToUpdate{
			Schema: mergedSchema,
		}, md.ETag))

		record := model.ScanRawRecord{
			Scan:      scan,
			Timestamp: scan.Timestamp.UnixMicro(),
		}
		gt.NoError(t, client.Insert(ctx, tblName, mergedSchema, record))
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

	client, err := bq.New(ctx, types.GoogleProjectID(projectID), types.BQDatasetID(datasetID), option.WithTokenSource(ts))
	gt.NoError(t, err)

	msg := struct {
		Msg string
	}{
		Msg: "Hello, BigQuery: " + time.Now().String(),
	}

	schema := gt.R1(bqs.Infer(msg)).NoError(t)

	tblName := types.BQTableID(time.Now().Format("impersonation_test_20060102_150405"))
	gt.NoError(t, client.CreateTable(ctx, tblName, &bigquery.TableMetadata{
		Name:   tblName.String(),
		Schema: schema,
	}))

	gt.NoError(t, client.Insert(ctx, tblName, schema, msg))
}

func TestClientErrors(t *testing.T) {
	t.Run("GetMetadata on non-existent table returns nil", func(t *testing.T) {
		projectID := testutil.GetEnvOrSkip(t, "TEST_BIGQUERY_PROJECT_ID")
		datasetID := testutil.GetEnvOrSkip(t, "TEST_BIGQUERY_DATASET_ID")

		ctx := context.Background()
		client, err := bq.New(ctx, types.GoogleProjectID(projectID), types.BQDatasetID(datasetID))
		gt.NoError(t, err)

		nonExistentTable := types.BQTableID("non_existent_table_999999")
		md, err := client.GetMetadata(ctx, nonExistentTable)
		gt.NoError(t, err)
		gt.V(t, md).Equal(nil)
	})

	t.Run("Insert with mismatched schema fails", func(t *testing.T) {
		projectID := testutil.GetEnvOrSkip(t, "TEST_BIGQUERY_PROJECT_ID")
		datasetID := testutil.GetEnvOrSkip(t, "TEST_BIGQUERY_DATASET_ID")

		ctx := context.Background()
		client, err := bq.New(ctx, types.GoogleProjectID(projectID), types.BQDatasetID(datasetID))
		gt.NoError(t, err)

		tblName := types.BQTableID(time.Now().Format("mismatch_test_20060102_150405"))

		// Create table with specific schema
		schema := bigquery.Schema{
			{Name: "field1", Type: bigquery.StringFieldType},
		}
		gt.NoError(t, client.CreateTable(ctx, tblName, &bigquery.TableMetadata{
			Name:   tblName.String(),
			Schema: schema,
		}))

		// Try to insert data with different structure
		wrongData := struct {
			WrongField int
		}{
			WrongField: 123,
		}

		err = client.Insert(ctx, tblName, schema, wrongData)
		gt.Error(t, err)
	})
}

func TestNewClient(t *testing.T) {
	t.Run("New creates client successfully", func(t *testing.T) {
		projectID := testutil.GetEnvOrSkip(t, "TEST_BIGQUERY_PROJECT_ID")
		datasetID := testutil.GetEnvOrSkip(t, "TEST_BIGQUERY_DATASET_ID")

		ctx := context.Background()
		client, err := bq.New(ctx, types.GoogleProjectID(projectID), types.BQDatasetID(datasetID))
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
