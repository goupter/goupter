package app

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// HealthStatus 健康状态
type HealthStatus string

const (
	StatusUp   HealthStatus = "up"
	StatusDown HealthStatus = "down"
)

// CheckResult 检查结果
type CheckResult struct {
	Status    HealthStatus  `json:"status"`
	Message   string        `json:"message,omitempty"`
	Duration  time.Duration `json:"duration,omitempty"`
	Timestamp time.Time     `json:"timestamp"`
}

// HealthChecker 健康检查器接口
type HealthChecker interface {
	Name() string
	Check(ctx context.Context) CheckResult
}

// HealthCheckFunc 健康检查函数类型
type HealthCheckFunc func(ctx context.Context) CheckResult

// FuncChecker 函数式健康检查器
type FuncChecker struct {
	name  string
	check HealthCheckFunc
}

// NewFuncChecker 创建函数式健康检查器
func NewFuncChecker(name string, check HealthCheckFunc) *FuncChecker {
	return &FuncChecker{name: name, check: check}
}

func (c *FuncChecker) Name() string                         { return c.name }
func (c *FuncChecker) Check(ctx context.Context) CheckResult { return c.check(ctx) }

// PingChecker 通用 Ping 检查器
type PingChecker struct {
	name    string
	pingFn  func(ctx context.Context) error
	timeout time.Duration
}

// NewPingChecker 创建 Ping 检查器
func NewPingChecker(name string, pingFn func(ctx context.Context) error) *PingChecker {
	return &PingChecker{name: name, pingFn: pingFn, timeout: 3 * time.Second}
}

func (c *PingChecker) Name() string { return c.name }

func (c *PingChecker) Check(ctx context.Context) CheckResult {
	start := time.Now()
	result := CheckResult{Timestamp: start}

	checkCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	if err := c.pingFn(checkCtx); err != nil {
		result.Status = StatusDown
		result.Message = err.Error()
	} else {
		result.Status = StatusUp
	}

	result.Duration = time.Since(start)
	return result
}

// HealthResponse 健康检查响应
type HealthResponse struct {
	Status    HealthStatus            `json:"status"`
	Timestamp time.Time               `json:"timestamp"`
	Duration  string                  `json:"duration"`
	Checks    map[string]*CheckResult `json:"checks,omitempty"`
}

// ReadyResponse 就绪检查响应
type ReadyResponse struct {
	Ready     bool                    `json:"ready"`
	Status    HealthStatus            `json:"status"`
	Timestamp time.Time               `json:"timestamp"`
	Duration  string                  `json:"duration"`
	Checks    map[string]*CheckResult `json:"checks,omitempty"`
}

// HealthConfig 健康检查配置
type HealthConfig struct {
	LivenessPath  string
	ReadinessPath string
	Timeout       time.Duration
	CacheTTL      time.Duration
}

// DefaultHealthConfig 默认健康检查配置
func DefaultHealthConfig() *HealthConfig {
	return &HealthConfig{
		LivenessPath:  "/health",
		ReadinessPath: "/ready",
		Timeout:       5 * time.Second,
		CacheTTL:      time.Second,
	}
}

// HealthManager 健康检查管理器
type HealthManager struct {
	config            *HealthConfig
	livenessCheckers  []HealthChecker
	readinessCheckers []HealthChecker
	livenessCache     *HealthResponse
	readinessCache    *ReadyResponse
	livenessTTL       time.Time
	readinessTTL      time.Time
	mu                sync.RWMutex
}

// NewHealthManager 创建健康检查管理器
func NewHealthManager(cfg *HealthConfig) *HealthManager {
	if cfg == nil {
		cfg = DefaultHealthConfig()
	}
	return &HealthManager{
		config:            cfg,
		livenessCheckers:  make([]HealthChecker, 0),
		readinessCheckers: make([]HealthChecker, 0),
	}
}

// RegisterLiveness 注册存活检查器
func (m *HealthManager) RegisterLiveness(checker HealthChecker) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.livenessCheckers = append(m.livenessCheckers, checker)
}

// RegisterReadiness 注册就绪检查器
func (m *HealthManager) RegisterReadiness(checker HealthChecker) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.readinessCheckers = append(m.readinessCheckers, checker)
}

