package log

import (
	"context"
)

// loggerKey is the context key for storing Logger
type loggerKey struct{}

// ContextWithLogger stores Logger in context
func ContextWithLogger(ctx context.Context, logger Logger) context.Context {
	return context.WithValue(ctx, loggerKey{}, logger)
}

// LoggerFromContext retrieves Logger from context, returns default if not found
func LoggerFromContext(ctx context.Context) Logger {
	if logger, ok := ctx.Value(loggerKey{}).(Logger); ok {
		return logger
	}
	return Default()
}

// L is a shortcut for LoggerFromContext
func L(ctx context.Context) Logger {
	return LoggerFromContext(ctx)
}
