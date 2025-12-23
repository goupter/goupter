package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSecurity_DefaultHeaders(t *testing.T) {
	router := gin.New()
	router.Use(Security())
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	if w.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Error("应该设置X-Content-Type-Options")
	}
	if w.Header().Get("X-Frame-Options") != "DENY" {
		t.Error("应该设置X-Frame-Options")
	}
	if w.Header().Get("X-XSS-Protection") != "1; mode=block" {
		t.Error("应该设置X-XSS-Protection")
	}
	hsts := w.Header().Get("Strict-Transport-Security")
	if !strings.Contains(hsts, "max-age=") {
		t.Error("应该设置HSTS")
	}
	if w.Header().Get("Referrer-Policy") == "" {
		t.Error("应该设置Referrer-Policy")
	}
}

func TestSecurityWithConfig_CustomHeaders(t *testing.T) {
	cfg := SecurityConfig{
		ContentTypeNosniff: true,
		FrameOptions:       "SAMEORIGIN",
		HSTSMaxAge:         86400,
		CustomHeaders: map[string]string{
			"X-Custom-Header": "custom-value",
		},
	}

	router := gin.New()
	router.Use(SecurityWithConfig(cfg))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	if w.Header().Get("X-Frame-Options") != "SAMEORIGIN" {
		t.Errorf("X-Frame-Options应该是SAMEORIGIN: %s", w.Header().Get("X-Frame-Options"))
	}
	if w.Header().Get("X-Custom-Header") != "custom-value" {
		t.Error("应该设置自定义头")
	}
}

func TestIPWhitelist_Allowed(t *testing.T) {
	router := gin.New()
	router.Use(IPWhitelist("192.168.1.1", "10.0.0.1"))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("白名单IP应该允许访问，实际: %d", w.Code)
	}
}

func TestIPWhitelist_Denied(t *testing.T) {
	router := gin.New()
	router.Use(IPWhitelist("192.168.1.1"))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.2:12345"
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("非白名单IP应该拒绝，实际: %d", w.Code)
	}
}

func TestIPBlacklist_Blocked(t *testing.T) {
	router := gin.New()
	router.Use(IPBlacklist("192.168.1.1"))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("黑名单IP应该拒绝，实际: %d", w.Code)
	}
}

func TestIPBlacklist_Allowed(t *testing.T) {
	router := gin.New()
	router.Use(IPBlacklist("192.168.1.1"))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.2:12345"
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("非黑名单IP应该允许，实际: %d", w.Code)
	}
}

func TestDefaultSecurityConfig(t *testing.T) {
	cfg := DefaultSecurityConfig()

	if !cfg.ContentTypeNosniff {
		t.Error("默认应该启用ContentTypeNosniff")
	}
	if cfg.FrameOptions != "DENY" {
		t.Errorf("默认FrameOptions应该是DENY: %s", cfg.FrameOptions)
	}
	if cfg.HSTSMaxAge != 31536000 {
		t.Errorf("默认HSTS应该是1年: %d", cfg.HSTSMaxAge)
	}
}
