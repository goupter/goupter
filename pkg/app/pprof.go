package app

import (
	"net/http/pprof"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/goupter/goupter/pkg/server"
)

// PProfConfig pprof 配置
type PProfConfig struct {
	Enabled  bool
	BasePath string // 默认 /debug/pprof
}

// DefaultPProfConfig 默认 pprof 配置
func DefaultPProfConfig() *PProfConfig {
	return &PProfConfig{
		Enabled:  true,
		BasePath: "/debug/pprof",
	}
}

func registerPProf(httpServer *server.HTTPServer, cfg *PProfConfig) {
	if cfg == nil {
		cfg = DefaultPProfConfig()
	}

	base := normalizeBasePath(cfg.BasePath, "/debug/pprof")

	httpServer.GET(base+"/", gin.WrapF(pprof.Index))
	httpServer.GET(base+"/cmdline", gin.WrapF(pprof.Cmdline))
	httpServer.GET(base+"/profile", gin.WrapF(pprof.Profile))
	httpServer.GET(base+"/symbol", gin.WrapF(pprof.Symbol))
	httpServer.POST(base+"/symbol", gin.WrapF(pprof.Symbol))
	httpServer.GET(base+"/trace", gin.WrapF(pprof.Trace))

	handlers := []string{
		"allocs",
		"block",
		"goroutine",
		"heap",
		"mutex",
		"threadcreate",
	}
	for _, name := range handlers {
		httpServer.GET(base+"/"+name, gin.WrapH(pprof.Handler(name)))
	}
}

func normalizeBasePath(path, defaultPath string) string {
	p := strings.TrimSpace(path)
	if p == "" {
		p = defaultPath
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	return strings.TrimRight(p, "/")
}

