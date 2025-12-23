package errors

// 通用错误码
const (
	// CodeSuccess 成功
	CodeSuccess = 0

	// 客户端错误 (4xx)
	CodeBadRequest          = 400 // 请求参数错误
	CodeUnauthorized        = 401 // 未授权
	CodePaymentRequired     = 402 // 需要付费
	CodeForbidden           = 403 // 禁止访问
	CodeNotFound            = 404 // 资源不存在
	CodeMethodNotAllowed    = 405 // 方法不允许
	CodeNotAcceptable       = 406 // 不可接受
	CodeRequestTimeout      = 408 // 请求超时
	CodeConflict            = 409 // 资源冲突
	CodeGone                = 410 // 资源已删除
	CodeUnprocessableEntity = 422 // 无法处理的实体
	CodeTooManyRequests     = 429 // 请求过多

	// 服务端错误 (5xx)
	CodeInternalError     = 500 // 内部错误
	CodeNotImplemented    = 501 // 未实现
	CodeBadGateway        = 502 // 网关错误
	CodeServiceUnavailable = 503 // 服务不可用
	CodeGatewayTimeout    = 504 // 网关超时

	// 业务错误码 (10000+)
	CodeInvalidParam     = 10001 // 参数无效
	CodeInvalidToken     = 10002 // 令牌无效
	CodeTokenExpired     = 10003 // 令牌过期
	CodePermissionDenied = 10004 // 权限不足
	CodeUserNotFound     = 10005 // 用户不存在
	CodeUserDisabled     = 10006 // 用户已禁用
	CodePasswordError    = 10007 // 密码错误
	CodeAccountLocked    = 10008 // 账户已锁定

	// 数据库错误码 (20000+)
	CodeDBError        = 20001 // 数据库错误
	CodeDBNotFound     = 20002 // 记录不存在
	CodeDBDuplicate    = 20003 // 记录重复
	CodeDBDeadlock     = 20004 // 死锁
	CodeDBTimeout      = 20005 // 数据库超时

	// 缓存错误码 (30000+)
	CodeCacheError   = 30001 // 缓存错误
	CodeCacheMiss    = 30002 // 缓存未命中
	CodeCacheExpired = 30003 // 缓存已过期

	// 外部服务错误码 (40000+)
	CodeExternalError   = 40001 // 外部服务错误
	CodeExternalTimeout = 40002 // 外部服务超时

	// 消息队列错误码 (50000+)
	CodeMQError     = 50001 // 消息队列错误
	CodeMQTimeout   = 50002 // 消息队列超时
	CodeMQPublish   = 50003 // 消息发布失败
	CodeMQSubscribe = 50004 // 消息订阅失败
)

// 预定义错误
var (
	// 通用错误
	ErrSuccess           = New(CodeSuccess, "success")
	ErrBadRequest        = New(CodeBadRequest, "bad request")
	ErrUnauthorized      = New(CodeUnauthorized, "unauthorized")
	ErrForbidden         = New(CodeForbidden, "forbidden")
	ErrNotFound          = New(CodeNotFound, "not found")
	ErrMethodNotAllowed  = New(CodeMethodNotAllowed, "method not allowed")
	ErrTooManyRequests   = New(CodeTooManyRequests, "too many requests")
	ErrInternalError     = New(CodeInternalError, "internal error")
	ErrServiceUnavailable = New(CodeServiceUnavailable, "service unavailable")

	// 业务错误
	ErrInvalidParam     = New(CodeInvalidParam, "invalid parameter")
	ErrInvalidToken     = New(CodeInvalidToken, "invalid token")
	ErrTokenExpired     = New(CodeTokenExpired, "token expired")
	ErrPermissionDenied = New(CodePermissionDenied, "permission denied")
	ErrUserNotFound     = New(CodeUserNotFound, "user not found")
	ErrUserDisabled     = New(CodeUserDisabled, "user disabled")
	ErrPasswordError    = New(CodePasswordError, "password error")
	ErrAccountLocked    = New(CodeAccountLocked, "account locked")

	// 数据库错误
	ErrDBError     = New(CodeDBError, "database error")
	ErrDBNotFound  = New(CodeDBNotFound, "record not found")
	ErrDBDuplicate = New(CodeDBDuplicate, "duplicate record")
	ErrDBDeadlock  = New(CodeDBDeadlock, "database deadlock")
	ErrDBTimeout   = New(CodeDBTimeout, "database timeout")

	// 缓存错误
	ErrCacheError   = New(CodeCacheError, "cache error")
	ErrCacheMiss    = New(CodeCacheMiss, "cache miss")
	ErrCacheExpired = New(CodeCacheExpired, "cache expired")

	// 外部服务错误
	ErrExternalError   = New(CodeExternalError, "external service error")
	ErrExternalTimeout = New(CodeExternalTimeout, "external service timeout")

	// 消息队列错误
	ErrMQError     = New(CodeMQError, "message queue error")
	ErrMQTimeout   = New(CodeMQTimeout, "message queue timeout")
	ErrMQPublish   = New(CodeMQPublish, "message publish failed")
	ErrMQSubscribe = New(CodeMQSubscribe, "message subscribe failed")
)

// CodeMessage 错误码消息映射
var CodeMessage = map[int]string{
	CodeSuccess:           "success",
	CodeBadRequest:        "bad request",
	CodeUnauthorized:      "unauthorized",
	CodeForbidden:         "forbidden",
	CodeNotFound:          "not found",
	CodeMethodNotAllowed:  "method not allowed",
	CodeTooManyRequests:   "too many requests",
	CodeInternalError:     "internal error",
	CodeServiceUnavailable: "service unavailable",
	CodeInvalidParam:      "invalid parameter",
	CodeInvalidToken:      "invalid token",
	CodeTokenExpired:      "token expired",
	CodePermissionDenied:  "permission denied",
	CodeUserNotFound:      "user not found",
	CodeUserDisabled:      "user disabled",
	CodePasswordError:     "password error",
	CodeAccountLocked:     "account locked",
	CodeDBError:           "database error",
	CodeDBNotFound:        "record not found",
	CodeDBDuplicate:       "duplicate record",
	CodeDBDeadlock:        "database deadlock",
	CodeDBTimeout:         "database timeout",
	CodeCacheError:        "cache error",
	CodeCacheMiss:         "cache miss",
	CodeCacheExpired:      "cache expired",
	CodeExternalError:     "external service error",
	CodeExternalTimeout:   "external service timeout",
	CodeMQError:           "message queue error",
	CodeMQTimeout:         "message queue timeout",
	CodeMQPublish:         "message publish failed",
	CodeMQSubscribe:       "message subscribe failed",
}

// GetMessageByCode 根据错误码获取消息
func GetMessageByCode(code int) string {
	if msg, ok := CodeMessage[code]; ok {
		return msg
	}
	return "unknown error"
}
