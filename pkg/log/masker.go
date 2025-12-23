package log

import (
	"context"
	"regexp"
	"strings"
	"sync"
)

// SensitiveType 敏感信息类型
type SensitiveType int

const (
	// SensitivePhone 手机号
	SensitivePhone SensitiveType = iota
	// SensitiveIDCard 身份证
	SensitiveIDCard
	// SensitiveEmail 邮箱
	SensitiveEmail
	// SensitiveBankCard 银行卡
	SensitiveBankCard
	// SensitivePassword 密码
	SensitivePassword
	// SensitiveToken Token
	SensitiveToken
	// SensitiveCustom 自定义
	SensitiveCustom
)

// MaskRule 脱敏规则
type MaskRule struct {
	Type     SensitiveType
	Pattern  *regexp.Regexp
	MaskFunc func(string) string
}

// Masker 脱敏器接口
type Masker interface {
	// Mask 对字符串进行脱敏
	Mask(s string) string
	// MaskField 对字段值进行脱敏
	MaskField(key string, value interface{}) interface{}
}

// === 默认脱敏器 ===

// DefaultMasker 默认脱敏器
type DefaultMasker struct {
	mu             sync.RWMutex
	rules          []MaskRule
	sensitiveKeys  map[string]SensitiveType
	maskChar       string
}

// MaskerOption 脱敏器选项
type MaskerOption func(*DefaultMasker)

// WithMaskChar 设置脱敏字符
func WithMaskChar(char string) MaskerOption {
	return func(m *DefaultMasker) {
		m.maskChar = char
	}
}

// WithSensitiveKeys 设置敏感字段
func WithSensitiveKeys(keys map[string]SensitiveType) MaskerOption {
	return func(m *DefaultMasker) {
		for k, v := range keys {
			m.sensitiveKeys[strings.ToLower(k)] = v
		}
	}
}

// NewDefaultMasker 创建默认脱敏器
func NewDefaultMasker(opts ...MaskerOption) *DefaultMasker {
	m := &DefaultMasker{
		maskChar:      "*",
		sensitiveKeys: make(map[string]SensitiveType),
	}

	// 默认敏感字段
	m.sensitiveKeys["password"] = SensitivePassword
	m.sensitiveKeys["passwd"] = SensitivePassword
	m.sensitiveKeys["pwd"] = SensitivePassword
	m.sensitiveKeys["secret"] = SensitivePassword
	m.sensitiveKeys["token"] = SensitiveToken
	m.sensitiveKeys["access_token"] = SensitiveToken
	m.sensitiveKeys["refresh_token"] = SensitiveToken
	m.sensitiveKeys["api_key"] = SensitiveToken
	m.sensitiveKeys["apikey"] = SensitiveToken
	m.sensitiveKeys["phone"] = SensitivePhone
	m.sensitiveKeys["mobile"] = SensitivePhone
	m.sensitiveKeys["tel"] = SensitivePhone
	m.sensitiveKeys["id_card"] = SensitiveIDCard
	m.sensitiveKeys["idcard"] = SensitiveIDCard
	m.sensitiveKeys["identity"] = SensitiveIDCard
	m.sensitiveKeys["email"] = SensitiveEmail
	m.sensitiveKeys["mail"] = SensitiveEmail
	m.sensitiveKeys["bank_card"] = SensitiveBankCard
	m.sensitiveKeys["bankcard"] = SensitiveBankCard
	m.sensitiveKeys["card_no"] = SensitiveBankCard

	// 初始化规则
	m.initRules()

	for _, opt := range opts {
		opt(m)
	}

	return m
}

