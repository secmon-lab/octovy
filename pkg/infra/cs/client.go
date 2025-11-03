// Google Cloud Storage client
package cs

import (
	"compress/gzip"
	"context"
	"io"

	"cloud.google.com/go/storage"
	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/octovy/pkg/domain/interfaces"
	"github.com/m-mizutani/octovy/pkg/utils"
)

type Client struct {
	bucket string
	prefix string
	client *storage.Client
}

var _ interfaces.Storage = (*Client)(nil)

func New(ctx context.Context, bucket string, options ...Option) (*Client, error) {
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, err
	}

	return &Client{
		bucket: bucket,
		client: client,
	}, nil
}

// Option is a functional option for New function
type Option func(*Client)

func WithPrefix(prefix string) Option {
	return func(c *Client) {
		c.prefix = prefix
	}
}

// Get implements interfaces.Storage.
func (c *Client) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	obj := c.client.Bucket(c.bucket).Object(c.prefix + key)
	r, err := obj.NewReader(ctx)
	if err != nil {
		// check if the object does not exist
		if err == storage.ErrObjectNotExist {
			return nil, nil
		}
		return nil, goerr.Wrap(err, "Failed to create object reader")
	}

	return r, nil
}

// Put implements interfaces.Storage.
func (c *Client) Put(ctx context.Context, key string, r io.ReadCloser) error {
	obj := c.client.Bucket(c.bucket).Object(c.prefix + key)
	w := obj.NewWriter(ctx)
	w.ContentType = "application/json"
	w.ContentEncoding = "gzip"

	zw := gzip.NewWriter(w)

	defer func() {
		utils.SafeClose(zw)
		utils.SafeClose(w)
	}()

	if _, err := io.Copy(zw, r); err != nil {
		return goerr.Wrap(err, "Failed to write object")
	}

	return nil
}
