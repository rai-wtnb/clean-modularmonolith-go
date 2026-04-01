package spanner

import (
	"context"
	"log/slog"
	"time"
)

// txLog logs "transaction starting" and returns a function to log the outcome.
// The caller captures file/line via runtime.Caller(1) and passes them explicitly,
// avoiding fragile caller-skip depth assumptions.
func txLog(
	ctx context.Context,
	logger *slog.Logger,
	file string, line int,
	successMsg, errorMsg string,
	attrs ...slog.Attr,
) (finishLog func(error)) {
	caller := slog.Group("caller", slog.String("file", file), slog.Int("line", line))

	args := make([]any, 0, len(attrs)+1)
	for _, a := range attrs {
		args = append(args, a)
	}
	args = append(args, caller)

	logger.InfoContext(ctx, "transaction starting", args...)
	start := time.Now()

	return func(err error) {
		duration := time.Since(start)
		doneArgs := make([]any, 0, len(args)+2)
		doneArgs = append(doneArgs, slog.Duration("duration", duration))
		if err != nil {
			doneArgs = append(doneArgs, slog.Any("error", err))
		}
		doneArgs = append(doneArgs, args...)

		if err != nil {
			logger.ErrorContext(ctx, errorMsg, doneArgs...)
		} else {
			logger.InfoContext(ctx, successMsg, doneArgs...)
		}
	}
}
