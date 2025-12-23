package middleware

import (
	"net"
	"net/http/httputil"
	"os"
	"runtime/debug"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/goupter/goupter/pkg/errors"
	"github.com/goupter/goupter/pkg/log"
	"github.com/goupter/goupter/pkg/response"
)

// Recovery 恢复中间件
func Recovery(logger log.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// 检查是否是连接断开错误
				var brokenPipe bool
				if ne, ok := err.(*net.OpError); ok {
					if se, ok := ne.Err.(*os.SyscallError); ok {
						if strings.Contains(strings.ToLower(se.Error()), "broken pipe") ||
							strings.Contains(strings.ToLower(se.Error()), "connection reset by peer") {
							brokenPipe = true
						}
					}
				}

				httpRequest, _ := httputil.DumpRequest(c.Request, false)

				if brokenPipe {
					if logger != nil {
						logger.Error("broken pipe",
							log.String("path", c.Request.URL.Path),
							log.Any("error", err),
							log.String("request", string(httpRequest)),
						)
					}
					c.Error(err.(error))
					c.Abort()
					return
				}

				// 记录堆栈
				stack := string(debug.Stack())
				if logger != nil {
					logger.Error("panic recovered",
						log.Any("error", err),
						log.String("request", string(httpRequest)),
						log.String("stack", stack),
					)
				}

				// 返回500错误
				response.Abort(c, errors.ErrInternalError)
			}
		}()
		c.Next()
	}
}

// RecoveryWithWriter 带自定义Writer的恢复中间件
func RecoveryWithWriter(logger log.Logger, out ...gin.RecoveryFunc) gin.HandlerFunc {
	if len(out) > 0 {
		return gin.CustomRecoveryWithWriter(nil, out[0])
	}
	return Recovery(logger)
}
