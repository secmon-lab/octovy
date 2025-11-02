package bq

import (
	"context"
	"encoding/json"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/bigquery/storage/managedwriter"
	"cloud.google.com/go/bigquery/storage/managedwriter/adapt"
	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/octovy/pkg/domain/interfaces"
	"github.com/m-mizutani/octovy/pkg/domain/types"
	"github.com/m-mizutani/octovy/pkg/utils"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

type Client struct {
	bqClient *bigquery.Client
	mwClient *managedwriter.Client
	project  string
	dataset  string
}

var _ interfaces.BigQuery = (*Client)(nil)

func New(ctx context.Context, projectID types.GoogleProjectID, datasetID types.BQDatasetID, options ...option.ClientOption) (*Client, error) {
	mwClient, err := managedwriter.NewClient(ctx, projectID.String(), options...)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create bigquery client", goerr.V("projectID", projectID))
	}

	bqClient, err := bigquery.NewClient(ctx, string(projectID), options...)
	if err != nil {
		return nil, goerr.Wrap(err, "failed to create BigQuery client", goerr.V("projectID", projectID))
	}

	return &Client{
		bqClient: bqClient,
		mwClient: mwClient,
		project:  projectID.String(),
		dataset:  datasetID.String(),
	}, nil
}

// CreateTable implements interfaces.BigQuery.
func (x *Client) CreateTable(ctx context.Context, table types.BQTableID, md *bigquery.TableMetadata) error {
	if err := x.bqClient.Dataset(x.dataset).Table(table.String()).Create(ctx, md); err != nil {
		return goerr.Wrap(err, "failed to create table", goerr.V("dataset", x.dataset), goerr.V("table", table))
	}
	return nil
}

// GetMetadata implements interfaces.BigQuery. If the table does not exist, it returns nil.
func (x *Client) GetMetadata(ctx context.Context, table types.BQTableID) (*bigquery.TableMetadata, error) {
	md, err := x.bqClient.Dataset(x.dataset).Table(table.String()).Metadata(ctx)
	if err != nil {
		if gErr, ok := err.(*googleapi.Error); ok && gErr.Code == 404 {
			return nil, nil
		}
		return nil, goerr.Wrap(err, "failed to get table metadata", goerr.V("dataset", x.dataset), goerr.V("table", table))
	}

	return md, nil
}

// Insert implements interfaces.BigQuery.
func (x *Client) Insert(ctx context.Context, table types.BQTableID, schema bigquery.Schema, data any) error {
	convertedSchema, err := adapt.BQSchemaToStorageTableSchema(schema)
	if err != nil {
		return goerr.Wrap(err, "failed to convert schema")
	}

	descriptor, err := adapt.StorageSchemaToProto2Descriptor(convertedSchema, "root")
	if err != nil {
		return goerr.Wrap(err, "failed to convert schema to descriptor")
	}
	messageDescriptor, ok := descriptor.(protoreflect.MessageDescriptor)
	if !ok {
		return goerr.Wrap(err, "adapted descriptor is not a message descriptor")
	}
	descriptorProto, err := adapt.NormalizeDescriptor(messageDescriptor)
	if err != nil {
		return goerr.Wrap(err, "failed to normalize descriptor")
	}

	message := dynamicpb.NewMessage(messageDescriptor)

	raw, err := json.Marshal(data)
	if err != nil {
		return goerr.Wrap(err, "failed to Marshal json message", goerr.V("v", data))
	}

	// First, json->proto message
	err = protojson.Unmarshal(raw, message)
	if err != nil {
		return goerr.Wrap(err, "failed to Unmarshal json message", goerr.V("raw", string(raw)))
	}
	// Then, proto message -> bytes.
	b, err := proto.Marshal(message)
	if err != nil {
		return goerr.Wrap(err, "failed to Marshal proto message")
	}

	rows := [][]byte{b}

	ms, err := x.mwClient.NewManagedStream(ctx,
		managedwriter.WithDestinationTable(
			managedwriter.TableParentFromParts(
				x.project,
				x.dataset,
				table.String(),
			),
		),
		// managedwriter.WithType(managedwriter.CommittedStream),
		managedwriter.WithSchemaDescriptor(descriptorProto),
	)
	if err != nil {
		return goerr.Wrap(err, "failed to create managed stream")
	}
	defer utils.SafeClose(ms)

	arResult, err := ms.AppendRows(ctx, rows)
	if err != nil {
		return goerr.Wrap(err, "failed to append rows")
	}

	if _, err := arResult.FullResponse(ctx); err != nil {
		return goerr.Wrap(err, "failed to get append result")
	}

	return nil
}

// UpdateTable implements interfaces.BigQuery.
func (x *Client) UpdateTable(ctx context.Context, table types.BQTableID, md bigquery.TableMetadataToUpdate, eTag string) error {
	if _, err := x.bqClient.Dataset(x.dataset).Table(table.String()).Update(ctx, md, eTag); err != nil {
		return goerr.Wrap(err, "failed to update table", goerr.V("dataset", x.dataset), goerr.V("table", table), goerr.V("meta", md))
	}

	return nil
}
