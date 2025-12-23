package log

import (
	"context"
	"testing"
)

func TestContextWithLogger(t *testing.T) {
	ctx := context.Background()
	logger := Default()

	newCtx := ContextWithLogger(ctx, logger)

	got := LoggerFromContext(newCtx)
	if got == nil {
		t.Fatal("LoggerFromContext should return logger")
	}
}

func TestLoggerFromContext_Empty(t *testing.T) {
	ctx := context.Background()

	got := LoggerFromContext(ctx)
	if got == nil {
		t.Fatal("LoggerFromContext should return default logger for empty context")
	}
}

func TestL(t *testing.T) {
	ctx := context.Background()
	logger := Default()
	ctx = ContextWithLogger(ctx, logger)

	got := L(ctx)
	if got == nil {
		t.Fatal("L() should return logger from context")
	}
}

func TestL_WithCustomLogger(t *testing.T) {
	ctx := context.Background()
	customLogger := Default().With(String("custom", "field"))
	ctx = ContextWithLogger(ctx, customLogger)

	logger := L(ctx)
	if logger == nil {
		t.Fatal("L() should return custom logger")
	}

	logger.Info("test with custom logger")
}
