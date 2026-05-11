package middleware

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

var errTimeoutWriterHijackUnsupported = errors.New("timeout writer does not support hijack")

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
		executeWithTimeout(c, cfg.Timeout, func(writer gin.ResponseWriter, _ *gin.Context) {
			writeJSONResponse(writer, cfg.StatusCode, gin.H{
				"code":    cfg.StatusCode,
				"message": cfg.Message,
			})
		})
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

		if timeoutHeader := c.GetHeader("X-Request-Timeout"); timeoutHeader != "" {
			if d, err := time.ParseDuration(timeoutHeader); err == nil {
				timeout = d
			}
		}

		if maxTimeout > 0 && timeout > maxTimeout {
			timeout = maxTimeout
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()

		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

type bufferedResponseWriter struct {
	original gin.ResponseWriter
	header   http.Header
	body     bytes.Buffer
	status   int
	size     int
	timedOut bool
	mu       sync.Mutex
}

func newBufferedResponseWriter(original gin.ResponseWriter) *bufferedResponseWriter {
	return &bufferedResponseWriter{
		original: original,
		header:   make(http.Header),
		status:   http.StatusOK,
		size:     -1,
	}
}

func (w *bufferedResponseWriter) Header() http.Header {
	return w.header
}

func (w *bufferedResponseWriter) WriteHeader(code int) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if code > 0 && w.status != code {
		if w.size != -1 {
			return
		}
		w.status = code
	}
}

func (w *bufferedResponseWriter) WriteHeaderNow() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.size == -1 {
		w.size = 0
	}
}

func (w *bufferedResponseWriter) Write(data []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.timedOut {
		return 0, context.DeadlineExceeded
	}
	if w.size == -1 {
		w.size = 0
	}

	n, err := w.body.Write(data)
	w.size += n
	return n, err
}

func (w *bufferedResponseWriter) WriteString(s string) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.timedOut {
		return 0, context.DeadlineExceeded
	}
	if w.size == -1 {
		w.size = 0
	}

	n, err := w.body.WriteString(s)
	w.size += n
	return n, err
}

func (w *bufferedResponseWriter) Status() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.status
}

func (w *bufferedResponseWriter) Size() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.size
}

func (w *bufferedResponseWriter) Written() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.size != -1
}

func (w *bufferedResponseWriter) Flush() {
	w.WriteHeaderNow()
}

func (w *bufferedResponseWriter) CloseNotify() <-chan bool {
	return w.original.CloseNotify()
}

func (w *bufferedResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return nil, nil, errTimeoutWriterHijackUnsupported
}

func (w *bufferedResponseWriter) Pusher() http.Pusher {
	return w.original.Pusher()
}

func (w *bufferedResponseWriter) markTimedOut() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.timedOut = true
}

func (w *bufferedResponseWriter) commitToOriginal() error {
	w.mu.Lock()
	if w.timedOut {
		w.mu.Unlock()
		return context.DeadlineExceeded
	}

	header := cloneHeader(w.header)
	status := w.status
	size := w.size
	body := append([]byte(nil), w.body.Bytes()...)
	w.mu.Unlock()

	dstHeader := w.original.Header()
	for key, values := range header {
		dstHeader[key] = append([]string(nil), values...)
	}

	if size != -1 || status != http.StatusOK || len(header) > 0 {
		w.original.WriteHeader(status)
	}
	if len(body) == 0 {
		return nil
	}

	_, err := w.original.Write(body)
	return err
}

func executeWithTimeout(c *gin.Context, timeout time.Duration, onTimeout func(gin.ResponseWriter, *gin.Context)) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
	defer cancel()

	originalWriter := c.Writer
	bufferedWriter := newBufferedResponseWriter(originalWriter)
	workerContext := cloneContext(c, bufferedWriter, c.Request.WithContext(ctx))

	done := make(chan struct{})
	panicChan := make(chan any, 1)

	go func() {
		defer close(done)
		defer func() {
			if p := recover(); p != nil {
				panicChan <- p
			}
		}()
		workerContext.Next()
	}()

	select {
	case p := <-panicChan:
		panic(p)
	case <-done:
		select {
		case p := <-panicChan:
			panic(p)
		default:
		}
		_ = bufferedWriter.commitToOriginal()
		return
	case <-ctx.Done():
		bufferedWriter.markTimedOut()
		onTimeout(originalWriter, cloneContext(c, originalWriter, c.Request.WithContext(ctx)))
		return
	}
}

func cloneContext(c *gin.Context, writer gin.ResponseWriter, request *http.Request) *gin.Context {
	cloned := *c
	cloned.Writer = writer
	cloned.Request = request
	cloned.Params = append(gin.Params(nil), c.Params...)
	cloned.Keys = cloneKeys(c.Keys)
	cloned.Accepted = append([]string(nil), c.Accepted...)
	return &cloned
}

func writeJSONResponse(writer gin.ResponseWriter, statusCode int, payload gin.H) {
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	writer.WriteHeader(statusCode)
	_ = json.NewEncoder(writer).Encode(payload)
}

func cloneHeader(header http.Header) http.Header {
	cloned := make(http.Header, len(header))
	for key, values := range header {
		cloned[key] = append([]string(nil), values...)
	}
	return cloned
}

func cloneKeys(keys map[any]any) map[any]any {
	if len(keys) == 0 {
		return nil
	}

	cloned := make(map[any]any, len(keys))
	for key, value := range keys {
		cloned[key] = value
	}
	return cloned
}

// TimeoutWithCustomResponse 支持自定义超时响应的中间件
func TimeoutWithCustomResponse(timeout time.Duration, timeoutHandler gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		executeWithTimeout(c, timeout, func(writer gin.ResponseWriter, timeoutContext *gin.Context) {
			if timeoutHandler != nil {
				timeoutHandler(timeoutContext)
				return
			}

			writeJSONResponse(writer, http.StatusGatewayTimeout, gin.H{
				"code":    http.StatusGatewayTimeout,
				"message": "Request timeout",
			})
		})
	}
}
