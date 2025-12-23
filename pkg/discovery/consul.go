package discovery

import (
	"context"
	"fmt"
	"sync"

	"github.com/goupter/goupter/pkg/config"
	"github.com/goupter/goupter/pkg/log"
	"github.com/hashicorp/consul/api"
)

// ConsulDiscovery Consul服务发现
type ConsulDiscovery struct {
	client   *api.Client
	config   *config.ConsulConfig
	logger   log.Logger
	watchers map[string]context.CancelFunc
	mu       sync.RWMutex
}

// ConsulOption Consul选项
type ConsulOption func(*ConsulDiscovery)

// WithConsulConfig 设置Consul配置
func WithConsulConfig(cfg *config.ConsulConfig) ConsulOption {
	return func(d *ConsulDiscovery) { d.config = cfg }
}

// WithConsulLogger 设置日志
func WithConsulLogger(logger log.Logger) ConsulOption {
	return func(d *ConsulDiscovery) { d.logger = logger }
}

// NewConsulDiscovery 创建Consul服务发现
func NewConsulDiscovery(opts ...ConsulOption) (*ConsulDiscovery, error) {
	d := &ConsulDiscovery{
		config:   &config.ConsulConfig{Host: "localhost", Port: 8500},
		watchers: make(map[string]context.CancelFunc),
	}

	for _, opt := range opts {
		opt(d)
	}

	consulConfig := api.DefaultConfig()
	consulConfig.Address = fmt.Sprintf("%s:%d", d.config.Host, d.config.Port)
	if d.config.Token != "" {
		consulConfig.Token = d.config.Token
	}
	if d.config.Datacenter != "" {
		consulConfig.Datacenter = d.config.Datacenter
	}

	client, err := api.NewClient(consulConfig)
	if err != nil {
		return nil, fmt.Errorf("create consul client failed: %w", err)
	}

	d.client = client
	return d, nil
}

// Register 注册服务
func (d *ConsulDiscovery) Register(ctx context.Context, service *ServiceInfo) error {
	registration := &api.AgentServiceRegistration{
		ID:      service.ID,
		Name:    service.Name,
		Address: service.Address,
		Port:    service.Port,
		Tags:    service.Tags,
		Meta:    service.Metadata,
	}

	if d.config.Service.CheckInterval != "" {
		check := &api.AgentServiceCheck{
			Interval:                       d.config.Service.CheckInterval,
			Timeout:                        d.config.Service.CheckTimeout,
			DeregisterCriticalServiceAfter: d.config.Service.DeregisterCriticalServiceAfter,
		}
		if grpcAddr, ok := service.Metadata["grpc_addr"]; ok {
			check.GRPC = grpcAddr
		} else {
			check.HTTP = fmt.Sprintf("http://%s:%d/health", service.Address, service.Port)
		}
		registration.Check = check
	}

	if err := d.client.Agent().ServiceRegister(registration); err != nil {
		return fmt.Errorf("register service failed: %w", err)
	}

	if d.logger != nil {
		d.logger.Info("service registered",
			log.String("id", service.ID),
			log.String("name", service.Name),
		)
	}
	return nil
}

// Deregister 注销服务
func (d *ConsulDiscovery) Deregister(ctx context.Context, serviceID string) error {
	if err := d.client.Agent().ServiceDeregister(serviceID); err != nil {
		return fmt.Errorf("deregister service failed: %w", err)
	}
	if d.logger != nil {
		d.logger.Info("service deregistered", log.String("id", serviceID))
	}
	return nil
}

// GetService 获取服务实例列表
func (d *ConsulDiscovery) GetService(ctx context.Context, name string) ([]*ServiceInfo, error) {
	services, _, err := d.client.Health().Service(name, "", true, nil)
	if err != nil {
		return nil, fmt.Errorf("get service failed: %w", err)
	}

	result := make([]*ServiceInfo, 0, len(services))
	for _, s := range services {
		result = append(result, &ServiceInfo{
			ID:       s.Service.ID,
			Name:     s.Service.Service,
			Address:  s.Service.Address,
			Port:     s.Service.Port,
			Tags:     s.Service.Tags,
			Metadata: s.Service.Meta,
			Health:   getHealthStatus(s.Checks),
		})
	}
	return result, nil
}

// Watch 监听服务变化
func (d *ConsulDiscovery) Watch(ctx context.Context, name string, callback func([]*ServiceInfo)) error {
	d.mu.Lock()
	if cancel, ok := d.watchers[name]; ok {
		cancel()
	}
	watchCtx, cancel := context.WithCancel(ctx)
	d.watchers[name] = cancel
	d.mu.Unlock()

	go func() {
		var lastIndex uint64
		for {
			select {
			case <-watchCtx.Done():
				return
			default:
				services, meta, err := d.client.Health().Service(name, "", true, &api.QueryOptions{
					WaitIndex: lastIndex,
				})
				if err != nil {
					if d.logger != nil {
						d.logger.Error("watch service failed", log.String("service", name), log.Error(err))
					}
					continue
				}

				if meta.LastIndex > lastIndex {
					lastIndex = meta.LastIndex
					result := make([]*ServiceInfo, 0, len(services))
					for _, s := range services {
						result = append(result, &ServiceInfo{
							ID:       s.Service.ID,
							Name:     s.Service.Service,
							Address:  s.Service.Address,
							Port:     s.Service.Port,
							Tags:     s.Service.Tags,
							Metadata: s.Service.Meta,
							Health:   getHealthStatus(s.Checks),
						})
					}
					callback(result)
				}
			}
		}
	}()
	return nil
}

// Close 关闭连接
func (d *ConsulDiscovery) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, cancel := range d.watchers {
		cancel()
	}
	d.watchers = make(map[string]context.CancelFunc)
	return nil
}

// Client 获取Consul客户端
func (d *ConsulDiscovery) Client() *api.Client {
	return d.client
}

func getHealthStatus(checks api.HealthChecks) HealthStatus {
	for _, check := range checks {
		switch check.Status {
		case api.HealthCritical:
			return HealthStatusCritical
		case api.HealthWarning:
			return HealthStatusWarning
		}
	}
	return HealthStatusPassing
}
