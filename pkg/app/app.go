package app

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/goupter/goupter/pkg/auth"
	"github.com/goupter/goupter/pkg/cache"
	"github.com/goupter/goupter/pkg/config"
	"github.com/goupter/goupter/pkg/discovery"
	"github.com/goupter/goupter/pkg/log"
	"github.com/goupter/goupter/pkg/mq"
	"github.com/goupter/goupter/pkg/server"
	"github.com/goupter/goupter/pkg/server/middleware"
	"gorm.io/gorm"
)

// App 应用
type App struct {
	name       string
	config     *config.Config
	logger     log.Logger
	httpServer *server.HTTPServer
	grpcServer *server.GRPCServer
	db         *gorm.DB
	cache      cache.Cache
	discovery  discovery.Discovery
	mq         mq.MessageQueue
	auth       auth.Authenticator

	// 健康检查
	health *HealthManager

	// 指标与诊断
	metricsConfig *MetricsConfig
	metrics       *middleware.RequestMetrics
	pprofConfig   *PProfConfig

	// 生命周期钩子
	beforeStart []func() error
	afterStart  []func() error
	beforeStop  []func() error
	afterStop   []func() error

	// 状态
	running bool
	mu      sync.RWMutex
}

// Option 应用选项
type Option func(*App)

// WithName 设置应用名称
func WithName(name string) Option {
	return func(a *App) {
		a.name = name
	}
}

// WithConfig 设置配置
func WithConfig(cfg *config.Config) Option {
	return func(a *App) {
		a.config = cfg
	}
}

// WithLogger 设置日志
func WithLogger(logger log.Logger) Option {
	return func(a *App) {
		a.logger = logger
	}
}

// WithHTTPServer 设置HTTP服务器
func WithHTTPServer(s *server.HTTPServer) Option {
	return func(a *App) {
		a.httpServer = s
	}
}

// WithGRPCServer 设置gRPC服务器
func WithGRPCServer(s *server.GRPCServer) Option {
	return func(a *App) {
		a.grpcServer = s
	}
}

// WithDatabase 设置数据库
func WithDatabase(db *gorm.DB) Option {
	return func(a *App) {
		a.db = db
	}
}

// WithCache 设置缓存
func WithCache(c cache.Cache) Option {
	return func(a *App) {
		a.cache = c
	}
}

// WithDiscovery 设置服务发现
func WithDiscovery(d discovery.Discovery) Option {
	return func(a *App) {
		a.discovery = d
	}
}

// WithMessageQueue 设置消息队列
func WithMessageQueue(mq mq.MessageQueue) Option {
	return func(a *App) {
		a.mq = mq
	}
}

// WithAuthenticator 设置鉴权器
func WithAuthenticator(auth auth.Authenticator) Option {
	return func(a *App) {
		a.auth = auth
	}
}

// WithHealthManager 设置健康检查管理器
func WithHealthManager(h *HealthManager) Option {
	return func(a *App) {
		a.health = h
	}
}

// WithHealthConfig 设置健康检查配置
func WithHealthConfig(cfg *HealthConfig) Option {
	return func(a *App) {
		a.health = NewHealthManager(cfg)
	}
}

// WithMetricsConfig 设置指标配置
func WithMetricsConfig(cfg *MetricsConfig) Option {
	return func(a *App) {
		a.metricsConfig = cfg
	}
}

// WithPProfConfig 设置 PProf 配置
func WithPProfConfig(cfg *PProfConfig) Option {
	return func(a *App) {
		a.pprofConfig = cfg
	}
}

// BeforeStart 添加启动前钩子
func BeforeStart(fn func() error) Option {
	return func(a *App) {
		a.beforeStart = append(a.beforeStart, fn)
	}
}

// AfterStart 添加启动后钩子
func AfterStart(fn func() error) Option {
	return func(a *App) {
		a.afterStart = append(a.afterStart, fn)
	}
}

// BeforeStop 添加停止前钩子
func BeforeStop(fn func() error) Option {
	return func(a *App) {
		a.beforeStop = append(a.beforeStop, fn)
	}
}

// AfterStop 添加停止后钩子
func AfterStop(fn func() error) Option {
	return func(a *App) {
		a.afterStop = append(a.afterStop, fn)
	}
}

// New 创建应用
func New(opts ...Option) *App {
	app := &App{
		name:        "flit-app",
		health:      NewHealthManager(nil),
		beforeStart: make([]func() error, 0),
		afterStart:  make([]func() error, 0),
		beforeStop:  make([]func() error, 0),
		afterStop:   make([]func() error, 0),
	}

	for _, opt := range opts {
		opt(app)
	}

	return app
}

// Name 获取应用名称
func (a *App) Name() string {
	return a.name
}

// Config 获取配置
func (a *App) Config() *config.Config {
	return a.config
}

// Logger 获取日志
func (a *App) Logger() log.Logger {
	return a.logger
}

// HTTPServer 获取HTTP服务器
func (a *App) HTTPServer() *server.HTTPServer {
	return a.httpServer
}

// GRPCServer 获取gRPC服务器
func (a *App) GRPCServer() *server.GRPCServer {
	return a.grpcServer
}

// Cache 获取缓存
func (a *App) Cache() cache.Cache {
	return a.cache
}

// DB 获取数据库
func (a *App) DB() *gorm.DB {
	return a.db
}

// Discovery 获取服务发现
func (a *App) Discovery() discovery.Discovery {
	return a.discovery
}

// MQ 获取消息队列
func (a *App) MQ() mq.MessageQueue {
	return a.mq
}

// Auth 获取鉴权器
func (a *App) Auth() auth.Authenticator {
	return a.auth
}

