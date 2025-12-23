package config

import (
	"fmt"
	"net"
	"reflect"
	"regexp"
	"strings"
	"time"
)

// ValidationError 校验错误
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Value   any    `json:"value,omitempty"`
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("配置校验失败 [%s]: %s", e.Field, e.Message)
}

// ValidationErrors 多个校验错误
type ValidationErrors []*ValidationError

func (e ValidationErrors) Error() string {
	if len(e) == 0 {
		return ""
	}
	var msgs []string
	for _, err := range e {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, "; ")
}

// HasErrors 是否有错误
func (e ValidationErrors) HasErrors() bool {
	return len(e) > 0
}

// Validator 配置校验器接口
type Validator interface {
	Validate() ValidationErrors
}

// ValidationRule 校验规则
type ValidationRule func(value any, field string) *ValidationError

// ConfigValidator 配置校验器
type ConfigValidator struct {
	rules  map[string][]ValidationRule
	errors ValidationErrors
}

// NewConfigValidator 创建配置校验器
func NewConfigValidator() *ConfigValidator {
	return &ConfigValidator{
		rules:  make(map[string][]ValidationRule),
		errors: make(ValidationErrors, 0),
	}
}

// AddRule 添加校验规则
func (v *ConfigValidator) AddRule(field string, rules ...ValidationRule) *ConfigValidator {
	v.rules[field] = append(v.rules[field], rules...)
	return v
}

// ValidateConfig 校验配置
func (v *ConfigValidator) ValidateConfig(cfg *Config) ValidationErrors {
	v.errors = make(ValidationErrors, 0)

	if cfg == nil {
		v.errors = append(v.errors, &ValidationError{
			Field:   "config",
			Message: "配置不能为空",
		})
		return v.errors
	}

	// 校验 App 配置
	v.validateApp(&cfg.App)

	// 校验 Server 配置
	v.validateServer(&cfg.Server)

	// 校验 Database 配置
	v.validateDatabase(&cfg.Database)

	// 校验 Redis 配置
	v.validateRedis(&cfg.Redis)

	// 校验 Consul 配置
	v.validateConsul(&cfg.Consul)

	// 校验 NATS 配置
	v.validateNATS(&cfg.NATS)

	// 校验 Log 配置
	v.validateLog(&cfg.Log)

	// 校验 Trace 配置
	v.validateTrace(&cfg.Trace)

	return v.errors
}

// validateApp 校验应用配置
func (v *ConfigValidator) validateApp(cfg *AppConfig) {
	if cfg.Name == "" {
		v.errors = append(v.errors, &ValidationError{
			Field:   "app.name",
			Message: "应用名称不能为空",
		})
	}

	if cfg.Version == "" {
		v.errors = append(v.errors, &ValidationError{
			Field:   "app.version",
			Message: "应用版本不能为空",
		})
	}

	validEnvs := []string{"development", "staging", "production", "test"}
	if cfg.Env != "" && !containsString(validEnvs, cfg.Env) {
		v.errors = append(v.errors, &ValidationError{
			Field:   "app.env",
			Message: fmt.Sprintf("无效的环境值，必须是: %s", strings.Join(validEnvs, ", ")),
			Value:   cfg.Env,
		})
	}
}

// validateServer 校验服务器配置
func (v *ConfigValidator) validateServer(cfg *ServerConfig) {
	// HTTP 配置校验
	if cfg.HTTP.Port < 0 || cfg.HTTP.Port > 65535 {
		v.errors = append(v.errors, &ValidationError{
			Field:   "server.http.port",
			Message: "端口必须在 0-65535 范围内",
			Value:   cfg.HTTP.Port,
		})
	}

	if cfg.HTTP.ReadTimeout < 0 {
		v.errors = append(v.errors, &ValidationError{
			Field:   "server.http.read_timeout",
			Message: "读取超时不能为负数",
			Value:   cfg.HTTP.ReadTimeout,
		})
	}

	if cfg.HTTP.WriteTimeout < 0 {
		v.errors = append(v.errors, &ValidationError{
			Field:   "server.http.write_timeout",
			Message: "写入超时不能为负数",
			Value:   cfg.HTTP.WriteTimeout,
		})
	}

	// gRPC 配置校验
	if cfg.GRPC.Port < 0 || cfg.GRPC.Port > 65535 {
		v.errors = append(v.errors, &ValidationError{
			Field:   "server.grpc.port",
			Message: "端口必须在 0-65535 范围内",
			Value:   cfg.GRPC.Port,
		})
	}
}

