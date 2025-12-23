package log

import (
	"sync"
	"testing"
	"time"
)

func TestNewEntry(t *testing.T) {
	entry := &Entry{
		Level:   InfoLevel,
		Message: "test message",
		Fields:  []Field{String("key", "value")},
		Time:    time.Now().UnixNano(),
		Caller:  "test_file.go:123",
	}

	if entry.Level != InfoLevel {
		t.Errorf("Expected InfoLevel, got %v", entry.Level)
	}
	if entry.Message != "test message" {
		t.Errorf("Expected 'test message', got %s", entry.Message)
	}
	if len(entry.Fields) != 1 {
		t.Errorf("Expected 1 field, got %d", len(entry.Fields))
	}
}

func TestNewEntry_Function(t *testing.T) {
	entry := NewEntry(InfoLevel, "test", String("key", "value"))
	if entry == nil {
		t.Fatal("NewEntry should return non-nil")
	}
	if entry.Level != InfoLevel {
		t.Errorf("Expected InfoLevel, got %v", entry.Level)
	}
	if entry.Message != "test" {
		t.Errorf("Expected 'test', got %s", entry.Message)
	}
}

func TestEntry_GetField(t *testing.T) {
	entry := NewEntry(InfoLevel, "test", String("key1", "value1"), Int("key2", 42))

	val, ok := entry.GetField("key1")
	if !ok {
		t.Error("GetField should find key1")
	}
	if val != "value1" {
		t.Errorf("Expected 'value1', got %v", val)
	}

	val, ok = entry.GetField("key2")
	if !ok {
		t.Error("GetField should find key2")
	}
	if val != 42 {
		t.Errorf("Expected 42, got %v", val)
	}

	_, ok = entry.GetField("nonexistent")
	if ok {
		t.Error("GetField should not find nonexistent key")
	}
}

func TestNewHookManager(t *testing.T) {
	manager := NewHookManager()
	if manager == nil {
		t.Fatal("NewHookManager should return non-nil")
	}
}

func TestHookManager_Add(t *testing.T) {
	manager := NewHookManager()

	hook := &testHook{levels: []Level{InfoLevel}}
	manager.Add(hook)

	entry := &Entry{Level: InfoLevel, Message: "test"}
	manager.Fire(InfoLevel, entry)

	if hook.fireCount != 1 {
		t.Errorf("Expected hook to be fired once, got %d", hook.fireCount)
	}
}

func TestHookManager_Fire(t *testing.T) {
	manager := NewHookManager()

	hook1 := &testHook{levels: []Level{InfoLevel, WarnLevel}}
	hook2 := &testHook{levels: []Level{ErrorLevel}}

	manager.Add(hook1)
	manager.Add(hook2)

	manager.Fire(InfoLevel, &Entry{Level: InfoLevel})
	if hook1.fireCount != 1 || hook2.fireCount != 0 {
		t.Errorf("Expected hook1=1, hook2=0, got hook1=%d, hook2=%d", hook1.fireCount, hook2.fireCount)
	}

	manager.Fire(ErrorLevel, &Entry{Level: ErrorLevel})
	if hook1.fireCount != 1 || hook2.fireCount != 1 {
		t.Errorf("Expected hook1=1, hook2=1, got hook1=%d, hook2=%d", hook1.fireCount, hook2.fireCount)
	}
}

func TestNewFuncHook(t *testing.T) {
	called := false
	hook := NewFuncHook([]Level{InfoLevel, WarnLevel}, func(entry *Entry) error {
		called = true
		return nil
	})

	if hook == nil {
		t.Fatal("NewFuncHook should return non-nil")
	}

	levels := hook.Levels()
	if len(levels) != 2 {
		t.Errorf("Expected 2 levels, got %d", len(levels))
	}

	hook.Fire(&Entry{Level: InfoLevel})
	if !called {
		t.Error("FuncHook should call the function")
	}
}