// Health 获取健康检查管理器
func (a *App) Health() *HealthManager {
	return a.health
}

// Run 启动应用
func (a *App) Run() error {
	a.mu.Lock()
	if a.running {
		a.mu.Unlock()
		return fmt.Errorf("app is already running")
	}
	a.running = true
	a.mu.Unlock()

	// 执行启动前钩子
	for _, fn := range a.beforeStart {
		if err := fn(); err != nil {
			return fmt.Errorf("before start hook failed: %w", err)
		}
	}

	// 启动HTTP服务器
	if a.httpServer != nil {
		a.registerBuiltinHTTPEndpoints()

		go func() {
			if err := a.httpServer.Start(); err != nil {
				if a.logger != nil {
					a.logger.Error("HTTP server error", log.Error(err))
				}
			}
		}()
	}

	// 启动gRPC服务器
	if a.grpcServer != nil {
		go func() {
			if err := a.grpcServer.Start(); err != nil {
				if a.logger != nil {
					a.logger.Error("gRPC server error", log.Error(err))
				}
			}
		}()
	}

	// 注册服务
	if a.discovery != nil && a.config != nil {
		address := a.config.Server.HTTP.Host
		if a.config.Server.HTTP.AdvertiseHost != "" {
			address = a.config.Server.HTTP.AdvertiseHost
		}

		serviceInfo := &discovery.ServiceInfo{
			ID:      a.config.Consul.Service.ID,
			Name:    a.config.Consul.Service.Name,
			Address: address,
			Port:    a.config.Server.HTTP.Port,
			Tags:    a.config.Consul.Service.Tags,
		}
		if err := a.discovery.Register(context.Background(), serviceInfo); err != nil {
			if a.logger != nil {
				a.logger.Warn("register service failed", log.Error(err))
			}
		}
	}

	// 执行启动后钩子
	for _, fn := range a.afterStart {
		if err := fn(); err != nil {
			return fmt.Errorf("after start hook failed: %w", err)
		}
	}

	if a.logger != nil {
		a.logger.Info("application started", log.String("name", a.name))
	}

	// 等待信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	if a.logger != nil {
		a.logger.Info("shutting down application...")
	}

	return a.Shutdown(context.Background())
}

func (a *App) registerBuiltinHTTPEndpoints() {
	// 注册 /metrics
	if a.metricsConfig != nil && a.metricsConfig.Enabled {
		cfg := *a.metricsConfig

		// 默认跳过内置诊断端点，避免污染业务指标
		if a.health != nil {
			cfg.SkipPaths = append(cfg.SkipPaths,
				a.health.Config().LivenessPath,
				a.health.Config().ReadinessPath,
			)
		}
		if a.pprofConfig != nil && a.pprofConfig.Enabled {
			base := normalizeBasePath(a.pprofConfig.BasePath, "/debug/pprof")
			cfg.SkipPaths = append(cfg.SkipPaths, base, base+"/")
		}

		a.metrics = registerMetrics(a.httpServer, &cfg)
	}

	// 注册健康检查端点
	if a.health != nil {
		a.httpServer.GET(a.health.Config().LivenessPath, a.health.LivenessHandler())
		a.httpServer.GET(a.health.Config().ReadinessPath, a.health.ReadinessHandler())
	}

	// 注册 /debug/pprof
	if a.pprofConfig != nil && a.pprofConfig.Enabled {
		registerPProf(a.httpServer, a.pprofConfig)
	}
}

// Shutdown 关闭应用
func (a *App) Shutdown(ctx context.Context) error {
	a.mu.Lock()
	if !a.running {
		a.mu.Unlock()
		return nil
	}
	a.running = false
	a.mu.Unlock()

	// 创建超时上下文
	shutdownCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// 执行停止前钩子
	for _, fn := range a.beforeStop {
		if err := fn(); err != nil {
			if a.logger != nil {
				a.logger.Warn("before stop hook failed", log.Error(err))
			}
		}
	}

	// 注销服务
	if a.discovery != nil && a.config != nil {
		_ = a.discovery.Deregister(shutdownCtx, a.config.Consul.Service.ID)
	}

	// 关闭HTTP服务器
	if a.httpServer != nil {
		if err := a.httpServer.Stop(shutdownCtx); err != nil {
			if a.logger != nil {
				a.logger.Warn("stop HTTP server failed", log.Error(err))
			}
		}
	}

	// 关闭gRPC服务器
	if a.grpcServer != nil {
		if err := a.grpcServer.Stop(shutdownCtx); err != nil {
			if a.logger != nil {
				a.logger.Warn("stop gRPC server failed", log.Error(err))
			}
		}
	}

	// 关闭消息队列
	if a.mq != nil {
		if err := a.mq.Close(); err != nil {
			if a.logger != nil {
				a.logger.Warn("close message queue failed", log.Error(err))
			}
		}
	}

	// 关闭缓存
	if a.cache != nil {
		if err := a.cache.Close(); err != nil {
			if a.logger != nil {
				a.logger.Warn("close cache failed", log.Error(err))
			}
		}
	}

	// 关闭数据库
	if a.db != nil {
		sqlDB, err := a.db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	}

	// 关闭服务发现
	if a.discovery != nil {
		_ = a.discovery.Close()
	}

	// 执行停止后钩子
	for _, fn := range a.afterStop {
		if err := fn(); err != nil {
			if a.logger != nil {
				a.logger.Warn("after stop hook failed", log.Error(err))
			}
		}
	}

	// 同步日志
	if a.logger != nil {
		_ = a.logger.Sync()
		a.logger.Info("application stopped", log.String("name", a.name))
	}

	return nil
}

// IsRunning 是否正在运行
func (a *App) IsRunning() bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.running
}