// validateDatabase 校验数据库配置
func (v *ConfigValidator) validateDatabase(cfg *DatabaseConfig) {
	validDrivers := []string{"mysql", "postgres", "sqlite", "sqlserver"}
	if cfg.Driver != "" && !containsString(validDrivers, cfg.Driver) {
		v.errors = append(v.errors, &ValidationError{
			Field:   "database.driver",
			Message: fmt.Sprintf("无效的数据库驱动，必须是: %s", strings.Join(validDrivers, ", ")),
			Value:   cfg.Driver,
		})
	}

	if cfg.Port < 0 || cfg.Port > 65535 {
		v.errors = append(v.errors, &ValidationError{
			Field:   "database.port",
			Message: "端口必须在 0-65535 范围内",
			Value:   cfg.Port,
		})
	}

	if cfg.MaxIdleConns < 0 {
		v.errors = append(v.errors, &ValidationError{
			Field:   "database.max_idle_conns",
			Message: "最大空闲连接数不能为负数",
			Value:   cfg.MaxIdleConns,
		})
	}

	if cfg.MaxOpenConns < 0 {
		v.errors = append(v.errors, &ValidationError{
			Field:   "database.max_open_conns",
			Message: "最大打开连接数不能为负数",
			Value:   cfg.MaxOpenConns,
		})
	}

	if cfg.MaxIdleConns > 0 && cfg.MaxOpenConns > 0 && cfg.MaxIdleConns > cfg.MaxOpenConns {
		v.errors = append(v.errors, &ValidationError{
			Field:   "database.max_idle_conns",
			Message: "最大空闲连接数不能大于最大打开连接数",
			Value:   cfg.MaxIdleConns,
		})
	}

	if cfg.ConnMaxLifetime < 0 {
		v.errors = append(v.errors, &ValidationError{
			Field:   "database.conn_max_lifetime",
			Message: "连接最大生存时间不能为负数",
			Value:   cfg.ConnMaxLifetime,
		})
	}

	validLogLevels := []string{"silent", "error", "warn", "info"}
	if cfg.LogLevel != "" && !containsString(validLogLevels, cfg.LogLevel) {
		v.errors = append(v.errors, &ValidationError{
			Field:   "database.log_level",
			Message: fmt.Sprintf("无效的日志级别，必须是: %s", strings.Join(validLogLevels, ", ")),
			Value:   cfg.LogLevel,
		})
	}

	if cfg.SlowQueryThreshold < 0 {
		v.errors = append(v.errors, &ValidationError{
			Field:   "database.slow_query_threshold",
			Message: "慢查询阈值不能为负数",
			Value:   cfg.SlowQueryThreshold,
		})
	}
}

// validateRedis 校验Redis配置
func (v *ConfigValidator) validateRedis(cfg *RedisConfig) {
	if cfg.Port < 0 || cfg.Port > 65535 {
		v.errors = append(v.errors, &ValidationError{
			Field:   "redis.port",
			Message: "端口必须在 0-65535 范围内",
			Value:   cfg.Port,
		})
	}

	if cfg.DB < 0 || cfg.DB > 15 {
		v.errors = append(v.errors, &ValidationError{
			Field:   "redis.db",
			Message: "数据库索引必须在 0-15 范围内",
			Value:   cfg.DB,
		})
	}

	if cfg.PoolSize < 0 {
		v.errors = append(v.errors, &ValidationError{
			Field:   "redis.pool_size",
			Message: "连接池大小不能为负数",
			Value:   cfg.PoolSize,
		})
	}

	validModes := []string{"", "standalone", "cluster", "sentinel"}
	if !containsString(validModes, cfg.Mode) {
		v.errors = append(v.errors, &ValidationError{
			Field:   "redis.mode",
			Message: fmt.Sprintf("无效的模式，必须是: %s", strings.Join(validModes[1:], ", ")),
			Value:   cfg.Mode,
		})
	}

	if cfg.Mode == "cluster" && len(cfg.Addrs) == 0 {
		v.errors = append(v.errors, &ValidationError{
			Field:   "redis.addrs",
			Message: "集群模式下必须配置节点地址",
		})
	}

	if cfg.Mode == "sentinel" {
		if cfg.MasterName == "" {
			v.errors = append(v.errors, &ValidationError{
				Field:   "redis.master_name",
				Message: "哨兵模式下必须配置 master 名称",
			})
		}
		if len(cfg.SentinelAddrs) == 0 {
			v.errors = append(v.errors, &ValidationError{
				Field:   "redis.sentinel_addrs",
				Message: "哨兵模式下必须配置哨兵地址",
			})
		}
	}

	if cfg.DialTimeout < 0 {
		v.errors = append(v.errors, &ValidationError{
			Field:   "redis.dial_timeout",
			Message: "连接超时不能为负数",
			Value:   cfg.DialTimeout,
		})
	}
}

