package log

import (
	"context"
	"io"
)

// Level 日志级别
type Level int8

const (
	// DebugLevel 调试级别
	DebugLevel Level = iota - 1
	// InfoLevel 信息级别
	InfoLevel
	// WarnLevel 警告级别
	WarnLevel
	// ErrorLevel 错误级别
	ErrorLevel
	// FatalLevel 致命错误级别
	FatalLevel
)

// String 返回日志级别字符串
func (l Level) String() string {
	switch l {
	case DebugLevel:
		return "debug"
	case InfoLevel:
		return "info"
	case WarnLevel:
		return "warn"
	case ErrorLevel:
		return "error"
	case FatalLevel:
		return "fatal"
	default:
		return "unknown"
	}
}

// ParseLevel 解析日志级别字符串
func ParseLevel(level string) Level {
	switch level {
	case "debug":
		return DebugLevel
	case "info":
		return InfoLevel
	case "warn", "warning":
		return WarnLevel
	case "error":
		return ErrorLevel
	case "fatal":
		return FatalLevel
	default:
		return InfoLevel
	}
}

// Field 日志字段
type Field struct {
	Key   string
	Value interface{}
}

// String 创建字符串字段
func String(key string, val string) Field {
	return Field{Key: key, Value: val}
}

// Int 创建整数字段
func Int(key string, val int) Field {
	return Field{Key: key, Value: val}
}

// Int64 创建int64字段
func Int64(key string, val int64) Field {
	return Field{Key: key, Value: val}
}

// Float64 创建float64字段
func Float64(key string, val float64) Field {
	return Field{Key: key, Value: val}
}

// Bool 创建布尔字段
func Bool(key string, val bool) Field {
	return Field{Key: key, Value: val}
}

// Error 创建错误字段
func Error(err error) Field {
	return Field{Key: "error", Value: err}
}

// Any 创建任意类型字段
func Any(key string, val interface{}) Field {
	return Field{Key: key, Value: val}
}

// Logger 日志接口
type Logger interface {
	// Debug 输出调试日志
	Debug(msg string, fields ...Field)
	// Info 输出信息日志
	Info(msg string, fields ...Field)
	// Warn 输出警告日志
	Warn(msg string, fields ...Field)
	// Error 输出错误日志
	Error(msg string, fields ...Field)
	// Fatal 输出致命错误日志并退出程序
	Fatal(msg string, fields ...Field)

	// With 返回带有预设字段的新Logger
	With(fields ...Field) Logger
	// WithContext 返回带有上下文的新Logger
	WithContext(ctx context.Context) Logger

	// SetLevel 设置日志级别
	SetLevel(level Level)
	// GetLevel 获取日志级别
	GetLevel() Level

	// Sync 同步日志缓冲
	Sync() error
}

// LoggerConfig 日志配置
type LoggerConfig struct {
	Level      Level
	Format     string // json, console
	Output     string // stdout, stderr, file, both
	Filename   string
	MaxSize    int  // MB
	MaxBackups int
	MaxAge     int  // days
	Compress   bool
}

// Writer 日志写入器接口
type Writer interface {
	io.Writer
	Sync() error
}

// contextKey 上下文键类型
type contextKey struct{}

// traceIDKey 追踪ID的上下文键
var traceIDKey = contextKey{}

// requestIDKey 请求ID的上下文键
var requestIDKey = contextKey{}

// WithTraceID 设置追踪ID到上下文
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceIDKey, traceID)
}

// TraceIDFromContext 从上下文获取追踪ID
func TraceIDFromContext(ctx context.Context) string {
	if traceID, ok := ctx.Value(traceIDKey).(string); ok {
		return traceID
	}
	return ""
}

// WithRequestID 设置请求ID到上下文
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

// RequestIDFromContext 从上下文获取请求ID
func RequestIDFromContext(ctx context.Context) string {
	if requestID, ok := ctx.Value(requestIDKey).(string); ok {
		return requestID
	}
	return ""
}
