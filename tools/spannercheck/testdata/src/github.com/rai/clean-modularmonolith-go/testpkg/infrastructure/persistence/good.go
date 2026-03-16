package persistence

import (
	"context"

	"cloud.google.com/go/spanner"
	platformspanner "github.com/rai/clean-modularmonolith-go/internal/platform/spanner"
)

// All of these should NOT trigger any diagnostics.

func (r *Repo) goodWrite(ctx context.Context) {
	platformspanner.Write(ctx, r.client, spanner.Statement{})
}

func (r *Repo) goodWriteMultiple(ctx context.Context) {
	platformspanner.Write(ctx, r.client, spanner.Statement{}, spanner.Statement{})
}

func (r *Repo) goodRead(ctx context.Context) {
	platformspanner.Read(ctx, r.client, func(_ context.Context, _ interface{}) (interface{}, error) {
		return nil, nil
	})
}

func (r *Repo) goodReadConsistent(ctx context.Context) {
	platformspanner.ReadConsistent(ctx, r.client, func(_ context.Context, _ interface{}) (interface{}, error) {
		return nil, nil
	})
}

// Comments and strings should NOT trigger diagnostics.
// client.Apply, ReadWriteTxFromContext, BufferWrite
var _ = "client.Apply is fine in a string"
