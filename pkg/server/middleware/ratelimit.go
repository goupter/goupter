package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimiter 限流器接口
type RateLimiter interface {
	// Allow 判断是否允许请求
	Allow(key string) bool
	// AllowN 判断是否允许N个请求
	AllowN(key string, n int) bool
}

// === 令牌桶限流器 ===

// TokenBucket 令牌桶
type TokenBucket struct {
	mu         sync.Mutex
	rate       float64   // 令牌生成速率（每秒）
	capacity   int       // 桶容量
	tokens     float64   // 当前令牌数
	lastUpdate time.Time // 上次更新时间
}

// NewTokenBucket 创建令牌桶
func NewTokenBucket(rate float64, capacity int) *TokenBucket {
	return &TokenBucket{
		rate:       rate,
		capacity:   capacity,
		tokens:     float64(capacity),
		lastUpdate: time.Now(),
	}
}

// Allow 判断是否允许一个请求
func (tb *TokenBucket) Allow(key string) bool {
	return tb.AllowN(key, 1)
}

// AllowN 判断是否允许N个请求
func (tb *TokenBucket) AllowN(key string, n int) bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(tb.lastUpdate).Seconds()
	tb.lastUpdate = now

	// 添加新令牌
	tb.tokens += elapsed * tb.rate
	if tb.tokens > float64(tb.capacity) {
		tb.tokens = float64(tb.capacity)
	}

	// 检查是否有足够令牌
	if tb.tokens >= float64(n) {
		tb.tokens -= float64(n)
		return true
	}

	return false
}

// === 滑动窗口限流器 ===

// SlidingWindowLimiter 滑动窗口限流器
type SlidingWindowLimiter struct {
	mu        sync.Mutex
	windows   map[string]*window
	limit     int           // 窗口内最大请求数
	windowLen time.Duration // 窗口长度
}

type window struct {
	count     int
	startTime time.Time
}

// NewSlidingWindowLimiter 创建滑动窗口限流器
func NewSlidingWindowLimiter(limit int, windowLen time.Duration) *SlidingWindowLimiter {
	limiter := &SlidingWindowLimiter{
		windows:   make(map[string]*window),
		limit:     limit,
		windowLen: windowLen,
	}
	// 定期清理过期窗口
	go limiter.cleanup()
	return limiter
}

// Allow 判断是否允许请求
func (l *SlidingWindowLimiter) Allow(key string) bool {
	return l.AllowN(key, 1)
}

// AllowN 判断是否允许N个请求
func (l *SlidingWindowLimiter) AllowN(key string, n int) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	w, ok := l.windows[key]

	if !ok || now.Sub(w.startTime) >= l.windowLen {
		// 创建新窗口
		l.windows[key] = &window{
			count:     n,
			startTime: now,
		}
		return n <= l.limit
	}

	// 检查是否超过限制
	if w.count+n <= l.limit {
		w.count += n
		return true
	}

	return false
}

// cleanup 清理过期窗口
func (l *SlidingWindowLimiter) cleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		l.mu.Lock()
		now := time.Now()
		for key, w := range l.windows {
			if now.Sub(w.startTime) >= l.windowLen*2 {
				delete(l.windows, key)
			}
		}
		l.mu.Unlock()
	}
}

// === 基于IP的限流器 ===

// IPRateLimiter 基于IP的限流器
type IPRateLimiter struct {
	mu       sync.RWMutex
	limiters map[string]*TokenBucket
	rate     float64
	capacity int
}

// NewIPRateLimiter 创建IP限流器
func NewIPRateLimiter(rate float64, capacity int) *IPRateLimiter {
	limiter := &IPRateLimiter{
		limiters: make(map[string]*TokenBucket),
		rate:     rate,
		capacity: capacity,
	}
	// 定期清理
	go limiter.cleanup()
	return limiter
}

