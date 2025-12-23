# Log 日志包

基于 [zap](https://github.com/uber-go/zap) 的高性能日志库，提供结构化日志、采样、钩子、敏感数据脱敏和框架适配器。

## 特性

- **结构化日志**：类型安全的字段式日志
- **多输出目标**：stdout、stderr、文件或组合输出
- **日志轮转**：通过 lumberjack 内置文件轮转
- **日志采样**：频率和时间窗口采样
- **钩子系统**：可扩展的钩子系统
- **敏感数据脱敏**：自动脱敏手机号、身份证、邮箱、密码、Token
- **命名日志**：层级化的日志组织
- **框架适配器**：GORM、gRPC、标准库

## 快速开始

```go
// 初始化
log.Init(&log.LoggerConfig{
    Level:  log.InfoLevel,
    Format: "json",
    Output: "stdout",
})

// 记录日志
log.Info("服务启动", log.String("addr", ":8080"))
log.Error("请求失败", log.Error(err), log.Int("status", 500))

defer log.Sync()
```

## 带上下文

```go
ctx := log.WithTraceID(context.Background(), "trace-123")
ctx = log.WithRequestID(ctx, "req-456")

logger := log.L(ctx)
logger.Info("处理请求")
```

## 字段类型

```go
// 基本类型
log.String("key", "value")
log.Int("count", 42)
log.Int64("id", 123456789)
log.Float64("rate", 0.95)
log.Bool("enabled", true)
log.Error(err)
log.Any("data", obj)

// 时间类型
log.Duration("elapsed", time.Second)
log.Time("created_at", time.Now())

// 数组类型
log.Strings("tags", []string{"a", "b"})
log.Ints("ids", []int{1, 2, 3})

// 业务字段
log.UserID("user-123")
log.RequestID("req-456")
log.TraceID("trace-789")
log.Method("POST")
log.Path("/api/users")
log.StatusCode(200)
log.LatencyMs(time.Millisecond * 50)
```

## 日志采样

```go
// 频率采样器：每 100 条消息只输出 1 条
sampler := log.NewRateSampler(100)
logger := log.NewSampledLogger(log.Default(), sampler)

// 时间窗口采样器：每分钟最多输出 10 条相同消息
sampler := log.NewWindowSampler(time.Minute, 10)
logger := log.NewSampledLogger(log.Default(), sampler)
```

## 钩子

```go
logger := log.NewHookedLogger(log.Default())

// 错误钩子
logger.AddHook(log.NewErrorHook(func(entry *log.Entry) error {
    // 发送告警、写入外部系统
    return nil
}))

// 异步钩子
asyncHook := log.NewAsyncHook(myHook, 1000)
defer asyncHook.Close()
logger.AddHook(asyncHook)
```

## 敏感数据脱敏

```go
masker := log.NewDefaultMasker()
logger := log.NewMaskedLogger(log.Default(), masker)

logger.Info("用户登录",
    log.String("phone", "13812345678"),    // -> 138****5678
    log.String("password", "secret123"),   // -> ********
)
```

## 命名日志

```go
userLogger := log.Named("user.service")
userLogger.Info("用户创建")
// 输出: {"logger":"user.service","msg":"用户创建"}
```

## 框架适配器

### GORM

```go
gormLogger := log.NewGormLogger(log.Default(),
    log.WithGormSlowThreshold(200*time.Millisecond),
    log.WithGormIgnoreNotFound(true),
)
db, _ := gorm.Open(mysql.Open(dsn), &gorm.Config{
    Logger: gormLogger,
})
```

### gRPC

```go
grpcLogger := log.NewGrpcLogger(log.Default())
grpclog.SetLoggerV2(grpcLogger)
```

### 标准库

```go
stdLogger := log.NewStdLogAdapter(log.Default(), log.InfoLevel)
http.Server{ErrorLog: stdLogger}
```

## API 参考

### Logger 接口

```go
type Logger interface {
    Debug(msg string, fields ...Field)
    Info(msg string, fields ...Field)
    Warn(msg string, fields ...Field)
    Error(msg string, fields ...Field)
    Fatal(msg string, fields ...Field)
    With(fields ...Field) Logger
    WithContext(ctx context.Context) Logger
    SetLevel(level Level)
    GetLevel() Level
    Sync() error
}
```

### 日志级别

```go
const (
    DebugLevel Level = iota - 1
    InfoLevel
    WarnLevel
    ErrorLevel
    FatalLevel
)
```

## 许可证

MIT License
