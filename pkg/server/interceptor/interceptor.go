package interceptor

import (
	"context"
	"runtime/debug"
	"time"

	"github.com/goupter/goupter/pkg/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

// === 日志拦截器 ===

// LoggingConfig 日志拦截器配置
type LoggingConfig struct {
	Logger           log.Logger
	SlowThreshold    time.Duration // 慢请求阈值
	SkipMethods      []string      // 跳过的方法
	LogPayload       bool          // 是否记录请求/响应内容
	MaxPayloadLength int           // 最大记录长度
}

// DefaultLoggingConfig 默认日志配置
func DefaultLoggingConfig() LoggingConfig {
	return LoggingConfig{
		Logger:           log.Default(),
		SlowThreshold:    200 * time.Millisecond,
		SkipMethods:      []string{"/grpc.health.v1.Health/Check", "/grpc.health.v1.Health/Watch"},
		LogPayload:       false,
		MaxPayloadLength: 1024,
	}
}

// UnaryLogging 一元调用日志拦截器
func UnaryLogging(cfg LoggingConfig) grpc.UnaryServerInterceptor {
	skipMethods := make(map[string]bool)
	for _, m := range cfg.SkipMethods {
		skipMethods[m] = true
	}

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// 跳过指定方法
		if skipMethods[info.FullMethod] {
			return handler(ctx, req)
		}

		start := time.Now()

		// 获取客户端信息
		clientIP := getClientIP(ctx)

		// 执行处理
		resp, err := handler(ctx, req)

		// 计算耗时
		duration := time.Since(start)

		// 构建日志字段
		fields := []log.Field{
			log.String("method", info.FullMethod),
			log.String("client_ip", clientIP),
			log.Duration("duration", duration),
		}

		// 添加状态码
		code := codes.OK
		if err != nil {
			code = status.Code(err)
			fields = append(fields, log.String("error", err.Error()))
		}
		fields = append(fields, log.String("code", code.String()))

		// 记录请求/响应内容
		if cfg.LogPayload {
			fields = append(fields, log.Any("request", truncatePayload(req, cfg.MaxPayloadLength)))
			if resp != nil {
				fields = append(fields, log.Any("response", truncatePayload(resp, cfg.MaxPayloadLength)))
			}
		}

		// 根据情况选择日志级别
		switch {
		case err != nil:
			cfg.Logger.Error("gRPC request failed", fields...)
		case duration > cfg.SlowThreshold:
			cfg.Logger.Warn("gRPC slow request", fields...)
		default:
			cfg.Logger.Info("gRPC request", fields...)
		}

		return resp, err
	}
}

// StreamLogging 流式调用日志拦截器
func StreamLogging(cfg LoggingConfig) grpc.StreamServerInterceptor {
	skipMethods := make(map[string]bool)
	for _, m := range cfg.SkipMethods {
		skipMethods[m] = true
	}

	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		// 跳过指定方法
		if skipMethods[info.FullMethod] {
			return handler(srv, ss)
		}

		start := time.Now()
		clientIP := getClientIP(ss.Context())

		err := handler(srv, ss)

		duration := time.Since(start)

		fields := []log.Field{
			log.String("method", info.FullMethod),
			log.String("client_ip", clientIP),
			log.Duration("duration", duration),
			log.Bool("client_stream", info.IsClientStream),
			log.Bool("server_stream", info.IsServerStream),
		}

		code := codes.OK
		if err != nil {
			code = status.Code(err)
			fields = append(fields, log.String("error", err.Error()))
		}
		fields = append(fields, log.String("code", code.String()))

		switch {
		case err != nil:
			cfg.Logger.Error("gRPC stream failed", fields...)
		case duration > cfg.SlowThreshold:
			cfg.Logger.Warn("gRPC slow stream", fields...)
		default:
			cfg.Logger.Info("gRPC stream", fields...)
		}

		return err
	}
}

// === 恢复拦截器 ===

// RecoveryConfig 恢复拦截器配置
type RecoveryConfig struct {
	Logger         log.Logger
	EnableStack    bool
	RecoveryFunc   func(ctx context.Context, p interface{}) error
	StackSkipCount int
}

// DefaultRecoveryConfig 默认恢复配置
func DefaultRecoveryConfig() RecoveryConfig {
	return RecoveryConfig{
		Logger:         log.Default(),
		EnableStack:    true,
		StackSkipCount: 3,
	}
}

// UnaryRecovery 一元调用恢复拦截器
func UnaryRecovery(cfg RecoveryConfig) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		defer func() {
			if p := recover(); p != nil {
				err = handlePanic(ctx, p, info.FullMethod, cfg)
			}
		}()
		return handler(ctx, req)
	}
}

// StreamRecovery 流式调用恢复拦截器
func StreamRecovery(cfg RecoveryConfig) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
		defer func() {
			if p := recover(); p != nil {
				err = handlePanic(ss.Context(), p, info.FullMethod, cfg)
			}
		}()
		return handler(srv, ss)
	}
}

