package transaction_test

import (
	"context"
	"errors"
	"testing"

	"github.com/rai/clean-modularmonolith-go/internal/platform/transaction"
)

type mockTransactionScope struct {
	executeFn func(ctx context.Context, fn func(ctx context.Context) error) error
}

func (m *mockTransactionScope) Execute(ctx context.Context, fn func(ctx context.Context) error) error {
	return m.executeFn(ctx, fn)
}

func TestExecuteWithResult_Success(t *testing.T) {
	scope := &mockTransactionScope{
		executeFn: func(ctx context.Context, fn func(ctx context.Context) error) error {
			return fn(ctx)
		},
	}

	result, err := transaction.ExecuteWithResult(context.Background(), scope, func(ctx context.Context) (string, error) {
		return "success", nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "success" {
		t.Errorf("expected 'success', got %q", result)
	}
}

func TestExecuteWithResult_FnError(t *testing.T) {
	scope := &mockTransactionScope{
		executeFn: func(ctx context.Context, fn func(ctx context.Context) error) error {
			return fn(ctx)
		},
	}

	errFn := errors.New("fn error")
	result, err := transaction.ExecuteWithResult(context.Background(), scope, func(ctx context.Context) (string, error) {
		return "", errFn
	})

	if !errors.Is(err, errFn) {
		t.Errorf("expected errFn, got %v", err)
	}
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestExecuteWithResult_TransactionError(t *testing.T) {
	errTx := errors.New("transaction error")
	scope := &mockTransactionScope{
		executeFn: func(ctx context.Context, fn func(ctx context.Context) error) error {
			_ = fn(ctx) // fnは成功するが、トランザクション自体が失敗
			return errTx
		},
	}

	result, err := transaction.ExecuteWithResult(context.Background(), scope, func(ctx context.Context) (int, error) {
		return 42, nil
	})

	if !errors.Is(err, errTx) {
		t.Errorf("expected errTx, got %v", err)
	}
	// 結果は設定されているが、エラーがあるので使用すべきでない
	if result != 42 {
		t.Errorf("expected 42, got %d", result)
	}
}

func TestExecuteWithResult_StructResult(t *testing.T) {
	type Result struct {
		ID   string
		Name string
	}

	scope := &mockTransactionScope{
		executeFn: func(ctx context.Context, fn func(ctx context.Context) error) error {
			return fn(ctx)
		},
	}

	result, err := transaction.ExecuteWithResult(context.Background(), scope, func(ctx context.Context) (Result, error) {
		return Result{ID: "123", Name: "test"}, nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != "123" || result.Name != "test" {
		t.Errorf("unexpected result: %+v", result)
	}
}
