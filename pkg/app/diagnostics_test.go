package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/goupter/goupter/pkg/server"
)

func TestApp_RegisterBuiltinEndpoints_MetricsAndPProf(t *testing.T) {
	gin.SetMode(gin.TestMode)

	httpServer := server.NewHTTPServer()
	a := New(
		WithHTTPServer(httpServer),
		WithHealthConfig(DefaultHealthConfig()),
		WithMetricsConfig(&MetricsConfig{
			Enabled: true,
			Path:    "/metrics",
			Format:  MetricsFormatJSON,
		}),
		WithPProfConfig(&PProfConfig{
			Enabled:  true,
			BasePath: "/debug/pprof",
		}),
	)

	a.registerBuiltinHTTPEndpoints()

	// 注册一个业务路由，用于产生指标
	httpServer.GET("/ping", func(c *gin.Context) {
		c.String(200, "ok")
	})

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	httpServer.Engine().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected /ping 200, got %d", rr.Code)
	}

	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/metrics", nil)
	httpServer.Engine().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected /metrics 200, got %d", rr.Code)
	}

	var payload map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal /metrics response failed: %v", err)
	}

	totalRequests, ok := payload["total_requests"].(float64)
	if !ok {
		t.Fatalf("expected total_requests number, got %T", payload["total_requests"])
	}
	if totalRequests != 1 {
		t.Fatalf("expected total_requests == 1 (metrics endpoint skipped), got %v", totalRequests)
	}

	rr = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/debug/pprof/", nil)
	httpServer.Engine().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected /debug/pprof/ 200, got %d", rr.Code)
	}
}

func TestBuilder_InitMetricsAndPProf(t *testing.T) {
	b := NewBuilder().InitMetrics(nil).InitPProf(nil)
	a, err := b.Build()
	if err != nil {
		t.Fatalf("build failed: %v", err)
	}
	if a.metricsConfig == nil || !a.metricsConfig.Enabled {
		t.Fatalf("expected metrics enabled by default config")
	}
	if a.pprofConfig == nil || !a.pprofConfig.Enabled {
		t.Fatalf("expected pprof enabled by default config")
	}
}