// initRules 初始化脱敏规则
func (m *DefaultMasker) initRules() {
	// 手机号规则：保留前3后4
	m.rules = append(m.rules, MaskRule{
		Type:    SensitivePhone,
		Pattern: regexp.MustCompile(`1[3-9]\d{9}`),
		MaskFunc: func(s string) string {
			if len(s) != 11 {
				return m.maskAll(s)
			}
			return s[:3] + strings.Repeat(m.maskChar, 4) + s[7:]
		},
	})

	// 身份证规则：保留前6后4
	m.rules = append(m.rules, MaskRule{
		Type:    SensitiveIDCard,
		Pattern: regexp.MustCompile(`\d{17}[\dXx]|\d{15}`),
		MaskFunc: func(s string) string {
			if len(s) < 10 {
				return m.maskAll(s)
			}
			return s[:6] + strings.Repeat(m.maskChar, len(s)-10) + s[len(s)-4:]
		},
	})

	// 邮箱规则：保留首字母和@后域名
	m.rules = append(m.rules, MaskRule{
		Type:    SensitiveEmail,
		Pattern: regexp.MustCompile(`[\w.-]+@[\w.-]+\.\w+`),
		MaskFunc: func(s string) string {
			atIndex := strings.Index(s, "@")
			if atIndex <= 0 {
				return m.maskAll(s)
			}
			masked := string(s[0]) + strings.Repeat(m.maskChar, atIndex-1) + s[atIndex:]
			return masked
		},
	})

	// 银行卡规则：保留前6后4
	m.rules = append(m.rules, MaskRule{
		Type:    SensitiveBankCard,
		Pattern: regexp.MustCompile(`\d{16,19}`),
		MaskFunc: func(s string) string {
			if len(s) < 10 {
				return m.maskAll(s)
			}
			return s[:6] + strings.Repeat(m.maskChar, len(s)-10) + s[len(s)-4:]
		},
	})
}

// maskAll 全部脱敏
func (m *DefaultMasker) maskAll(s string) string {
	return strings.Repeat(m.maskChar, len(s))
}

// Mask 对字符串进行脱敏
func (m *DefaultMasker) Mask(s string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := s
	for _, rule := range m.rules {
		result = rule.Pattern.ReplaceAllStringFunc(result, rule.MaskFunc)
	}
	return result
}

// MaskField 对字段值进行脱敏
func (m *DefaultMasker) MaskField(key string, value interface{}) interface{} {
	m.mu.RLock()
	sensitiveType, isSensitive := m.sensitiveKeys[strings.ToLower(key)]
	m.mu.RUnlock()

	if !isSensitive {
		return value
	}

	str, ok := value.(string)
	if !ok {
		return value
	}

	switch sensitiveType {
	case SensitivePassword, SensitiveToken:
		// 密码和Token完全脱敏
		return strings.Repeat(m.maskChar, 8)
	case SensitivePhone:
		return m.maskPhone(str)
	case SensitiveIDCard:
		return m.maskIDCard(str)
	case SensitiveEmail:
		return m.maskEmail(str)
	case SensitiveBankCard:
		return m.maskBankCard(str)
	default:
		return m.maskAll(str)
	}
}

// maskPhone 手机号脱敏
func (m *DefaultMasker) maskPhone(s string) string {
	if len(s) != 11 {
		return m.maskAll(s)
	}
	return s[:3] + strings.Repeat(m.maskChar, 4) + s[7:]
}

// maskIDCard 身份证脱敏
func (m *DefaultMasker) maskIDCard(s string) string {
	if len(s) < 10 {
		return m.maskAll(s)
	}
	return s[:6] + strings.Repeat(m.maskChar, len(s)-10) + s[len(s)-4:]
}

// maskEmail 邮箱脱敏
func (m *DefaultMasker) maskEmail(s string) string {
	atIndex := strings.Index(s, "@")
	if atIndex <= 0 {
		return m.maskAll(s)
	}
	return string(s[0]) + strings.Repeat(m.maskChar, atIndex-1) + s[atIndex:]
}

