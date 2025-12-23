package app

import (
	"fmt"

	"github.com/goupter/goupter/pkg/auth"
	_ "github.com/goupter/goupter/pkg/auth/jwt"
	"github.com/goupter/goupter/pkg/cache"
	"github.com/goupter/goupter/pkg/config"
	"github.com/goupter/goupter/pkg/database"
	"github.com/goupter/goupter/pkg/discovery"
	"github.com/goupter/goupter/pkg/log"
	"github.com/goupter/goupter/pkg/mq"
	"github.com/goupter/goupter/pkg/server"
	"github.com/goupter/goupter/pkg/server/middleware"
)

// Builder 应用构建器
type Builder struct {
	opts []Option
	err  error
}

// NewBuilder 创建应用构建器
func NewBuilder() *Builder {
	return &Builder{
		opts: make([]Option, 0),
	}
}

// Name 设置应用名称
func (b *Builder) Name(name string) *Builder {
	b.opts = append(b.opts, WithName(name))
	return b
}

// LoadConfig 加载配置
func (b *Builder) LoadConfig(opts ...config.Option) *Builder {
	if b.err != nil {
		return b
	}

	cfg, err := config.Load(opts...)
	if err != nil {
		b.err = fmt.Errorf("load config failed: %w", err)
		return b
	}

	b.opts = append(b.opts, WithConfig(cfg))
	return b
}

// InitLogger 初始化日志
func (b *Builder) InitLogger() *Builder {
	if b.err != nil {
		return b
	}

	cfg := config.Get()
	if cfg == nil {
		b.err = fmt.Errorf("config not loaded")
		return b
	}

	logger, err := log.NewZapLogger(&log.LoggerConfig{
		Level:      log.ParseLevel(cfg.Log.Level),
		Format:     cfg.Log.Format,
		Output:     cfg.Log.Output,
		Filename:   cfg.Log.Filename,
		MaxSize:    cfg.Log.MaxSize,
		MaxBackups: cfg.Log.MaxBackups,
		MaxAge:     cfg.Log.MaxAge,
		Compress:   cfg.Log.Compress,
	})
	if err != nil {
		b.err = fmt.Errorf("init logger failed: %w", err)
		return b
	}

	log.SetDefault(logger)
	b.opts = append(b.opts, WithLogger(logger))
	return b
}

// InitDatabase 初始化数据库
func (b *Builder) InitDatabase() *Builder {
	if b.err != nil {
		return b
	}

	cfg := config.Get()
	if cfg == nil {
		b.err = fmt.Errorf("config not loaded")
		return b
	}

	mysql, err := database.NewMySQL(
		database.WithConfig(&cfg.Database),
		database.WithLogger(log.Default()),
	)
	if err != nil {
		b.err = fmt.Errorf("init database failed: %w", err)
		return b
	}

	b.opts = append(b.opts, WithDatabase(mysql.DB()))
	return b
}

// InitCache 初始化缓存
func (b *Builder) InitCache() *Builder {
	if b.err != nil {
		return b
	}

	cfg := config.Get()
	if cfg == nil {
		b.err = fmt.Errorf("config not loaded")
		return b
	}

	redisCache, err := cache.NewRedisCache(
		cache.WithRedisConfig(&cfg.Redis),
	)
	if err != nil {
		b.err = fmt.Errorf("init cache failed: %w", err)
		return b
	}

	b.opts = append(b.opts, WithCache(redisCache))
	return b
}

// InitDiscovery 初始化服务发现
func (b *Builder) InitDiscovery() *Builder {
	if b.err != nil {
		return b
	}

	cfg := config.Get()
	if cfg == nil {
		b.err = fmt.Errorf("config not loaded")
		return b
	}

	consulDiscovery, err := discovery.NewConsulDiscovery(
		discovery.WithConsulConfig(&cfg.Consul),
		discovery.WithConsulLogger(log.Default()),
	)
	if err != nil {
		b.err = fmt.Errorf("init discovery failed: %w", err)
		return b
	}

	b.opts = append(b.opts, WithDiscovery(consulDiscovery))
	return b
}

