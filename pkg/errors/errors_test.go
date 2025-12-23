package errors

import (
	"errors"
	"net/http"
	"testing"
)

func TestError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *Error
		contains string
	}{
		{
			name:     "without cause",
			err:      New(CodeBadRequest, "bad request"),
			contains: "code: 400, message: bad request",
		},
		{
			name:     "with cause",
			err:      Wrap(errors.New("underlying error"), CodeInternalError, "internal error"),
			contains: "cause: underlying error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			if len(got) == 0 {
				t.Error("Error() should not return empty string")
			}
		})
	}
}

func TestError_Unwrap(t *testing.T) {
	cause := errors.New("original error")
	err := Wrap(cause, CodeInternalError, "wrapped")

	unwrapped := err.Unwrap()
	if unwrapped != cause {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, cause)
	}
}

func TestError_Unwrap_NoCause(t *testing.T) {
	err := New(CodeBadRequest, "no cause")
	if err.Unwrap() != nil {
		t.Error("Unwrap() should return nil when no cause")
	}
}

func TestError_WithCause(t *testing.T) {
	original := New(CodeBadRequest, "bad request")
	cause := errors.New("underlying")

	newErr := original.WithCause(cause)

	if newErr.Cause != cause {
		t.Error("WithCause should set cause")
	}
	if newErr.Code != original.Code {
		t.Error("WithCause should preserve code")
	}
	if newErr.Message != original.Message {
		t.Error("WithCause should preserve message")
	}
	if newErr == original {
		t.Error("WithCause should return new instance")
	}
}

func TestError_WithDetails(t *testing.T) {
	original := New(CodeBadRequest, "bad request")
	details := map[string]string{"field": "email", "reason": "invalid format"}

	newErr := original.WithDetails(details)

	if newErr.Details == nil {
		t.Error("WithDetails should set details")
	}
	if newErr.Code != original.Code {
		t.Error("WithDetails should preserve code")
	}
	if newErr.Message != original.Message {
		t.Error("WithDetails should preserve message")
	}
	if newErr == original {
		t.Error("WithDetails should return new instance")
	}
}

func TestError_WithMessage(t *testing.T) {
	original := New(CodeBadRequest, "bad request")
	newMessage := "custom message"

	newErr := original.WithMessage(newMessage)

	if newErr.Message != newMessage {
		t.Errorf("WithMessage() message = %s, want %s", newErr.Message, newMessage)
	}
	if newErr.Code != original.Code {
		t.Error("WithMessage should preserve code")
	}
	if newErr == original {
		t.Error("WithMessage should return new instance")
	}
}

func TestError_HTTPStatus(t *testing.T) {
	tests := []struct {
		name     string
		err      *Error
		expected int
	}{
		{"400 code", New(400, "bad request"), http.StatusBadRequest},
		{"401 code", New(401, "unauthorized"), http.StatusUnauthorized},
		{"404 code", New(404, "not found"), http.StatusNotFound},
		{"500 code", New(500, "internal error"), http.StatusInternalServerError},
		{"success code", New(CodeSuccess, "success"), http.StatusOK},
		{"business code", New(10001, "invalid param"), http.StatusInternalServerError},
		{"db code", New(20001, "db error"), http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.HTTPStatus()
			if got != tt.expected {
				t.Errorf("HTTPStatus() = %d, want %d", got, tt.expected)
			}
		})
	}
}

func TestNew(t *testing.T) {
	err := New(CodeBadRequest, "bad request")

	if err == nil {
		t.Fatal("New should return non-nil")
	}
	if err.Code != CodeBadRequest {
		t.Errorf("Code = %d, want %d", err.Code, CodeBadRequest)
	}
	if err.Message != "bad request" {
		t.Errorf("Message = %s, want bad request", err.Message)
	}
	if err.Cause != nil {
		t.Error("Cause should be nil")
	}
	if err.Details != nil {
		t.Error("Details should be nil")
	}
}

func TestNewf(t *testing.T) {
	err := Newf(CodeBadRequest, "field %s is %s", "email", "invalid")

	if err == nil {
		t.Fatal("Newf should return non-nil")
	}
	if err.Code != CodeBadRequest {
		t.Errorf("Code = %d, want %d", err.Code, CodeBadRequest)
	}
	if err.Message != "field email is invalid" {
		t.Errorf("Message = %s, want 'field email is invalid'", err.Message)
	}
}

