package log

import (
	"context"
	"testing"
)

func TestLevel_String(t *testing.T) {
	tests := []struct {
		level    Level
		expected string
	}{
		{DebugLevel, "debug"},
		{InfoLevel, "info"},
		{WarnLevel, "warn"},
		{ErrorLevel, "error"},
		{FatalLevel, "fatal"},
		{Level(100), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.level.String(); got != tt.expected {
				t.Errorf("Level.String() = %s, want %s", got, tt.expected)
			}
		})
	}
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected Level
	}{
		{"debug", DebugLevel},
		{"info", InfoLevel},
		{"warn", WarnLevel},
		{"warning", WarnLevel},
		{"error", ErrorLevel},
		{"fatal", FatalLevel},
		{"unknown", InfoLevel},
		{"", InfoLevel},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := ParseLevel(tt.input); got != tt.expected {
				t.Errorf("ParseLevel(%s) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestField_Constructors(t *testing.T) {
	// Test String field
	f := String("key", "value")
	if f.Key != "key" || f.Value != "value" {
		t.Errorf("String() = {%s, %v}, want {key, value}", f.Key, f.Value)
	}

	// Test Int field
	f = Int("count", 42)
	if f.Key != "count" || f.Value != 42 {
		t.Errorf("Int() = {%s, %v}, want {count, 42}", f.Key, f.Value)
	}

	// Test Int64 field
	f = Int64("big", 9223372036854775807)
	if f.Key != "big" || f.Value != int64(9223372036854775807) {
		t.Errorf("Int64() = {%s, %v}, want {big, 9223372036854775807}", f.Key, f.Value)
	}

	// Test Float64 field
	f = Float64("price", 19.99)
	if f.Key != "price" || f.Value != 19.99 {
		t.Errorf("Float64() = {%s, %v}, want {price, 19.99}", f.Key, f.Value)
	}

	// Test Bool field
	f = Bool("enabled", true)
	if f.Key != "enabled" || f.Value != true {
		t.Errorf("Bool() = {%s, %v}, want {enabled, true}", f.Key, f.Value)
	}

	// Test Any field
	f = Any("data", map[string]int{"a": 1})
	if f.Key != "data" {
		t.Errorf("Any() key = %s, want data", f.Key)
	}

	// Test Error field
	f = Error(nil)
	if f.Key != "error" {
		t.Errorf("Error() key = %s, want error", f.Key)
	}
}

func TestWithTraceID(t *testing.T) {
	ctx := context.Background()
	traceID := "trace-123"

	ctx = WithTraceID(ctx, traceID)
	got := TraceIDFromContext(ctx)

	if got != traceID {
		t.Errorf("TraceIDFromContext() = %s, want %s", got, traceID)
	}
}

func TestTraceIDFromContext_Empty(t *testing.T) {
	ctx := context.Background()
	got := TraceIDFromContext(ctx)

	if got != "" {
		t.Errorf("TraceIDFromContext() from empty context = %s, want empty", got)
	}
}

func TestWithRequestID(t *testing.T) {
	ctx := context.Background()
	requestID := "req-456"

	ctx = WithRequestID(ctx, requestID)
	got := RequestIDFromContext(ctx)

	if got != requestID {
		t.Errorf("RequestIDFromContext() = %s, want %s", got, requestID)
	}
}

func TestRequestIDFromContext_Empty(t *testing.T) {
	ctx := context.Background()
	got := RequestIDFromContext(ctx)

	if got != "" {
		t.Errorf("RequestIDFromContext() from empty context = %s, want empty", got)
	}
}

func TestLoggerConfig(t *testing.T) {
	cfg := &LoggerConfig{
		Level:      InfoLevel,
		Format:     "json",
		Output:     "stdout",
		Filename:   "/var/log/app.log",
		MaxSize:    100,
		MaxBackups: 5,
		MaxAge:     30,
		Compress:   true,
	}

	if cfg.Level != InfoLevel {
		t.Errorf("Level = %v, want InfoLevel", cfg.Level)
	}
	if cfg.Format != "json" {
		t.Errorf("Format = %s, want json", cfg.Format)
	}
	if cfg.Output != "stdout" {
		t.Errorf("Output = %s, want stdout", cfg.Output)
	}
	if cfg.Filename != "/var/log/app.log" {
		t.Errorf("Filename = %s, want /var/log/app.log", cfg.Filename)
	}
	if cfg.MaxSize != 100 {
		t.Errorf("MaxSize = %d, want 100", cfg.MaxSize)
	}
	if cfg.MaxBackups != 5 {
		t.Errorf("MaxBackups = %d, want 5", cfg.MaxBackups)
	}
	if cfg.MaxAge != 30 {
		t.Errorf("MaxAge = %d, want 30", cfg.MaxAge)
	}
	if !cfg.Compress {
		t.Error("Compress should be true")
	}
}

func TestLevel_Enabled(t *testing.T) {
	// Test level comparison (lower value = more verbose)
	if DebugLevel >= InfoLevel {
		t.Error("DebugLevel should be less than InfoLevel")
	}
	if InfoLevel >= WarnLevel {
		t.Error("InfoLevel should be less than WarnLevel")
	}
	if WarnLevel >= ErrorLevel {
		t.Error("WarnLevel should be less than ErrorLevel")
	}
	if ErrorLevel >= FatalLevel {
		t.Error("ErrorLevel should be less than FatalLevel")
	}
}
