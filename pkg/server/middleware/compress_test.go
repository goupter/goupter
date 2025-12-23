package middleware

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestGzip_NoAcceptEncoding(t *testing.T) {
	router := gin.New()
	router.Use(Gzip(gzip.DefaultCompression))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "Hello World")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望200，实际: %d", w.Code)
	}
	if w.Header().Get("Content-Encoding") == "gzip" {
		t.Error("没有Accept-Encoding不应该压缩")
	}
}

func TestGzip_WithAcceptEncoding(t *testing.T) {
	router := gin.New()
	cfg := DefaultGzipConfig()
	cfg.MinLength = 0
	router.Use(GzipWithConfig(cfg))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, strings.Repeat("Hello World ", 100))
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望200，实际: %d", w.Code)
	}
}

func TestGzip_ExcludedPath(t *testing.T) {
	cfg := DefaultGzipConfig()
	cfg.ExcludedPaths = []string{"/excluded"}
	cfg.MinLength = 0

	router := gin.New()
	router.Use(GzipWithConfig(cfg))
	router.GET("/excluded", func(c *gin.Context) {
		c.String(http.StatusOK, strings.Repeat("data", 100))
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/excluded", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	router.ServeHTTP(w, req)

	if w.Header().Get("Content-Encoding") == "gzip" {
		t.Error("排除的路径不应该压缩")
	}
}

func TestGzip_BelowMinLength(t *testing.T) {
	cfg := DefaultGzipConfig()
	cfg.MinLength = 10000

	router := gin.New()
	router.Use(GzipWithConfig(cfg))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "Short content")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	router.ServeHTTP(w, req)

	if w.Header().Get("Content-Encoding") == "gzip" {
		t.Error("短内容不应该压缩")
	}
}

func TestDecompress_Gzip(t *testing.T) {
	router := gin.New()
	router.Use(Decompress())
	router.POST("/test", func(c *gin.Context) {
		body, _ := io.ReadAll(c.Request.Body)
		c.String(http.StatusOK, string(body))
	})

	var buf bytes.Buffer
	gzWriter := gzip.NewWriter(&buf)
	gzWriter.Write([]byte("Hello Gzip"))
	gzWriter.Close()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/test", &buf)
	req.Header.Set("Content-Encoding", "gzip")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望200，实际: %d", w.Code)
	}
	if w.Body.String() != "Hello Gzip" {
		t.Errorf("解压内容不正确: %s", w.Body.String())
	}
}

func TestDecompress_NoEncoding(t *testing.T) {
	router := gin.New()
	router.Use(Decompress())
	router.POST("/test", func(c *gin.Context) {
		body, _ := io.ReadAll(c.Request.Body)
		c.String(http.StatusOK, string(body))
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/test", strings.NewReader("Plain text"))
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望200，实际: %d", w.Code)
	}
	if w.Body.String() != "Plain text" {
		t.Errorf("内容不正确: %s", w.Body.String())
	}
}

func TestDecompress_InvalidGzip(t *testing.T) {
	router := gin.New()
	router.Use(Decompress())
	router.POST("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/test", strings.NewReader("not gzip data"))
	req.Header.Set("Content-Encoding", "gzip")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("无效gzip应该返回400，实际: %d", w.Code)
	}
}

func TestDefaultGzipConfig(t *testing.T) {
	cfg := DefaultGzipConfig()

	if cfg.Level != gzip.DefaultCompression {
		t.Errorf("默认压缩级别不正确: %d", cfg.Level)
	}
	if cfg.MinLength != 1024 {
		t.Errorf("默认最小长度不正确: %d", cfg.MinLength)
	}
	if len(cfg.IncludedMimes) == 0 {
		t.Error("应该有默认MIME类型")
	}
}
