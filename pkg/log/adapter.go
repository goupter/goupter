package log

import (
	"context"
	"fmt"
	"io"
	stdlog "log"
	"time"

	"gorm.io/gorm/logger"
)

// WriterAdapter adapts Logger to io.Writer
type WriterAdapter struct {
	logger Logger
	level  Level
}

// NewWriterAdapter creates an io.Writer adapter
func NewWriterAdapter(logger Logger, level Level) io.Writer {
	return &WriterAdapter{
		logger: logger,
		level:  level,
	}
}

// Write implements io.Writer
func (w *WriterAdapter) Write(p []byte) (n int, err error) {
	msg := string(p)
	if len(msg) > 0 && msg[len(msg)-1] == '\n' {
		msg = msg[:len(msg)-1]
	}

	switch w.level {
	case DebugLevel:
		w.logger.Debug(msg)
	case InfoLevel:
		w.logger.Info(msg)
	case WarnLevel:
		w.logger.Warn(msg)
	case ErrorLevel:
		w.logger.Error(msg)
	case FatalLevel:
		w.logger.Fatal(msg)
	default:
		w.logger.Info(msg)
	}

	return len(p), nil
}

// StdLogAdapter adapts Logger to standard library log.Logger
type StdLogAdapter struct {
	logger Logger
	level  Level
}

// NewStdLogAdapter creates a standard library adapter
func NewStdLogAdapter(logger Logger, level Level) *stdlog.Logger {
	adapter := &StdLogAdapter{
		logger: logger,
		level:  level,
	}
	return stdlog.New(adapter, "", 0)
}

// Write implements io.Writer
func (a *StdLogAdapter) Write(p []byte) (n int, err error) {
	msg := string(p)
	if len(msg) > 0 && msg[len(msg)-1] == '\n' {
		msg = msg[:len(msg)-1]
	}

	switch a.level {
	case DebugLevel:
		a.logger.Debug(msg)
	case InfoLevel:
		a.logger.Info(msg)
	case WarnLevel:
		a.logger.Warn(msg)
	case ErrorLevel:
		a.logger.Error(msg)
	default:
		a.logger.Info(msg)
	}

	return len(p), nil
}

// GormLogger adapts Logger to GORM logger interface
type GormLogger struct {
	logger                    Logger
	level                     logger.LogLevel
	slowThreshold             time.Duration
	ignoreRecordNotFoundError bool
}

// GormLoggerOption configures GormLogger
type GormLoggerOption func(*GormLogger)

// WithGormSlowThreshold sets slow query threshold
func WithGormSlowThreshold(threshold time.Duration) GormLoggerOption {
	return func(l *GormLogger) {
		l.slowThreshold = threshold
	}
}

// WithGormIgnoreNotFound ignores record not found errors
func WithGormIgnoreNotFound(ignore bool) GormLoggerOption {
	return func(l *GormLogger) {
		l.ignoreRecordNotFoundError = ignore
	}
}

// NewGormLogger creates a GORM logger adapter
func NewGormLogger(log Logger, opts ...GormLoggerOption) logger.Interface {
	l := &GormLogger{
		logger:                    log,
		level:                     logger.Info,
		slowThreshold:             200 * time.Millisecond,
		ignoreRecordNotFoundError: true,
	}

	for _, opt := range opts {
		opt(l)
	}

	return l
}

// LogMode sets log level
func (l *GormLogger) LogMode(level logger.LogLevel) logger.Interface {
	newLogger := *l
	newLogger.level = level
	return &newLogger
}

// Info logs info messages
func (l *GormLogger) Info(ctx context.Context, msg string, args ...interface{}) {
	if l.level >= logger.Info {
		l.logger.WithContext(ctx).Info(fmt.Sprintf(msg, args...))
	}
}

// Warn logs warning messages
func (l *GormLogger) Warn(ctx context.Context, msg string, args ...interface{}) {
	if l.level >= logger.Warn {
		l.logger.WithContext(ctx).Warn(fmt.Sprintf(msg, args...))
	}
}

// Error logs error messages
func (l *GormLogger) Error(ctx context.Context, msg string, args ...interface{}) {
	if l.level >= logger.Error {
		l.logger.WithContext(ctx).Error(fmt.Sprintf(msg, args...))
	}
}

// Trace logs SQL queries
func (l *GormLogger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	if l.level <= logger.Silent {
		return
	}

	elapsed := time.Since(begin)
	sql, rows := fc()

	fields := []Field{
		Duration("elapsed", elapsed),
		Int64("rows", rows),
		String("sql", sql),
	}

	switch {
	case err != nil && l.level >= logger.Error:
		if l.ignoreRecordNotFoundError && err.Error() == "record not found" {
			return
		}
		l.logger.WithContext(ctx).Error("gorm query error", append(fields, Error(err))...)
	case elapsed > l.slowThreshold && l.slowThreshold != 0 && l.level >= logger.Warn:
		l.logger.WithContext(ctx).Warn("gorm slow query", fields...)
	case l.level >= logger.Info:
		l.logger.WithContext(ctx).Debug("gorm query", fields...)
	}
}

// GrpcLogger adapts Logger to gRPC logger interface
type GrpcLogger struct {
	logger Logger
	v      int
}

// NewGrpcLogger creates a gRPC logger adapter
func NewGrpcLogger(logger Logger) *GrpcLogger {
	return &GrpcLogger{
		logger: logger,
		v:      2,
	}
}

func (l *GrpcLogger) Info(args ...interface{}) {
	l.logger.Info(fmt.Sprint(args...))
}

func (l *GrpcLogger) Infoln(args ...interface{}) {
	l.logger.Info(fmt.Sprint(args...))
}

func (l *GrpcLogger) Infof(format string, args ...interface{}) {
	l.logger.Info(fmt.Sprintf(format, args...))
}

func (l *GrpcLogger) Warning(args ...interface{}) {
	l.logger.Warn(fmt.Sprint(args...))
}

func (l *GrpcLogger) Warningln(args ...interface{}) {
	l.logger.Warn(fmt.Sprint(args...))
}

func (l *GrpcLogger) Warningf(format string, args ...interface{}) {
	l.logger.Warn(fmt.Sprintf(format, args...))
}

func (l *GrpcLogger) Error(args ...interface{}) {
	l.logger.Error(fmt.Sprint(args...))
}

func (l *GrpcLogger) Errorln(args ...interface{}) {
	l.logger.Error(fmt.Sprint(args...))
}

func (l *GrpcLogger) Errorf(format string, args ...interface{}) {
	l.logger.Error(fmt.Sprintf(format, args...))
}

func (l *GrpcLogger) Fatal(args ...interface{}) {
	l.logger.Fatal(fmt.Sprint(args...))
}

func (l *GrpcLogger) Fatalln(args ...interface{}) {
	l.logger.Fatal(fmt.Sprint(args...))
}

func (l *GrpcLogger) Fatalf(format string, args ...interface{}) {
	l.logger.Fatal(fmt.Sprintf(format, args...))
}

// V returns whether the specified verbosity level is enabled
func (l *GrpcLogger) V(level int) bool {
	return level <= l.v
}

// SetV sets the verbosity level
func (l *GrpcLogger) SetV(v int) {
	l.v = v
}
