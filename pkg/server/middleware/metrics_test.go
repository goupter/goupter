package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestRequestMetrics_Record(t *testing.T) {
	metrics := NewRequestMetrics()
	metrics.Record(200, "GET", 100*time.Millisecond)

	snapshot := metrics.GetSnapshot()
	if snapshot.TotalRequests != 1 {
		t.Errorf("TotalRequests应该是1，实际: %d", snapshot.TotalRequests)
	}
	if snapshot.SuccessRequests != 1 {
		t.Errorf("SuccessRequests应该是1，实际: %d", snapshot.SuccessRequests)
	}
	if snapshot.FailedRequests != 0 {
		t.Errorf("FailedRequests应该是0，实际: %d", snapshot.FailedRequests)
	}
}

func TestRequestMetrics_RecordError(t *testing.T) {
	metrics := NewRequestMetrics()
	metrics.Record(500, "POST", 50*time.Millisecond)

	snapshot := metrics.GetSnapshot()
	if snapshot.TotalRequests != 1 {
		t.Errorf("TotalRequests应该是1，实际: %d", snapshot.TotalRequests)
	}
	if snapshot.SuccessRequests != 0 {
		t.Errorf("SuccessRequests应该是0，实际: %d", snapshot.SuccessRequests)
	}
	if snapshot.FailedRequests != 1 {
		t.Errorf("FailedRequests应该是1，实际: %d", snapshot.FailedRequests)
	}
}

func TestRequestMetrics_InFlight(t *testing.T) {
	metrics := NewRequestMetrics()

	metrics.IncrInFlight()
	metrics.IncrInFlight()

	snapshot := metrics.GetSnapshot()
	if snapshot.InFlightRequests != 2 {
		t.Errorf("InFlightRequests应该是2，实际: %d", snapshot.InFlightRequests)
	}

	metrics.DecrInFlight()

	snapshot = metrics.GetSnapshot()
	if snapshot.InFlightRequests != 1 {
		t.Errorf("InFlightRequests应该是1，实际: %d", snapshot.InFlightRequests)
	}
}

func TestRequestMetrics_StatusCodes(t *testing.T) {
	metrics := NewRequestMetrics()

	metrics.Record(200, "GET", 10*time.Millisecond)
	metrics.Record(200, "GET", 10*time.Millisecond)
	metrics.Record(404, "GET", 10*time.Millisecond)
	metrics.Record(500, "GET", 10*time.Millisecond)

	snapshot := metrics.GetSnapshot()
	if snapshot.StatusCodeCounts[200] != 2 {
		t.Errorf("200状态码应该是2，实际: %d", snapshot.StatusCodeCounts[200])
	}
	if snapshot.StatusCodeCounts[404] != 1 {
		t.Errorf("404状态码应该是1，实际: %d", snapshot.StatusCodeCounts[404])
	}
	if snapshot.StatusCodeCounts[500] != 1 {
		t.Errorf("500状态码应该是1，实际: %d", snapshot.StatusCodeCounts[500])
	}
}

func TestRequestMetrics_Methods(t *testing.T) {
	metrics := NewRequestMetrics()

	metrics.Record(200, "GET", 10*time.Millisecond)
	metrics.Record(200, "POST", 10*time.Millisecond)
	metrics.Record(200, "GET", 10*time.Millisecond)

	snapshot := metrics.GetSnapshot()
	if snapshot.MethodCounts["GET"] != 2 {
		t.Errorf("GET方法应该是2，实际: %d", snapshot.MethodCounts["GET"])
	}
	if snapshot.MethodCounts["POST"] != 1 {
		t.Errorf("POST方法应该是1，实际: %d", snapshot.MethodCounts["POST"])
	}
}

func TestRequestMetrics_Latency(t *testing.T) {
	metrics := NewRequestMetrics()

	metrics.Record(200, "GET", 100*time.Millisecond)
	metrics.Record(200, "GET", 200*time.Millisecond)
	metrics.Record(200, "GET", 300*time.Millisecond)

	snapshot := metrics.GetSnapshot()
	if snapshot.MinLatency != 100*time.Millisecond {
		t.Errorf("MinLatency应该是100ms，实际: %v", snapshot.MinLatency)
	}
	if snapshot.MaxLatency != 300*time.Millisecond {
		t.Errorf("MaxLatency应该是300ms，实际: %v", snapshot.MaxLatency)
	}
	if snapshot.AvgLatency != 200*time.Millisecond {
		t.Errorf("AvgLatency应该是200ms，实际: %v", snapshot.AvgLatency)
	}
}

