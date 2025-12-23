package middleware

import (
	"compress/flate"
	"compress/gzip"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

// GzipConfig Gzip压缩配置
type GzipConfig struct {
	Level         int
	MinLength     int
	ExcludedPaths []string
	IncludedMimes []string
}

// DefaultGzipConfig 默认Gzip配置
func DefaultGzipConfig() GzipConfig {
	return GzipConfig{
		Level:     gzip.DefaultCompression,
		MinLength: 1024,
		IncludedMimes: []string{
			"text/html", "text/css", "text/plain", "text/javascript",
			"application/javascript", "application/json", "application/xml",
		},
	}
}

// Gzip Gzip压缩中间件
func Gzip(level int) gin.HandlerFunc {
	cfg := DefaultGzipConfig()
	cfg.Level = level
	return GzipWithConfig(cfg)
}

// GzipWithConfig 带配置的Gzip压缩中间件
func GzipWithConfig(cfg GzipConfig) gin.HandlerFunc {
	excludedPaths := make(map[string]bool)
	for _, p := range cfg.ExcludedPaths {
		excludedPaths[p] = true
	}

	includedMimes := make(map[string]bool)
	for _, m := range cfg.IncludedMimes {
		includedMimes[m] = true
	}

	pool := sync.Pool{
		New: func() interface{} {
			w, _ := gzip.NewWriterLevel(io.Discard, cfg.Level)
			return w
		},
	}

	return func(c *gin.Context) {
		if excludedPaths[c.Request.URL.Path] {
			c.Next()
			return
		}

		if !strings.Contains(c.GetHeader("Accept-Encoding"), "gzip") {
			c.Next()
			return
		}

		gzWriter := pool.Get().(*gzip.Writer)
		gzWriter.Reset(c.Writer)

		cw := &gzipWriter{
			ResponseWriter: c.Writer,
			writer:         gzWriter,
			minLength:      cfg.MinLength,
			includedMimes:  includedMimes,
			pool:           &pool,
		}

		c.Writer = cw
		defer cw.Close()

		c.Next()
	}
}

type gzipWriter struct {
	gin.ResponseWriter
	writer        *gzip.Writer
	minLength     int
	includedMimes map[string]bool
	pool          *sync.Pool
	started       bool
	buf           []byte
}

func (w *gzipWriter) Write(data []byte) (int, error) {
	if !w.started {
		w.buf = append(w.buf, data...)
		if len(w.buf) >= w.minLength {
			w.start()
			return w.writer.Write(w.buf)
		}
		return len(data), nil
	}
	return w.writer.Write(data)
}

func (w *gzipWriter) start() {
	if w.started {
		return
	}
	w.started = true

	contentType := w.Header().Get("Content-Type")
	if idx := strings.Index(contentType, ";"); idx > 0 {
		contentType = contentType[:idx]
	}
	contentType = strings.TrimSpace(contentType)

	if len(w.includedMimes) > 0 && !w.includedMimes[contentType] {
		return
	}

	w.Header().Set("Content-Encoding", "gzip")
	w.Header().Set("Vary", "Accept-Encoding")
	w.Header().Del("Content-Length")
}

func (w *gzipWriter) WriteString(s string) (int, error) {
	return w.Write([]byte(s))
}

func (w *gzipWriter) Close() error {
	if !w.started && len(w.buf) > 0 {
		_, err := w.ResponseWriter.Write(w.buf)
		return err
	}

	if w.writer != nil {
		w.writer.Close()
		w.pool.Put(w.writer)
	}
	return nil
}

func (w *gzipWriter) Flush() {
	if w.started {
		w.writer.Flush()
	}
	w.ResponseWriter.Flush()
}

// Decompress 请求解压中间件
func Decompress() gin.HandlerFunc {
	return func(c *gin.Context) {
		encoding := c.GetHeader("Content-Encoding")
		if encoding == "" {
			c.Next()
			return
		}

		var reader io.ReadCloser
		var err error

		switch strings.ToLower(encoding) {
		case "gzip":
			reader, err = gzip.NewReader(c.Request.Body)
		case "deflate":
			reader = flate.NewReader(c.Request.Body)
		default:
			c.Next()
			return
		}

		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"code":    http.StatusBadRequest,
				"message": "invalid compressed request body",
			})
			return
		}

		c.Request.Body = reader
		c.Request.Header.Del("Content-Encoding")
		c.Request.Header.Del("Content-Length")

		c.Next()
	}
}
