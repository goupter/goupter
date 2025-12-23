package mq

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/goupter/goupter/pkg/config"
	"github.com/goupter/goupter/pkg/log"
	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
)

// NATSClient NATS消息队列客户端
type NATSClient struct {
	conn          *nats.Conn
	config        *config.NATSConfig
	logger        log.Logger
	subscriptions map[string]*nats.Subscription
	mu            sync.RWMutex
}

// NATSOption NATS选项
type NATSOption func(*NATSClient)

// WithNATSConfig 设置NATS配置
func WithNATSConfig(cfg *config.NATSConfig) NATSOption {
	return func(c *NATSClient) {
		c.config = cfg
	}
}

// WithNATSLogger 设置日志
func WithNATSLogger(logger log.Logger) NATSOption {
	return func(c *NATSClient) {
		c.logger = logger
	}
}

// NewNATSClient 创建NATS客户端
func NewNATSClient(opts ...NATSOption) (*NATSClient, error) {
	c := &NATSClient{
		config: &config.NATSConfig{
			URL:            "nats://localhost:4222",
			MaxReconnects:  10,
			ReconnectWait:  2 * time.Second,
			ConnectTimeout: 5 * time.Second,
		},
		subscriptions: make(map[string]*nats.Subscription),
	}

	for _, opt := range opts {
		opt(c)
	}

	// 连接选项
	natsOpts := []nats.Option{
		nats.MaxReconnects(c.config.MaxReconnects),
		nats.ReconnectWait(c.config.ReconnectWait),
		nats.Timeout(c.config.ConnectTimeout),
	}

	// 设置重连回调
	if c.logger != nil {
		natsOpts = append(natsOpts,
			nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
				c.logger.Warn("NATS disconnected", log.Error(err))
			}),
			nats.ReconnectHandler(func(nc *nats.Conn) {
				c.logger.Info("NATS reconnected", log.String("url", nc.ConnectedUrl()))
			}),
			nats.ClosedHandler(func(nc *nats.Conn) {
				c.logger.Info("NATS connection closed")
			}),
		)
	}

	// 连接NATS
	conn, err := nats.Connect(c.config.URL, natsOpts...)
	if err != nil {
		return nil, fmt.Errorf("连接NATS失败: %w", err)
	}

	c.conn = conn

	if c.logger != nil {
		c.logger.Info("NATS connected", log.String("url", c.config.URL))
	}

	return c, nil
}

// Publish 发布消息
func (c *NATSClient) Publish(ctx context.Context, topic string, msg *Message) error {
	if msg.ID == "" {
		msg.ID = uuid.New().String()
	}
	if msg.Timestamp.IsZero() {
		msg.Timestamp = time.Now()
	}
	msg.Topic = topic

	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("序列化消息失败: %w", err)
	}

	if err := c.conn.Publish(topic, data); err != nil {
		return fmt.Errorf("发布消息失败: %w", err)
	}

	return nil
}

// PublishRaw 发布原始数据
func (c *NATSClient) PublishRaw(ctx context.Context, topic string, data []byte) error {
	return c.conn.Publish(topic, data)
}

// Subscribe 订阅主题
func (c *NATSClient) Subscribe(topic string, handler Handler) error {
	sub, err := c.conn.Subscribe(topic, func(msg *nats.Msg) {
		var m Message
		if err := json.Unmarshal(msg.Data, &m); err != nil {
			// 如果不是JSON格式，使用原始数据
			m = Message{
				Topic:     msg.Subject,
				Payload:   msg.Data,
				Timestamp: time.Now(),
			}
		}

		ctx := context.Background()
		if err := handler(ctx, &m); err != nil {
			if c.logger != nil {
				c.logger.Error("处理消息失败",
					log.String("topic", topic),
					log.Error(err),
				)
			}
		}
	})

	if err != nil {
		return fmt.Errorf("订阅主题失败: %w", err)
	}

	c.mu.Lock()
	c.subscriptions[topic] = sub
	c.mu.Unlock()

	if c.logger != nil {
		c.logger.Info("订阅主题成功", log.String("topic", topic))
	}

	return nil
}

// QueueSubscribe 队列订阅
func (c *NATSClient) QueueSubscribe(topic, queue string, handler Handler) error {
	sub, err := c.conn.QueueSubscribe(topic, queue, func(msg *nats.Msg) {
		var m Message
		if err := json.Unmarshal(msg.Data, &m); err != nil {
			m = Message{
				Topic:     msg.Subject,
				Payload:   msg.Data,
				Timestamp: time.Now(),
			}
		}

		ctx := context.Background()
		if err := handler(ctx, &m); err != nil {
			if c.logger != nil {
				c.logger.Error("处理消息失败",
					log.String("topic", topic),
					log.String("queue", queue),
					log.Error(err),
				)
			}
		}
	})

	if err != nil {
		return fmt.Errorf("队列订阅失败: %w", err)
	}

	key := topic + ":" + queue
	c.mu.Lock()
	c.subscriptions[key] = sub
	c.mu.Unlock()

	if c.logger != nil {
		c.logger.Info("队列订阅成功",
			log.String("topic", topic),
			log.String("queue", queue),
		)
	}

	return nil
}

// Request 请求响应模式
func (c *NATSClient) Request(ctx context.Context, topic string, msg *Message, timeout time.Duration) (*Message, error) {
	if msg.ID == "" {
		msg.ID = uuid.New().String()
	}
	if msg.Timestamp.IsZero() {
		msg.Timestamp = time.Now()
	}
	msg.Topic = topic

	data, err := json.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("序列化消息失败: %w", err)
	}

	resp, err := c.conn.Request(topic, data, timeout)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}

	var response Message
	if err := json.Unmarshal(resp.Data, &response); err != nil {
		response = Message{
			Topic:     resp.Subject,
			Payload:   resp.Data,
			Timestamp: time.Now(),
		}
	}

	return &response, nil
}

// Unsubscribe 取消订阅
func (c *NATSClient) Unsubscribe(topic string) error {
	c.mu.Lock()
	sub, ok := c.subscriptions[topic]
	if ok {
		delete(c.subscriptions, topic)
	}
	c.mu.Unlock()

	if !ok {
		return nil
	}

	if err := sub.Unsubscribe(); err != nil {
		return fmt.Errorf("取消订阅失败: %w", err)
	}

	if c.logger != nil {
		c.logger.Info("取消订阅成功", log.String("topic", topic))
	}

	return nil
}

// Close 关闭连接
func (c *NATSClient) Close() error {
	c.mu.Lock()
	for _, sub := range c.subscriptions {
		_ = sub.Unsubscribe()
	}
	c.subscriptions = make(map[string]*nats.Subscription)
	c.mu.Unlock()

	c.conn.Close()

	if c.logger != nil {
		c.logger.Info("NATS connection closed")
	}

	return nil
}

// Ping 检查连接
func (c *NATSClient) Ping(ctx context.Context) error {
	if !c.conn.IsConnected() {
		return fmt.Errorf("NATS not connected")
	}
	return nil
}

// Conn 获取原始NATS连接
func (c *NATSClient) Conn() *nats.Conn {
	return c.conn
}

// Flush 刷新缓冲
func (c *NATSClient) Flush() error {
	return c.conn.Flush()
}

// FlushTimeout 带超时的刷新
func (c *NATSClient) FlushTimeout(timeout time.Duration) error {
	return c.conn.FlushTimeout(timeout)
}
