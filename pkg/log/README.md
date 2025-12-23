# Log Package

A high-performance logging library based on [zap](https://github.com/uber-go/zap), providing structured logging, sampling, hooks, sensitive data masking, and framework adapters.

## Features

- **Structured Logging**: Type-safe field-based logging
- **Multiple Output**: stdout, stderr, file, or combined
- **Log Rotation**: Built-in file rotation via lumberjack
- **Log Sampling**: Rate and time window sampling
- **Hooks**: Extensible hook system
- **Sensitive Data Masking**: Auto-mask phone, ID card, email, password, token
- **Named Logger**: Hierarchical logger organization
- **Framework Adapters**: GORM, gRPC, standard library

## Quick Start

```go
// Initialize
log.Init(&log.LoggerConfig{
    Level:  log.InfoLevel,
    Format: "json",
    Output: "stdout",
})

// Log messages
log.Info("server started", log.String("addr", ":8080"))
log.Error("request failed", log.Error(err), log.Int("status", 500))

defer log.Sync()
```

## With Context

```go
ctx := log.WithTraceID(context.Background(), "trace-123")
ctx = log.WithRequestID(ctx, "req-456")

logger := log.L(ctx)
logger.Info("processing request")
```

## Field Types

```go
// Basic types
log.String("key", "value")
log.Int("count", 42)
log.Int64("id", 123456789)
log.Float64("rate", 0.95)
log.Bool("enabled", true)
log.Error(err)
log.Any("data", obj)

// Time types
log.Duration("elapsed", time.Second)
log.Time("created_at", time.Now())

// Arrays
log.Strings("tags", []string{"a", "b"})
log.Ints("ids", []int{1, 2, 3})

// Business fields
log.UserID("user-123")
log.RequestID("req-456")
log.TraceID("trace-789")
log.Method("POST")
log.Path("/api/users")
log.StatusCode(200)
log.LatencyMs(time.Millisecond * 50)
```

## Log Sampling

```go
// Rate sampler: 1 out of every 100 messages
sampler := log.NewRateSampler(100)
logger := log.NewSampledLogger(log.Default(), sampler)

// Window sampler: max 10 same messages per minute
sampler := log.NewWindowSampler(time.Minute, 10)
logger := log.NewSampledLogger(log.Default(), sampler)
```

## Hooks

```go
logger := log.NewHookedLogger(log.Default())

// Error hook
logger.AddHook(log.NewErrorHook(func(entry *log.Entry) error {
    // Send alert, write to external system
    return nil
}))

// Async hook
asyncHook := log.NewAsyncHook(myHook, 1000)
defer asyncHook.Close()
logger.AddHook(asyncHook)
```

## Sensitive Data Masking

```go
masker := log.NewDefaultMasker()
logger := log.NewMaskedLogger(log.Default(), masker)

logger.Info("user login",
    log.String("phone", "13812345678"),    // -> 138****5678
    log.String("password", "secret123"),   // -> ********
)
```

## Named Logger

```go
userLogger := log.Named("user.service")
userLogger.Info("user created")
// Output: {"logger":"user.service","msg":"user created"}
```

## Framework Adapters

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

### Standard Library

```go
stdLogger := log.NewStdLogAdapter(log.Default(), log.InfoLevel)
http.Server{ErrorLog: stdLogger}
```

## API Reference

### Logger Interface

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

### Log Levels

```go
const (
    DebugLevel Level = iota - 1
    InfoLevel
    WarnLevel
    ErrorLevel
    FatalLevel
)
```

## License

MIT License
