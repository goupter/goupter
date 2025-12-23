package interceptor

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/goupter/goupter/pkg/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// === 模拟对象 ===

type mockUnaryHandler func(ctx context.Context, req interface{}) (interface{}, error)

func (h mockUnaryHandler) Handle(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	return handler(ctx, req)
}

// === 日志拦截器测试 ===

func TestUnaryLogging_Success(t *testing.T) {
	cfg := DefaultLoggingConfig()
	cfg.Logger = log.Default()

	interceptor := UnaryLogging(cfg)

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/Method",
	}

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "response", nil
	}

	resp, err := interceptor(context.Background(), "request", info, handler)
	if err != nil {
		t.Errorf("不应该返回错误: %v", err)
	}
	if resp != "response" {
		t.Errorf("响应不正确: %v", resp)
	}
}

func TestUnaryLogging_Error(t *testing.T) {
	cfg := DefaultLoggingConfig()
	cfg.Logger = log.Default()

	interceptor := UnaryLogging(cfg)

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/Method",
	}

	expectedErr := status.Error(codes.Internal, "internal error")
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, expectedErr
	}

	_, err := interceptor(context.Background(), "request", info, handler)
	if err != expectedErr {
		t.Errorf("应该返回预期错误: %v", err)
	}
}

func TestUnaryLogging_SkipMethods(t *testing.T) {
	cfg := DefaultLoggingConfig()
	cfg.SkipMethods = []string{"/grpc.health.v1.Health/Check"}

	interceptor := UnaryLogging(cfg)

	info := &grpc.UnaryServerInfo{
		FullMethod: "/grpc.health.v1.Health/Check",
	}

	called := false
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		called = true
		return "ok", nil
	}

	_, err := interceptor(context.Background(), "request", info, handler)
	if err != nil {
		t.Errorf("不应该返回错误: %v", err)
	}
	if !called {
		t.Error("handler应该被调用")
	}
}

// === 恢复拦截器测试 ===

func TestUnaryRecovery_NoPanic(t *testing.T) {
	cfg := DefaultRecoveryConfig()
	interceptor := UnaryRecovery(cfg)

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/Method",
	}

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "ok", nil
	}

	resp, err := interceptor(context.Background(), "request", info, handler)
	if err != nil {
		t.Errorf("不应该返回错误: %v", err)
	}
	if resp != "ok" {
		t.Errorf("响应不正确: %v", resp)
	}
}

func TestUnaryRecovery_WithPanic(t *testing.T) {
	cfg := DefaultRecoveryConfig()
	cfg.Logger = log.Default()
	interceptor := UnaryRecovery(cfg)

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/Method",
	}

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		panic("test panic")
	}

	_, err := interceptor(context.Background(), "request", info, handler)
	if err == nil {
		t.Error("应该返回错误")
	}

	st, ok := status.FromError(err)
	if !ok {
		t.Error("应该是gRPC状态错误")
	}
	if st.Code() != codes.Internal {
		t.Errorf("应该返回Internal错误，实际: %v", st.Code())
	}
}

func TestUnaryRecovery_CustomRecoveryFunc(t *testing.T) {
	customErr := status.Error(codes.Aborted, "custom recovery")
	cfg := RecoveryConfig{
		Logger: log.Default(),
		RecoveryFunc: func(ctx context.Context, p interface{}) error {
			return customErr
		},
	}
	interceptor := UnaryRecovery(cfg)

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/Method",
	}

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		panic("test panic")
	}

	_, err := interceptor(context.Background(), "request", info, handler)
	if err != customErr {
		t.Errorf("应该返回自定义错误: %v", err)
	}
}

// === 认证拦截器测试 ===

func TestUnaryAuth_Success(t *testing.T) {
	cfg := DefaultAuthConfig()
	cfg.TokenValidator = func(ctx context.Context, token string) (string, error) {
		if token == "valid_token" {
			return "user123", nil
		}
		return "", errors.New("invalid token")
	}

	interceptor := UnaryAuth(cfg)

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/Method",
	}

	// 创建带有metadata的上下文
	md := metadata.New(map[string]string{
		"authorization": "Bearer valid_token",
	})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		userID, ok := GetUserID(ctx)
		if !ok {
			t.Error("应该能获取userID")
		}
		if userID != "user123" {
			t.Errorf("userID不正确: %s", userID)
		}
		return "ok", nil
	}

	_, err := interceptor(ctx, "request", info, handler)
	if err != nil {
		t.Errorf("不应该返回错误: %v", err)
	}
}