// validateConsul 校验Consul配置
func (v *ConfigValidator) validateConsul(cfg *ConsulConfig) {
	if cfg.Port < 0 || cfg.Port > 65535 {
		v.errors = append(v.errors, &ValidationError{
			Field:   "consul.port",
			Message: "端口必须在 0-65535 范围内",
			Value:   cfg.Port,
		})
	}

	// 校验服务配置
	if cfg.Service.CheckInterval != "" {
		if _, err := time.ParseDuration(cfg.Service.CheckInterval); err != nil {
			v.errors = append(v.errors, &ValidationError{
				Field:   "consul.service.check_interval",
				Message: "无效的时间间隔格式",
				Value:   cfg.Service.CheckInterval,
			})
		}
	}

	if cfg.Service.CheckTimeout != "" {
		if _, err := time.ParseDuration(cfg.Service.CheckTimeout); err != nil {
			v.errors = append(v.errors, &ValidationError{
				Field:   "consul.service.check_timeout",
				Message: "无效的时间间隔格式",
				Value:   cfg.Service.CheckTimeout,
			})
		}
	}
}

// validateNATS 校验NATS配置
func (v *ConfigValidator) validateNATS(cfg *NATSConfig) {
	if cfg.URL != "" {
		if !strings.HasPrefix(cfg.URL, "nats://") && !strings.HasPrefix(cfg.URL, "tls://") {
			v.errors = append(v.errors, &ValidationError{
				Field:   "nats.url",
				Message: "NATS URL 必须以 nats:// 或 tls:// 开头",
				Value:   cfg.URL,
			})
		}
	}

	if cfg.MaxReconnects < -1 {
		v.errors = append(v.errors, &ValidationError{
			Field:   "nats.max_reconnects",
			Message: "最大重连次数不能小于 -1（-1 表示无限重连）",
			Value:   cfg.MaxReconnects,
		})
	}

	if cfg.ReconnectWait < 0 {
		v.errors = append(v.errors, &ValidationError{
			Field:   "nats.reconnect_wait",
			Message: "重连等待时间不能为负数",
			Value:   cfg.ReconnectWait,
		})
	}

	if cfg.ConnectTimeout < 0 {
		v.errors = append(v.errors, &ValidationError{
			Field:   "nats.connect_timeout",
			Message: "连接超时不能为负数",
			Value:   cfg.ConnectTimeout,
		})
	}
}

