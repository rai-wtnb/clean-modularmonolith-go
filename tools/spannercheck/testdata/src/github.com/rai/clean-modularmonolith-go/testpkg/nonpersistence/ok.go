package nonpersistence

import (
	"context"

	"cloud.google.com/go/spanner"
	platformspanner "github.com/rai/clean-modularmonolith-go/internal/platform/spanner"
)

// No diagnostics expected — this is NOT a persistence package.

func rawCallsAreOKHere(ctx context.Context, client *spanner.Client) {
	client.Apply(ctx, nil)
	client.Single()
	client.ReadOnlyTransaction()
	client.ReadWriteTransaction(ctx, nil)
	platformspanner.ReadWriteTxFromContext(ctx)
	platformspanner.ReadTransactionFromContext(ctx)
}
