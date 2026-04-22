package lorm

import (
	"context"
	"log/slog"
	"os"
)

// Logger is the minimal structured logger interface used by Engine.
type Logger interface {
	DebugContext(ctx context.Context, msg string, args ...any)
	InfoContext(ctx context.Context, msg string, args ...any)
	WarnContext(ctx context.Context, msg string, args ...any)
	ErrorContext(ctx context.Context, msg string, args ...any)
}

var defaultLogger = slog.New(slog.NewJSONHandler(os.Stdout, nil))

type noopLogger struct{}

func (noopLogger) DebugContext(ctx context.Context, msg string, args ...any) {}
func (noopLogger) InfoContext(ctx context.Context, msg string, args ...any)  {}
func (noopLogger) WarnContext(ctx context.Context, msg string, args ...any)  {}
func (noopLogger) ErrorContext(ctx context.Context, msg string, args ...any) {}
