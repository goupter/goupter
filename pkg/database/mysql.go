package database

import (
	"context"
	"fmt"
	"time"

	"github.com/goupter/goupter/pkg/config"
	"github.com/goupter/goupter/pkg/log"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

// DB 数据库连接
var db *gorm.DB

// MySQL MySQL数据库
type MySQL struct {
	db     *gorm.DB
	config *config.DatabaseConfig
	logger log.Logger
}

// Option 数据库选项
type Option func(*MySQL)

// WithConfig 设置数据库配置
func WithConfig(cfg *config.DatabaseConfig) Option {
	return func(m *MySQL) {
		m.config = cfg
	}
}

// WithLogger 设置日志
func WithLogger(logger log.Logger) Option {
	return func(m *MySQL) {
		m.logger = logger
	}
}

// NewMySQL 创建MySQL连接
func NewMySQL(opts ...Option) (*MySQL, error) {
	m := &MySQL{
		config: &config.DatabaseConfig{
			Driver:             "mysql",
			Host:               "localhost",
			Port:               3306,
			Charset:            "utf8mb4",
			MaxIdleConns:       10,
			MaxOpenConns:       100,
			ConnMaxLifetime:    time.Hour,
			ConnMaxIdleTime:    10 * time.Minute,
			LogLevel:           "warn",
			SlowQueryThreshold: 200 * time.Millisecond,
			SlowQueryLog:       true,
			SlowQueryWithStack: false,
		},
	}

	for _, opt := range opts {
		opt(m)
	}

	// 构建DSN
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local",
		m.config.Username,
		m.config.Password,
		m.config.Host,
		m.config.Port,
		m.config.Database,
		m.config.Charset,
	)

	// 配置GORM日志
	var gormLogger logger.Interface
	if m.config.SlowQueryLog {
		gormLogger = NewSlowQueryLogger(m.logger, m.config)
	} else {
		gormLogger = newGormLogger(m.logger, m.config.LogLevel)
	}

	// 打开数据库连接
	var err error
	m.db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: gormLogger,
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true, // 使用单数表名
		},
		DisableForeignKeyConstraintWhenMigrating: true, // 禁用外键约束
	})
	if err != nil {
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}

	// 获取底层sql.DB
	sqlDB, err := m.db.DB()
	if err != nil {
		return nil, fmt.Errorf("获取数据库连接失败: %w", err)
	}

	// 设置连接池参数
	sqlDB.SetMaxIdleConns(m.config.MaxIdleConns)
	sqlDB.SetMaxOpenConns(m.config.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(m.config.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(m.config.ConnMaxIdleTime)

	// 设置全局db
	db = m.db

	return m, nil
}

// DB 获取GORM实例
func (m *MySQL) DB() *gorm.DB {
	return m.db
}

// Close 关闭数据库连接
func (m *MySQL) Close() error {
	sqlDB, err := m.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// Ping 检查数据库连接
func (m *MySQL) Ping(ctx context.Context) error {
	sqlDB, err := m.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}

// GetDB 获取全局数据库实例
func GetDB() *gorm.DB {
	return db
}

// SetDB 设置全局数据库实例
func SetDB(database *gorm.DB) {
	db = database
}

// Transaction 事务
func Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
	return db.WithContext(ctx).Transaction(fn)
}

// gormLogger GORM日志适配器
type gormLogger struct {
	log   log.Logger
	level logger.LogLevel
}

// newGormLogger 创建GORM日志适配器
func newGormLogger(l log.Logger, level string) logger.Interface {
	var logLevel logger.LogLevel
	switch level {
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

	return &gormLogger{
		log:   l,
		level: logLevel,
	}
}

func (l *gormLogger) LogMode(level logger.LogLevel) logger.Interface {
	return &gormLogger{
		log:   l.log,
		level: level,
	}
}

func (l *gormLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.level >= logger.Info && l.log != nil {
		l.log.Info(fmt.Sprintf(msg, data...))
	}
}

func (l *gormLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.level >= logger.Warn && l.log != nil {
		l.log.Warn(fmt.Sprintf(msg, data...))
	}
}

func (l *gormLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.level >= logger.Error && l.log != nil {
		l.log.Error(fmt.Sprintf(msg, data...))
	}
}

func (l *gormLogger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
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
	case elapsed > 200*time.Millisecond && l.level >= logger.Warn:
		l.log.Warn("SQL slow query", fields...)
	case l.level >= logger.Info:
		l.log.Debug("SQL trace", fields...)
	}
}
