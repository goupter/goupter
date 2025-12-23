package middleware

import (
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
)

// RequestMetrics 请求指标
type RequestMetrics struct {
	mu               sync.RWMutex
	TotalRequests    int64
	SuccessRequests  int64
	FailedRequests   int64
	InFlightRequests int64
	TotalLatency     time.Duration
	MinLatency       time.Duration
	MaxLatency       time.Duration
	StatusCodeCounts map[int]int64
	MethodCounts     map[string]int64
	LatencyBuckets   map[string]int64
	StartTime        time.Time
	LastTime         time.Time
}

// NewRequestMetrics 创建请求指标
func NewRequestMetrics() *RequestMetrics {
	return &RequestMetrics{
		StatusCodeCounts: make(map[int]int64),
		MethodCounts:     make(map[string]int64),
		LatencyBuckets:   make(map[string]int64),
		StartTime:        time.Now(),
		MinLatency:       time.Duration(1<<63 - 1),
	}
}

// Record 记录请求
func (m *RequestMetrics) Record(statusCode int, method string, latency time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.TotalRequests++
	if statusCode >= 200 && statusCode < 400 {
		m.SuccessRequests++
	} else {
		m.FailedRequests++
	}

	m.StatusCodeCounts[statusCode]++
	m.MethodCounts[method]++
	m.TotalLatency += latency

	if latency < m.MinLatency {
		m.MinLatency = latency
	}
	if latency > m.MaxLatency {
		m.MaxLatency = latency
	}

	m.LatencyBuckets[getLatencyBucket(latency)]++
	m.LastTime = time.Now()
}

// IncrInFlight 增加正在处理的请求数
func (m *RequestMetrics) IncrInFlight() { atomic.AddInt64(&m.InFlightRequests, 1) }

// DecrInFlight 减少正在处理的请求数
func (m *RequestMetrics) DecrInFlight() { atomic.AddInt64(&m.InFlightRequests, -1) }

// GetSnapshot 获取指标快照
func (m *RequestMetrics) GetSnapshot() MetricsSnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()

	snapshot := MetricsSnapshot{
		TotalRequests:    m.TotalRequests,
		SuccessRequests:  m.SuccessRequests,
		FailedRequests:   m.FailedRequests,
		InFlightRequests: atomic.LoadInt64(&m.InFlightRequests),
		StartTime:        m.StartTime,
		LastTime:         m.LastTime,
		StatusCodeCounts: make(map[int]int64),
		MethodCounts:     make(map[string]int64),
		LatencyBuckets:   make(map[string]int64),
	}

	if m.TotalRequests > 0 {
		snapshot.AvgLatency = m.TotalLatency / time.Duration(m.TotalRequests)
		snapshot.MinLatency = m.MinLatency
		snapshot.MaxLatency = m.MaxLatency
	}

	for k, v := range m.StatusCodeCounts {
		snapshot.StatusCodeCounts[k] = v
	}
	for k, v := range m.MethodCounts {
		snapshot.MethodCounts[k] = v
	}
	for k, v := range m.LatencyBuckets {
		snapshot.LatencyBuckets[k] = v
	}

	return snapshot
}

// Reset 重置指标
func (m *RequestMetrics) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.TotalRequests = 0
	m.SuccessRequests = 0
	m.FailedRequests = 0
	m.StatusCodeCounts = make(map[int]int64)
	m.MethodCounts = make(map[string]int64)
	m.TotalLatency = 0
	m.MinLatency = time.Duration(1<<63 - 1)
	m.MaxLatency = 0
	m.LatencyBuckets = make(map[string]int64)
	m.StartTime = time.Now()
}

// MetricsSnapshot 指标快照
type MetricsSnapshot struct {
	TotalRequests    int64
	SuccessRequests  int64
	FailedRequests   int64
	InFlightRequests int64
	AvgLatency       time.Duration
	MinLatency       time.Duration
	MaxLatency       time.Duration
	StartTime        time.Time
	LastTime         time.Time
	StatusCodeCounts map[int]int64
	MethodCounts     map[string]int64
	LatencyBuckets   map[string]int64
}

// SuccessRate 成功率
func (s MetricsSnapshot) SuccessRate() float64 {
	if s.TotalRequests == 0 {
		return 0
	}
	return float64(s.SuccessRequests) / float64(s.TotalRequests)
}

