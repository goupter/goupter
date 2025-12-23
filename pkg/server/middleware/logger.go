package middleware

import (
	"bytes"
	"io"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/goupter/goupter/pkg/log"
	"github.com/google/uuid"
)

// responseWriter 自定义响应写入器
type responseWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *responseWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// Logger 日志中间件
func Logger(logger log.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 开始时间
		start := time.Now()

		// 生成请求ID
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)

		// 读取请求体
		var requestBody string
		if c.Request.Body != nil {
			bodyBytes, _ := io.ReadAll(c.Request.Body)
			requestBody = string(bodyBytes)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}

		// 包装响应写入器
		w := &responseWriter{
			ResponseWriter: c.Writer,
			body:           bytes.NewBuffer(nil),
		}
		c.Writer = w

		// 处理请求
		c.Next()

		// 结束时间
		latency := time.Since(start)

		// 日志字段
		fields := []log.Field{
			log.String("request_id", requestID),
			log.String("method", c.Request.Method),
			log.String("path", c.Request.URL.Path),
			log.String("query", c.Request.URL.RawQuery),
			log.String("ip", c.ClientIP()),
			log.String("user_agent", c.Request.UserAgent()),
			log.Int("status", c.Writer.Status()),
			log.Int("size", c.Writer.Size()),
			log.String("latency", latency.String()),
		}

		// 仅在debug模式记录请求和响应体
		cfg := log.Default().GetLevel()
		if cfg == log.DebugLevel {
			if len(requestBody) > 0 && len(requestBody) < 10000 {
				fields = append(fields, log.String("request_body", requestBody))
			}
			responseBody := w.body.String()
			if len(responseBody) > 0 && len(responseBody) < 10000 {
				fields = append(fields, log.String("response_body", responseBody))
			}
		}

		// 记录错误
		if len(c.Errors) > 0 {
			fields = append(fields, log.String("errors", c.Errors.String()))
		}

		// 根据状态码选择日志级别
		status := c.Writer.Status()
		switch {
		case status >= 500:
			logger.Error("HTTP request", fields...)
		case status >= 400:
			logger.Warn("HTTP request", fields...)
		default:
			logger.Info("HTTP request", fields...)
		}
	}
}

// LoggerWithConfig 带配置的日志中间件
type LoggerConfig struct {
	Logger          log.Logger
	SkipPaths       []string // 跳过日志的路径
	SkipPathsRegexp []string // 跳过日志的路径正则
	LogRequestBody  bool     // 是否记录请求体
	LogResponseBody bool     // 是否记录响应体
	MaxBodySize     int      // 最大记录体大小
}

// LoggerWithConfig 带配置的日志中间件
func LoggerWithConfig(cfg LoggerConfig) gin.HandlerFunc {
	if cfg.Logger == nil {
		cfg.Logger = log.Default()
	}
	if cfg.MaxBodySize == 0 {
		cfg.MaxBodySize = 10000
	}

	skipPaths := make(map[string]bool, len(cfg.SkipPaths))
	for _, path := range cfg.SkipPaths {
		skipPaths[path] = true
	}

	return func(c *gin.Context) {
		// 检查是否跳过
		if skipPaths[c.Request.URL.Path] {
			c.Next()
			return
		}

		// 开始时间
		start := time.Now()

		// 生成请求ID
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)

		// 读取请求体
		var requestBody string
		if cfg.LogRequestBody && c.Request.Body != nil {
			bodyBytes, _ := io.ReadAll(c.Request.Body)
			if len(bodyBytes) <= cfg.MaxBodySize {
				requestBody = string(bodyBytes)
			}
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}

		// 包装响应写入器
		var w *responseWriter
		if cfg.LogResponseBody {
			w = &responseWriter{
				ResponseWriter: c.Writer,
				body:           bytes.NewBuffer(nil),
			}
			c.Writer = w
		}

		// 处理请求
		c.Next()

		// 结束时间
		latency := time.Since(start)

		// 日志字段
		fields := []log.Field{
			log.String("request_id", requestID),
			log.String("method", c.Request.Method),
			log.String("path", c.Request.URL.Path),
			log.String("ip", c.ClientIP()),
			log.Int("status", c.Writer.Status()),
			log.String("latency", latency.String()),
		}

		if cfg.LogRequestBody && len(requestBody) > 0 {
			fields = append(fields, log.String("request_body", requestBody))
		}

		if cfg.LogResponseBody && w != nil {
			responseBody := w.body.String()
			if len(responseBody) <= cfg.MaxBodySize {
				fields = append(fields, log.String("response_body", responseBody))
			}
		}

		// 记录错误
		if len(c.Errors) > 0 {
			fields = append(fields, log.String("errors", c.Errors.String()))
		}

		// 根据状态码选择日志级别
		status := c.Writer.Status()
		switch {
		case status >= 500:
			cfg.Logger.Error("HTTP request", fields...)
		case status >= 400:
			cfg.Logger.Warn("HTTP request", fields...)
		default:
			cfg.Logger.Info("HTTP request", fields...)
		}
	}
}
