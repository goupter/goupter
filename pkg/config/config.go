package config

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/spf13/viper"
)

// Config 应用配置结构
type Config struct {
	App      AppConfig      `mapstructure:"app"`
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	Consul   ConsulConfig   `mapstructure:"consul"`
	NATS     NATSConfig     `mapstructure:"nats"`
	Auth     AuthConfig     `mapstructure:"auth"`
	Log      LogConfig      `mapstructure:"log"`
	Trace    TraceConfig    `mapstructure:"trace"`
}

// AppConfig 应用配置
type AppConfig struct {
	Name    string `mapstructure:"name"`
	Version string `mapstructure:"version"`
	Env     string `mapstructure:"env"` // development, staging, production
}

// ServerConfig 服务器配置
type ServerConfig struct {
	HTTP HTTPConfig `mapstructure:"http"`
	GRPC GRPCConfig `mapstructure:"grpc"`
}

// HTTPConfig HTTP服务器配置
type HTTPConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
	// AdvertiseHost 用于服务注册/对外暴露的地址（监听仍使用 Host）
	// 常见场景：容器内监听 0.0.0.0，但注册到 Consul 时希望使用服务名或节点 IP。
	AdvertiseHost string        `mapstructure:"advertise_host"`
	ReadTimeout   time.Duration `mapstructure:"read_timeout"`
	WriteTimeout  time.Duration `mapstructure:"write_timeout"`
	IdleTimeout   time.Duration `mapstructure:"idle_timeout"`
}

