package errors

import (
	"testing"
)

func TestErrorCodes_Constants(t *testing.T) {
	// 验证错误码常量值
	tests := []struct {
		name     string
		code     int
		expected int
	}{
		// 通用错误码
		{"CodeSuccess", CodeSuccess, 0},
		{"CodeBadRequest", CodeBadRequest, 400},
		{"CodeUnauthorized", CodeUnauthorized, 401},
		{"CodePaymentRequired", CodePaymentRequired, 402},
		{"CodeForbidden", CodeForbidden, 403},
		{"CodeNotFound", CodeNotFound, 404},
		{"CodeMethodNotAllowed", CodeMethodNotAllowed, 405},
		{"CodeNotAcceptable", CodeNotAcceptable, 406},
		{"CodeRequestTimeout", CodeRequestTimeout, 408},
		{"CodeConflict", CodeConflict, 409},
		{"CodeGone", CodeGone, 410},
		{"CodeUnprocessableEntity", CodeUnprocessableEntity, 422},
		{"CodeTooManyRequests", CodeTooManyRequests, 429},

		// 服务端错误
		{"CodeInternalError", CodeInternalError, 500},
		{"CodeNotImplemented", CodeNotImplemented, 501},
		{"CodeBadGateway", CodeBadGateway, 502},
		{"CodeServiceUnavailable", CodeServiceUnavailable, 503},
		{"CodeGatewayTimeout", CodeGatewayTimeout, 504},

		// 业务错误码
		{"CodeInvalidParam", CodeInvalidParam, 10001},
		{"CodeInvalidToken", CodeInvalidToken, 10002},
		{"CodeTokenExpired", CodeTokenExpired, 10003},
		{"CodePermissionDenied", CodePermissionDenied, 10004},
		{"CodeUserNotFound", CodeUserNotFound, 10005},
		{"CodeUserDisabled", CodeUserDisabled, 10006},
		{"CodePasswordError", CodePasswordError, 10007},
		{"CodeAccountLocked", CodeAccountLocked, 10008},

		// 数据库错误码
		{"CodeDBError", CodeDBError, 20001},
		{"CodeDBNotFound", CodeDBNotFound, 20002},
		{"CodeDBDuplicate", CodeDBDuplicate, 20003},
		{"CodeDBDeadlock", CodeDBDeadlock, 20004},
		{"CodeDBTimeout", CodeDBTimeout, 20005},

		// 缓存错误码
		{"CodeCacheError", CodeCacheError, 30001},
		{"CodeCacheMiss", CodeCacheMiss, 30002},
		{"CodeCacheExpired", CodeCacheExpired, 30003},

		// 外部服务错误码
		{"CodeExternalError", CodeExternalError, 40001},
		{"CodeExternalTimeout", CodeExternalTimeout, 40002},

		// 消息队列错误码
		{"CodeMQError", CodeMQError, 50001},
		{"CodeMQTimeout", CodeMQTimeout, 50002},
		{"CodeMQPublish", CodeMQPublish, 50003},
		{"CodeMQSubscribe", CodeMQSubscribe, 50004},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.code != tt.expected {
				t.Errorf("%s = %d, want %d", tt.name, tt.code, tt.expected)
			}
		})
	}
}

func TestPredefinedErrors(t *testing.T) {
	tests := []struct {
		name    string
		err     *Error
		code    int
		message string
	}{
		// 通用错误
		{"ErrSuccess", ErrSuccess, CodeSuccess, "success"},
		{"ErrBadRequest", ErrBadRequest, CodeBadRequest, "bad request"},
		{"ErrUnauthorized", ErrUnauthorized, CodeUnauthorized, "unauthorized"},
		{"ErrForbidden", ErrForbidden, CodeForbidden, "forbidden"},
		{"ErrNotFound", ErrNotFound, CodeNotFound, "not found"},
		{"ErrMethodNotAllowed", ErrMethodNotAllowed, CodeMethodNotAllowed, "method not allowed"},
		{"ErrTooManyRequests", ErrTooManyRequests, CodeTooManyRequests, "too many requests"},
		{"ErrInternalError", ErrInternalError, CodeInternalError, "internal error"},
		{"ErrServiceUnavailable", ErrServiceUnavailable, CodeServiceUnavailable, "service unavailable"},

		// 业务错误
		{"ErrInvalidParam", ErrInvalidParam, CodeInvalidParam, "invalid parameter"},
		{"ErrInvalidToken", ErrInvalidToken, CodeInvalidToken, "invalid token"},
		{"ErrTokenExpired", ErrTokenExpired, CodeTokenExpired, "token expired"},
		{"ErrPermissionDenied", ErrPermissionDenied, CodePermissionDenied, "permission denied"},
		{"ErrUserNotFound", ErrUserNotFound, CodeUserNotFound, "user not found"},
		{"ErrUserDisabled", ErrUserDisabled, CodeUserDisabled, "user disabled"},
		{"ErrPasswordError", ErrPasswordError, CodePasswordError, "password error"},
		{"ErrAccountLocked", ErrAccountLocked, CodeAccountLocked, "account locked"},

		// 数据库错误
		{"ErrDBError", ErrDBError, CodeDBError, "database error"},
		{"ErrDBNotFound", ErrDBNotFound, CodeDBNotFound, "record not found"},
		{"ErrDBDuplicate", ErrDBDuplicate, CodeDBDuplicate, "duplicate record"},
		{"ErrDBDeadlock", ErrDBDeadlock, CodeDBDeadlock, "database deadlock"},
		{"ErrDBTimeout", ErrDBTimeout, CodeDBTimeout, "database timeout"},

		// 缓存错误
		{"ErrCacheError", ErrCacheError, CodeCacheError, "cache error"},
		{"ErrCacheMiss", ErrCacheMiss, CodeCacheMiss, "cache miss"},
		{"ErrCacheExpired", ErrCacheExpired, CodeCacheExpired, "cache expired"},

		// 外部服务错误
		{"ErrExternalError", ErrExternalError, CodeExternalError, "external service error"},
		{"ErrExternalTimeout", ErrExternalTimeout, CodeExternalTimeout, "external service timeout"},

		// 消息队列错误
		{"ErrMQError", ErrMQError, CodeMQError, "message queue error"},
		{"ErrMQTimeout", ErrMQTimeout, CodeMQTimeout, "message queue timeout"},
		{"ErrMQPublish", ErrMQPublish, CodeMQPublish, "message publish failed"},
		{"ErrMQSubscribe", ErrMQSubscribe, CodeMQSubscribe, "message subscribe failed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil {
				t.Fatalf("%s should not be nil", tt.name)
			}
			if tt.err.Code != tt.code {
				t.Errorf("%s.Code = %d, want %d", tt.name, tt.err.Code, tt.code)
			}
			if tt.err.Message != tt.message {
				t.Errorf("%s.Message = %s, want %s", tt.name, tt.err.Message, tt.message)
			}
		})
	}
}