func TestNewErrorHook(t *testing.T) {
	var capturedEntry *Entry
	hook := NewErrorHook(func(entry *Entry) error {
		capturedEntry = entry
		return nil
	})

	if hook == nil {
		t.Fatal("NewErrorHook should return non-nil")
	}

	levels := hook.Levels()
	if len(levels) != 2 {
		t.Errorf("Expected 2 levels, got %d", len(levels))
	}

	entry := &Entry{Level: ErrorLevel, Message: "error message"}
	hook.Fire(entry)

	if capturedEntry == nil {
		t.Error("ErrorHook should capture error entries")
	}
	if capturedEntry.Message != "error message" {
		t.Errorf("Expected 'error message', got %s", capturedEntry.Message)
	}
}

func TestNewAsyncHook(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)

	fired := false
	innerHook := &testHook{
		levels: []Level{InfoLevel},
		onFire: func() {
			fired = true
			wg.Done()
		},
	}

	hook := NewAsyncHook(innerHook, 10)
	if hook == nil {
		t.Fatal("NewAsyncHook should return non-nil")
	}
	defer hook.Close()

	hook.Fire(&Entry{Level: InfoLevel})

	wg.Wait()

	if !fired {
		t.Error("AsyncHook should fire inner hook asynchronously")
	}
}

func TestAsyncHook_Levels(t *testing.T) {
	innerHook := &testHook{levels: []Level{InfoLevel, WarnLevel}}
	hook := NewAsyncHook(innerHook, 10)
	defer hook.Close()

	levels := hook.Levels()
	if len(levels) != 2 {
		t.Errorf("Expected 2 levels, got %d", len(levels))
	}
}

func TestNewHookedLogger(t *testing.T) {
	logger := Default()

	hookedLogger := NewHookedLogger(logger)
	if hookedLogger == nil {
		t.Fatal("NewHookedLogger should return non-nil")
	}
}

func TestHookedLogger_AddHook(t *testing.T) {
	logger := Default()
	hookedLogger := NewHookedLogger(logger)

	hook := &testHook{levels: []Level{InfoLevel}}
	result := hookedLogger.AddHook(hook)

	if result != hookedLogger {
		t.Error("AddHook should return the same logger for chaining")
	}
}

func TestHookedLogger_Methods(t *testing.T) {
	logger := Default()
	hookedLogger := NewHookedLogger(logger)

	hook := &testHook{levels: []Level{InfoLevel, WarnLevel, ErrorLevel, DebugLevel}}
	hookedLogger.AddHook(hook)

	hookedLogger.Debug("debug msg")
	hookedLogger.Info("info msg")
	hookedLogger.Warn("warn msg")
	hookedLogger.Error("error msg")

	if hook.fireCount != 4 {
		t.Errorf("Expected 4 hook fires, got %d", hook.fireCount)
	}
}

func TestHookedLogger_With(t *testing.T) {
	logger := Default()
	hookedLogger := NewHookedLogger(logger)

	newLogger := hookedLogger.With(String("key", "value"))
	if newLogger == nil {
		t.Fatal("With should return non-nil logger")
	}
}

func TestHookedLogger_Unwrap(t *testing.T) {
	logger := Default()
	hookedLogger := NewHookedLogger(logger)

	unwrapped := hookedLogger.Unwrap()
	if unwrapped == nil {
		t.Fatal("Unwrap should return non-nil logger")
	}
}

func TestHookedLogger_SetGetLevel(t *testing.T) {
	logger := Default()
	hookedLogger := NewHookedLogger(logger)

	hookedLogger.SetLevel(WarnLevel)
	level := hookedLogger.GetLevel()

	if level != WarnLevel {
		t.Errorf("Expected WarnLevel, got %v", level)
	}
}

func TestHookedLogger_Sync(t *testing.T) {
	logger := Default()
	hookedLogger := NewHookedLogger(logger)

	err := hookedLogger.Sync()
	if err != nil {
		t.Logf("Sync returned error (may be expected): %v", err)
	}
}

// testHook is a simple Hook for testing
type testHook struct {
	levels    []Level
	fireCount int
	returnErr error
	onFire    func()
	mu        sync.Mutex
}

func (h *testHook) Levels() []Level {
	return h.levels
}

func (h *testHook) Fire(entry *Entry) error {
	h.mu.Lock()
	h.fireCount++
	h.mu.Unlock()

	if h.onFire != nil {
		h.onFire()
	}
	return h.returnErr
}
