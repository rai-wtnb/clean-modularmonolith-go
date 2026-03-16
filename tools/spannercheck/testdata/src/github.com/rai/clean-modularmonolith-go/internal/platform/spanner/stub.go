package spanner

import (
	"context"

	cloudspanner "cloud.google.com/go/spanner"
)

func ReadWriteTxFromContext(ctx context.Context) (*cloudspanner.ReadWriteTransaction, bool) {
	return nil, false
}

func ReadTransactionFromContext(ctx context.Context) (interface{}, bool) {
	return nil, false
}

func Write(ctx context.Context, client *cloudspanner.Client, stmts ...cloudspanner.Statement) error {
	return nil
}

func Read[T any](ctx context.Context, client *cloudspanner.Client, fn func(context.Context, interface{}) (T, error)) (T, error) {
	var zero T
	return zero, nil
}

func ReadConsistent[T any](ctx context.Context, client *cloudspanner.Client, fn func(context.Context, interface{}) (T, error)) (T, error) {
	var zero T
	return zero, nil
}