func TestCodeMessage_Map(t *testing.T) {
	// 验证 CodeMessage 映射包含所有预定义错误码
	expectedCodes := []int{
		CodeSuccess,
		CodeBadRequest,
		CodeUnauthorized,
		CodeForbidden,
		CodeNotFound,
		CodeMethodNotAllowed,
		CodeTooManyRequests,
		CodeInternalError,
		CodeServiceUnavailable,
		CodeInvalidParam,
		CodeInvalidToken,
		CodeTokenExpired,
		CodePermissionDenied,
		CodeUserNotFound,
		CodeUserDisabled,
		CodePasswordError,
		CodeAccountLocked,
		CodeDBError,
		CodeDBNotFound,
		CodeDBDuplicate,
		CodeDBDeadlock,
		CodeDBTimeout,
		CodeCacheError,
		CodeCacheMiss,
		CodeCacheExpired,
		CodeExternalError,
		CodeExternalTimeout,
		CodeMQError,
		CodeMQTimeout,
		CodeMQPublish,
		CodeMQSubscribe,
	}

	for _, code := range expectedCodes {
		if _, ok := CodeMessage[code]; !ok {
			t.Errorf("CodeMessage should contain code %d", code)
		}
	}
}

func TestGetMessageByCode(t *testing.T) {
	tests := []struct {
		name     string
		code     int
		expected string
	}{
		{"success", CodeSuccess, "success"},
		{"bad request", CodeBadRequest, "bad request"},
		{"unauthorized", CodeUnauthorized, "unauthorized"},
		{"not found", CodeNotFound, "not found"},
		{"internal error", CodeInternalError, "internal error"},
		{"invalid param", CodeInvalidParam, "invalid parameter"},
		{"db error", CodeDBError, "database error"},
		{"cache miss", CodeCacheMiss, "cache miss"},
		{"mq error", CodeMQError, "message queue error"},
		{"unknown code", 99999, "unknown error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetMessageByCode(tt.code)
			if got != tt.expected {
				t.Errorf("GetMessageByCode(%d) = %s, want %s", tt.code, got, tt.expected)
			}
		})
	}
}

func TestErrorCode_HTTPMapping(t *testing.T) {
	// 验证 HTTP 错误码映射到正确的 HTTP 状态
	httpCodes := []struct {
		code       int
		httpStatus int
	}{
		{CodeBadRequest, 400},
		{CodeUnauthorized, 401},
		{CodeForbidden, 403},
		{CodeNotFound, 404},
		{CodeMethodNotAllowed, 405},
		{CodeTooManyRequests, 429},
		{CodeInternalError, 500},
		{CodeServiceUnavailable, 503},
	}

	for _, tc := range httpCodes {
		err := New(tc.code, "test")
		if got := err.HTTPStatus(); got != tc.httpStatus {
			t.Errorf("HTTPStatus for code %d = %d, want %d", tc.code, got, tc.httpStatus)
		}
	}
}

func TestPredefinedErrors_Immutability(t *testing.T) {
	// 验证预定义错误的 WithMessage 等方法返回新实例而不修改原始错误
	original := ErrBadRequest
	originalMessage := original.Message

	newErr := original.WithMessage("custom message")

	if original.Message != originalMessage {
		t.Error("Original error should not be modified")
	}
	if newErr == original {
		t.Error("WithMessage should return a new instance")
	}
	if newErr.Message == originalMessage {
		t.Error("New error should have different message")
	}
}

func TestPredefinedErrors_UsagePatterns(t *testing.T) {
	// 测试常见使用模式
	t.Run("check specific error", func(t *testing.T) {
		err := ErrNotFound
		if !Is(err, ErrNotFound) {
			t.Error("Should match ErrNotFound")
		}
		if Is(err, ErrBadRequest) {
			t.Error("Should not match ErrBadRequest")
		}
	})

	t.Run("customize error message", func(t *testing.T) {
		err := ErrNotFound.WithMessage("user with id 123 not found")
		if !IsCode(err, CodeNotFound) {
			t.Error("Should have NotFound code")
		}
		if err.Message != "user with id 123 not found" {
			t.Errorf("Message = %s, want custom message", err.Message)
		}
	})

	t.Run("wrap with details", func(t *testing.T) {
		details := map[string]interface{}{
			"field": "email",
			"value": "invalid-email",
		}
		err := ErrInvalidParam.WithDetails(details)
		if err.Details == nil {
			t.Error("Details should be set")
		}
	})
}
