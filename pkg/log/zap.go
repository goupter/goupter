package log

import (
	"context"
	"io"
	"os"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	// defaultLogger 默认日志实例
	defaultLogger Logger
	once          sync.Once
)

// zapLogger zap日志实现
type zapLogger struct {
	logger *zap.Logger
	level  zap.AtomicLevel
	fields []Field
}

// NewZapLogger 创建zap日志实例
func NewZapLogger(cfg *LoggerConfig) (Logger, error) {
	level := zap.NewAtomicLevelAt(toZapLevel(cfg.Level))

	var writers []io.Writer
	var encoder zapcore.Encoder

	// 配置编码器
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	if cfg.Format == "console" {
		encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	}

	// 配置输出
	switch cfg.Output {
	case "stdout":
		writers = append(writers, os.Stdout)
	case "stderr":
		writers = append(writers, os.Stderr)
	case "file":
		if cfg.Filename != "" {
			writers = append(writers, newFileWriter(cfg))
		}
	case "both":
		writers = append(writers, os.Stdout)
		if cfg.Filename != "" {
			writers = append(writers, newFileWriter(cfg))
		}
	default:
		writers = append(writers, os.Stdout)
	}

	// 创建多输出WriteSyncer
	var writeSyncers []zapcore.WriteSyncer
	for _, w := range writers {
		writeSyncers = append(writeSyncers, zapcore.AddSync(w))
	}
	multiWriter := zapcore.NewMultiWriteSyncer(writeSyncers...)

	core := zapcore.NewCore(encoder, multiWriter, level)
	logger := zap.New(core,
		zap.AddCaller(),
		zap.AddCallerSkip(1),
		zap.AddStacktrace(zapcore.ErrorLevel),
	)

	return &zapLogger{
		logger: logger,
		level:  level,
		fields: make([]Field, 0),
	}, nil
}

// newFileWriter 创建文件写入器
func newFileWriter(cfg *LoggerConfig) io.Writer {
	return &lumberjack.Logger{
		Filename:   cfg.Filename,
		MaxSize:    cfg.MaxSize,
		MaxBackups: cfg.MaxBackups,
		MaxAge:     cfg.MaxAge,
		Compress:   cfg.Compress,
	}
}

// toZapLevel 转换日志级别
func toZapLevel(level Level) zapcore.Level {
	switch level {
	case DebugLevel:
		return zapcore.DebugLevel
	case InfoLevel:
		return zapcore.InfoLevel
	case WarnLevel:
		return zapcore.WarnLevel
	case ErrorLevel:
		return zapcore.ErrorLevel
	case FatalLevel:
		return zapcore.FatalLevel
	default:
		return zapcore.InfoLevel
	}
}

// toZapFields 转换字段
func toZapFields(fields []Field) []zap.Field {
	zapFields := make([]zap.Field, 0, len(fields))
	for _, f := range fields {
		zapFields = append(zapFields, zap.Any(f.Key, f.Value))
	}
	return zapFields
}

func (l *zapLogger) Debug(msg string, fields ...Field) {
	l.logger.Debug(msg, toZapFields(append(l.fields, fields...))...)
}

func (l *zapLogger) Info(msg string, fields ...Field) {
	l.logger.Info(msg, toZapFields(append(l.fields, fields...))...)
}

func (l *zapLogger) Warn(msg string, fields ...Field) {
	l.logger.Warn(msg, toZapFields(append(l.fields, fields...))...)
}

func (l *zapLogger) Error(msg string, fields ...Field) {
	l.logger.Error(msg, toZapFields(append(l.fields, fields...))...)
}

func (l *zapLogger) Fatal(msg string, fields ...Field) {
	l.logger.Fatal(msg, toZapFields(append(l.fields, fields...))...)
}

func (l *zapLogger) With(fields ...Field) Logger {
	newFields := make([]Field, 0, len(l.fields)+len(fields))
	newFields = append(newFields, l.fields...)
	newFields = append(newFields, fields...)
	return &zapLogger{
		logger: l.logger,
		level:  l.level,
		fields: newFields,
	}
}

func (l *zapLogger) WithContext(ctx context.Context) Logger {
	var fields []Field

	if traceID := TraceIDFromContext(ctx); traceID != "" {
		fields = append(fields, String("trace_id", traceID))
	}

	if requestID := RequestIDFromContext(ctx); requestID != "" {
		fields = append(fields, String("request_id", requestID))
	}

	if len(fields) == 0 {
		return l
	}

	return l.With(fields...)
}

func (l *zapLogger) SetLevel(level Level) {
	l.level.SetLevel(toZapLevel(level))
}

func (l *zapLogger) GetLevel() Level {
	switch l.level.Level() {
	case zapcore.DebugLevel:
		return DebugLevel
	case zapcore.InfoLevel:
		return InfoLevel
	case zapcore.WarnLevel:
		return WarnLevel
	case zapcore.ErrorLevel:
		return ErrorLevel
	case zapcore.FatalLevel:
		return FatalLevel
	default:
		return InfoLevel
	}
}

func (l *zapLogger) Sync() error {
	return l.logger.Sync()
}

// Init 初始化默认日志实例
func Init(cfg *LoggerConfig) error {
	var err error
	once.Do(func() {
		defaultLogger, err = NewZapLogger(cfg)
	})
	return err
}

// Default 获取默认日志实例
func Default() Logger {
	if defaultLogger == nil {
		// 如果未初始化，使用默认配置
		defaultLogger, _ = NewZapLogger(&LoggerConfig{
			Level:  InfoLevel,
			Format: "json",
			Output: "stdout",
		})
	}
	return defaultLogger
}

// SetDefault 设置默认日志实例
func SetDefault(logger Logger) {
	defaultLogger = logger
}

// Debug 使用默认Logger输出调试日志
func Debug(msg string, fields ...Field) {
	Default().Debug(msg, fields...)
}

// Info 使用默认Logger输出信息日志
func Info(msg string, fields ...Field) {
	Default().Info(msg, fields...)
}

// Warn 使用默认Logger输出警告日志
func Warn(msg string, fields ...Field) {
	Default().Warn(msg, fields...)
}

// Err 使用默认Logger输出错误日志
func Err(msg string, fields ...Field) {
	Default().Error(msg, fields...)
}

// Fatal 使用默认Logger输出致命错误日志
func Fatal(msg string, fields ...Field) {
	Default().Fatal(msg, fields...)
}

// WithFields 创建带字段的Logger
func WithFields(fields ...Field) Logger {
	return Default().With(fields...)
}

// WithContext 创建带上下文的Logger
func WithContext(ctx context.Context) Logger {
	return Default().WithContext(ctx)
}

// Sync 同步默认日志缓冲
func Sync() error {
	return Default().Sync()
}
