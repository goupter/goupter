# Goupter

Goupter 是一个轻量级 Go 微服务框架与 CLI 代码生成工具，提供应用生命周期管理、HTTP/gRPC 服务器封装、配置与日志体系、缓存/数据库/服务发现/消息队列等基础能力。

## 安装 CLI

### 方式 1: go install（推荐）

```bash
go install github.com/goupter/goupter/cmd/goupter@latest
```

### 方式 2: 安装脚本

```bash
curl -sSL https://raw.githubusercontent.com/goupter/goupter/main/install.sh | bash
```

### 方式 3: Release 下载

访问 [Releases](https://github.com/goupter/goupter/releases) 下载对应平台的二进制文件。

### 方式 4: 源码构建

```bash
git clone https://github.com/goupter/goupter.git
cd goupter && make build
# 二进制文件: bin/goupter
```

验证安装：

```bash
goupter --version
goupter --help
```

## 快速开始

### 创建服务

```bash
# 创建新项目
mkdir myproject && cd myproject
goupter new user              # 创建 user 服务
goupter new order             # 创建 order 服务（共享 go.mod）

# 运行
go mod tidy
go run ./cmd/user
```

### 运行示例

```bash
# 本地运行（无外部依赖）
cd examples/http-api && go run .

# Docker Compose（带 MySQL/Redis/Consul/NATS）
docker compose -f examples/http-api/docker-compose.yml up --build
```

访问：
- 首页：`http://localhost:8080/`
- 健康检查：`http://localhost:8080/health`
- 指标：`http://localhost:8080/metrics`
- pprof：`http://localhost:8080/debug/pprof/`

## CLI 命令

| 命令 | 说明 |
|------|------|
| `goupter new <name>` | 创建新服务 |
| `goupter gen model --dsn "..."` | 从数据库生成 GORM 模型 |
| `goupter gen crud --dsn "..." --service <name>` | 生成完整 CRUD（Model + Handler + Routes） |
| `goupter gen httpbook --service <name>` | 生成 HTTP API 调试文件 |

### 生成模型

```bash
goupter gen model --dsn "user:pass@tcp(localhost:3306)/dbname" --out ./model
```

### 生成 CRUD

```bash
# 生成到指定服务目录
goupter gen crud --dsn "user:pass@tcp(localhost:3306)/dbname" --service user

# 指定表
goupter gen crud --dsn "..." --tables users,orders --service user
```

### 生成 HTTP 调试文件

```bash
goupter gen httpbook --service user --out-dir ./httpbook
```

## 使用指南

### 配置

```go
cfg, err := config.Load(
    config.WithConfigFile("config"),
    config.WithConfigType("yaml"),
    config.WithConfigPaths(".", "./config"),
)
```

配置热更新：

```go
hot := config.NewHotReloader()
hot.OnChange(func(event *config.ChangeEvent) {
    logger.Info("config changed", log.String("source", event.Source))
})
_ = hot.Start()
defer hot.Stop()
```

### HTTP Server

```go
httpServer := server.NewHTTPServer(
    server.WithHTTPConfig(&cfg.Server.HTTP),
    server.WithHTTPLogger(logger),
    server.WithMiddleware(
        middleware.Recovery(logger),
        middleware.Logger(logger),
        middleware.CORS(),
    ),
)

httpServer.GET("/health", func(c *gin.Context) {
    c.JSON(200, gin.H{"status": "ok"})
})
```

### 日志

```go
logger, _ := log.NewZapLogger(&log.LoggerConfig{
    Level:  log.InfoLevel,
    Format: "json",
    Output: "stdout",
})
logger.Info("hello", log.String("module", "demo"))
```

### 缓存

```go
// 内存缓存
c := cache.NewMemoryCache()
_ = c.Set(ctx, "key", "value", time.Minute)

// Redis 缓存
c, _ := cache.NewRedisCache(cache.WithRedisConfig(&cfg.Redis))

// 多级缓存
l1 := cache.NewMemoryCache()
l2, _ := cache.NewRedisCache(cache.WithRedisConfig(&cfg.Redis))
ml := cache.NewMultiLevelCache(cache.WithLevels(l1, l2))
```

### 数据库

```go
mysqlClient, err := database.NewMySQL(
    database.WithConfig(&cfg.Database),
    database.WithLogger(logger),
)
db := mysqlClient.DB()
```

### 鉴权

```go
import (
    _ "github.com/goupter/goupter/pkg/auth/jwt"
    _ "github.com/goupter/goupter/pkg/auth/apikey"
)

authenticator, _ := auth.NewAuthenticator(cfg.Auth.Type, cfg.Auth.Config)
claims, _ := authenticator.Authenticate(ctx, "token")
```

### 服务发现

```go
disc, _ := discovery.NewConsulDiscovery(
    discovery.WithConsulConfig(&cfg.Consul),
    discovery.WithConsulLogger(logger),
)
```

### 消息队列

```go
client, _ := mq.NewNATSClient(
    mq.WithNATSConfig(&cfg.NATS),
    mq.WithNATSLogger(logger),
)
_ = client.Publish(ctx, "topic", &mq.Message{Payload: []byte("hi")})
```

### 统一响应

```go
func handler(c *gin.Context) {
    response.Success(c, gin.H{"ok": true})
}

// 错误响应
err := errors.New(errors.CodeBadRequest, "invalid param")
response.Error(c, err)
```

### 应用生命周期

```go
application := app.New(
    app.WithName(cfg.App.Name),
    app.WithConfig(cfg),
    app.WithLogger(logger),
    app.WithHTTPServer(httpServer),
    app.WithHealthConfig(app.DefaultHealthConfig()),
    app.WithMetricsConfig(app.DefaultMetricsConfig()),
    app.WithPProfConfig(app.DefaultPProfConfig()),
)

// 注册健康检查
application.RegisterReadinessChecker(app.NewFuncChecker("db", func(ctx context.Context) app.CheckResult {
    return app.CheckResult{Status: app.StatusUp}
}))

_ = application.Run()
```

## 模块总览

| 模块 | 说明 | 状态 |
|------|------|------|
| `pkg/app` | 应用生命周期、健康检查、指标、pprof | 已实现 |
| `pkg/server` | HTTP/gRPC 服务器封装 | 已实现 |
| `pkg/server/middleware` | 日志、恢复、安全、限流、超时、压缩、鉴权 | 已实现 |
| `pkg/server/interceptor` | gRPC 拦截器 | 已实现 |
| `pkg/config` | 配置加载、校验、热更新 | 已实现 |
| `pkg/log` | Zap 日志封装 | 已实现 |
| `pkg/cache` | 内存/Redis/多级缓存 | 已实现 |
| `pkg/database` | MySQL、慢查询记录 | 已实现 |
| `pkg/auth` | JWT、APIKey 鉴权 | 已实现 |
| `pkg/discovery` | Consul 服务发现 | 已实现 |
| `pkg/mq` | NATS 消息队列 | 已实现 |
| `pkg/errors` | 业务错误码 | 已实现 |
| `pkg/response` | 统一响应结构 | 已实现 |

## 目录结构

```
├── cmd/goupter/     # CLI 工具
├── pkg/             # 框架核心模块
├── examples/        # 示例项目
├── model/           # 生成的模型（示例）
└── doc/             # 文档
```

## 示例 API

| 接口 | 说明 |
|------|------|
| `GET /api/v1/ping` | Ping |
| `GET /api/v1/config` | 获取配置 |
| `POST /api/v1/auth/login` | 登录 |
| `GET /api/v1/me` | 当前用户（需 Token） |
| `GET /api/v1/admin/only` | 管理员接口 |
| `PUT /api/v1/cache/:key` | 设置缓存 |
| `GET /api/v1/cache/:key` | 获取缓存 |
| `DELETE /api/v1/cache/:key` | 删除缓存 |

## 进一步阅读

- 发布指南：`doc/RELEASE.md`