func TestUnaryAuth_MissingToken(t *testing.T) {
	cfg := DefaultAuthConfig()
	cfg.TokenValidator = func(ctx context.Context, token string) (string, error) {
		return "user", nil
	}

	interceptor := UnaryAuth(cfg)

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/Method",
	}

	// 没有metadata
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "ok", nil
	}

	_, err := interceptor(context.Background(), "request", info, handler)
	if err == nil {
		t.Error("应该返回错误")
	}

	st, _ := status.FromError(err)
	if st.Code() != codes.Unauthenticated {
		t.Errorf("应该返回Unauthenticated，实际: %v", st.Code())
	}
}

func TestUnaryAuth_InvalidToken(t *testing.T) {
	cfg := DefaultAuthConfig()
	cfg.TokenValidator = func(ctx context.Context, token string) (string, error) {
		return "", errors.New("invalid")
	}

	interceptor := UnaryAuth(cfg)

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/Method",
	}

	md := metadata.New(map[string]string{
		"authorization": "Bearer invalid_token",
	})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "ok", nil
	}

	_, err := interceptor(ctx, "request", info, handler)
	if err == nil {
		t.Error("应该返回错误")
	}

	st, _ := status.FromError(err)
	if st.Code() != codes.Unauthenticated {
		t.Errorf("应该返回Unauthenticated，实际: %v", st.Code())
	}
}

func TestUnaryAuth_SkipMethods(t *testing.T) {
	cfg := DefaultAuthConfig()
	cfg.TokenValidator = func(ctx context.Context, token string) (string, error) {
		return "", errors.New("should not be called")
	}

	interceptor := UnaryAuth(cfg)

	info := &grpc.UnaryServerInfo{
		FullMethod: "/grpc.health.v1.Health/Check",
	}

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "ok", nil
	}

	// 不需要认证
	_, err := interceptor(context.Background(), "request", info, handler)
	if err != nil {
		t.Errorf("跳过的方法不应该返回错误: %v", err)
	}
}

// === 超时拦截器测试 ===

func TestUnaryTimeout_Success(t *testing.T) {
	interceptor := UnaryTimeout(time.Second)

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/Method",
	}

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		// 检查上下文是否有deadline
		deadline, ok := ctx.Deadline()
		if !ok {
			t.Error("上下文应该有deadline")
		}
		if time.Until(deadline) > time.Second+10*time.Millisecond {
			t.Error("deadline不正确")
		}
		return "ok", nil
	}

	_, err := interceptor(context.Background(), "request", info, handler)
	if err != nil {
		t.Errorf("不应该返回错误: %v", err)
	}
}

// === 限流拦截器测试 ===

type mockRateLimiter struct {
	allowed bool
}

func (m *mockRateLimiter) Allow() bool {
	return m.allowed
}

func TestUnaryRateLimit_Allowed(t *testing.T) {
	limiter := &mockRateLimiter{allowed: true}
	interceptor := UnaryRateLimit(limiter)

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/Method",
	}

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "ok", nil
	}

	_, err := interceptor(context.Background(), "request", info, handler)
	if err != nil {
		t.Errorf("不应该返回错误: %v", err)
	}
}

func TestUnaryRateLimit_Denied(t *testing.T) {
	limiter := &mockRateLimiter{allowed: false}
	interceptor := UnaryRateLimit(limiter)

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/Method",
	}

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "ok", nil
	}

	_, err := interceptor(context.Background(), "request", info, handler)
	if err == nil {
		t.Error("应该返回错误")
	}

	st, _ := status.FromError(err)
	if st.Code() != codes.ResourceExhausted {
		t.Errorf("应该返回ResourceExhausted，实际: %v", st.Code())
	}
}

// === 请求ID拦截器测试 ===

