package log

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestNewRateSampler(t *testing.T) {
	sampler := NewRateSampler(10)
	if sampler == nil {
		t.Fatal("NewRateSampler should return non-nil")
	}
}

func TestRateSampler_Sample(t *testing.T) {
	sampler := NewRateSampler(3)

	results := make([]bool, 9)
	for i := 0; i < 9; i++ {
		results[i] = sampler.Sample(InfoLevel, "test message")
	}

	count := 0
	for _, r := range results {
		if r {
			count++
		}
	}
	if count != 3 {
		t.Errorf("Expected 3 samples from 9 calls with rate 3, got %d", count)
	}
}

func TestRateSampler_ZeroRate(t *testing.T) {
	sampler := NewRateSampler(0)
	_ = sampler
}

func TestNewWindowSampler(t *testing.T) {
	sampler := NewWindowSampler(100*time.Millisecond, 5)
	if sampler == nil {
		t.Fatal("NewWindowSampler should return non-nil")
	}
}

func TestWindowSampler_Sample(t *testing.T) {
	sampler := NewWindowSampler(100*time.Millisecond, 3)

	for i := 0; i < 3; i++ {
		if !sampler.Sample(InfoLevel, "test") {
			t.Errorf("Message %d should be sampled", i+1)
		}
	}

	if sampler.Sample(InfoLevel, "test") {
		t.Error("Message 4 should not be sampled (exceeded window limit)")
	}

	time.Sleep(150 * time.Millisecond)

	if !sampler.Sample(InfoLevel, "test") {
		t.Error("Message after window reset should be sampled")
	}
}

func TestNewSampledLogger(t *testing.T) {
	logger := Default()
	sampler := NewRateSampler(2)

	sampledLogger := NewSampledLogger(logger, sampler)
	if sampledLogger == nil {
		t.Fatal("NewSampledLogger should return non-nil")
	}
}

func TestSampledLogger_Methods(t *testing.T) {
	counter := &logCounter{}
	sampler := NewRateSampler(2)

	sampledLogger := NewSampledLogger(counter, sampler)

	sampledLogger.Info("test1")
	sampledLogger.Info("test2")
	sampledLogger.Info("test3")
	sampledLogger.Info("test4")

	if counter.infoCount != 2 {
		t.Errorf("Expected 2 info logs, got %d", counter.infoCount)
	}
}

func TestSampledLogger_With(t *testing.T) {
	logger := Default()
	sampler := NewRateSampler(2)

	sampledLogger := NewSampledLogger(logger, sampler)
	newLogger := sampledLogger.With(String("key", "value"))

	if newLogger == nil {
		t.Fatal("With should return non-nil logger")
	}
}

func TestSampledLogger_Concurrent(t *testing.T) {
	counter := &logCounter{}
	sampler := NewRateSampler(10)
	sampledLogger := NewSampledLogger(counter, sampler)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sampledLogger.Info("concurrent test")
		}()
	}
	wg.Wait()

	if counter.infoCount < 5 || counter.infoCount > 15 {
		t.Errorf("Expected around 10 logs, got %d", counter.infoCount)
	}
}

// logCounter is a simple Logger for testing
type logCounter struct {
	mu         sync.Mutex
	debugCount int
	infoCount  int
	warnCount  int
	errorCount int
	level      Level
}

func (l *logCounter) Debug(msg string, fields ...Field) {
	l.mu.Lock()
	l.debugCount++
	l.mu.Unlock()
}

func (l *logCounter) Info(msg string, fields ...Field) {
	l.mu.Lock()
	l.infoCount++
	l.mu.Unlock()
}

func (l *logCounter) Warn(msg string, fields ...Field) {
	l.mu.Lock()
	l.warnCount++
	l.mu.Unlock()
}

func (l *logCounter) Error(msg string, fields ...Field) {
	l.mu.Lock()
	l.errorCount++
	l.mu.Unlock()
}

func (l *logCounter) Fatal(msg string, fields ...Field) {}

func (l *logCounter) With(fields ...Field) Logger {
	return l
}

func (l *logCounter) WithContext(ctx context.Context) Logger {
	return l
}

func (l *logCounter) SetLevel(level Level) {
	l.level = level
}

func (l *logCounter) GetLevel() Level {
	return l.level
}

func (l *logCounter) Sync() error {
	return nil
}
