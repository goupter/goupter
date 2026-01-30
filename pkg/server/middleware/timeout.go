package middleware

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// TimeoutConfig 超时配置
type TimeoutConfig struct {
	Timeout    time.Duration // 超时时间
	Message    string        // 超时提示消息
	StatusCode int           // 超时状态码
}

// DefaultTimeoutConfig 默认超时配置
func DefaultTimeoutConfig() TimeoutConfig {
	return TimeoutConfig{
		Timeout:    30 * time.Second,
		Message:    "Request timeout",
		StatusCode: http.StatusGatewayTimeout,
	}
}

// Timeout 超时中间件
func Timeout(timeout time.Duration) gin.HandlerFunc {
	cfg := DefaultTimeoutConfig()
	cfg.Timeout = timeout
	return TimeoutWithConfig(cfg)
}

// TimeoutWithConfig 带配置的超时中间件
// Note: 使用 sync.Once 确保响应只写一次，避免竞态条件
func TimeoutWithConfig(cfg TimeoutConfig) gin.HandlerFunc {
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}
	if cfg.Message == "" {
		cfg.Message = "Request timeout"
	}
	if cfg.StatusCode == 0 {
		cfg.StatusCode = http.StatusGatewayTimeout
	}

	return func(c *gin.Context) {
		// 创建超时上下文
		ctx, cancel := context.WithTimeout(c.Request.Context(), cfg.Timeout)
		defer cancel()

		// 替换请求上下文
		c.Request = c.Request.WithContext(ctx)

		// 使用 sync.Once 确保响应只写一次
		var once sync.Once
		var finished bool
		var mu sync.Mutex

		done := make(chan struct{})
		panicChan := make(chan interface{}, 1)

		go func() {
			defer func() {
				if p := recover(); p != nil {
					panicChan <- p
				}
			}()
			c.Next()
			mu.Lock()
			finished = true
			mu.Unlock()
			close(done)
		}()

		select {
		case <-done:
			// 正常完成
			return
		case p := <-panicChan:
			// 发生panic
			panic(p)
		case <-ctx.Done():
			// 超时 - 使用 once 确保只写一次响应
			mu.Lock()
			isFinished := finished
			mu.Unlock()
			if !isFinished {
				once.Do(func() {
					c.AbortWithStatusJSON(cfg.StatusCode, gin.H{
						"code":    cfg.StatusCode,
						"message": cfg.Message,
					})
				})
			}
			return
		}
	}
}

// ContextTimeout 上下文超时中间件（推荐使用）
// 只设置上下文超时，不拦截响应
func ContextTimeout(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

// DeadlineMiddleware 截止时间中间件
func DeadlineMiddleware(deadline time.Time) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithDeadline(c.Request.Context(), deadline)
		defer cancel()

		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

// RequestTimeout 请求超时控制（支持从Header读取）
func RequestTimeout(defaultTimeout time.Duration, maxTimeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		timeout := defaultTimeout

		// 尝试从Header获取自定义超时
		if timeoutHeader := c.GetHeader("X-Request-Timeout"); timeoutHeader != "" {
			if d, err := time.ParseDuration(timeoutHeader); err == nil {
				timeout = d
			}
		}

		// 限制最大超时时间
		if maxTimeout > 0 && timeout > maxTimeout {
			timeout = maxTimeout
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

// TimeoutResponse 超时响应包装器
type TimeoutResponse struct {
	gin.ResponseWriter
	timedOut bool
	mu       sync.Mutex
}

// Write 实现Write方法
func (tr *TimeoutResponse) Write(b []byte) (int, error) {
	tr.mu.Lock()
	timedOut := tr.timedOut
	tr.mu.Unlock()
	if timedOut {
		return 0, context.DeadlineExceeded
	}
	return tr.ResponseWriter.Write(b)
}

// WriteHeader 实现WriteHeader方法
func (tr *TimeoutResponse) WriteHeader(code int) {
	tr.mu.Lock()
	timedOut := tr.timedOut
	tr.mu.Unlock()
	if !timedOut {
		tr.ResponseWriter.WriteHeader(code)
	}
}

// WriteString 实现WriteString方法
func (tr *TimeoutResponse) WriteString(s string) (int, error) {
	tr.mu.Lock()
	timedOut := tr.timedOut
	tr.mu.Unlock()
	if timedOut {
		return 0, context.DeadlineExceeded
	}
	return tr.ResponseWriter.WriteString(s)
}

// TimeoutWithCustomResponse 支持自定义超时响应的中间件
// Note: 使用 sync.Mutex 保护 timedOut 标志，避免竞态条件
func TimeoutWithCustomResponse(timeout time.Duration, timeoutHandler gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		c.Request = c.Request.WithContext(ctx)

		tr := &TimeoutResponse{ResponseWriter: c.Writer}
		c.Writer = tr

		var finished bool
		var mu sync.Mutex
		done := make(chan struct{})

		go func() {
			defer close(done)
			c.Next()
			mu.Lock()
			finished = true
			mu.Unlock()
		}()

		select {
		case <-done:
			return
		case <-ctx.Done():
			mu.Lock()
			isFinished := finished
			mu.Unlock()
			if !isFinished {
				tr.mu.Lock()
				tr.timedOut = true
				tr.mu.Unlock()
				if timeoutHandler != nil {
					timeoutHandler(c)
				} else {
					c.AbortWithStatusJSON(http.StatusGatewayTimeout, gin.H{
						"code":    http.StatusGatewayTimeout,
						"message": "Request timeout",
					})
				}
			}
		}
	}
}
