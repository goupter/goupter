package app

import (
	"strings"

	"github.com/goupter/goupter/pkg/server"
	"github.com/goupter/goupter/pkg/server/middleware"
)

// MetricsFormat 指标输出格式
type MetricsFormat string

const (
	MetricsFormatPrometheus MetricsFormat = "prometheus"
	MetricsFormatJSON       MetricsFormat = "json"
)

// MetricsConfig 指标配置
type MetricsConfig struct {
	Enabled   bool
	Path      string
	Format    MetricsFormat
	Namespace string
	SkipPaths []string
	Metrics   *middleware.RequestMetrics
}

// DefaultMetricsConfig 默认指标配置
func DefaultMetricsConfig() *MetricsConfig {
	return &MetricsConfig{
		Enabled:   true,
		Path:      "/metrics",
		Format:    MetricsFormatPrometheus,
		Namespace: "http_server",
	}
}

func registerMetrics(httpServer *server.HTTPServer, cfg *MetricsConfig) *middleware.RequestMetrics {
	if cfg == nil {
		cfg = DefaultMetricsConfig()
	}

	path := normalizePath(cfg.Path, "/metrics")

	metrics := cfg.Metrics
	if metrics == nil {
		metrics = middleware.GetGlobalMetrics()
	}

	skipPaths := make([]string, 0, len(cfg.SkipPaths)+1)
	skipPaths = append(skipPaths, cfg.SkipPaths...)
	skipPaths = append(skipPaths, path)

	httpServer.Use(middleware.MetricsWithConfig(middleware.MetricsConfig{
		Metrics:   metrics,
		SkipPaths: skipPaths,
	}))

	switch cfg.Format {
	case MetricsFormatJSON:
		httpServer.GET(path, middleware.MetricsHandler(metrics))
	default:
		httpServer.GET(path, middleware.PrometheusHandler(metrics, cfg.Namespace))
	}

	return metrics
}

func normalizePath(path, defaultPath string) string {
	p := strings.TrimSpace(path)
	if p == "" {
		p = defaultPath
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	return p
}
