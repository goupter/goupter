package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/goupter/goupter/pkg/errors"
)

// Response 统一响应结构
type Response struct {
	Code    int         `json:"code"`               // 业务错误码
	Message string      `json:"message"`            // 响应消息
	Data    interface{} `json:"data,omitempty"`     // 响应数据
	TraceID string      `json:"trace_id,omitempty"` // 链路追踪ID
}

// PageData 分页数据
type PageData struct {
	List     interface{} `json:"list"`      // 数据列表
	Total    int64       `json:"total"`     // 总数
	Page     int         `json:"page"`      // 当前页
	PageSize int         `json:"page_size"` // 每页数量
}

// JSON 返回JSON响应
func JSON(c *gin.Context, code int, message string, data interface{}) {
	resp := Response{
		Code:    code,
		Message: message,
		Data:    data,
	}

	if traceID, exists := c.Get("trace_id"); exists {
		if id, ok := traceID.(string); ok {
			resp.TraceID = id
		}
	}

	c.JSON(http.StatusOK, resp)
}

// Success 成功响应
func Success(c *gin.Context, data interface{}) {
	JSON(c, errors.CodeSuccess, "success", data)
}

// Error 错误响应
func Error(c *gin.Context, err error) {
	e := errors.FromError(err)
	JSON(c, e.Code, e.Message, e.Details)
}

// Abort 中止请求并返回错误
func Abort(c *gin.Context, err error) {
	e := errors.FromError(err)
	c.AbortWithStatusJSON(http.StatusOK, Response{
		Code:    e.Code,
		Message: e.Message,
		Data:    e.Details,
	})
}