// GRPCConfig gRPC服务器配置
type GRPCConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Driver          string        `mapstructure:"driver"`
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	Database        string        `mapstructure:"database"`
	Username        string        `mapstructure:"username"`
	Password        string        `mapstructure:"password"`
	Charset         string        `mapstructure:"charset"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns"`
	MaxOpenConns    int           `mapstructure:"max_open_conns"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
	ConnMaxIdleTime time.Duration `mapstructure:"conn_max_idle_time"`
	LogLevel        string        `mapstructure:"log_level"` // silent, error, warn, info
	// 慢查询配置
	SlowQueryThreshold time.Duration `mapstructure:"slow_query_threshold"`  // 慢查询阈值，默认200ms
	SlowQueryLog       bool          `mapstructure:"slow_query_log"`        // 是否启用慢查询日志
	SlowQueryWithStack bool          `mapstructure:"slow_query_with_stack"` // 是否记录调用栈
}

// RedisConfig Redis配置
type RedisConfig struct {
	// 单节点配置
	Host         string        `mapstructure:"host"`
	Port         int           `mapstructure:"port"`
	Password     string        `mapstructure:"password"`
	DB           int           `mapstructure:"db"`
	PoolSize     int           `mapstructure:"pool_size"`
	MinIdleConns int           `mapstructure:"min_idle_conns"`
	DialTimeout  time.Duration `mapstructure:"dial_timeout"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`

	// 集群/哨兵模式
	Mode           string   `mapstructure:"mode"`             // standalone, cluster, sentinel
	Addrs          []string `mapstructure:"addrs"`            // 集群地址列表
	MasterName     string   `mapstructure:"master_name"`      // 哨兵模式master名称
	SentinelAddrs  []string `mapstructure:"sentinel_addrs"`   // 哨兵地址列表
	RouteByLatency bool     `mapstructure:"route_by_latency"` // 集群模式按延迟路由
	RouteRandomly  bool     `mapstructure:"route_randomly"`   // 集群模式随机路由
}

// ConsulConfig Consul配置
type ConsulConfig struct {
	Host       string `mapstructure:"host"`
	Port       int    `mapstructure:"port"`
	Token      string `mapstructure:"token"`
	Datacenter string `mapstructure:"datacenter"`
	// 服务注册配置
	Service ConsulServiceConfig `mapstructure:"service"`
	// 配置中心
	ConfigPath string `mapstructure:"config_path"`
}

// ConsulServiceConfig Consul服务配置
type ConsulServiceConfig struct {
	ID                             string   `mapstructure:"id"`
	Name                           string   `mapstructure:"name"`
	Tags                           []string `mapstructure:"tags"`
	CheckInterval                  string   `mapstructure:"check_interval"`
	CheckTimeout                   string   `mapstructure:"check_timeout"`
	DeregisterCriticalServiceAfter string   `mapstructure:"deregister_critical_service_after"`
}

// NATSConfig NATS配置
type NATSConfig struct {
	URL            string        `mapstructure:"url"`
	ClusterID      string        `mapstructure:"cluster_id"`
	ClientID       string        `mapstructure:"client_id"`
	MaxReconnects  int           `mapstructure:"max_reconnects"`
	ReconnectWait  time.Duration `mapstructure:"reconnect_wait"`
	ConnectTimeout time.Duration `mapstructure:"connect_timeout"`
}

// AuthConfig 鉴权配置
type AuthConfig struct {
	Type   string                 `mapstructure:"type"` // jwt, apikey, oauth2, session
	Config map[string]interface{} `mapstructure:"config"`
}

// LogConfig 日志配置
type LogConfig struct {
	Level      string `mapstructure:"level"`  // debug, info, warn, error, fatal
	Format     string `mapstructure:"format"` // json, console
	Output     string `mapstructure:"output"` // stdout, file, both
	Filename   string `mapstructure:"filename"`
	MaxSize    int    `mapstructure:"max_size"` // MB
	MaxBackups int    `mapstructure:"max_backups"`
	MaxAge     int    `mapstructure:"max_age"` // days
	Compress   bool   `mapstructure:"compress"`
}

// TraceConfig 链路追踪配置
type TraceConfig struct {
	Enabled     bool    `mapstructure:"enabled"`
	ServiceName string  `mapstructure:"service_name"`
	Endpoint    string  `mapstructure:"endpoint"`
	SampleRate  float64 `mapstructure:"sample_rate"`
}

// Option 配置选项
type Option func(*options)

type options struct {
	configFile   string
	configType   string
	configPaths  []string
	envPrefix    string
	defaultValue *Config
}

// WithConfigFile 设置配置文件路径
func WithConfigFile(file string) Option {
	return func(o *options) {
		o.configFile = file
	}
}

// WithConfigType 设置配置文件类型
func WithConfigType(t string) Option {
	return func(o *options) {
		o.configType = t
	}
}

// WithConfigPaths 设置配置文件搜索路径
func WithConfigPaths(paths ...string) Option {
	return func(o *options) {
		o.configPaths = paths
	}
}

// WithEnvPrefix 设置环境变量前缀
func WithEnvPrefix(prefix string) Option {
	return func(o *options) {
		o.envPrefix = prefix
	}
}

// WithDefault 设置默认配置
func WithDefault(cfg *Config) Option {
	return func(o *options) {
		o.defaultValue = cfg
	}
}

var (
	globalConfig *Config
	configMu     sync.RWMutex
	v            *viper.Viper
)

// Load 加载配置
func Load(opts ...Option) (*Config, error) {
	o := &options{
		configFile:  "config",
		configType:  "yaml",
		configPaths: []string{".", "./config", "/etc/flit"},
		envPrefix:   "FLIT",
	}

	for _, opt := range opts {
		opt(o)
	}

	v = viper.New()

	// 设置配置文件
	v.SetConfigName(o.configFile)
	v.SetConfigType(o.configType)
	for _, path := range o.configPaths {
		v.AddConfigPath(path)
	}

	// 设置环境变量
	v.SetEnvPrefix(o.envPrefix)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// 设置默认值
	setDefaults(v, o.defaultValue)

	// 读取配置文件
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("读取配置文件失败: %w", err)
		}
		// 配置文件不存在时使用默认值
	}

	// 解析配置
	cfg := &Config{}
	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("解析配置失败: %w", err)
	}

	configMu.Lock()
	globalConfig = cfg
	configMu.Unlock()

	return cfg, nil
}

// Get 获取全局配置
func Get() *Config {
	configMu.RLock()
	defer configMu.RUnlock()
	return globalConfig
}

// setDefaults 设置默认值
func setDefaults(v *viper.Viper, defaults *Config) {
	// 应用默认值
	v.SetDefault("app.name", "flit-app")
	v.SetDefault("app.version", "1.0.0")
	v.SetDefault("app.env", "development")

	// HTTP服务器默认值
	v.SetDefault("server.http.host", "0.0.0.0")
	v.SetDefault("server.http.port", 8080)
	v.SetDefault("server.http.advertise_host", "")
	v.SetDefault("server.http.read_timeout", "30s")
	v.SetDefault("server.http.write_timeout", "30s")
	v.SetDefault("server.http.idle_timeout", "120s")

	// gRPC服务器默认值
	v.SetDefault("server.grpc.host", "0.0.0.0")
	v.SetDefault("server.grpc.port", 9090)

	// 数据库默认值
	v.SetDefault("database.driver", "mysql")
	v.SetDefault("database.host", "localhost")
	v.SetDefault("database.port", 3306)
	v.SetDefault("database.charset", "utf8mb4")
	v.SetDefault("database.max_idle_conns", 10)
	v.SetDefault("database.max_open_conns", 100)
	v.SetDefault("database.conn_max_lifetime", "1h")
	v.SetDefault("database.conn_max_idle_time", "10m")
	v.SetDefault("database.log_level", "warn")
	v.SetDefault("database.slow_query_threshold", "200ms")
	v.SetDefault("database.slow_query_log", true)
	v.SetDefault("database.slow_query_with_stack", false)

	// Redis默认值
	v.SetDefault("redis.host", "localhost")
	v.SetDefault("redis.port", 6379)
	v.SetDefault("redis.db", 0)
	v.SetDefault("redis.pool_size", 10)
	v.SetDefault("redis.min_idle_conns", 5)
	v.SetDefault("redis.dial_timeout", "5s")
	v.SetDefault("redis.read_timeout", "3s")
	v.SetDefault("redis.write_timeout", "3s")

	// Consul默认值
	v.SetDefault("consul.host", "localhost")
	v.SetDefault("consul.port", 8500)
	v.SetDefault("consul.datacenter", "dc1")
	v.SetDefault("consul.service.check_interval", "10s")
	v.SetDefault("consul.service.check_timeout", "5s")
	v.SetDefault("consul.service.deregister_critical_service_after", "30s")

	// NATS默认值
	v.SetDefault("nats.url", "nats://localhost:4222")
	v.SetDefault("nats.max_reconnects", 10)
	v.SetDefault("nats.reconnect_wait", "2s")
	v.SetDefault("nats.connect_timeout", "5s")

	// 鉴权默认值
	v.SetDefault("auth.type", "jwt")

	// 日志默认值
	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "json")
	v.SetDefault("log.output", "stdout")
	v.SetDefault("log.max_size", 100)
	v.SetDefault("log.max_backups", 3)
	v.SetDefault("log.max_age", 7)
	v.SetDefault("log.compress", true)

	// 链路追踪默认值
	v.SetDefault("trace.enabled", false)
	v.SetDefault("trace.sample_rate", 1.0)

	// 如果有用户自定义默认值，覆盖
	if defaults != nil {
		// 可以在这里处理用户自定义默认值
	}
}

// GetString 获取字符串配置
func GetString(key string) string {
	return v.GetString(key)
}

// GetInt 获取整数配置
func GetInt(key string) int {
	return v.GetInt(key)
}

// GetBool 获取布尔配置
func GetBool(key string) bool {
	return v.GetBool(key)
}

// GetDuration 获取时间间隔配置
func GetDuration(key string) time.Duration {
	return v.GetDuration(key)
}

// GetEnv 获取环境变量，如果不存在则返回默认值
func GetEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// IsDevelopment 是否为开发环境
func (c *Config) IsDevelopment() bool {
	return c.App.Env == "development"
}

// IsProduction 是否为生产环境
func (c *Config) IsProduction() bool {
	return c.App.Env == "production"
}

// DatabaseDSN 获取数据库连接字符串
func (c *Config) DatabaseDSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local",
		c.Database.Username,
		c.Database.Password,
		c.Database.Host,
		c.Database.Port,
		c.Database.Database,
		c.Database.Charset,
	)
}

// RedisAddr 获取Redis地址
func (c *Config) RedisAddr() string {
	return fmt.Sprintf("%s:%d", c.Redis.Host, c.Redis.Port)
}

// ConsulAddr 获取Consul地址
func (c *Config) ConsulAddr() string {
	return fmt.Sprintf("%s:%d", c.Consul.Host, c.Consul.Port)
}

// HTTPAddr 获取HTTP服务地址
func (c *Config) HTTPAddr() string {
	return fmt.Sprintf("%s:%d", c.Server.HTTP.Host, c.Server.HTTP.Port)
}

// GRPCAddr 获取gRPC服务地址
func (c *Config) GRPCAddr() string {
	return fmt.Sprintf("%s:%d", c.Server.GRPC.Host, c.Server.GRPC.Port)
}
