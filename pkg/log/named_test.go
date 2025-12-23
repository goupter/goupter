package log

import (
	"testing"
)

func TestNewNamedLogger(t *testing.T) {
	logger := Default()
	named := NewNamedLogger("test", logger)

	if named == nil {
		t.Fatal("NewNamedLogger should return non-nil")
	}
}

func TestNamedLogger_Name(t *testing.T) {
	logger := Default()
	named := NewNamedLogger("my-service", logger)

	if named.Name() != "my-service" {
		t.Errorf("Expected name 'my-service', got '%s'", named.Name())
	}
}

func TestNamedLogger_Methods(t *testing.T) {
	logger := Default()
	named := NewNamedLogger("test", logger)

	named.Debug("debug msg")
	named.Info("info msg")
	named.Warn("warn msg")
	named.Error("error msg")
}

func TestNamedLogger_With(t *testing.T) {
	logger := Default()
	named := NewNamedLogger("test", logger)

	newLogger := named.With(String("key", "value"))
	if newLogger == nil {
		t.Fatal("With should return non-nil logger")
	}
}

func TestNamedLogger_Named(t *testing.T) {
	logger := Default()
	named := NewNamedLogger("parent", logger)

	child := named.Named("child")
	if child == nil {
		t.Fatal("Named should return non-nil")
	}
	if child.Name() != "parent.child" {
		t.Errorf("Expected name 'parent.child', got '%s'", child.Name())
	}
}

func TestNamed(t *testing.T) {
	logger := Named("test-service")
	if logger == nil {
		t.Fatal("Named should return non-nil logger")
	}

	logger2 := Named("test-service")
	if logger != logger2 {
		t.Error("Named should return same instance for same name")
	}
}

func TestRegisterNamed(t *testing.T) {
	customLogger := NewNamedLogger("custom", Default())
	RegisterNamed("custom-service", customLogger)

	got := Named("custom-service")
	if got != customLogger {
		t.Error("RegisterNamed should update named logger")
	}
}