// QPS 每秒请求数
func (s MetricsSnapshot) QPS() float64 {
	duration := s.LastTime.Sub(s.StartTime).Seconds()
	if duration == 0 {
		return 0
	}
	return float64(s.TotalRequests) / duration
}

func getLatencyBucket(latency time.Duration) string {
	ms := latency.Milliseconds()
	switch {
	case ms < 10:
		return "<10ms"
	case ms < 50:
		return "10-50ms"
	case ms < 100:
		return "50-100ms"
	case ms < 500:
		return "100-500ms"
	case ms < 1000:
		return "500ms-1s"
	default:
		return ">1s"
	}
}

// MetricsConfig 指标配置
type MetricsConfig struct {
	Metrics   *RequestMetrics
	SkipPaths []string
}

var globalMetrics = NewRequestMetrics()

// GetGlobalMetrics 获取全局指标
func GetGlobalMetrics() *RequestMetrics { return globalMetrics }

// Metrics 指标中间件
func Metrics() gin.HandlerFunc {
	return MetricsWithConfig(MetricsConfig{Metrics: globalMetrics})
}

// MetricsWithConfig 带配置的指标中间件
func MetricsWithConfig(cfg MetricsConfig) gin.HandlerFunc {
	if cfg.Metrics == nil {
		cfg.Metrics = globalMetrics
	}

	skipPaths := make(map[string]bool)
	for _, p := range cfg.SkipPaths {
		skipPaths[p] = true
	}

	return func(c *gin.Context) {
		if skipPaths[c.Request.URL.Path] {
			c.Next()
			return
		}

		cfg.Metrics.IncrInFlight()
		defer cfg.Metrics.DecrInFlight()

		start := time.Now()
		c.Next()

		cfg.Metrics.Record(c.Writer.Status(), c.Request.Method, time.Since(start))
	}
}

// MetricsHandler 指标输出处理器
func MetricsHandler(metrics *RequestMetrics) gin.HandlerFunc {
	if metrics == nil {
		metrics = globalMetrics
	}

	return func(c *gin.Context) {
		snapshot := metrics.GetSnapshot()
		c.JSON(200, gin.H{
			"total_requests":     snapshot.TotalRequests,
			"success_requests":   snapshot.SuccessRequests,
			"failed_requests":    snapshot.FailedRequests,
			"in_flight_requests": snapshot.InFlightRequests,
			"success_rate":       snapshot.SuccessRate(),
			"qps":                snapshot.QPS(),
			"avg_latency_ms":     snapshot.AvgLatency.Milliseconds(),
			"min_latency_ms":     snapshot.MinLatency.Milliseconds(),
			"max_latency_ms":     snapshot.MaxLatency.Milliseconds(),
			"status_codes":       snapshot.StatusCodeCounts,
			"methods":            snapshot.MethodCounts,
			"latency_buckets":    snapshot.LatencyBuckets,
		})
	}
}

// PrometheusHandler Prometheus格式输出
func PrometheusHandler(metrics *RequestMetrics, namespace string) gin.HandlerFunc {
	if metrics == nil {
		metrics = globalMetrics
	}
	if namespace == "" {
		namespace = "http_server"
	}

	return func(c *gin.Context) {
		snapshot := metrics.GetSnapshot()
		var output string

		output += promMetric(namespace+"_requests_total", float64(snapshot.TotalRequests))
		output += promMetric(namespace+"_requests_success", float64(snapshot.SuccessRequests))
		output += promMetric(namespace+"_requests_failed", float64(snapshot.FailedRequests))
		output += promMetric(namespace+"_requests_in_flight", float64(snapshot.InFlightRequests))
		output += promMetric(namespace+"_latency_avg_seconds", snapshot.AvgLatency.Seconds())
		output += promMetric(namespace+"_qps", snapshot.QPS())

		for code, count := range snapshot.StatusCodeCounts {
			output += namespace + "_requests_by_status{code=\"" + strconv.Itoa(code) + "\"} " +
				strconv.FormatInt(count, 10) + "\n"
		}

		c.Header("Content-Type", "text/plain; charset=utf-8")
		c.String(200, output)
	}
}

func promMetric(name string, value float64) string {
	return name + " " + strconv.FormatFloat(value, 'f', -1, 64) + "\n"
}
