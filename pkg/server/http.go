package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/goupter/goupter/pkg/config"
	"github.com/goupter/goupter/pkg/log"
)

// HTTPServer HTTP服务器
type HTTPServer struct {
	engine     *gin.Engine
	server     *http.Server
	config     *config.HTTPConfig
	logger     log.Logger
	middleware []gin.HandlerFunc
}

// HTTPOption HTTP服务器选项
type HTTPOption func(*HTTPServer)

// WithHTTPConfig 设置HTTP配置
func WithHTTPConfig(cfg *config.HTTPConfig) HTTPOption {
	return func(s *HTTPServer) {
		s.config = cfg
	}
}

// WithHTTPLogger 设置日志
func WithHTTPLogger(logger log.Logger) HTTPOption {
	return func(s *HTTPServer) {
		s.logger = logger
	}
}

// WithMiddleware 添加中间件
func WithMiddleware(middleware ...gin.HandlerFunc) HTTPOption {
	return func(s *HTTPServer) {
		s.middleware = append(s.middleware, middleware...)
	}
}

// NewHTTPServer 创建HTTP服务器
func NewHTTPServer(opts ...HTTPOption) *HTTPServer {
	s := &HTTPServer{
		config: &config.HTTPConfig{
			Host:         "0.0.0.0",
			Port:         8080,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  120 * time.Second,
		},
		middleware: make([]gin.HandlerFunc, 0),
	}

	for _, opt := range opts {
		opt(s)
	}

	// 根据环境设置Gin模式
	cfg := config.Get()
	if cfg != nil && cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	s.engine = gin.New()

	// 应用中间件
	for _, m := range s.middleware {
		s.engine.Use(m)
	}

	return s
}

// Engine 获取Gin引擎
func (s *HTTPServer) Engine() *gin.Engine {
	return s.engine
}

// Use 添加中间件
func (s *HTTPServer) Use(middleware ...gin.HandlerFunc) {
	s.engine.Use(middleware...)
}

// Group 创建路由组
func (s *HTTPServer) Group(relativePath string, handlers ...gin.HandlerFunc) *gin.RouterGroup {
	return s.engine.Group(relativePath, handlers...)
}

// GET 注册GET路由
func (s *HTTPServer) GET(relativePath string, handlers ...gin.HandlerFunc) gin.IRoutes {
	return s.engine.GET(relativePath, handlers...)
}

// POST 注册POST路由
func (s *HTTPServer) POST(relativePath string, handlers ...gin.HandlerFunc) gin.IRoutes {
	return s.engine.POST(relativePath, handlers...)
}

// PUT 注册PUT路由
func (s *HTTPServer) PUT(relativePath string, handlers ...gin.HandlerFunc) gin.IRoutes {
	return s.engine.PUT(relativePath, handlers...)
}

// DELETE 注册DELETE路由
func (s *HTTPServer) DELETE(relativePath string, handlers ...gin.HandlerFunc) gin.IRoutes {
	return s.engine.DELETE(relativePath, handlers...)
}

// PATCH 注册PATCH路由
func (s *HTTPServer) PATCH(relativePath string, handlers ...gin.HandlerFunc) gin.IRoutes {
	return s.engine.PATCH(relativePath, handlers...)
}

// OPTIONS 注册OPTIONS路由
func (s *HTTPServer) OPTIONS(relativePath string, handlers ...gin.HandlerFunc) gin.IRoutes {
	return s.engine.OPTIONS(relativePath, handlers...)
}

// HEAD 注册HEAD路由
func (s *HTTPServer) HEAD(relativePath string, handlers ...gin.HandlerFunc) gin.IRoutes {
	return s.engine.HEAD(relativePath, handlers...)
}

// Any 注册所有方法路由
func (s *HTTPServer) Any(relativePath string, handlers ...gin.HandlerFunc) gin.IRoutes {
	return s.engine.Any(relativePath, handlers...)
}

// Handle 注册自定义方法路由
func (s *HTTPServer) Handle(httpMethod, relativePath string, handlers ...gin.HandlerFunc) gin.IRoutes {
	return s.engine.Handle(httpMethod, relativePath, handlers...)
}

// Static 提供静态文件服务
func (s *HTTPServer) Static(relativePath, root string) gin.IRoutes {
	return s.engine.Static(relativePath, root)
}

// StaticFile 提供单个静态文件
func (s *HTTPServer) StaticFile(relativePath, filepath string) gin.IRoutes {
	return s.engine.StaticFile(relativePath, filepath)
}

// Start 启动HTTP服务器
func (s *HTTPServer) Start() error {
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)

	s.server = &http.Server{
		Addr:         addr,
		Handler:      s.engine,
		ReadTimeout:  s.config.ReadTimeout,
		WriteTimeout: s.config.WriteTimeout,
		IdleTimeout:  s.config.IdleTimeout,
	}

	if s.logger != nil {
		s.logger.Info("HTTP server starting", log.String("addr", addr))
	}

	return s.server.ListenAndServe()
}

// Stop 停止HTTP服务器
func (s *HTTPServer) Stop(ctx context.Context) error {
	if s.server == nil {
		return nil
	}

	if s.logger != nil {
		s.logger.Info("HTTP server stopping")
	}

	return s.server.Shutdown(ctx)
}

// Addr 获取服务器地址
func (s *HTTPServer) Addr() string {
	return fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
}

// Router 路由接口
type Router interface {
	Routes() []Route
}

// Route 路由信息
type Route struct {
	Method  string
	Path    string
	Handler gin.HandlerFunc
}

// RegisterRouter 注册路由
func (s *HTTPServer) RegisterRouter(router Router) {
	for _, route := range router.Routes() {
		s.engine.Handle(route.Method, route.Path, route.Handler)
	}
}

// RegisterRoutes 批量注册路由
func (s *HTTPServer) RegisterRoutes(routes []Route) {
	for _, route := range routes {
		s.engine.Handle(route.Method, route.Path, route.Handler)
	}
}

// HealthCheck 健康检查路由
func (s *HTTPServer) HealthCheck(path string) {
	if path == "" {
		path = "/health"
	}
	s.engine.GET(path, func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
			"time":   time.Now().Format(time.RFC3339),
		})
	})
}

// NoRoute 404处理
func (s *HTTPServer) NoRoute(handlers ...gin.HandlerFunc) {
	s.engine.NoRoute(handlers...)
}

// NoMethod 405处理
func (s *HTTPServer) NoMethod(handlers ...gin.HandlerFunc) {
	s.engine.NoMethod(handlers...)
}
