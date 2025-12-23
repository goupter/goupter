package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestConfigStructs(t *testing.T) {
	t.Run("Config", func(t *testing.T) {
		cfg := &Config{
			App: AppConfig{
				Name:    "test-app",
				Version: "1.0.0",
				Env:     "development",
			},
			Server: ServerConfig{
				HTTP: HTTPConfig{
					Host: "0.0.0.0",
					Port: 8080,
				},
				GRPC: GRPCConfig{
					Host: "0.0.0.0",
					Port: 9090,
				},
			},
		}

		if cfg.App.Name != "test-app" {
			t.Errorf("App.Name = %s, want test-app", cfg.App.Name)
		}
		if cfg.Server.HTTP.Port != 8080 {
			t.Errorf("Server.HTTP.Port = %d, want 8080", cfg.Server.HTTP.Port)
		}
	})

	t.Run("AppConfig", func(t *testing.T) {
		app := AppConfig{
			Name:    "my-app",
			Version: "2.0.0",
			Env:     "production",
		}

		if app.Name != "my-app" {
			t.Errorf("Name = %s, want my-app", app.Name)
		}
		if app.Version != "2.0.0" {
			t.Errorf("Version = %s, want 2.0.0", app.Version)
		}
		if app.Env != "production" {
			t.Errorf("Env = %s, want production", app.Env)
		}
	})

	t.Run("ServerConfig", func(t *testing.T) {
		server := ServerConfig{
			HTTP: HTTPConfig{
				Host:         "localhost",
				Port:         3000,
				ReadTimeout:  30 * time.Second,
				WriteTimeout: 30 * time.Second,
				IdleTimeout:  120 * time.Second,
			},
			GRPC: GRPCConfig{
				Host: "localhost",
				Port: 50051,
			},
		}

		if server.HTTP.Host != "localhost" {
			t.Errorf("HTTP.Host = %s, want localhost", server.HTTP.Host)
		}
		if server.HTTP.Port != 3000 {
			t.Errorf("HTTP.Port = %d, want 3000", server.HTTP.Port)
		}
		if server.HTTP.ReadTimeout != 30*time.Second {
			t.Errorf("HTTP.ReadTimeout = %v, want 30s", server.HTTP.ReadTimeout)
		}
		if server.GRPC.Port != 50051 {
			t.Errorf("GRPC.Port = %d, want 50051", server.GRPC.Port)
		}
	})

	t.Run("DatabaseConfig", func(t *testing.T) {
		db := DatabaseConfig{
			Driver:          "mysql",
			Host:            "localhost",
			Port:            3306,
			Database:        "testdb",
			Username:        "root",
			Password:        "password",
			Charset:         "utf8mb4",
			MaxIdleConns:    10,
			MaxOpenConns:    100,
			ConnMaxLifetime: time.Hour,
			ConnMaxIdleTime: 10 * time.Minute,
			LogLevel:        "warn",
		}

		if db.Driver != "mysql" {
			t.Errorf("Driver = %s, want mysql", db.Driver)
		}
		if db.Port != 3306 {
			t.Errorf("Port = %d, want 3306", db.Port)
		}
		if db.MaxOpenConns != 100 {
			t.Errorf("MaxOpenConns = %d, want 100", db.MaxOpenConns)
		}
	})

	t.Run("RedisConfig", func(t *testing.T) {
		redis := RedisConfig{
			Host:           "localhost",
			Port:           6379,
			Password:       "secret",
			DB:             0,
			PoolSize:       10,
			MinIdleConns:   5,
			DialTimeout:    5 * time.Second,
			ReadTimeout:    3 * time.Second,
			WriteTimeout:   3 * time.Second,
			Mode:           "standalone",
			Addrs:          []string{"node1:6379", "node2:6379"},
			MasterName:     "mymaster",
			SentinelAddrs:  []string{"sentinel1:26379"},
			RouteByLatency: true,
			RouteRandomly:  false,
		}

		if redis.Host != "localhost" {
			t.Errorf("Host = %s, want localhost", redis.Host)
		}
		if redis.Port != 6379 {
			t.Errorf("Port = %d, want 6379", redis.Port)
		}
		if redis.Mode != "standalone" {
			t.Errorf("Mode = %s, want standalone", redis.Mode)
		}
		if len(redis.Addrs) != 2 {
			t.Errorf("Addrs length = %d, want 2", len(redis.Addrs))
		}
	})

	t.Run("ConsulConfig", func(t *testing.T) {
		consul := ConsulConfig{
			Host:       "localhost",
			Port:       8500,
			Token:      "my-token",
			Datacenter: "dc1",
			Service: ConsulServiceConfig{
				ID:                             "svc-1",
				Name:                           "my-service",
				Tags:                           []string{"api", "v1"},
				CheckInterval:                  "10s",
				CheckTimeout:                   "5s",
				DeregisterCriticalServiceAfter: "30s",
			},
			ConfigPath: "config/my-service",
		}

		if consul.Host != "localhost" {
			t.Errorf("Host = %s, want localhost", consul.Host)
		}
		if consul.Service.Name != "my-service" {
			t.Errorf("Service.Name = %s, want my-service", consul.Service.Name)
		}
		if len(consul.Service.Tags) != 2 {
			t.Errorf("Service.Tags length = %d, want 2", len(consul.Service.Tags))
		}
	})

	t.Run("NATSConfig", func(t *testing.T) {
		nats := NATSConfig{
			URL:            "nats://localhost:4222",
			ClusterID:      "test-cluster",
			ClientID:       "test-client",
			MaxReconnects:  10,
			ReconnectWait:  2 * time.Second,
			ConnectTimeout: 5 * time.Second,
		}

		if nats.URL != "nats://localhost:4222" {
			t.Errorf("URL = %s, want nats://localhost:4222", nats.URL)
		}
		if nats.MaxReconnects != 10 {
			t.Errorf("MaxReconnects = %d, want 10", nats.MaxReconnects)
		}
	})

	t.Run("AuthConfig", func(t *testing.T) {
		auth := AuthConfig{
			Type: "jwt",
			Config: map[string]interface{}{
				"secret":      "my-secret",
				"expire_time": "24h",
			},
		}

		if auth.Type != "jwt" {
			t.Errorf("Type = %s, want jwt", auth.Type)
		}
		if auth.Config["secret"] != "my-secret" {
			t.Errorf("Config[secret] = %v, want my-secret", auth.Config["secret"])
		}
	})

	t.Run("LogConfig", func(t *testing.T) {
		log := LogConfig{
			Level:      "info",
			Format:     "json",
			Output:     "stdout",
			Filename:   "/var/log/app.log",
			MaxSize:    100,
			MaxBackups: 3,
			MaxAge:     7,
			Compress:   true,
		}

		if log.Level != "info" {
			t.Errorf("Level = %s, want info", log.Level)
		}
		if log.Format != "json" {
			t.Errorf("Format = %s, want json", log.Format)
		}
		if log.MaxSize != 100 {
			t.Errorf("MaxSize = %d, want 100", log.MaxSize)
		}
	})

	t.Run("TraceConfig", func(t *testing.T) {
		trace := TraceConfig{
			Enabled:     true,
			ServiceName: "my-service",
			Endpoint:    "http://jaeger:14268/api/traces",
			SampleRate:  0.5,
		}

		if !trace.Enabled {
			t.Error("Enabled should be true")
		}
		if trace.ServiceName != "my-service" {
			t.Errorf("ServiceName = %s, want my-service", trace.ServiceName)
		}
		if trace.SampleRate != 0.5 {
			t.Errorf("SampleRate = %f, want 0.5", trace.SampleRate)
		}
	})
}