// validateLog 校验日志配置
func (v *ConfigValidator) validateLog(cfg *LogConfig) {
	validLevels := []string{"debug", "info", "warn", "error", "fatal", ""}
	if !containsString(validLevels, cfg.Level) {
		v.errors = append(v.errors, &ValidationError{
			Field:   "log.level",
			Message: fmt.Sprintf("无效的日志级别，必须是: %s", strings.Join(validLevels[:len(validLevels)-1], ", ")),
			Value:   cfg.Level,
		})
	}

	validFormats := []string{"json", "console", ""}
	if !containsString(validFormats, cfg.Format) {
		v.errors = append(v.errors, &ValidationError{
			Field:   "log.format",
			Message: "无效的日志格式，必须是: json, console",
			Value:   cfg.Format,
		})
	}

	validOutputs := []string{"stdout", "file", "both", ""}
	if !containsString(validOutputs, cfg.Output) {
		v.errors = append(v.errors, &ValidationError{
			Field:   "log.output",
			Message: "无效的输出方式，必须是: stdout, file, both",
			Value:   cfg.Output,
		})
	}

	if cfg.MaxSize < 0 {
		v.errors = append(v.errors, &ValidationError{
			Field:   "log.max_size",
			Message: "最大文件大小不能为负数",
			Value:   cfg.MaxSize,
		})
	}

	if cfg.MaxBackups < 0 {
		v.errors = append(v.errors, &ValidationError{
			Field:   "log.max_backups",
			Message: "最大备份数不能为负数",
			Value:   cfg.MaxBackups,
		})
	}

	if cfg.MaxAge < 0 {
		v.errors = append(v.errors, &ValidationError{
			Field:   "log.max_age",
			Message: "最大保留天数不能为负数",
			Value:   cfg.MaxAge,
		})
	}
}

// validateTrace 校验链路追踪配置
func (v *ConfigValidator) validateTrace(cfg *TraceConfig) {
	if cfg.Enabled {
		if cfg.ServiceName == "" {
			v.errors = append(v.errors, &ValidationError{
				Field:   "trace.service_name",
				Message: "启用链路追踪时必须配置服务名称",
			})
		}

		if cfg.Endpoint == "" {
			v.errors = append(v.errors, &ValidationError{
				Field:   "trace.endpoint",
				Message: "启用链路追踪时必须配置上报地址",
			})
		}
	}

	if cfg.SampleRate < 0 || cfg.SampleRate > 1 {
		v.errors = append(v.errors, &ValidationError{
			Field:   "trace.sample_rate",
			Message: "采样率必须在 0-1 范围内",
			Value:   cfg.SampleRate,
		})
	}
}

// containsString 检查字符串是否在切片中
func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

// ==================== 通用校验规则 ====================

// Required 必填校验
func Required(value any, field string) *ValidationError {
	if isEmptyValue(value) {
		return &ValidationError{
			Field:   field,
			Message: "该字段为必填项",
		}
	}
	return nil
}

// Min 最小值校验
func Min(min int) ValidationRule {
	return func(value any, field string) *ValidationError {
		v := reflect.ValueOf(value)
		switch v.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if v.Int() < int64(min) {
				return &ValidationError{
					Field:   field,
					Message: fmt.Sprintf("值不能小于 %d", min),
					Value:   value,
				}
			}
		case reflect.String:
			if len(v.String()) < min {
				return &ValidationError{
					Field:   field,
					Message: fmt.Sprintf("长度不能小于 %d", min),
					Value:   value,
				}
			}
		case reflect.Slice, reflect.Array:
			if v.Len() < min {
				return &ValidationError{
					Field:   field,
					Message: fmt.Sprintf("元素数量不能少于 %d", min),
					Value:   value,
				}
			}
		}
		return nil
	}
}

// Max 最大值校验
func Max(max int) ValidationRule {
	return func(value any, field string) *ValidationError {
		v := reflect.ValueOf(value)
		switch v.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if v.Int() > int64(max) {
				return &ValidationError{
					Field:   field,
					Message: fmt.Sprintf("值不能大于 %d", max),
					Value:   value,
				}
			}
		case reflect.String:
			if len(v.String()) > max {
				return &ValidationError{
					Field:   field,
					Message: fmt.Sprintf("长度不能大于 %d", max),
					Value:   value,
				}
			}
		case reflect.Slice, reflect.Array:
			if v.Len() > max {
				return &ValidationError{
					Field:   field,
					Message: fmt.Sprintf("元素数量不能多于 %d", max),
					Value:   value,
				}
			}
		}
		return nil
	}
}

// Range 范围校验
func Range(min, max int) ValidationRule {
	return func(value any, field string) *ValidationError {
		v := reflect.ValueOf(value)
		switch v.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			val := v.Int()
			if val < int64(min) || val > int64(max) {
				return &ValidationError{
					Field:   field,
					Message: fmt.Sprintf("值必须在 %d 到 %d 之间", min, max),
					Value:   value,
				}
			}
		}
		return nil
	}
}