// maskBankCard 银行卡脱敏
func (m *DefaultMasker) maskBankCard(s string) string {
	if len(s) < 10 {
		return m.maskAll(s)
	}
	return s[:6] + strings.Repeat(m.maskChar, len(s)-10) + s[len(s)-4:]
}

// AddRule 添加自定义规则
func (m *DefaultMasker) AddRule(pattern string, maskFunc func(string) string) error {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return err
	}

	m.mu.Lock()
	m.rules = append(m.rules, MaskRule{
		Type:     SensitiveCustom,
		Pattern:  re,
		MaskFunc: maskFunc,
	})
	m.mu.Unlock()

	return nil
}

// AddSensitiveKey 添加敏感字段
func (m *DefaultMasker) AddSensitiveKey(key string, sensitiveType SensitiveType) {
	m.mu.Lock()
	m.sensitiveKeys[strings.ToLower(key)] = sensitiveType
	m.mu.Unlock()
}

// === 带脱敏的Logger ===

// MaskedLogger 带脱敏功能的Logger
type MaskedLogger struct {
	logger Logger
	masker Masker
}

// NewMaskedLogger 创建带脱敏的Logger
func NewMaskedLogger(logger Logger, masker Masker) *MaskedLogger {
	if masker == nil {
		masker = NewDefaultMasker()
	}
	return &MaskedLogger{
		logger: logger,
		masker: masker,
	}
}

// maskFields 对字段进行脱敏
func (l *MaskedLogger) maskFields(fields []Field) []Field {
	masked := make([]Field, len(fields))
	for i, f := range fields {
		masked[i] = Field{
			Key:   f.Key,
			Value: l.masker.MaskField(f.Key, f.Value),
		}
	}
	return masked
}

func (l *MaskedLogger) Debug(msg string, fields ...Field) {
	l.logger.Debug(l.masker.Mask(msg), l.maskFields(fields)...)
}

func (l *MaskedLogger) Info(msg string, fields ...Field) {
	l.logger.Info(l.masker.Mask(msg), l.maskFields(fields)...)
}

func (l *MaskedLogger) Warn(msg string, fields ...Field) {
	l.logger.Warn(l.masker.Mask(msg), l.maskFields(fields)...)
}

func (l *MaskedLogger) Error(msg string, fields ...Field) {
	l.logger.Error(l.masker.Mask(msg), l.maskFields(fields)...)
}

func (l *MaskedLogger) Fatal(msg string, fields ...Field) {
	l.logger.Fatal(l.masker.Mask(msg), l.maskFields(fields)...)
}

func (l *MaskedLogger) With(fields ...Field) Logger {
	return &MaskedLogger{
		logger: l.logger.With(l.maskFields(fields)...),
		masker: l.masker,
	}
}

func (l *MaskedLogger) WithContext(ctx context.Context) Logger {
	return &MaskedLogger{
		logger: l.logger.WithContext(ctx),
		masker: l.masker,
	}
}

func (l *MaskedLogger) SetLevel(level Level) {
	l.logger.SetLevel(level)
}

func (l *MaskedLogger) GetLevel() Level {
	return l.logger.GetLevel()
}

func (l *MaskedLogger) Sync() error {
	return l.logger.Sync()
}

// Unwrap 获取底层Logger
func (l *MaskedLogger) Unwrap() Logger {
	return l.logger
}

// === 全局默认脱敏器 ===

var globalMasker Masker = NewDefaultMasker()

// SetGlobalMasker 设置全局脱敏器
func SetGlobalMasker(masker Masker) {
	globalMasker = masker
}

// GetGlobalMasker 获取全局脱敏器
func GetGlobalMasker() Masker {
	return globalMasker
}

// MaskString 使用全局脱敏器脱敏字符串
func MaskString(s string) string {
	return globalMasker.Mask(s)
}

// MaskValue 使用全局脱敏器脱敏字段值
func MaskValue(key string, value interface{}) interface{} {
	return globalMasker.MaskField(key, value)
}
