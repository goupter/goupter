package database

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/goupter/goupter/pkg/config"
	"github.com/goupter/goupter/pkg/log"
)

type mockLogger struct {
	messages []string
	level    log.Level
}

func (m *mockLogger) Debug(msg string, fields ...log.Field) {
	m.messages = append(m.messages, "DEBUG: "+msg)
}

func (m *mockLogger) Info(msg string, fields ...log.Field) {
	m.messages = append(m.messages, "INFO: "+msg)
}

func (m *mockLogger) Warn(msg string, fields ...log.Field) {
	m.messages = append(m.messages, "WARN: "+msg)
}

func (m *mockLogger) Error(msg string, fields ...log.Field) {
	m.messages = append(m.messages, "ERROR: "+msg)
}

func (m *mockLogger) Fatal(msg string, fields ...log.Field) {
	m.messages = append(m.messages, "FATAL: "+msg)
}

func (m *mockLogger) With(fields ...log.Field) log.Logger       { return m }
func (m *mockLogger) WithContext(ctx context.Context) log.Logger { return m }
func (m *mockLogger) SetLevel(level log.Level)                   { m.level = level }
func (m *mockLogger) GetLevel() log.Level                        { return m.level }
func (m *mockLogger) Sync() error                                { return nil }

func TestNewSlowQueryLogger(t *testing.T) {
	mockLog := &mockLogger{}
	cfg := &config.DatabaseConfig{
		LogLevel:           "info",
		SlowQueryThreshold: 100 * time.Millisecond,
	}

	sqLogger := NewSlowQueryLogger(mockLog, cfg)
	if sqLogger == nil {
		t.Fatal("NewSlowQueryLogger should return non-nil")
	}
}

func TestNewSlowQueryLogger_DefaultThreshold(t *testing.T) {
	mockLog := &mockLogger{}
	cfg := &config.DatabaseConfig{
		LogLevel:           "warn",
		SlowQueryThreshold: 0,
	}

	sqLogger := NewSlowQueryLogger(mockLog, cfg)
	if sqLogger.threshold != 200*time.Millisecond {
		t.Errorf("Default threshold = %v, want 200ms", sqLogger.threshold)
	}
}

func TestSlowQueryLogger_LogMode(t *testing.T) {
	mockLog := &mockLogger{}
	cfg := &config.DatabaseConfig{LogLevel: "warn"}
	sqLogger := NewSlowQueryLogger(mockLog, cfg)

	newLogger := sqLogger.LogMode(4) // logger.Info
	if newLogger == sqLogger {
		t.Error("LogMode should return new instance")
	}
}

func TestSlowQueryLogger_LogMethods(t *testing.T) {
	mockLog := &mockLogger{}
	cfg := &config.DatabaseConfig{LogLevel: "info"}
	sqLogger := NewSlowQueryLogger(mockLog, cfg)
	ctx := context.Background()

	sqLogger.Info(ctx, "test info %s", "message")
	sqLogger.Warn(ctx, "test warn %s", "message")
	sqLogger.Error(ctx, "test error %s", "message")

	if len(mockLog.messages) != 3 {
		t.Errorf("messages count = %d, want 3", len(mockLog.messages))
	}
}

func TestSlowQueryLogger_Trace(t *testing.T) {
	t.Run("slow query", func(t *testing.T) {
		mockLog := &mockLogger{}
		cfg := &config.DatabaseConfig{
			LogLevel:           "warn",
			SlowQueryThreshold: 100 * time.Millisecond,
		}
		sqLogger := NewSlowQueryLogger(mockLog, cfg)

		ctx := context.Background()
		begin := time.Now().Add(-200 * time.Millisecond)
		fc := func() (string, int64) { return "SELECT * FROM users", 10 }

		sqLogger.Trace(ctx, begin, fc, nil)

		found := false
		for _, msg := range mockLog.messages {
			if msg == "WARN: SQL slow query" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Should log slow query warning")
		}
	})

	t.Run("fast query", func(t *testing.T) {
		mockLog := &mockLogger{}
		cfg := &config.DatabaseConfig{
			LogLevel:           "warn",
			SlowQueryThreshold: 500 * time.Millisecond,
		}
		sqLogger := NewSlowQueryLogger(mockLog, cfg)

		ctx := context.Background()
		begin := time.Now().Add(-100 * time.Millisecond)
		fc := func() (string, int64) { return "SELECT 1", 1 }

		sqLogger.Trace(ctx, begin, fc, nil)

		for _, msg := range mockLog.messages {
			if msg == "WARN: SQL slow query" {
				t.Error("Fast query should not trigger slow query warning")
			}
		}
	})

	t.Run("query with error", func(t *testing.T) {
		mockLog := &mockLogger{}
		cfg := &config.DatabaseConfig{
			LogLevel:           "error",
			SlowQueryThreshold: 100 * time.Millisecond,
		}
		sqLogger := NewSlowQueryLogger(mockLog, cfg)

		ctx := context.Background()
		begin := time.Now().Add(-200 * time.Millisecond)
		fc := func() (string, int64) { return "SELECT * FROM nonexistent", 0 }

		sqLogger.Trace(ctx, begin, fc, errors.New("table not found"))

		found := false
		for _, msg := range mockLog.messages {
			if msg == "ERROR: SQL error" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Should log SQL error")
		}
	})

	t.Run("silent mode", func(t *testing.T) {
		mockLog := &mockLogger{}
		cfg := &config.DatabaseConfig{
			LogLevel:           "silent",
			SlowQueryThreshold: 100 * time.Millisecond,
		}
		sqLogger := NewSlowQueryLogger(mockLog, cfg)

		ctx := context.Background()
		begin := time.Now().Add(-200 * time.Millisecond)
		fc := func() (string, int64) { return "SELECT * FROM users", 10 }

		sqLogger.Trace(ctx, begin, fc, nil)

		if len(mockLog.messages) != 0 {
			t.Errorf("Silent mode should not log, got %d messages", len(mockLog.messages))
		}
	})
}
