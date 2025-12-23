package log

import (
	"context"
	"sync"
)

// Hook is the log hook interface
type Hook interface {
	Fire(entry *Entry) error
	Levels() []Level
}

// Entry represents a log entry
type Entry struct {
	Level   Level
	Message string
	Fields  []Field
	Time    int64
	Caller  string
}

// NewEntry creates a log entry
func NewEntry(level Level, msg string, fields ...Field) *Entry {
	return &Entry{
		Level:   level,
		Message: msg,
		Fields:  fields,
	}
}

// GetField retrieves a field value by key
func (e *Entry) GetField(key string) (interface{}, bool) {
	for _, f := range e.Fields {
		if f.Key == key {
			return f.Value, true
		}
	}
	return nil, false
}

// HookManager manages hooks by level
type HookManager struct {
	mu    sync.RWMutex
	hooks map[Level][]Hook
}

// NewHookManager creates a hook manager
func NewHookManager() *HookManager {
	return &HookManager{
		hooks: make(map[Level][]Hook),
	}
}

// Add registers a hook
func (m *HookManager) Add(hook Hook) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, level := range hook.Levels() {
		m.hooks[level] = append(m.hooks[level], hook)
	}
}

// Fire triggers all hooks for a level
func (m *HookManager) Fire(level Level, entry *Entry) error {
	m.mu.RLock()
	hooks := m.hooks[level]
	m.mu.RUnlock()

	for _, hook := range hooks {
		if err := hook.Fire(entry); err != nil {
			return err
		}
	}
	return nil
}

// FuncHook wraps a function as a hook
type FuncHook struct {
	levels []Level
	fn     func(*Entry) error
}

// NewFuncHook creates a function hook
func NewFuncHook(levels []Level, fn func(*Entry) error) *FuncHook {
	return &FuncHook{
		levels: levels,
		fn:     fn,
	}
}

func (h *FuncHook) Fire(entry *Entry) error {
	return h.fn(entry)
}

func (h *FuncHook) Levels() []Level {
	return h.levels
}

// ErrorHook handles Error and Fatal levels
type ErrorHook struct {
	fn func(*Entry) error
}

// NewErrorHook creates an error hook
func NewErrorHook(fn func(*Entry) error) *ErrorHook {
	return &ErrorHook{fn: fn}
}

func (h *ErrorHook) Fire(entry *Entry) error {
	return h.fn(entry)
}

func (h *ErrorHook) Levels() []Level {
	return []Level{ErrorLevel, FatalLevel}
}

// AsyncHook wraps a hook for async execution
type AsyncHook struct {
	hook   Hook
	ch     chan *Entry
	ctx    context.Context
	cancel context.CancelFunc
}

// NewAsyncHook creates an async hook
func NewAsyncHook(hook Hook, bufferSize int) *AsyncHook {
	ctx, cancel := context.WithCancel(context.Background())
	h := &AsyncHook{
		hook:   hook,
		ch:     make(chan *Entry, bufferSize),
		ctx:    ctx,
		cancel: cancel,
	}
	go h.run()
	return h
}

func (h *AsyncHook) run() {
	for {
		select {
		case <-h.ctx.Done():
			return
		case entry := <-h.ch:
			_ = h.hook.Fire(entry)
		}
	}
}

func (h *AsyncHook) Fire(entry *Entry) error {
	select {
	case h.ch <- entry:
	default:
	}
	return nil
}

func (h *AsyncHook) Levels() []Level {
	return h.hook.Levels()
}

// Close stops the async hook
func (h *AsyncHook) Close() {
	h.cancel()
}

// HookedLogger wraps a Logger with hooks
type HookedLogger struct {
	logger Logger
	hooks  *HookManager
}

// NewHookedLogger creates a hooked logger
func NewHookedLogger(logger Logger) *HookedLogger {
	return &HookedLogger{
		logger: logger,
		hooks:  NewHookManager(),
	}
}

// AddHook adds a hook
func (l *HookedLogger) AddHook(hook Hook) *HookedLogger {
	l.hooks.Add(hook)
	return l
}

func (l *HookedLogger) fireHooks(level Level, msg string, fields ...Field) {
	entry := &Entry{
		Level:   level,
		Message: msg,
		Fields:  fields,
	}
	_ = l.hooks.Fire(level, entry)
}

func (l *HookedLogger) Debug(msg string, fields ...Field) {
	l.fireHooks(DebugLevel, msg, fields...)
	l.logger.Debug(msg, fields...)
}

func (l *HookedLogger) Info(msg string, fields ...Field) {
	l.fireHooks(InfoLevel, msg, fields...)
	l.logger.Info(msg, fields...)
}

func (l *HookedLogger) Warn(msg string, fields ...Field) {
	l.fireHooks(WarnLevel, msg, fields...)
	l.logger.Warn(msg, fields...)
}

func (l *HookedLogger) Error(msg string, fields ...Field) {
	l.fireHooks(ErrorLevel, msg, fields...)
	l.logger.Error(msg, fields...)
}

func (l *HookedLogger) Fatal(msg string, fields ...Field) {
	l.fireHooks(FatalLevel, msg, fields...)
	l.logger.Fatal(msg, fields...)
}

func (l *HookedLogger) With(fields ...Field) Logger {
	return &HookedLogger{
		logger: l.logger.With(fields...),
		hooks:  l.hooks,
	}
}

func (l *HookedLogger) WithContext(ctx context.Context) Logger {
	return &HookedLogger{
		logger: l.logger.WithContext(ctx),
		hooks:  l.hooks,
	}
}

func (l *HookedLogger) SetLevel(level Level) {
	l.logger.SetLevel(level)
}

func (l *HookedLogger) GetLevel() Level {
	return l.logger.GetLevel()
}

func (l *HookedLogger) Sync() error {
	return l.logger.Sync()
}

// Unwrap returns the underlying Logger
func (l *HookedLogger) Unwrap() Logger {
	return l.logger
}
