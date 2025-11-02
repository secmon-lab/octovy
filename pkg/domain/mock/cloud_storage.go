package mock

import (
	"bytes"
	"context"
	"encoding/json"
	"io"

	"github.com/m-mizutani/goerr/v2"
	"github.com/m-mizutani/octovy/pkg/domain/interfaces"
)

type StorageMock struct {
	Data map[string][]byte
}

var _ interfaces.Storage = (*StorageMock)(nil)

func NewStorageMock() *StorageMock {
	return &StorageMock{
		Data: make(map[string][]byte),
	}
}

// Get implements Storage.
func (s *StorageMock) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	if data, ok := s.Data[key]; ok {
		return io.NopCloser(bytes.NewReader(data)), nil
	}
	return nil, nil
}

// Put implements Storage.
func (s *StorageMock) Put(ctx context.Context, key string, r io.ReadCloser) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	s.Data[key] = data
	return nil
}

func (s *StorageMock) Unmarshal(key string, v interface{}) error {
	data, ok := s.Data[key]
	if !ok {
		return io.EOF
	}

	if err := json.Unmarshal(data, v); err != nil {
		return goerr.Wrap(err, "Failed to unmarshal data")
	}

	return nil
}
