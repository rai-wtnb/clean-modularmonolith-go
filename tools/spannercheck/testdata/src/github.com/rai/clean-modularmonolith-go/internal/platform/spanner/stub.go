package spanner

import (
	"context"

	cloudspanner "cloud.google.com/go/spanner"
)

func Write(ctx context.Context, client *cloudspanner.Client, stmts ...cloudspanner.Statement) error {
	return nil
}

func SingleRead[T any](ctx context.Context, client *cloudspanner.Client, fn func(context.Context, interface{}) (T, error)) (T, error) {
	var zero T
	return zero, nil
}

func ConsistentRead[T any](ctx context.Context, client *cloudspanner.Client, fn func(context.Context, interface{}) (T, error)) (T, error) {
	var zero T
	return zero, nil
}
