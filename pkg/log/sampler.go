package log

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// Sampler determines whether a log entry should be sampled
type Sampler interface {
	Sample(level Level, msg string) bool
}

// RateSampler samples 1 out of every N messages
type RateSampler struct {
	rate    int64
	counter int64
}

// NewRateSampler creates a rate sampler
func NewRateSampler(rate int64) *RateSampler {
	if rate <= 0 {
		rate = 1
	}
	return &RateSampler{rate: rate}
}

// Sample implements Sampler
func (s *RateSampler) Sample(level Level, msg string) bool {
	n := atomic.AddInt64(&s.counter, 1)
	return n%s.rate == 1
}

// WindowSampler limits messages within a time window
type WindowSampler struct {
	mu       sync.Mutex
	window   time.Duration
	maxCount int
	counts   map[string]*windowCount
}

type windowCount struct {
	count     int
	resetTime time.Time
}

// NewWindowSampler creates a window sampler
func NewWindowSampler(window time.Duration, maxCount int) *WindowSampler {
	if maxCount <= 0 {
		maxCount = 1
	}
	s := &WindowSampler{
		window:   window,
		maxCount: maxCount,
		counts:   make(map[string]*windowCount),
	}
	go s.cleanup()
	return s
}

// Sample implements Sampler
func (s *WindowSampler) Sample(level Level, msg string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	key := level.String() + ":" + msg

	wc, ok := s.counts[key]
	if !ok {
		s.counts[key] = &windowCount{
			count:     1,
			resetTime: now.Add(s.window),
		}
		return true
	}

	if now.After(wc.resetTime) {
		wc.count = 1
		wc.resetTime = now.Add(s.window)
		return true
	}

	if wc.count < s.maxCount {
		wc.count++
		return true
	}

	return false
}

func (s *WindowSampler) cleanup() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.Lock()
		now := time.Now()
		for key, wc := range s.counts {
			if now.After(wc.resetTime) {
				delete(s.counts, key)
			}
		}
		s.mu.Unlock()
	}
}

// SampledLogger wraps a Logger with sampling
type SampledLogger struct {
	logger  Logger
	sampler Sampler
}

// NewSampledLogger creates a sampled logger
func NewSampledLogger(logger Logger, sampler Sampler) *SampledLogger {
	return &SampledLogger{
		logger:  logger,
		sampler: sampler,
	}
}

func (l *SampledLogger) Debug(msg string, fields ...Field) {
	if l.sampler.Sample(DebugLevel, msg) {
		l.logger.Debug(msg, fields...)
	}
}

func (l *SampledLogger) Info(msg string, fields ...Field) {
	if l.sampler.Sample(InfoLevel, msg) {
		l.logger.Info(msg, fields...)
	}
}

func (l *SampledLogger) Warn(msg string, fields ...Field) {
	if l.sampler.Sample(WarnLevel, msg) {
		l.logger.Warn(msg, fields...)
	}
}

func (l *SampledLogger) Error(msg string, fields ...Field) {
	if l.sampler.Sample(ErrorLevel, msg) {
		l.logger.Error(msg, fields...)
	}
}

func (l *SampledLogger) Fatal(msg string, fields ...Field) {
	l.logger.Fatal(msg, fields...)
}

func (l *SampledLogger) With(fields ...Field) Logger {
	return &SampledLogger{
		logger:  l.logger.With(fields...),
		sampler: l.sampler,
	}
}

func (l *SampledLogger) WithContext(ctx context.Context) Logger {
	return &SampledLogger{
		logger:  l.logger.WithContext(ctx),
		sampler: l.sampler,
	}
}

func (l *SampledLogger) SetLevel(level Level) {
	l.logger.SetLevel(level)
}

func (l *SampledLogger) GetLevel() Level {
	return l.logger.GetLevel()
}

func (l *SampledLogger) Sync() error {
	return l.logger.Sync()
}

// Unwrap returns the underlying Logger
func (l *SampledLogger) Unwrap() Logger {
	return l.logger
}