func handlePanic(ctx context.Context, p interface{}, method string, cfg RecoveryConfig) error {
	fields := []log.Field{
		log.String("method", method),
		log.Any("panic", p),
	}

	if cfg.EnableStack {
		fields = append(fields, log.String("stack", string(debug.Stack())))
	}

	cfg.Logger.Error("gRPC panic recovered", fields...)

	if cfg.RecoveryFunc != nil {
		return cfg.RecoveryFunc(ctx, p)
	}

	return status.Errorf(codes.Internal, "internal server error")
}

// === 认证拦截器 ===

// AuthConfig 认证拦截器配置
type AuthConfig struct {
	// TokenValidator 验证Token并返回用户信息
	TokenValidator func(ctx context.Context, token string) (userID string, err error)
	// SkipMethods 跳过认证的方法
	SkipMethods []string
	// AuthScheme 认证方案（默认Bearer）
	AuthScheme string
	// MetadataKey 元数据键（默认authorization）
	MetadataKey string
}

// DefaultAuthConfig 默认认证配置
func DefaultAuthConfig() AuthConfig {
	return AuthConfig{
		AuthScheme:  "Bearer",
		MetadataKey: "authorization",
		SkipMethods: []string{
			"/grpc.health.v1.Health/Check",
			"/grpc.health.v1.Health/Watch",
			"/grpc.reflection.v1alpha.ServerReflection/ServerReflectionInfo",
		},
	}
}

// 上下文键类型
type userIDKey struct{}

// UnaryAuth 一元调用认证拦截器
func UnaryAuth(cfg AuthConfig) grpc.UnaryServerInterceptor {
	if cfg.MetadataKey == "" {
		cfg.MetadataKey = "authorization"
	}
	if cfg.AuthScheme == "" {
		cfg.AuthScheme = "Bearer"
	}

	skipMethods := make(map[string]bool)
	for _, m := range cfg.SkipMethods {
		skipMethods[m] = true
	}

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// 跳过指定方法
		if skipMethods[info.FullMethod] {
			return handler(ctx, req)
		}

		// 提取Token
		token, err := extractToken(ctx, cfg.MetadataKey, cfg.AuthScheme)
		if err != nil {
			return nil, err
		}

		// 验证Token
		if cfg.TokenValidator == nil {
			return nil, status.Error(codes.Internal, "token validator not configured")
		}

		userID, err := cfg.TokenValidator(ctx, token)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, err.Error())
		}

		// 将用户ID添加到上下文
		ctx = context.WithValue(ctx, userIDKey{}, userID)
		return handler(ctx, req)
	}
}

// StreamAuth 流式调用认证拦截器
func StreamAuth(cfg AuthConfig) grpc.StreamServerInterceptor {
	if cfg.MetadataKey == "" {
		cfg.MetadataKey = "authorization"
	}
	if cfg.AuthScheme == "" {
		cfg.AuthScheme = "Bearer"
	}

	skipMethods := make(map[string]bool)
	for _, m := range cfg.SkipMethods {
		skipMethods[m] = true
	}

	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		// 跳过指定方法
		if skipMethods[info.FullMethod] {
			return handler(srv, ss)
		}

		ctx := ss.Context()

		// 提取Token
		token, err := extractToken(ctx, cfg.MetadataKey, cfg.AuthScheme)
		if err != nil {
			return err
		}

		// 验证Token
		if cfg.TokenValidator == nil {
			return status.Error(codes.Internal, "token validator not configured")
		}

		userID, err := cfg.TokenValidator(ctx, token)
		if err != nil {
			return status.Error(codes.Unauthenticated, err.Error())
		}

		// 使用包装的Stream传递认证信息
		wrapped := &wrappedServerStream{
			ServerStream: ss,
			ctx:          context.WithValue(ctx, userIDKey{}, userID),
		}

		return handler(srv, wrapped)
	}
}

// wrappedServerStream 包装的ServerStream
type wrappedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedServerStream) Context() context.Context {
	return w.ctx
}

// extractToken 从上下文提取Token
func extractToken(ctx context.Context, metadataKey, scheme string) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", status.Error(codes.Unauthenticated, "missing metadata")
	}

	values := md.Get(metadataKey)
	if len(values) == 0 {
		return "", status.Error(codes.Unauthenticated, "missing authorization token")
	}

	auth := values[0]
	prefix := scheme + " "
	if len(auth) < len(prefix) {
		return "", status.Error(codes.Unauthenticated, "invalid authorization format")
	}

	return auth[len(prefix):], nil
}

// GetUserID 从上下文获取用户ID
func GetUserID(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(userIDKey{}).(string)
	return userID, ok
}

// === 超时拦截器 ===

// UnaryTimeout 一元调用超时拦截器
func UnaryTimeout(timeout time.Duration) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		return handler(ctx, req)
	}
}

// === 限流拦截器 ===

// RateLimiter 限流器接口
type RateLimiter interface {
	Allow() bool
}

