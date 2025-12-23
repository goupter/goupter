package log

import (
	"context"
	"testing"
	"time"
)

func TestNewWriterAdapter(t *testing.T) {
	logger := Default()
	adapter := NewWriterAdapter(logger, InfoLevel)

	if adapter == nil {
		t.Fatal("NewWriterAdapter should return non-nil")
	}
}

func TestWriterAdapter_Write(t *testing.T) {
	logger := Default()
	adapter := NewWriterAdapter(logger, InfoLevel)

	n, err := adapter.Write([]byte("test message"))
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if n != len("test message") {
		t.Errorf("Expected n=%d, got %d", len("test message"), n)
	}
}

func TestWriterAdapter_WriteDifferentLevels(t *testing.T) {
	logger := Default()

	debugAdapter := NewWriterAdapter(logger, DebugLevel)
	debugAdapter.Write([]byte("debug"))

	warnAdapter := NewWriterAdapter(logger, WarnLevel)
	warnAdapter.Write([]byte("warn"))

	errorAdapter := NewWriterAdapter(logger, ErrorLevel)
	errorAdapter.Write([]byte("error"))
}

func TestNewStdLogAdapter(t *testing.T) {
	logger := Default()
	stdLogger := NewStdLogAdapter(logger, InfoLevel)

	if stdLogger == nil {
		t.Fatal("NewStdLogAdapter should return non-nil")
	}
}

func TestStdLogAdapter_Print(t *testing.T) {
	logger := Default()
	stdLogger := NewStdLogAdapter(logger, InfoLevel)

	stdLogger.Print("test message")
}

func TestNewGormLogger(t *testing.T) {
	logger := Default()
	gormLogger := NewGormLogger(logger)

	if gormLogger == nil {
		t.Fatal("NewGormLogger should return non-nil")
	}
}

func TestGormLogger_LogMode(t *testing.T) {
	logger := Default()
	gormLogger := NewGormLogger(logger)

	newLogger := gormLogger.LogMode(4)
	if newLogger == nil {
		t.Fatal("LogMode should return non-nil")
	}
}

func TestGormLogger_Info(t *testing.T) {
	logger := Default()
	gormLogger := NewGormLogger(logger)

	ctx := context.Background()
	gormLogger.Info(ctx, "test %s", "message")
}

func TestGormLogger_Warn(t *testing.T) {
	logger := Default()
	gormLogger := NewGormLogger(logger)

	ctx := context.Background()
	gormLogger.Warn(ctx, "test %s", "message")
}

func TestGormLogger_Error(t *testing.T) {
	logger := Default()
	gormLogger := NewGormLogger(logger)

	ctx := context.Background()
	gormLogger.Error(ctx, "test %s", "message")
}

func TestGormLogger_Trace(t *testing.T) {
	logger := Default()
	gormLogger := NewGormLogger(logger)

	ctx := context.Background()
	begin := time.Now()

	gormLogger.Trace(ctx, begin, func() (sql string, rowsAffected int64) {
		return "SELECT * FROM users", 10
	}, nil)
}

func TestGormLogger_WithOptions(t *testing.T) {
	logger := Default()
	gormLogger := NewGormLogger(logger,
		WithGormSlowThreshold(500*time.Millisecond),
		WithGormIgnoreNotFound(true),
	)

	if gormLogger == nil {
		t.Fatal("NewGormLogger with options should return non-nil")
	}
}

func TestNewGrpcLogger(t *testing.T) {
	logger := Default()
	grpcLogger := NewGrpcLogger(logger)

	if grpcLogger == nil {
		t.Fatal("NewGrpcLogger should return non-nil")
	}
}

func TestGrpcLogger_Info(t *testing.T) {
	logger := Default()
	grpcLogger := NewGrpcLogger(logger)

	grpcLogger.Info("test", "arg1", "arg2")
}

func TestGrpcLogger_Infoln(t *testing.T) {
	logger := Default()
	grpcLogger := NewGrpcLogger(logger)

	grpcLogger.Infoln("test", "arg1", "arg2")
}

func TestGrpcLogger_Infof(t *testing.T) {
	logger := Default()
	grpcLogger := NewGrpcLogger(logger)

	grpcLogger.Infof("test %s", "message")
}

func TestGrpcLogger_Warning(t *testing.T) {
	logger := Default()
	grpcLogger := NewGrpcLogger(logger)

	grpcLogger.Warning("test", "arg1")
}

func TestGrpcLogger_Warningln(t *testing.T) {
	logger := Default()
	grpcLogger := NewGrpcLogger(logger)

	grpcLogger.Warningln("test", "arg1")
}

func TestGrpcLogger_Warningf(t *testing.T) {
	logger := Default()
	grpcLogger := NewGrpcLogger(logger)

	grpcLogger.Warningf("test %s", "message")
}

func TestGrpcLogger_Error(t *testing.T) {
	logger := Default()
	grpcLogger := NewGrpcLogger(logger)

	grpcLogger.Error("test", "arg1")
}

func TestGrpcLogger_Errorln(t *testing.T) {
	logger := Default()
	grpcLogger := NewGrpcLogger(logger)

	grpcLogger.Errorln("test", "arg1")
}

func TestGrpcLogger_Errorf(t *testing.T) {
	logger := Default()
	grpcLogger := NewGrpcLogger(logger)

	grpcLogger.Errorf("test %s", "message")
}

func TestGrpcLogger_V(t *testing.T) {
	logger := Default()
	grpcLogger := NewGrpcLogger(logger)

	_ = grpcLogger.V(0)
	_ = grpcLogger.V(1)
	_ = grpcLogger.V(2)
}

func TestGrpcLogger_SetV(t *testing.T) {
	logger := Default()
	grpcLogger := NewGrpcLogger(logger)

	grpcLogger.SetV(3)
	if !grpcLogger.V(3) {
		t.Error("V(3) should return true after SetV(3)")
	}
}
