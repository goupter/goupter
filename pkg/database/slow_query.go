package database

import (
	"context"
	"fmt"
	"time"

	"github.com/goupter/goupter/pkg/config"
	"github.com/goupter/goupter/pkg/log"
	"gorm.io/gorm/logger"
)

// SlowQueryLogger 慢查询日志记录器
type SlowQueryLogger struct {
	log       log.Logger
	level     logger.LogLevel
	threshold time.Duration
}

// NewSlowQueryLogger 创建慢查询日志记录器
func NewSlowQueryLogger(l log.Logger, cfg *config.DatabaseConfig) *SlowQueryLogger {
	var logLevel logger.LogLevel
	switch cfg.LogLevel {
	case "silent":
		logLevel = logger.Silent
	case "error":
		logLevel = logger.Error
	case "warn":
		logLevel = logger.Warn
	case "info":
		logLevel = logger.Info
	default:
		logLevel = logger.Warn
	}

	threshold := cfg.SlowQueryThreshold
	if threshold == 0 {
		threshold = 200 * time.Millisecond
	}

	return &SlowQueryLogger{
		log:       l,
		level:     logLevel,
		threshold: threshold,
	}
}

// LogMode 实现 gorm logger.Interface
func (l *SlowQueryLogger) LogMode(level logger.LogLevel) logger.Interface {
	newLogger := *l
	newLogger.level = level
	return &newLogger
}

// Info 实现 gorm logger.Interface
func (l *SlowQueryLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.level >= logger.Info && l.log != nil {
		l.log.Info(fmt.Sprintf(msg, data...))
	}
}

// Warn 实现 gorm logger.Interface
func (l *SlowQueryLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.level >= logger.Warn && l.log != nil {
		l.log.Warn(fmt.Sprintf(msg, data...))
	}
}

// Error 实现 gorm logger.Interface
func (l *SlowQueryLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.level >= logger.Error && l.log != nil {
		l.log.Error(fmt.Sprintf(msg, data...))
	}
}

// Trace 实现 gorm logger.Interface
func (l *SlowQueryLogger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	if l.level <= logger.Silent {
		return
	}

	elapsed := time.Since(begin)
	sql, rows := fc()

	if l.log == nil {
		return
	}

	fields := []log.Field{
		log.String("sql", sql),
		log.Int64("rows", rows),
		log.String("elapsed", elapsed.String()),
	}

	switch {
	case err != nil && l.level >= logger.Error:
		fields = append(fields, log.Error(err))
		l.log.Error("SQL error", fields...)
	case elapsed >= l.threshold && l.level >= logger.Warn:
		l.log.Warn("SQL slow query", fields...)
	case l.level >= logger.Info:
		l.log.Debug("SQL trace", fields...)
	}
}