// UnaryRateLimit 一元调用限流拦截器
func UnaryRateLimit(limiter RateLimiter) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if !limiter.Allow() {
			return nil, status.Error(codes.ResourceExhausted, "rate limit exceeded")
		}
		return handler(ctx, req)
	}
}

// StreamRateLimit 流式调用限流拦截器
func StreamRateLimit(limiter RateLimiter) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if !limiter.Allow() {
			return status.Error(codes.ResourceExhausted, "rate limit exceeded")
		}
		return handler(srv, ss)
	}
}

// === 请求ID拦截器 ===

type requestIDKey struct{}

// UnaryRequestID 一元调用请求ID拦截器
func UnaryRequestID(generator func() string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		requestID := ""

		// 尝试从metadata获取
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			if values := md.Get("x-request-id"); len(values) > 0 {
				requestID = values[0]
			}
		}

		// 如果没有则生成
		if requestID == "" && generator != nil {
			requestID = generator()
		}

		ctx = context.WithValue(ctx, requestIDKey{}, requestID)

		// 添加到响应metadata
		if err := grpc.SetHeader(ctx, metadata.Pairs("x-request-id", requestID)); err != nil {
			// 忽略错误
		}

		return handler(ctx, req)
	}
}

// StreamRequestID 流式调用请求ID拦截器
func StreamRequestID(generator func() string) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx := ss.Context()
		requestID := ""

		// 尝试从metadata获取
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			if values := md.Get("x-request-id"); len(values) > 0 {
				requestID = values[0]
			}
		}

		// 如果没有则生成
		if requestID == "" && generator != nil {
			requestID = generator()
		}

		ctx = context.WithValue(ctx, requestIDKey{}, requestID)

		// 添加到响应metadata
		if err := ss.SetHeader(metadata.Pairs("x-request-id", requestID)); err != nil {
			// 忽略错误
		}

		wrapped := &wrappedServerStream{
			ServerStream: ss,
			ctx:          ctx,
		}

		return handler(srv, wrapped)
	}
}

// GetRequestID 从上下文获取请求ID
func GetRequestID(ctx context.Context) (string, bool) {
	requestID, ok := ctx.Value(requestIDKey{}).(string)
	return requestID, ok
}

// === 辅助函数 ===

// getClientIP 获取客户端IP
func getClientIP(ctx context.Context) string {
	// 尝试从metadata获取
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		// X-Forwarded-For
		if values := md.Get("x-forwarded-for"); len(values) > 0 {
			return values[0]
		}
		// X-Real-IP
		if values := md.Get("x-real-ip"); len(values) > 0 {
			return values[0]
		}
	}

	// 从peer获取
	if p, ok := peer.FromContext(ctx); ok {
		return p.Addr.String()
	}

	return ""
}

// truncatePayload 截断负载
func truncatePayload(payload interface{}, maxLength int) interface{} {
	// 简单实现，实际可以根据需要处理
	return payload
}

// === 拦截器链 ===

// ChainUnaryInterceptors 链接多个一元拦截器
func ChainUnaryInterceptors(interceptors ...grpc.UnaryServerInterceptor) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		chain := handler
		for i := len(interceptors) - 1; i >= 0; i-- {
			interceptor := interceptors[i]
			next := chain
			chain = func(ctx context.Context, req interface{}) (interface{}, error) {
				return interceptor(ctx, req, info, next)
			}
		}
		return chain(ctx, req)
	}
}

// ChainStreamInterceptors 链接多个流拦截器
func ChainStreamInterceptors(interceptors ...grpc.StreamServerInterceptor) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		chain := handler
		for i := len(interceptors) - 1; i >= 0; i-- {
			interceptor := interceptors[i]
			next := chain
			chain = func(srv interface{}, ss grpc.ServerStream) error {
				return interceptor(srv, ss, info, next)
			}
		}
		return chain(srv, ss)
	}
}

// === 默认拦截器组合 ===

// DefaultUnaryInterceptors 默认一元拦截器组合
func DefaultUnaryInterceptors(logger log.Logger) []grpc.UnaryServerInterceptor {
	recoveryCfg := DefaultRecoveryConfig()
	recoveryCfg.Logger = logger

	loggingCfg := DefaultLoggingConfig()
	loggingCfg.Logger = logger

	return []grpc.UnaryServerInterceptor{
		UnaryRecovery(recoveryCfg),
		UnaryLogging(loggingCfg),
	}
}

// DefaultStreamInterceptors 默认流拦截器组合
func DefaultStreamInterceptors(logger log.Logger) []grpc.StreamServerInterceptor {
	recoveryCfg := DefaultRecoveryConfig()
	recoveryCfg.Logger = logger

	loggingCfg := DefaultLoggingConfig()
	loggingCfg.Logger = logger

	return []grpc.StreamServerInterceptor{
		StreamRecovery(recoveryCfg),
		StreamLogging(loggingCfg),
	}
}