func TestOptions(t *testing.T) {
	t.Run("WithConfigFile", func(t *testing.T) {
		o := &options{}
		WithConfigFile("app")(&options{})

		opt := WithConfigFile("myconfig")
		opt(o)

		if o.configFile != "myconfig" {
			t.Errorf("configFile = %s, want myconfig", o.configFile)
		}
	})

	t.Run("WithConfigType", func(t *testing.T) {
		o := &options{}
		opt := WithConfigType("json")
		opt(o)

		if o.configType != "json" {
			t.Errorf("configType = %s, want json", o.configType)
		}
	})

	t.Run("WithConfigPaths", func(t *testing.T) {
		o := &options{}
		opt := WithConfigPaths("./config", "/etc/app")
		opt(o)

		if len(o.configPaths) != 2 {
			t.Errorf("configPaths length = %d, want 2", len(o.configPaths))
		}
		if o.configPaths[0] != "./config" {
			t.Errorf("configPaths[0] = %s, want ./config", o.configPaths[0])
		}
	})

	t.Run("WithEnvPrefix", func(t *testing.T) {
		o := &options{}
		opt := WithEnvPrefix("MYAPP")
		opt(o)

		if o.envPrefix != "MYAPP" {
			t.Errorf("envPrefix = %s, want MYAPP", o.envPrefix)
		}
	})

	t.Run("WithDefault", func(t *testing.T) {
		o := &options{}
		defaultCfg := &Config{
			App: AppConfig{
				Name: "default-app",
			},
		}
		opt := WithDefault(defaultCfg)
		opt(o)

		if o.defaultValue == nil {
			t.Error("defaultValue should not be nil")
		}
		if o.defaultValue.App.Name != "default-app" {
			t.Errorf("defaultValue.App.Name = %s, want default-app", o.defaultValue.App.Name)
		}
	})
}

