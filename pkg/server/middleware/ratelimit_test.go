package middleware

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// === 令牌桶测试 ===

func TestTokenBucket_Allow(t *testing.T) {
	tb := NewTokenBucket(10, 10) // 每秒10个令牌，容量10

	// 连续请求应该成功（消耗初始令牌）
	for i := 0; i < 10; i++ {
		if !tb.Allow("test") {
			t.Errorf("请求 %d 应该被允许", i)
		}
	}

	// 第11个请求应该被拒绝
	if tb.Allow("test") {
		t.Error("第11个请求应该被拒绝")
	}
}

func TestTokenBucket_Refill(t *testing.T) {
	tb := NewTokenBucket(100, 10) // 每秒100个令牌，容量10

	// 消耗所有令牌
	for i := 0; i < 10; i++ {
		tb.Allow("test")
	}

	// 等待令牌恢复
	time.Sleep(100 * time.Millisecond)

	// 现在应该有新令牌
	if !tb.Allow("test") {
		t.Error("等待后应该有新令牌")
	}
}

func TestTokenBucket_AllowN(t *testing.T) {
	tb := NewTokenBucket(10, 10)

	// 请求5个令牌
	if !tb.AllowN("test", 5) {
		t.Error("应该允许5个令牌")
	}

	// 再请求5个
	if !tb.AllowN("test", 5) {
		t.Error("应该允许再5个令牌")
	}

	// 第11个应该失败
	if tb.AllowN("test", 1) {
		t.Error("应该拒绝第11个令牌")
	}
}

// === 滑动窗口测试 ===

func TestSlidingWindowLimiter_Allow(t *testing.T) {
	limiter := NewSlidingWindowLimiter(10, time.Second)

	// 10个请求应该成功
	for i := 0; i < 10; i++ {
		if !limiter.Allow("test") {
			t.Errorf("请求 %d 应该被允许", i)
		}
	}

	// 第11个应该失败
	if limiter.Allow("test") {
		t.Error("第11个请求应该被拒绝")
	}
}

func TestSlidingWindowLimiter_WindowReset(t *testing.T) {
	limiter := NewSlidingWindowLimiter(5, 100*time.Millisecond)

	// 消耗配额
	for i := 0; i < 5; i++ {
		limiter.Allow("test")
	}

	// 等待窗口重置
	time.Sleep(150 * time.Millisecond)

	// 现在应该可以再次请求
	if !limiter.Allow("test") {
		t.Error("窗口重置后应该允许请求")
	}
}

// === IP限流器测试 ===

func TestIPRateLimiter(t *testing.T) {
	limiter := NewIPRateLimiter(10, 5)

	// 不同IP应该有独立配额
	for i := 0; i < 5; i++ {
		if !limiter.Allow("192.168.1.1") {
			t.Errorf("IP1 请求 %d 应该被允许", i)
		}
		if !limiter.Allow("192.168.1.2") {
			t.Errorf("IP2 请求 %d 应该被允许", i)
		}
	}

	// IP1配额用完
	if limiter.Allow("192.168.1.1") {
		t.Error("IP1 应该被限流")
	}

	// IP2配额用完
	if limiter.Allow("192.168.1.2") {
		t.Error("IP2 应该被限流")
	}
}

// === 限流中间件测试 ===

func TestRateLimitMiddleware(t *testing.T) {
	limiter := NewTokenBucket(100, 5)

	router := gin.New()
	router.Use(RateLimit(limiter))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	// 前5个请求应该成功
	for i := 0; i < 5; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("请求 %d 应该返回200，实际返回 %d", i, w.Code)
		}
	}

	// 第6个请求应该被限流
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("第6个请求应该返回429，实际返回 %d", w.Code)
	}

	// 检查限流头
	if w.Header().Get("X-RateLimit-Exceeded") != "true" {
		t.Error("应该设置 X-RateLimit-Exceeded 头")
	}
}

