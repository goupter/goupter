package server

import (
	"context"
	"fmt"
	"net"

	"github.com/goupter/goupter/pkg/config"
	"github.com/goupter/goupter/pkg/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

// GRPCServer gRPC服务器
type GRPCServer struct {
	server       *grpc.Server
	config       *config.GRPCConfig
	logger       log.Logger
	healthServer *health.Server
	listener     net.Listener
	opts         []grpc.ServerOption
	unaryInts    []grpc.UnaryServerInterceptor
	streamInts   []grpc.StreamServerInterceptor
}

// GRPCOption gRPC服务器选项
type GRPCOption func(*GRPCServer)

// WithGRPCConfig 设置gRPC配置
func WithGRPCConfig(cfg *config.GRPCConfig) GRPCOption {
	return func(s *GRPCServer) {
		s.config = cfg
	}
}

// WithGRPCLogger 设置日志
func WithGRPCLogger(logger log.Logger) GRPCOption {
	return func(s *GRPCServer) {
		s.logger = logger
	}
}

// WithServerOptions 设置gRPC服务器选项
func WithServerOptions(opts ...grpc.ServerOption) GRPCOption {
	return func(s *GRPCServer) {
		s.opts = append(s.opts, opts...)
	}
}

// WithUnaryInterceptors 添加一元拦截器
func WithUnaryInterceptors(interceptors ...grpc.UnaryServerInterceptor) GRPCOption {
	return func(s *GRPCServer) {
		s.unaryInts = append(s.unaryInts, interceptors...)
	}
}

// WithStreamInterceptors 添加流拦截器
func WithStreamInterceptors(interceptors ...grpc.StreamServerInterceptor) GRPCOption {
	return func(s *GRPCServer) {
		s.streamInts = append(s.streamInts, interceptors...)
	}
}

// NewGRPCServer 创建gRPC服务器
func NewGRPCServer(opts ...GRPCOption) *GRPCServer {
	s := &GRPCServer{
		config: &config.GRPCConfig{
			Host: "0.0.0.0",
			Port: 9090,
		},
		opts:       make([]grpc.ServerOption, 0),
		unaryInts:  make([]grpc.UnaryServerInterceptor, 0),
		streamInts: make([]grpc.StreamServerInterceptor, 0),
	}

	for _, opt := range opts {
		opt(s)
	}

	// 构建服务器选项
	serverOpts := s.opts

	// 添加拦截器
	if len(s.unaryInts) > 0 {
		serverOpts = append(serverOpts, grpc.ChainUnaryInterceptor(s.unaryInts...))
	}
	if len(s.streamInts) > 0 {
		serverOpts = append(serverOpts, grpc.ChainStreamInterceptor(s.streamInts...))
	}

	s.server = grpc.NewServer(serverOpts...)

	// 注册健康检查
	s.healthServer = health.NewServer()
	grpc_health_v1.RegisterHealthServer(s.server, s.healthServer)

	// 注册反射服务（便于调试）
	reflection.Register(s.server)

	return s
}

// Server 获取gRPC服务器
func (s *GRPCServer) Server() *grpc.Server {
	return s.server
}

// RegisterService 注册服务
func (s *GRPCServer) RegisterService(sd *grpc.ServiceDesc, ss interface{}) {
	s.server.RegisterService(sd, ss)
	// 设置服务健康状态
	s.healthServer.SetServingStatus(sd.ServiceName, grpc_health_v1.HealthCheckResponse_SERVING)
}

// Start 启动gRPC服务器
func (s *GRPCServer) Start() error {
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)

	var err error
	s.listener, err = net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("gRPC server listen failed: %w", err)
	}

	if s.logger != nil {
		s.logger.Info("gRPC server starting", log.String("addr", addr))
	}

	return s.server.Serve(s.listener)
}

// Stop 停止gRPC服务器
func (s *GRPCServer) Stop(ctx context.Context) error {
	if s.server == nil {
		return nil
	}

	if s.logger != nil {
		s.logger.Info("gRPC server stopping")
	}

	// 设置所有服务为NOT_SERVING
	s.healthServer.Shutdown()

	// 创建一个channel用于等待优雅关闭
	done := make(chan struct{})
	go func() {
		s.server.GracefulStop()
		close(done)
	}()

	select {
	case <-ctx.Done():
		// 超时，强制关闭
		s.server.Stop()
		return ctx.Err()
	case <-done:
		return nil
	}
}

// Addr 获取服务器地址
func (s *GRPCServer) Addr() string {
	return fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
}

// SetServingStatus 设置服务健康状态
func (s *GRPCServer) SetServingStatus(service string, status grpc_health_v1.HealthCheckResponse_ServingStatus) {
	s.healthServer.SetServingStatus(service, status)
}

// Resume 恢复服务
func (s *GRPCServer) Resume() {
	s.healthServer.Resume()
}