func TestLoad(t *testing.T) {
	t.Run("load with defaults", func(t *testing.T) {
		// 重置全局状态
		configMu.Lock()
		globalConfig = nil
		configMu.Unlock()

		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}

		if cfg == nil {
			t.Fatal("Load should return non-nil config")
		}

		// 验证默认值
		if cfg.App.Name != "flit-app" {
			t.Errorf("App.Name = %s, want flit-app", cfg.App.Name)
		}
		if cfg.App.Version != "1.0.0" {
			t.Errorf("App.Version = %s, want 1.0.0", cfg.App.Version)
		}
		if cfg.App.Env != "development" {
			t.Errorf("App.Env = %s, want development", cfg.App.Env)
		}
		if cfg.Server.HTTP.Port != 8080 {
			t.Errorf("Server.HTTP.Port = %d, want 8080", cfg.Server.HTTP.Port)
		}
		if cfg.Server.GRPC.Port != 9090 {
			t.Errorf("Server.GRPC.Port = %d, want 9090", cfg.Server.GRPC.Port)
		}
		if cfg.Database.Driver != "mysql" {
			t.Errorf("Database.Driver = %s, want mysql", cfg.Database.Driver)
		}
		if cfg.Redis.Port != 6379 {
			t.Errorf("Redis.Port = %d, want 6379", cfg.Redis.Port)
		}
		if cfg.Log.Level != "info" {
			t.Errorf("Log.Level = %s, want info", cfg.Log.Level)
		}
	})

	t.Run("load with config file", func(t *testing.T) {
		// 创建临时配置文件
		tmpDir := t.TempDir()
		configFile := filepath.Join(tmpDir, "config.yaml")
		content := `
app:
  name: test-app
  version: 2.0.0
  env: production
server:
  http:
    port: 3000
  grpc:
    port: 50051
`
		if err := os.WriteFile(configFile, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create config file: %v", err)
		}

		cfg, err := Load(
			WithConfigFile("config"),
			WithConfigType("yaml"),
			WithConfigPaths(tmpDir),
		)
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}

		if cfg.App.Name != "test-app" {
			t.Errorf("App.Name = %s, want test-app", cfg.App.Name)
		}
		if cfg.App.Version != "2.0.0" {
			t.Errorf("App.Version = %s, want 2.0.0", cfg.App.Version)
		}
		if cfg.Server.HTTP.Port != 3000 {
			t.Errorf("Server.HTTP.Port = %d, want 3000", cfg.Server.HTTP.Port)
		}
	})

	t.Run("load with env prefix", func(t *testing.T) {
		// 设置环境变量
		os.Setenv("TEST_APP_NAME", "env-app")
		defer os.Unsetenv("TEST_APP_NAME")

		cfg, err := Load(WithEnvPrefix("TEST"))
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}

		if cfg.App.Name != "env-app" {
			t.Errorf("App.Name = %s, want env-app", cfg.App.Name)
		}
	})
}

func TestGet(t *testing.T) {
	// 先加载配置
	configMu.Lock()
	globalConfig = &Config{
		App: AppConfig{
			Name: "get-test-app",
		},
	}
	configMu.Unlock()

	cfg := Get()
	if cfg == nil {
		t.Fatal("Get should return non-nil config")
	}
	if cfg.App.Name != "get-test-app" {
		t.Errorf("App.Name = %s, want get-test-app", cfg.App.Name)
	}
}