func TestRequestMetrics_Reset(t *testing.T) {
	metrics := NewRequestMetrics()
	metrics.Record(200, "GET", 100*time.Millisecond)

	snapshot := metrics.GetSnapshot()
	if snapshot.TotalRequests != 1 {
		t.Error("重置前应该有记录")
	}

	metrics.Reset()

	snapshot = metrics.GetSnapshot()
	if snapshot.TotalRequests != 0 {
		t.Errorf("重置后TotalRequests应该是0，实际: %d", snapshot.TotalRequests)
	}
}

func TestRequestMetricsSnapshot_SuccessRate(t *testing.T) {
	metrics := NewRequestMetrics()

	metrics.Record(200, "GET", 10*time.Millisecond)
	metrics.Record(200, "GET", 10*time.Millisecond)
	metrics.Record(500, "GET", 10*time.Millisecond)
	metrics.Record(500, "GET", 10*time.Millisecond)

	snapshot := metrics.GetSnapshot()
	if snapshot.SuccessRate() != 0.5 {
		t.Errorf("成功率应该是0.5，实际: %f", snapshot.SuccessRate())
	}
}

func TestRequestMetricsSnapshot_SuccessRateZero(t *testing.T) {
	metrics := NewRequestMetrics()

	snapshot := metrics.GetSnapshot()
	if snapshot.SuccessRate() != 0 {
		t.Errorf("没有请求时成功率应该是0，实际: %f", snapshot.SuccessRate())
	}
}

func TestGetLatencyBucket(t *testing.T) {
	tests := []struct {
		latency  time.Duration
		expected string
	}{
		{5 * time.Millisecond, "<10ms"},
		{30 * time.Millisecond, "10-50ms"},
		{75 * time.Millisecond, "50-100ms"},
		{350 * time.Millisecond, "100-500ms"},
		{750 * time.Millisecond, "500ms-1s"},
		{2 * time.Second, ">1s"},
	}

	for _, tt := range tests {
		bucket := getLatencyBucket(tt.latency)
		if bucket != tt.expected {
			t.Errorf("延迟 %v 应该在桶 %s，实际: %s", tt.latency, tt.expected, bucket)
		}
	}
}

func TestMetricsMiddleware(t *testing.T) {
	metrics := NewRequestMetrics()
	cfg := MetricsConfig{
		Metrics: metrics,
	}

	router := gin.New()
	router.Use(MetricsWithConfig(cfg))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	for i := 0; i < 5; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		router.ServeHTTP(w, req)
	}

	snapshot := metrics.GetSnapshot()
	if snapshot.TotalRequests != 5 {
		t.Errorf("应该有5个请求，实际: %d", snapshot.TotalRequests)
	}
	if snapshot.SuccessRequests != 5 {
		t.Errorf("应该有5个成功请求，实际: %d", snapshot.SuccessRequests)
	}
}

func TestMetricsMiddleware_SkipPaths(t *testing.T) {
	metrics := NewRequestMetrics()
	cfg := MetricsConfig{
		Metrics:   metrics,
		SkipPaths: []string{"/health"},
	}

	router := gin.New()
	router.Use(MetricsWithConfig(cfg))
	router.GET("/health", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})
	router.GET("/api", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/health", nil)
	router.ServeHTTP(w, req)

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/api", nil)
	router.ServeHTTP(w, req)

	snapshot := metrics.GetSnapshot()
	if snapshot.TotalRequests != 1 {
		t.Errorf("应该只记录1个请求（跳过/health），实际: %d", snapshot.TotalRequests)
	}
}

func TestGlobalMetrics(t *testing.T) {
	globalMetrics.Reset()

	router := gin.New()
	router.Use(Metrics())
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	metrics := GetGlobalMetrics()
	snapshot := metrics.GetSnapshot()
	if snapshot.TotalRequests < 1 {
		t.Errorf("全局指标应该记录请求，实际: %d", snapshot.TotalRequests)
	}
}

func TestPrometheusHandler(t *testing.T) {
	metrics := NewRequestMetrics()
	metrics.Record(200, "GET", 100*time.Millisecond)

	router := gin.New()
	router.GET("/metrics", PrometheusHandler(metrics, "http_server"))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/metrics", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望200，实际: %d", w.Code)
	}

	body := w.Body.String()
	if !strings.Contains(body, "http_server_requests_total") {
		t.Error("应该包含requests_total指标")
	}
}

func TestMetricsHandler(t *testing.T) {
	metrics := NewRequestMetrics()
	metrics.Record(200, "GET", 100*time.Millisecond)

	router := gin.New()
	router.GET("/metrics", MetricsHandler(metrics))

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/metrics", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望200，实际: %d", w.Code)
	}

	if w.Header().Get("Content-Type") != "application/json; charset=utf-8" {
		t.Error("Content-Type应该是application/json")
	}
}