// RegisterLivenessFunc 注册存活检查函数
func (m *HealthManager) RegisterLivenessFunc(name string, check HealthCheckFunc) {
	m.RegisterLiveness(NewFuncChecker(name, check))
}

// RegisterReadinessFunc 注册就绪检查函数
func (m *HealthManager) RegisterReadinessFunc(name string, check HealthCheckFunc) {
	m.RegisterReadiness(NewFuncChecker(name, check))
}

// CheckLiveness 执行存活检查
func (m *HealthManager) CheckLiveness(ctx context.Context) *HealthResponse {
	m.mu.RLock()
	if m.livenessCache != nil && time.Now().Before(m.livenessTTL) {
		cache := m.livenessCache
		m.mu.RUnlock()
		return cache
	}
	m.mu.RUnlock()

	start := time.Now()
	response := &HealthResponse{
		Status:    StatusUp,
		Timestamp: start,
		Checks:    make(map[string]*CheckResult),
	}

	m.mu.RLock()
	checkers := m.livenessCheckers
	m.mu.RUnlock()

	if len(checkers) == 0 {
		response.Duration = time.Since(start).String()
		m.cacheResponse(response, nil)
		return response
	}

	checkCtx, cancel := context.WithTimeout(ctx, m.config.Timeout)
	defer cancel()

	for _, checker := range checkers {
		result := checker.Check(checkCtx)
		response.Checks[checker.Name()] = &result
		if result.Status == StatusDown {
			response.Status = StatusDown
		}
	}

	response.Duration = time.Since(start).String()
	m.cacheResponse(response, nil)
	return response
}

// CheckReadiness 执行就绪检查
func (m *HealthManager) CheckReadiness(ctx context.Context) *ReadyResponse {
	m.mu.RLock()
	if m.readinessCache != nil && time.Now().Before(m.readinessTTL) {
		cache := m.readinessCache
		m.mu.RUnlock()
		return cache
	}
	m.mu.RUnlock()

	start := time.Now()
	response := &ReadyResponse{
		Ready:     true,
		Status:    StatusUp,
		Timestamp: start,
		Checks:    make(map[string]*CheckResult),
	}

	m.mu.RLock()
	checkers := m.readinessCheckers
	m.mu.RUnlock()

	if len(checkers) == 0 {
		response.Duration = time.Since(start).String()
		m.cacheResponse(nil, response)
		return response
	}

	checkCtx, cancel := context.WithTimeout(ctx, m.config.Timeout)
	defer cancel()

	for _, checker := range checkers {
		result := checker.Check(checkCtx)
		response.Checks[checker.Name()] = &result
		if result.Status == StatusDown {
			response.Status = StatusDown
			response.Ready = false
		}
	}

	response.Duration = time.Since(start).String()
	m.cacheResponse(nil, response)
	return response
}

func (m *HealthManager) cacheResponse(liveness *HealthResponse, readiness *ReadyResponse) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if liveness != nil {
		m.livenessCache = liveness
		m.livenessTTL = time.Now().Add(m.config.CacheTTL)
	}
	if readiness != nil {
		m.readinessCache = readiness
		m.readinessTTL = time.Now().Add(m.config.CacheTTL)
	}
}

// ClearCache 清除缓存
func (m *HealthManager) ClearCache() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.livenessCache = nil
	m.readinessCache = nil
}

// LivenessHandler 返回存活检查的 Gin 处理器
func (m *HealthManager) LivenessHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		response := m.CheckLiveness(c.Request.Context())
		status := http.StatusOK
		if response.Status == StatusDown {
			status = http.StatusServiceUnavailable
		}
		c.JSON(status, response)
	}
}

// ReadinessHandler 返回就绪检查的 Gin 处理器
func (m *HealthManager) ReadinessHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		response := m.CheckReadiness(c.Request.Context())
		status := http.StatusOK
		if !response.Ready {
			status = http.StatusServiceUnavailable
		}
		c.JSON(status, response)
	}
}

// Config 返回健康检查配置
func (m *HealthManager) Config() *HealthConfig {
	return m.config
}