func TestGetHelpers(t *testing.T) {
	// 先加载配置以初始化 viper
	Load()

	t.Run("GetString", func(t *testing.T) {
		got := GetString("app.name")
		if got == "" {
			t.Error("GetString should return non-empty for app.name")
		}
	})

	t.Run("GetInt", func(t *testing.T) {
		got := GetInt("server.http.port")
		if got == 0 {
			t.Error("GetInt should return non-zero for server.http.port")
		}
	})

	t.Run("GetBool", func(t *testing.T) {
		got := GetBool("log.compress")
		if !got {
			t.Error("GetBool should return true for log.compress")
		}
	})

	t.Run("GetDuration", func(t *testing.T) {
		got := GetDuration("server.http.read_timeout")
		if got == 0 {
			t.Error("GetDuration should return non-zero for server.http.read_timeout")
		}
	})
}

func TestGetEnv(t *testing.T) {
	t.Run("existing env", func(t *testing.T) {
		os.Setenv("TEST_ENV_VAR", "test-value")
		defer os.Unsetenv("TEST_ENV_VAR")

		got := GetEnv("TEST_ENV_VAR", "default")
		if got != "test-value" {
			t.Errorf("GetEnv() = %s, want test-value", got)
		}
	})

	t.Run("non-existing env", func(t *testing.T) {
		got := GetEnv("NON_EXISTING_VAR", "default-value")
		if got != "default-value" {
			t.Errorf("GetEnv() = %s, want default-value", got)
		}
	})
}

func TestConfig_IsDevelopment(t *testing.T) {
	tests := []struct {
		name     string
		env      string
		expected bool
	}{
		{"development", "development", true},
		{"production", "production", false},
		{"staging", "staging", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				App: AppConfig{Env: tt.env},
			}
			if got := cfg.IsDevelopment(); got != tt.expected {
				t.Errorf("IsDevelopment() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestConfig_IsProduction(t *testing.T) {
	tests := []struct {
		name     string
		env      string
		expected bool
	}{
		{"production", "production", true},
		{"development", "development", false},
		{"staging", "staging", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				App: AppConfig{Env: tt.env},
			}
			if got := cfg.IsProduction(); got != tt.expected {
				t.Errorf("IsProduction() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestConfig_DatabaseDSN(t *testing.T) {
	cfg := &Config{
		Database: DatabaseConfig{
			Username: "root",
			Password: "password",
			Host:     "localhost",
			Port:     3306,
			Database: "testdb",
			Charset:  "utf8mb4",
		},
	}

	dsn := cfg.DatabaseDSN()
	expected := "root:password@tcp(localhost:3306)/testdb?charset=utf8mb4&parseTime=True&loc=Local"

	if dsn != expected {
		t.Errorf("DatabaseDSN() = %s, want %s", dsn, expected)
	}
}

func TestConfig_RedisAddr(t *testing.T) {
	cfg := &Config{
		Redis: RedisConfig{
			Host: "localhost",
			Port: 6379,
		},
	}

	addr := cfg.RedisAddr()
	expected := "localhost:6379"

	if addr != expected {
		t.Errorf("RedisAddr() = %s, want %s", addr, expected)
	}
}

func TestConfig_ConsulAddr(t *testing.T) {
	cfg := &Config{
		Consul: ConsulConfig{
			Host: "consul.local",
			Port: 8500,
		},
	}

	addr := cfg.ConsulAddr()
	expected := "consul.local:8500"

	if addr != expected {
		t.Errorf("ConsulAddr() = %s, want %s", addr, expected)
	}
}

func TestConfig_HTTPAddr(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{
			HTTP: HTTPConfig{
				Host: "0.0.0.0",
				Port: 8080,
			},
		},
	}

	addr := cfg.HTTPAddr()
	expected := "0.0.0.0:8080"

	if addr != expected {
		t.Errorf("HTTPAddr() = %s, want %s", addr, expected)
	}
}

func TestConfig_GRPCAddr(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{
			GRPC: GRPCConfig{
				Host: "0.0.0.0",
				Port: 9090,
			},
		},
	}

	addr := cfg.GRPCAddr()
	expected := "0.0.0.0:9090"

	if addr != expected {
		t.Errorf("GRPCAddr() = %s, want %s", addr, expected)
	}
}

func TestLoad_InvalidConfigFile(t *testing.T) {
	// 创建临时目录并添加无效配置文件
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yaml")

	// 写入无效的 YAML
	content := `
invalid yaml content
  - this is: [broken
`
	if err := os.WriteFile(configFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	_, err := Load(
		WithConfigFile("config"),
		WithConfigType("yaml"),
		WithConfigPaths(tmpDir),
	)

	if err == nil {
		t.Error("Load should return error for invalid config file")
	}
}