func TestUnaryRequestID_Generate(t *testing.T) {
	generator := func() string {
		return "generated-id-123"
	}

	interceptor := UnaryRequestID(generator)

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/Method",
	}

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		requestID, ok := GetRequestID(ctx)
		if !ok {
			t.Error("应该能获取requestID")
		}
		if requestID != "generated-id-123" {
			t.Errorf("requestID不正确: %s", requestID)
		}
		return "ok", nil
	}

	_, err := interceptor(context.Background(), "request", info, handler)
	if err != nil {
		t.Errorf("不应该返回错误: %v", err)
	}
}

func TestUnaryRequestID_FromMetadata(t *testing.T) {
	generator := func() string {
		return "should-not-use"
	}

	interceptor := UnaryRequestID(generator)

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/Method",
	}

	md := metadata.New(map[string]string{
		"x-request-id": "existing-id-456",
	})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		requestID, ok := GetRequestID(ctx)
		if !ok {
			t.Error("应该能获取requestID")
		}
		if requestID != "existing-id-456" {
			t.Errorf("应该使用已有的requestID: %s", requestID)
		}
		return "ok", nil
	}

	_, err := interceptor(ctx, "request", info, handler)
	if err != nil {
		t.Errorf("不应该返回错误: %v", err)
	}
}

// === 辅助函数测试 ===

func TestGetUserID(t *testing.T) {
	// 没有userID
	ctx := context.Background()
	_, ok := GetUserID(ctx)
	if ok {
		t.Error("没有设置userID时不应该返回ok")
	}

	// 有userID
	ctx = context.WithValue(ctx, userIDKey{}, "user123")
	userID, ok := GetUserID(ctx)
	if !ok {
		t.Error("应该返回ok")
	}
	if userID != "user123" {
		t.Errorf("userID不正确: %s", userID)
	}
}

func TestGetRequestID(t *testing.T) {
	// 没有requestID
	ctx := context.Background()
	_, ok := GetRequestID(ctx)
	if ok {
		t.Error("没有设置requestID时不应该返回ok")
	}

	// 有requestID
	ctx = context.WithValue(ctx, requestIDKey{}, "req-123")
	requestID, ok := GetRequestID(ctx)
	if !ok {
		t.Error("应该返回ok")
	}
	if requestID != "req-123" {
		t.Errorf("requestID不正确: %s", requestID)
	}
}

// === 默认配置测试 ===

func TestDefaultLoggingConfig(t *testing.T) {
	cfg := DefaultLoggingConfig()

	if cfg.SlowThreshold != 200*time.Millisecond {
		t.Errorf("SlowThreshold不正确: %v", cfg.SlowThreshold)
	}
	if cfg.LogPayload {
		t.Error("默认不应该记录payload")
	}
	if cfg.MaxPayloadLength != 1024 {
		t.Errorf("MaxPayloadLength不正确: %d", cfg.MaxPayloadLength)
	}
}

func TestDefaultRecoveryConfig(t *testing.T) {
	cfg := DefaultRecoveryConfig()

	if !cfg.EnableStack {
		t.Error("默认应该启用堆栈")
	}
	if cfg.RecoveryFunc != nil {
		t.Error("默认不应该有自定义恢复函数")
	}
}

func TestDefaultAuthConfig(t *testing.T) {
	cfg := DefaultAuthConfig()

	if cfg.AuthScheme != "Bearer" {
		t.Errorf("AuthScheme不正确: %s", cfg.AuthScheme)
	}
	if cfg.MetadataKey != "authorization" {
		t.Errorf("MetadataKey不正确: %s", cfg.MetadataKey)
	}
	if len(cfg.SkipMethods) == 0 {
		t.Error("应该有默认跳过的方法")
	}
}

// === 默认拦截器组合测试 ===

func TestDefaultUnaryInterceptors(t *testing.T) {
	interceptors := DefaultUnaryInterceptors(log.Default())
	if len(interceptors) != 2 {
		t.Errorf("应该有2个拦截器，实际: %d", len(interceptors))
	}
}

func TestDefaultStreamInterceptors(t *testing.T) {
	interceptors := DefaultStreamInterceptors(log.Default())
	if len(interceptors) != 2 {
		t.Errorf("应该有2个拦截器，实际: %d", len(interceptors))
	}
}
