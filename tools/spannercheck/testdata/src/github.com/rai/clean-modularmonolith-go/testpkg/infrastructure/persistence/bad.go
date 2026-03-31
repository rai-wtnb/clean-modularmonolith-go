package persistence

import (
	"context"
	"log/slog"

	"cloud.google.com/go/spanner"
)

type Repo struct {
	client *spanner.Client
	logger *slog.Logger
}

func (r *Repo) badApply(ctx context.Context) {
	r.client.Apply(ctx, nil) // want `direct call to \(\*spanner\.Client\)\.Apply in persistence package`
}

func (r *Repo) badSingle(ctx context.Context) {
	r.client.Single() // want `direct call to \(\*spanner\.Client\)\.Single in persistence package`
}

func (r *Repo) badReadOnlyTransaction(ctx context.Context) {
	r.client.ReadOnlyTransaction() // want `direct call to \(\*spanner\.Client\)\.ReadOnlyTransaction in persistence package`
}

func (r *Repo) badReadWriteTransaction(ctx context.Context) {
	r.client.ReadWriteTransaction(ctx, nil) // want `direct call to \(\*spanner\.Client\)\.ReadWriteTransaction in persistence package`
}

func (r *Repo) badBufferWrite(ctx context.Context, tx *spanner.ReadWriteTransaction) {
	tx.BufferWrite(nil) // want `direct call to \(\*spanner\.ReadWriteTransaction\)\.BufferWrite in persistence package`
}

