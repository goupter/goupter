package log

import (
	"context"
	"sync"
)

// NamedLogger wraps a Logger with a name field
type NamedLogger struct {
	Logger
	name string
}

// NewNamedLogger creates a named logger
func NewNamedLogger(name string, logger Logger) *NamedLogger {
	return &NamedLogger{
		Logger: logger.With(String("logger", name)),
		name:   name,
	}
}

// Name returns the logger name
func (l *NamedLogger) Name() string {
	return l.name
}

// With adds fields
func (l *NamedLogger) With(fields ...Field) Logger {
	return &NamedLogger{
		Logger: l.Logger.With(fields...),
		name:   l.name,
	}
}

// WithContext adds context
func (l *NamedLogger) WithContext(ctx context.Context) Logger {
	return &NamedLogger{
		Logger: l.Logger.WithContext(ctx),
		name:   l.name,
	}
}

// Named creates a child logger with sub-name
func (l *NamedLogger) Named(name string) *NamedLogger {
	childName := l.name + "." + name
	return &NamedLogger{
		Logger: l.Logger.With(String("logger", childName)),
		name:   childName,
	}
}

// Global named logger registry
var (
	namedLoggers = make(map[string]*NamedLogger)
	namedMu      sync.RWMutex
)

// Named returns a named logger, creating one if it doesn't exist
func Named(name string) *NamedLogger {
	namedMu.RLock()
	logger, ok := namedLoggers[name]
	namedMu.RUnlock()

	if ok {
		return logger
	}

	namedMu.Lock()
	defer namedMu.Unlock()

	if logger, ok := namedLoggers[name]; ok {
		return logger
	}

	logger = NewNamedLogger(name, Default())
	namedLoggers[name] = logger
	return logger
}

// RegisterNamed registers a named logger
func RegisterNamed(name string, logger *NamedLogger) {
	namedMu.Lock()
	namedLoggers[name] = logger
	namedMu.Unlock()
}