// OneOf 枚举值校验
func OneOf(values ...string) ValidationRule {
	return func(value any, field string) *ValidationError {
		s, ok := value.(string)
		if !ok {
			return nil
		}
		if !containsString(values, s) {
			return &ValidationError{
				Field:   field,
				Message: fmt.Sprintf("值必须是以下之一: %s", strings.Join(values, ", ")),
				Value:   value,
			}
		}
		return nil
	}
}

// Pattern 正则校验
func Pattern(pattern string) ValidationRule {
	re := regexp.MustCompile(pattern)
	return func(value any, field string) *ValidationError {
		s, ok := value.(string)
		if !ok {
			return nil
		}
		if s != "" && !re.MatchString(s) {
			return &ValidationError{
				Field:   field,
				Message: "格式不正确",
				Value:   value,
			}
		}
		return nil
	}
}

// Email 邮箱格式校验
func Email(value any, field string) *ValidationError {
	s, ok := value.(string)
	if !ok || s == "" {
		return nil
	}
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(s) {
		return &ValidationError{
			Field:   field,
			Message: "邮箱格式不正确",
			Value:   value,
		}
	}
	return nil
}

// URL 网址格式校验
func URL(value any, field string) *ValidationError {
	s, ok := value.(string)
	if !ok || s == "" {
		return nil
	}
	urlRegex := regexp.MustCompile(`^(http|https)://[a-zA-Z0-9][-a-zA-Z0-9]{0,62}(\.[a-zA-Z0-9][-a-zA-Z0-9]{0,62})+(:[0-9]{1,5})?(/.*)?$`)
	if !urlRegex.MatchString(s) {
		return &ValidationError{
			Field:   field,
			Message: "URL格式不正确",
			Value:   value,
		}
	}
	return nil
}

// IP 校验 IP 地址格式
func IP(value any, field string) *ValidationError {
	s, ok := value.(string)
	if !ok || s == "" {
		return nil
	}
	if net.ParseIP(s) == nil {
		return &ValidationError{
			Field:   field,
			Message: "IP地址格式不正确",
			Value:   value,
		}
	}
	return nil
}

// Port 端口校验
func Port(value any, field string) *ValidationError {
	var port int
	switch v := value.(type) {
	case int:
		port = v
	case int64:
		port = int(v)
	default:
		return nil
	}
	if port < 0 || port > 65535 {
		return &ValidationError{
			Field:   field,
			Message: "端口必须在 0-65535 范围内",
			Value:   value,
		}
	}
	return nil
}

// Duration 时间间隔格式校验
func Duration(value any, field string) *ValidationError {
	s, ok := value.(string)
	if !ok || s == "" {
		return nil
	}
	if _, err := time.ParseDuration(s); err != nil {
		return &ValidationError{
			Field:   field,
			Message: "时间间隔格式不正确",
			Value:   value,
		}
	}
	return nil
}

// isEmptyValue 检查值是否为空
func isEmptyValue(value any) bool {
	if value == nil {
		return true
	}
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.String:
		return v.String() == ""
	case reflect.Array, reflect.Slice, reflect.Map:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Ptr, reflect.Interface:
		return v.IsNil()
	}
	return false
}

// ==================== 便捷函数 ====================

// ValidateConfig 校验全局配置
func ValidateConfig(cfg *Config) ValidationErrors {
	validator := NewConfigValidator()
	return validator.ValidateConfig(cfg)
}

// MustValidateConfig 校验配置，有错误则 panic
func MustValidateConfig(cfg *Config) {
	errs := ValidateConfig(cfg)
	if errs.HasErrors() {
		panic(errs.Error())
	}
}

// ValidateAndLoad 加载并校验配置
func ValidateAndLoad(opts ...Option) (*Config, error) {
	cfg, err := Load(opts...)
	if err != nil {
		return nil, err
	}

	if errs := ValidateConfig(cfg); errs.HasErrors() {
		return nil, fmt.Errorf("配置校验失败: %w", errs)
	}

	return cfg, nil
}
