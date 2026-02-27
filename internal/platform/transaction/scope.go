// Package transaction provides transaction management abstractions.
package transaction

import "context"

// TransactionScope manages the lifecycle of a transaction.
// It provides a clean abstraction for executing business logic
// within a transactional boundary.
type TransactionScope interface {
	// Execute runs the given function within a transaction.
	// The transaction is committed if fn returns nil, rolled back otherwise.
	// The ctx passed to fn contains the transaction for repositories to use.
	Execute(ctx context.Context, fn func(ctx context.Context) error) error
}

// ExecuteWithResult runs fn within a transaction and returns the result.
// This is a generic helper that wraps TransactionScope.Execute for cases
// where the transaction needs to return a value.
func ExecuteWithResult[T any](ctx context.Context, scope TransactionScope, fn func(ctx context.Context) (T, error)) (T, error) {
	var result T
	err := scope.Execute(ctx, func(ctx context.Context) error {
		var fnErr error
		result, fnErr = fn(ctx)
		return fnErr
	})
	return result, err
}

// この設計の利点:
// - 実装側の変更不要: 既存の TransactionScope 実装をそのまま使える
// - 型安全: ジェネリクスの恩恵を受けられる
// - Goの制約に準拠: インターフェースにジェネリックメソッドを追加できない制約を回避
// これはGoでよく使われるパターンで、標準ライブラリでも sort.Slice などが同様のアプローチを取っています。
