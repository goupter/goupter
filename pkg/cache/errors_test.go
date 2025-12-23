package cache

import (
	"errors"
	"testing"
)

func TestErrNotFound(t *testing.T) {
	if ErrNotFound.Error() != "cache: key not found" {
		t.Errorf("Unexpected error message: %s", ErrNotFound.Error())
	}
}

func TestNotFoundError(t *testing.T) {
	err := &NotFoundError{Key: "mykey"}

	expected := "cache: key not found: mykey"
	if err.Error() != expected {
		t.Errorf("Expected '%s', got '%s'", expected, err.Error())
	}
}

func TestIsNotFound(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil error", nil, false},
		{"NotFoundError", &NotFoundError{Key: "test"}, true},
		{"ErrNotFound", ErrNotFound, true},
		{"redis nil", errors.New("redis: nil"), true},
		{"other error", errors.New("other error"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsNotFound(tt.err); got != tt.want {
				t.Errorf("IsNotFound(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}