// getLimiter 获取或创建限流器
func (l *IPRateLimiter) getLimiter(ip string) *TokenBucket {
	l.mu.RLock()
	limiter, ok := l.limiters[ip]
	l.mu.RUnlock()

	if ok {
		return limiter
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	// 双重检查
	if limiter, ok = l.limiters[ip]; ok {
		return limiter
	}

	limiter = NewTokenBucket(l.rate, l.capacity)
	l.limiters[ip] = limiter
	return limiter
}

// Allow 判断是否允许请求
func (l *IPRateLimiter) Allow(key string) bool {
	return l.getLimiter(key).Allow(key)
}

// AllowN 判断是否允许N个请求
func (l *IPRateLimiter) AllowN(key string, n int) bool {
	return l.getLimiter(key).AllowN(key, n)
}

// cleanup 清理过期的限流器
func (l *IPRateLimiter) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		l.mu.Lock()
		// 简单策略：如果令牌桶满了就删除
		for ip, limiter := range l.limiters {
			limiter.mu.Lock()
			if limiter.tokens >= float64(limiter.capacity) {
				delete(l.limiters, ip)
			}
			limiter.mu.Unlock()
		}
		l.mu.Unlock()
	}
}

// === 限流中间件 ===

// RateLimitConfig 限流配置
type RateLimitConfig struct {
	Limiter     RateLimiter
	KeyFunc     func(*gin.Context) string // 获取限流key的函数
	Message     string                    // 限流提示消息
	StatusCode  int                       // 限流状态码
	AbortOnFail bool                      // 限流时是否中断请求
}

// DefaultRateLimitConfig 默认限流配置
func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		Limiter:     NewIPRateLimiter(10, 100), // 每秒10个，最多100个
		KeyFunc:     func(c *gin.Context) string { return c.ClientIP() },
		Message:     "Too many requests",
		StatusCode:  http.StatusTooManyRequests,
		AbortOnFail: true,
	}
}

// RateLimit 限流中间件
func RateLimit(limiter RateLimiter) gin.HandlerFunc {
	cfg := DefaultRateLimitConfig()
	cfg.Limiter = limiter
	return RateLimitWithConfig(cfg)
}

// RateLimitWithConfig 带配置的限流中间件
func RateLimitWithConfig(cfg RateLimitConfig) gin.HandlerFunc {
	if cfg.KeyFunc == nil {
		cfg.KeyFunc = func(c *gin.Context) string { return c.ClientIP() }
	}
	if cfg.StatusCode == 0 {
		cfg.StatusCode = http.StatusTooManyRequests
	}
	if cfg.Message == "" {
		cfg.Message = "Too many requests"
	}

	return func(c *gin.Context) {
		key := cfg.KeyFunc(c)

		if !cfg.Limiter.Allow(key) {
			c.Header("X-RateLimit-Exceeded", "true")
			c.Header("Retry-After", "1")

			if cfg.AbortOnFail {
				c.AbortWithStatusJSON(cfg.StatusCode, gin.H{
					"code":    cfg.StatusCode,
					"message": cfg.Message,
				})
				return
			}
		}

		c.Next()
	}
}

// IPRateLimit 基于IP的限流中间件
func IPRateLimit(rate float64, capacity int) gin.HandlerFunc {
	limiter := NewIPRateLimiter(rate, capacity)
	return RateLimit(limiter)
}

// PathRateLimit 基于路径的限流中间件
func PathRateLimit(rate float64, capacity int) gin.HandlerFunc {
	limiter := NewIPRateLimiter(rate, capacity)
	cfg := DefaultRateLimitConfig()
	cfg.Limiter = limiter
	cfg.KeyFunc = func(c *gin.Context) string {
		return c.ClientIP() + ":" + c.Request.URL.Path
	}
	return RateLimitWithConfig(cfg)
}

// UserRateLimit 基于用户的限流中间件
func UserRateLimit(rate float64, capacity int, getUserID func(*gin.Context) string) gin.HandlerFunc {
	limiter := NewIPRateLimiter(rate, capacity)
	cfg := DefaultRateLimitConfig()
	cfg.Limiter = limiter
	cfg.KeyFunc = func(c *gin.Context) string {
		userID := getUserID(c)
		if userID == "" {
			return c.ClientIP()
		}
		return userID
	}
	return RateLimitWithConfig(cfg)
}
