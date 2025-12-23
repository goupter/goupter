package middleware

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// SecurityConfig 安全头配置
type SecurityConfig struct {
	ContentTypeNosniff        bool
	FrameOptions              string // DENY, SAMEORIGIN
	XSSProtection             string
	HSTSMaxAge                int
	HSTSIncludeSubdomains     bool
	HSTSPreload               bool
	ContentSecurityPolicy     string
	ReferrerPolicy            string
	CrossOriginOpenerPolicy   string
	CrossOriginResourcePolicy string
	CustomHeaders             map[string]string
}

// DefaultSecurityConfig 默认安全头配置
func DefaultSecurityConfig() SecurityConfig {
	return SecurityConfig{
		ContentTypeNosniff:        true,
		FrameOptions:              "DENY",
		XSSProtection:             "1; mode=block",
		HSTSMaxAge:                31536000,
		HSTSIncludeSubdomains:     true,
		ReferrerPolicy:            "strict-origin-when-cross-origin",
		CrossOriginOpenerPolicy:   "same-origin",
		CrossOriginResourcePolicy: "same-origin",
	}
}

// Security 安全头中间件
func Security() gin.HandlerFunc {
	return SecurityWithConfig(DefaultSecurityConfig())
}

// SecurityWithConfig 带配置的安全头中间件
func SecurityWithConfig(cfg SecurityConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		if cfg.ContentTypeNosniff {
			c.Header("X-Content-Type-Options", "nosniff")
		}
		if cfg.FrameOptions != "" {
			c.Header("X-Frame-Options", cfg.FrameOptions)
		}
		if cfg.XSSProtection != "" {
			c.Header("X-XSS-Protection", cfg.XSSProtection)
		}
		if cfg.HSTSMaxAge > 0 {
			hsts := "max-age=" + strconv.Itoa(cfg.HSTSMaxAge)
			if cfg.HSTSIncludeSubdomains {
				hsts += "; includeSubDomains"
			}
			if cfg.HSTSPreload {
				hsts += "; preload"
			}
			c.Header("Strict-Transport-Security", hsts)
		}
		if cfg.ContentSecurityPolicy != "" {
			c.Header("Content-Security-Policy", cfg.ContentSecurityPolicy)
		}
		if cfg.ReferrerPolicy != "" {
			c.Header("Referrer-Policy", cfg.ReferrerPolicy)
		}
		if cfg.CrossOriginOpenerPolicy != "" {
			c.Header("Cross-Origin-Opener-Policy", cfg.CrossOriginOpenerPolicy)
		}
		if cfg.CrossOriginResourcePolicy != "" {
			c.Header("Cross-Origin-Resource-Policy", cfg.CrossOriginResourcePolicy)
		}
		for key, value := range cfg.CustomHeaders {
			c.Header(key, value)
		}
		c.Next()
	}
}

// IPWhitelist IP白名单中间件
func IPWhitelist(allowedIPs ...string) gin.HandlerFunc {
	allowed := make(map[string]bool)
	for _, ip := range allowedIPs {
		allowed[ip] = true
	}

	return func(c *gin.Context) {
		if !allowed[c.ClientIP()] {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code":    http.StatusForbidden,
				"message": "IP not allowed",
			})
			return
		}
		c.Next()
	}
}

// IPBlacklist IP黑名单中间件
func IPBlacklist(blockedIPs ...string) gin.HandlerFunc {
	blocked := make(map[string]bool)
	for _, ip := range blockedIPs {
		blocked[ip] = true
	}

	return func(c *gin.Context) {
		if blocked[c.ClientIP()] {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code":    http.StatusForbidden,
				"message": "IP blocked",
			})
			return
		}
		c.Next()
	}
}
