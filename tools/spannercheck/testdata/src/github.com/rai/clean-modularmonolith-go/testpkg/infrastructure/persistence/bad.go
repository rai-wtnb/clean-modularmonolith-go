package persistence

import (
	"context"

	"cloud.google.com/go/spanner"
	platformspanner "github.com/rai/clean-modularmonolith-go/internal/platform/spanner"
)

type Repo struct {
	client *spanner.Client
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

func (r *Repo) badReadWriteTxFromContext(ctx context.Context) {
	platformspanner.ReadWriteTxFromContext(ctx) // want `direct call to ReadWriteTxFromContext in persistence package`
}

func (r *Repo) badReadTransactionFromContext(ctx context.Context) {
	platformspanner.ReadTransactionFromContext(ctx) // want `direct call to ReadTransactionFromContext in persistence package`
}
