package errors

import (
	"fmt"
	"net/http"
)

// Error 业务错误
type Error struct {
	Code    int         `json:"code"`              // 业务错误码
	Message string      `json:"message"`           // 错误消息
	Details interface{} `json:"details,omitempty"` // 错误详情
	Cause   error       `json:"-"`                 // 原始错误
}

// Error 实现error接口
func (e *Error) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("code: %d, message: %s, cause: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("code: %d, message: %s", e.Code, e.Message)
}

// Unwrap 返回原始错误
func (e *Error) Unwrap() error {
	return e.Cause
}

// WithCause 设置原始错误
func (e *Error) WithCause(cause error) *Error {
	return &Error{
		Code:    e.Code,
		Message: e.Message,
		Details: e.Details,
		Cause:   cause,
	}
}

// WithDetails 设置错误详情
func (e *Error) WithDetails(details interface{}) *Error {
	return &Error{
		Code:    e.Code,
		Message: e.Message,
		Details: details,
		Cause:   e.Cause,
	}
}

// WithMessage 设置错误消息
func (e *Error) WithMessage(message string) *Error {
	return &Error{
		Code:    e.Code,
		Message: message,
		Details: e.Details,
		Cause:   e.Cause,
	}
}

// HTTPStatus 获取HTTP状态码
func (e *Error) HTTPStatus() int {
	switch {
	case e.Code >= 400 && e.Code < 600:
		return e.Code
	case e.Code == CodeSuccess:
		return http.StatusOK
	default:
		return http.StatusInternalServerError
	}
}

// New 创建新错误
func New(code int, message string) *Error {
	return &Error{
		Code:    code,
		Message: message,
	}
}

// Newf 创建格式化消息的新错误
func Newf(code int, format string, args ...interface{}) *Error {
	return &Error{
		Code:    code,
		Message: fmt.Sprintf(format, args...),
	}
}

// Wrap 包装错误
func Wrap(err error, code int, message string) *Error {
	return &Error{
		Code:    code,
		Message: message,
		Cause:   err,
	}
}

// Wrapf 包装错误并格式化消息
func Wrapf(err error, code int, format string, args ...interface{}) *Error {
	return &Error{
		Code:    code,
		Message: fmt.Sprintf(format, args...),
		Cause:   err,
	}
}

// FromError 从标准error转换
func FromError(err error) *Error {
	if err == nil {
		return nil
	}

	// 如果已经是Error类型，直接返回
	if e, ok := err.(*Error); ok {
		return e
	}

	// 否则包装为内部错误
	return &Error{
		Code:    CodeInternalError,
		Message: err.Error(),
		Cause:   err,
	}
}

// Is 判断错误是否为指定错误
func Is(err error, target *Error) bool {
	if err == nil || target == nil {
		return err == nil && target == nil
	}

	e, ok := err.(*Error)
	if !ok {
		return false
	}

	return e.Code == target.Code
}

// IsCode 判断错误码是否匹配
func IsCode(err error, code int) bool {
	if err == nil {
		return false
	}

	e, ok := err.(*Error)
	if !ok {
		return false
	}

	return e.Code == code
}

// GetCode 获取错误码
func GetCode(err error) int {
	if err == nil {
		return CodeSuccess
	}

	if e, ok := err.(*Error); ok {
		return e.Code
	}

	return CodeInternalError
}

// GetMessage 获取错误消息
func GetMessage(err error) string {
	if err == nil {
		return "success"
	}

	if e, ok := err.(*Error); ok {
		return e.Message
	}

	return err.Error()
}
