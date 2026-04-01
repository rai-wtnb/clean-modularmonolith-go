package spanner

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"runtime"
	"strings"
	"time"

	"go.opentelemetry.io/otel/trace"
)

// businessCaller walks the call stack and returns the first frame outside
// the transaction/events infrastructure packages. This finds the actual
// command handler, query handler, or event handler that initiated the transaction.
// Returns a "caller" attribute in "package/file.go:line" format.
func businessCaller() slog.Attr {
	var pcs [10]uintptr
	n := runtime.Callers(2, pcs[:])
	frames := runtime.CallersFrames(pcs[:n])
	for {
		frame, more := frames.Next()
		if !strings.Contains(frame.Function, "internal/platform/spanner") &&
			!strings.Contains(frame.Function, "modules/shared/transaction") &&
			!strings.Contains(frame.Function, "modules/shared/events") {
			return slog.String("caller", shortFileLine(frame.File, frame.Line))
		}
		if !more {
			break
		}
	}
	return slog.String("caller", "unknown")
}

// shortFileLine returns "parent/file.go:line" from an absolute path.
func shortFileLine(file string, line int) string {
	// Find the last two path segments: "commands/create_user.go"
	short := file
	if i := strings.LastIndex(file, "/"); i >= 0 {
		if j := strings.LastIndex(file[:i], "/"); j >= 0 {
			short = file[j+1:]
		}
	}
	return fmt.Sprintf("%s:%d", short, line)
}

// transactionType represents the type of Spanner transaction for structured logging.
type transactionType string

const (
	TxReadWrite  transactionType = "read-write"
	TxReadOnly   transactionType = "read-only"
	TxSingleRead transactionType = "single-read"
)

// txLog logs "transaction starting" and returns a function to log the outcome.
// It uses businessCaller() to find the first frame outside infrastructure packages,
// so it works correctly from both Scope.Execute and standalone functions (Write/SingleRead/ConsistentRead).
func txLog(ctx context.Context, logger *slog.Logger, txType transactionType, op string) (finishLog func(error)) {
	var b [4]byte
	_, _ = rand.Read(b[:])
	txID := hex.EncodeToString(b[:])

	caller := businessCaller()
	args := []any{
		slog.String("tx_id", txID),
		slog.String("transaction_type", string(txType)),
		slog.String("op", op),
		caller,
	}

	if traceID := trace.SpanContextFromContext(ctx).TraceID(); traceID.IsValid() {
		args = append(args, slog.String("trace_id", traceID.String()))
	}

	logger.InfoContext(ctx, "transaction starting", args...)
	start := time.Now()

	return func(err error) {
		doneArgs := make([]any, 0, len(args)+2)
		doneArgs = append(doneArgs, args...)
		doneArgs = append(doneArgs, slog.Duration("duration", time.Since(start)))

		if err != nil {
			doneArgs = append(doneArgs, slog.Any("error", err))
			logger.ErrorContext(ctx, "transaction failed", doneArgs...)
			return
		}

		logger.InfoContext(ctx, "transaction finished", doneArgs...)
	}
}