func TestWrap(t *testing.T) {
	cause := errors.New("original error")
	err := Wrap(cause, CodeInternalError, "wrapped error")

	if err == nil {
		t.Fatal("Wrap should return non-nil")
	}
	if err.Code != CodeInternalError {
		t.Errorf("Code = %d, want %d", err.Code, CodeInternalError)
	}
	if err.Message != "wrapped error" {
		t.Errorf("Message = %s, want wrapped error", err.Message)
	}
	if err.Cause != cause {
		t.Error("Cause should be set")
	}
}

func TestWrapf(t *testing.T) {
	cause := errors.New("original error")
	err := Wrapf(cause, CodeInternalError, "operation %s failed", "save")

	if err == nil {
		t.Fatal("Wrapf should return non-nil")
	}
	if err.Code != CodeInternalError {
		t.Errorf("Code = %d, want %d", err.Code, CodeInternalError)
	}
	if err.Message != "operation save failed" {
		t.Errorf("Message = %s, want 'operation save failed'", err.Message)
	}
	if err.Cause != cause {
		t.Error("Cause should be set")
	}
}

func TestFromError(t *testing.T) {
	t.Run("nil error", func(t *testing.T) {
		result := FromError(nil)
		if result != nil {
			t.Error("FromError(nil) should return nil")
		}
	})

	t.Run("already Error type", func(t *testing.T) {
		original := New(CodeBadRequest, "bad request")
		result := FromError(original)
		if result != original {
			t.Error("FromError should return same Error instance")
		}
	})

	t.Run("standard error", func(t *testing.T) {
		stdErr := errors.New("standard error")
		result := FromError(stdErr)

		if result == nil {
			t.Fatal("FromError should return non-nil")
		}
		if result.Code != CodeInternalError {
			t.Errorf("Code = %d, want %d", result.Code, CodeInternalError)
		}
		if result.Message != "standard error" {
			t.Errorf("Message = %s, want 'standard error'", result.Message)
		}
		if result.Cause != stdErr {
			t.Error("Cause should be set to original error")
		}
	})
}

func TestIs(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		target   *Error
		expected bool
	}{
		{"both nil", nil, nil, true},
		{"err nil", nil, ErrBadRequest, false},
		{"target nil", ErrBadRequest, nil, false},
		{"same error", ErrBadRequest, ErrBadRequest, true},
		{"same code different instance", New(CodeBadRequest, "custom"), ErrBadRequest, true},
		{"different code", ErrBadRequest, ErrNotFound, false},
		{"standard error", errors.New("std"), ErrBadRequest, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Is(tt.err, tt.target)
			if got != tt.expected {
				t.Errorf("Is() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsCode(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		code     int
		expected bool
	}{
		{"nil error", nil, CodeBadRequest, false},
		{"matching code", ErrBadRequest, CodeBadRequest, true},
		{"non-matching code", ErrBadRequest, CodeNotFound, false},
		{"standard error", errors.New("std"), CodeBadRequest, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsCode(tt.err, tt.code)
			if got != tt.expected {
				t.Errorf("IsCode() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGetCode(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected int
	}{
		{"nil error", nil, CodeSuccess},
		{"Error type", ErrBadRequest, CodeBadRequest},
		{"standard error", errors.New("std"), CodeInternalError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetCode(tt.err)
			if got != tt.expected {
				t.Errorf("GetCode() = %d, want %d", got, tt.expected)
			}
		})
	}
}

func TestGetMessage(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{"nil error", nil, "success"},
		{"Error type", ErrBadRequest, "bad request"},
		{"standard error", errors.New("standard message"), "standard message"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetMessage(tt.err)
			if got != tt.expected {
				t.Errorf("GetMessage() = %s, want %s", got, tt.expected)
			}
		})
	}
}

func TestError_ChainMethods(t *testing.T) {
	// 测试链式调用
	err := New(CodeBadRequest, "original").
		WithMessage("custom message").
		WithDetails(map[string]string{"field": "name"}).
		WithCause(errors.New("underlying"))

	if err.Code != CodeBadRequest {
		t.Errorf("Code = %d, want %d", err.Code, CodeBadRequest)
	}
	if err.Message != "custom message" {
		t.Errorf("Message = %s, want custom message", err.Message)
	}
	if err.Details == nil {
		t.Error("Details should not be nil")
	}
	if err.Cause == nil {
		t.Error("Cause should not be nil")
	}
}

func TestError_Implements(t *testing.T) {
	var _ error = (*Error)(nil)
}