func TestIPRateLimitMiddleware(t *testing.T) {
	router := gin.New()
	router.Use(IPRateLimit(100, 3))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	// 测试限流
	for i := 0; i < 4; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		router.ServeHTTP(w, req)

		if i < 3 && w.Code != http.StatusOK {
			t.Errorf("请求 %d 应该返回200", i)
		}
		if i == 3 && w.Code != http.StatusTooManyRequests {
			t.Errorf("请求 %d 应该返回429", i)
		}
	}
}

func TestPathRateLimitMiddleware(t *testing.T) {
	router := gin.New()
	router.Use(PathRateLimit(100, 2))
	router.GET("/api/v1", func(c *gin.Context) {
		c.String(http.StatusOK, "v1")
	})
	router.GET("/api/v2", func(c *gin.Context) {
		c.String(http.StatusOK, "v2")
	})

	// /api/v1 的2个请求
	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1", nil)
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("/api/v1 请求 %d 应该返回200", i)
		}
	}

	// /api/v1 第3个请求应该被限流
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1", nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusTooManyRequests {
		t.Error("/api/v1 第3个请求应该返回429")
	}

	// /api/v2 应该有独立配额
	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api/v2", nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Error("/api/v2 应该返回200")
	}
}

// === 并发测试 ===

func TestTokenBucket_Concurrent(t *testing.T) {
	tb := NewTokenBucket(1000, 100)

	var wg sync.WaitGroup
	var allowed, denied int64
	var mu sync.Mutex

	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if tb.Allow("test") {
				mu.Lock()
				allowed++
				mu.Unlock()
			} else {
				mu.Lock()
				denied++
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	if allowed > 100 {
		t.Errorf("允许的请求数不应超过100，实际: %d", allowed)
	}

	t.Logf("并发测试: 允许=%d, 拒绝=%d", allowed, denied)
}

func TestSlidingWindowLimiter_Concurrent(t *testing.T) {
	limiter := NewSlidingWindowLimiter(50, time.Second)

	var wg sync.WaitGroup
	var allowed, denied int64
	var mu sync.Mutex

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if limiter.Allow("test") {
				mu.Lock()
				allowed++
				mu.Unlock()
			} else {
				mu.Lock()
				denied++
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	if allowed > 50 {
		t.Errorf("允许的请求数不应超过50，实际: %d", allowed)
	}

	t.Logf("并发测试: 允许=%d, 拒绝=%d", allowed, denied)
}

// === 配置测试 ===

func TestRateLimitWithConfig(t *testing.T) {
	// 使用IPRateLimiter使每个用户有独立的限流桶
	cfg := RateLimitConfig{
		Limiter:     NewIPRateLimiter(100, 2),
		KeyFunc:     func(c *gin.Context) string { return c.GetHeader("X-User-ID") },
		Message:     "Custom rate limit message",
		StatusCode:  http.StatusServiceUnavailable,
		AbortOnFail: true,
	}

	router := gin.New()
	router.Use(RateLimitWithConfig(cfg))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	// 用户1的请求
	for i := 0; i < 3; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		req.Header.Set("X-User-ID", "user1")
		router.ServeHTTP(w, req)

		if i < 2 && w.Code != http.StatusOK {
			t.Errorf("用户1 请求 %d 应该返回200", i)
		}
		if i == 2 && w.Code != http.StatusServiceUnavailable {
			t.Errorf("用户1 请求 %d 应该返回503，实际: %d", i, w.Code)
		}
	}

	// 用户2 应该有独立配额
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("X-User-ID", "user2")
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Error("用户2 应该返回200")
	}
}

func TestUserRateLimitMiddleware(t *testing.T) {
	getUserID := func(c *gin.Context) string {
		return c.GetHeader("Authorization")
	}

	router := gin.New()
	router.Use(UserRateLimit(100, 2, getUserID))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	// 有认证的用户
	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "token123")
		router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("认证用户请求 %d 应该返回200", i)
		}
	}

	// 第3个请求被限流
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "token123")
	router.ServeHTTP(w, req)
	if w.Code != http.StatusTooManyRequests {
		t.Error("认证用户第3个请求应该返回429")
	}
}
