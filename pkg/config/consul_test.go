package config

import (
	"testing"
)

func TestConsulSourceOption(t *testing.T) {
	t.Run("WithWatchCallback", func(t *testing.T) {
		called := false
		callback := func(data []byte) {
			called = true
		}

		s := &ConsulSource{}
		opt := WithWatchCallback(callback)
		opt(s)

		if s.onChangeFn == nil {
			t.Error("onChangeFn should be set")
		}

		// 调用回调验证
		s.onChangeFn([]byte("test"))
		if !called {
			t.Error("callback should be called")
		}
	})
}

func TestConsulSource_StopWatch(t *testing.T) {
	s := &ConsulSource{
		watchStop: make(chan struct{}),
	}

	// StopWatch 应该关闭 channel
	s.StopWatch()

	// 验证 channel 被关闭
	select {
	case _, ok := <-s.watchStop:
		if ok {
			t.Error("watchStop channel should be closed")
		}
		// channel 已关闭，预期行为
	default:
		t.Error("watchStop channel should be closed and readable")
	}
}

func TestConsulSource_Client(t *testing.T) {
	// 由于需要真实的 Consul 连接，这里只测试 nil 情况
	s := &ConsulSource{
		client: nil,
	}

	if s.Client() != nil {
		t.Error("Client() should return nil when not set")
	}
}

func TestConsulServiceConfig(t *testing.T) {
	svc := ConsulServiceConfig{
		ID:                             "service-1",
		Name:                           "my-service",
		Tags:                           []string{"api", "v1", "http"},
		CheckInterval:                  "10s",
		CheckTimeout:                   "5s",
		DeregisterCriticalServiceAfter: "30s",
	}

	if svc.ID != "service-1" {
		t.Errorf("ID = %s, want service-1", svc.ID)
	}

	if svc.Name != "my-service" {
		t.Errorf("Name = %s, want my-service", svc.Name)
	}

	if len(svc.Tags) != 3 {
		t.Errorf("Tags length = %d, want 3", len(svc.Tags))
	}

	if svc.CheckInterval != "10s" {
		t.Errorf("CheckInterval = %s, want 10s", svc.CheckInterval)
	}

	if svc.CheckTimeout != "5s" {
		t.Errorf("CheckTimeout = %s, want 5s", svc.CheckTimeout)
	}

	if svc.DeregisterCriticalServiceAfter != "30s" {
		t.Errorf("DeregisterCriticalServiceAfter = %s, want 30s", svc.DeregisterCriticalServiceAfter)
	}
}

func TestConsulConfig_Integration(t *testing.T) {
	// 测试 ConsulConfig 与 Config 的集成
	cfg := &Config{
		Consul: ConsulConfig{
			Host:       "consul.example.com",
			Port:       8500,
			Token:      "secret-token",
			Datacenter: "dc1",
			Service: ConsulServiceConfig{
				ID:            "svc-001",
				Name:          "api-service",
				Tags:          []string{"http", "api"},
				CheckInterval: "15s",
			},
			ConfigPath: "config/api-service/config",
		},
	}

	// 验证地址生成
	addr := cfg.ConsulAddr()
	if addr != "consul.example.com:8500" {
		t.Errorf("ConsulAddr() = %s, want consul.example.com:8500", addr)
	}

	// 验证配置结构
	if cfg.Consul.Token != "secret-token" {
		t.Errorf("Consul.Token = %s, want secret-token", cfg.Consul.Token)
	}

	if cfg.Consul.ConfigPath != "config/api-service/config" {
		t.Errorf("Consul.ConfigPath = %s, want config/api-service/config", cfg.Consul.ConfigPath)
	}
}

func TestConsulSource_Fields(t *testing.T) {
	// 测试 ConsulSource 字段
	s := &ConsulSource{
		path:      "config/myapp",
		lastIndex: 100,
	}

	if s.path != "config/myapp" {
		t.Errorf("path = %s, want config/myapp", s.path)
	}

	s.mu.Lock()
	s.lastIndex = 200
	s.mu.Unlock()

	s.mu.RLock()
	idx := s.lastIndex
	s.mu.RUnlock()

	if idx != 200 {
		t.Errorf("lastIndex = %d, want 200", idx)
	}
}
