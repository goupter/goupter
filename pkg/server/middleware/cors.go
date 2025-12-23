package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// CORSConfig CORS配置
type CORSConfig struct {
	AllowOrigins     []string // 允许的源
	AllowMethods     []string // 允许的方法
	AllowHeaders     []string // 允许的头
	ExposeHeaders    []string // 暴露的头
	AllowCredentials bool     // 是否允许凭证
	MaxAge           int      // 预检请求缓存时间（秒）
}

// DefaultCORSConfig 默认CORS配置
func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
			http.MethodHead,
			http.MethodOptions,
		},
		AllowHeaders: []string{
			"Origin",
			"Content-Type",
			"Accept",
			"Authorization",
			"X-Request-ID",
			"X-Trace-ID",
		},
		ExposeHeaders: []string{
			"Content-Length",
			"X-Request-ID",
			"X-Trace-ID",
		},
		AllowCredentials: true,
		MaxAge:           86400,
	}
}

// CORS 跨域中间件
func CORS() gin.HandlerFunc {
	return CORSWithConfig(DefaultCORSConfig())
}

// CORSWithConfig 带配置的跨域中间件
func CORSWithConfig(cfg CORSConfig) gin.HandlerFunc {
	// 处理配置
	allowAllOrigins := false
	for _, origin := range cfg.AllowOrigins {
		if origin == "*" {
			allowAllOrigins = true
			break
		}
	}

	allowOrigins := make(map[string]bool, len(cfg.AllowOrigins))
	for _, origin := range cfg.AllowOrigins {
		allowOrigins[origin] = true
	}

	allowMethods := strings.Join(cfg.AllowMethods, ", ")
	allowHeaders := strings.Join(cfg.AllowHeaders, ", ")
	exposeHeaders := strings.Join(cfg.ExposeHeaders, ", ")

	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")

		// 检查是否允许该源
		var allowOrigin string
		if allowAllOrigins {
			if cfg.AllowCredentials {
				allowOrigin = origin
			} else {
				allowOrigin = "*"
			}
		} else {
			if allowOrigins[origin] {
				allowOrigin = origin
			}
		}

		// 设置响应头
		if allowOrigin != "" {
			c.Header("Access-Control-Allow-Origin", allowOrigin)
		}

		if cfg.AllowCredentials {
			c.Header("Access-Control-Allow-Credentials", "true")
		}

		if exposeHeaders != "" {
			c.Header("Access-Control-Expose-Headers", exposeHeaders)
		}

		// 处理预检请求
		if c.Request.Method == http.MethodOptions {
			c.Header("Access-Control-Allow-Methods", allowMethods)
			c.Header("Access-Control-Allow-Headers", allowHeaders)
			if cfg.MaxAge > 0 {
				c.Header("Access-Control-Max-Age", string(rune(cfg.MaxAge)))
			}
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// CORSAllowAll 允许所有跨域请求
func CORSAllowAll() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		if origin == "" {
			origin = "*"
		}

		c.Header("Access-Control-Allow-Origin", origin)
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "*")
		c.Header("Access-Control-Expose-Headers", "*")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
