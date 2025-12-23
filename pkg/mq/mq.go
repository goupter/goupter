package mq

import (
	"context"
	"fmt"
	"time"
)

// MessageQueue 消息队列接口
type MessageQueue interface {
	// Publish 发布消息
	Publish(ctx context.Context, topic string, msg *Message) error
	// Subscribe 订阅主题
	Subscribe(topic string, handler Handler) error
	// QueueSubscribe 队列订阅（负载均衡）
	QueueSubscribe(topic, queue string, handler Handler) error
	// Request 请求响应模式
	Request(ctx context.Context, topic string, msg *Message, timeout time.Duration) (*Message, error)
	// Unsubscribe 取消订阅
	Unsubscribe(topic string) error
	// Close 关闭连接
	Close() error
	// Ping 检查连接
	Ping(ctx context.Context) error
}

// Handler 消息处理函数
type Handler func(ctx context.Context, msg *Message) error

// Message 消息结构
type Message struct {
	ID        string            `json:"id"`        // 消息ID
	Topic     string            `json:"topic"`     // 主题
	Payload   []byte            `json:"payload"`   // 消息内容
	Headers   map[string]string `json:"headers"`   // 消息头
	Timestamp time.Time         `json:"timestamp"` // 时间戳
	ReplyTo   string            `json:"reply_to"`  // 回复主题
}

// NewMessage 创建消息
func NewMessage(topic string, payload []byte) *Message {
	return &Message{
		ID:        generateID(),
		Topic:     topic,
		Payload:   payload,
		Headers:   make(map[string]string),
		Timestamp: time.Now(),
	}
}

// SetHeader 设置消息头
func (m *Message) SetHeader(key, value string) *Message {
	if m.Headers == nil {
		m.Headers = make(map[string]string)
	}
	m.Headers[key] = value
	return m
}

// GetHeader 获取消息头
func (m *Message) GetHeader(key string) string {
	if m.Headers == nil {
		return ""
	}
	return m.Headers[key]
}

// Subscription 订阅信息
type Subscription struct {
	Topic   string
	Queue   string
	Handler Handler
}

// generateID 生成消息ID
func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