// InitMessageQueue 初始化消息队列
func (b *Builder) InitMessageQueue() *Builder {
	if b.err != nil {
		return b
	}

	cfg := config.Get()
	if cfg == nil {
		b.err = fmt.Errorf("config not loaded")
		return b
	}

	natsClient, err := mq.NewNATSClient(
		mq.WithNATSConfig(&cfg.NATS),
		mq.WithNATSLogger(log.Default()),
	)
	if err != nil {
		b.err = fmt.Errorf("init message queue failed: %w", err)
		return b
	}

	b.opts = append(b.opts, WithMessageQueue(natsClient))
	return b
}

// InitAuth 初始化鉴权
func (b *Builder) InitAuth() *Builder {
	if b.err != nil {
		return b
	}

	cfg := config.Get()
	if cfg == nil {
		b.err = fmt.Errorf("config not loaded")
		return b
	}

	authenticator, err := auth.NewAuthenticator(cfg.Auth.Type, cfg.Auth.Config)
	if err != nil {
		b.err = fmt.Errorf("init auth failed: %w", err)
		return b
	}

	b.opts = append(b.opts, WithAuthenticator(authenticator))
	return b
}

// InitHTTPServer 初始化HTTP服务器
func (b *Builder) InitHTTPServer() *Builder {
	if b.err != nil {
		return b
	}

	cfg := config.Get()
	if cfg == nil {
		b.err = fmt.Errorf("config not loaded")
		return b
	}

	httpServer := server.NewHTTPServer(
		server.WithHTTPConfig(&cfg.Server.HTTP),
		server.WithHTTPLogger(log.Default()),
		server.WithMiddleware(
			middleware.Recovery(log.Default()),
			middleware.Logger(log.Default()),
			middleware.CORS(),
		),
	)

	b.opts = append(b.opts, WithHTTPServer(httpServer))
	return b
}

// InitGRPCServer 初始化gRPC服务器
func (b *Builder) InitGRPCServer() *Builder {
	if b.err != nil {
		return b
	}

	cfg := config.Get()
	if cfg == nil {
		b.err = fmt.Errorf("config not loaded")
		return b
	}

	grpcServer := server.NewGRPCServer(
		server.WithGRPCConfig(&cfg.Server.GRPC),
		server.WithGRPCLogger(log.Default()),
	)

	b.opts = append(b.opts, WithGRPCServer(grpcServer))
	return b
}

// InitHealth 初始化健康检查
func (b *Builder) InitHealth(cfg *HealthConfig) *Builder {
	if cfg == nil {
		cfg = DefaultHealthConfig()
	}
	b.opts = append(b.opts, WithHealthConfig(cfg))
	return b
}

// InitMetrics 初始化指标端点与采集
func (b *Builder) InitMetrics(cfg *MetricsConfig) *Builder {
	if cfg == nil {
		cfg = DefaultMetricsConfig()
	}
	b.opts = append(b.opts, WithMetricsConfig(cfg))
	return b
}

// InitPProf 初始化 pprof 诊断端点
func (b *Builder) InitPProf(cfg *PProfConfig) *Builder {
	if cfg == nil {
		cfg = DefaultPProfConfig()
	}
	b.opts = append(b.opts, WithPProfConfig(cfg))
	return b
}

// BeforeStart 添加启动前钩子
func (b *Builder) BeforeStart(fn func() error) *Builder {
	b.opts = append(b.opts, BeforeStart(fn))
	return b
}

// AfterStart 添加启动后钩子
func (b *Builder) AfterStart(fn func() error) *Builder {
	b.opts = append(b.opts, AfterStart(fn))
	return b
}

// BeforeStop 添加停止前钩子
func (b *Builder) BeforeStop(fn func() error) *Builder {
	b.opts = append(b.opts, BeforeStop(fn))
	return b
}

// AfterStop 添加停止后钩子
func (b *Builder) AfterStop(fn func() error) *Builder {
	b.opts = append(b.opts, AfterStop(fn))
	return b
}

// Build 构建应用
func (b *Builder) Build() (*App, error) {
	if b.err != nil {
		return nil, b.err
	}

	return New(b.opts...), nil
}

// MustBuild 构建应用，失败则panic
func (b *Builder) MustBuild() *App {
	app, err := b.Build()
	if err != nil {
		panic(err)
	}
	return app
}
