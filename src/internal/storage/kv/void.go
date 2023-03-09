package kv

import (
	"context"

	"github.com/pachyderm/pachyderm/v2/src/internal/pacherr"
	"github.com/pachyderm/pachyderm/v2/src/internal/stream"
)

type Void struct{}

func (v Void) Put(ctx context.Context, key, value []byte) error {
	return nil
}

func (v Void) Exists(ctx context.Context, key []byte) (bool, error) {
	return false, nil
}

func (v Void) Get(ctx context.Context, key, buf []byte) (int, error) {
	return 0, pacherr.NewNotExist("kv.Void", string(key))
}

func (v Void) Delete(ctx context.Context, key []byte) error {
	return nil
}

func (v Void) NewKeyIterator(span Span) stream.Iterator[[]byte] {
	return stream.NewSlice[[]byte](nil)
}
