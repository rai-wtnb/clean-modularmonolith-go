package spanner

import (
	"context"
	"log/slog"

	cloudspanner "cloud.google.com/go/spanner"
)

func Write(ctx context.Context, stmts ...cloudspanner.Statement) error {
	return nil
}

func ReadOrSingle[T any](ctx context.Context, client *cloudspanner.Client, logger *slog.Logger, fn func(context.Context, interface{}) (T, error)) (T, error) {
	var zero T
	return zero, nil
}

func Read[T any](ctx context.Context, client *cloudspanner.Client, logger *slog.Logger, fn func(context.Context, interface{}) (T, error)) (T, error) {
	var zero T
	return zero, nil
}
