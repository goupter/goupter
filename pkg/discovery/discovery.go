package discovery

import (
	"context"
)

// Discovery 服务发现接口
type Discovery interface {
	Register(ctx context.Context, service *ServiceInfo) error
	Deregister(ctx context.Context, serviceID string) error
	GetService(ctx context.Context, name string) ([]*ServiceInfo, error)
	Watch(ctx context.Context, name string, callback func([]*ServiceInfo)) error
	Close() error
}

// ServiceInfo 服务信息
type ServiceInfo struct {
	ID       string            `json:"id"`
	Name     string            `json:"name"`
	Address  string            `json:"address"`
	Port     int               `json:"port"`
	Tags     []string          `json:"tags"`
	Metadata map[string]string `json:"metadata"`
	Weight   int               `json:"weight"`
	Health   HealthStatus      `json:"health"`
}

// HealthStatus 健康状态
type HealthStatus string

const (
	HealthStatusPassing  HealthStatus = "passing"
	HealthStatusWarning  HealthStatus = "warning"
	HealthStatusCritical HealthStatus = "critical"
)

// HealthCheck 健康检查配置
type HealthCheck struct {
	Type                           string `json:"type"`
	Endpoint                       string `json:"endpoint"`
	Interval                       string `json:"interval"`
	Timeout                        string `json:"timeout"`
	DeregisterCriticalServiceAfter string `json:"deregister"`
}

// LoadBalancer 负载均衡器
type LoadBalancer interface {
	Select(services []*ServiceInfo) *ServiceInfo
}

// RoundRobinBalancer 轮询负载均衡器
type RoundRobinBalancer struct {
	index int
}

// NewRoundRobinBalancer 创建轮询负载均衡器
func NewRoundRobinBalancer() *RoundRobinBalancer {
	return &RoundRobinBalancer{}
}

// Select 轮询选择
func (b *RoundRobinBalancer) Select(services []*ServiceInfo) *ServiceInfo {
	if len(services) == 0 {
		return nil
	}
	service := services[b.index%len(services)]
	b.index++
	return service
}
