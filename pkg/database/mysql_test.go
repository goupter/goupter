package database

import (
	"testing"
	"time"

	"github.com/goupter/goupter/pkg/config"
)

func TestMySQLOptions(t *testing.T) {
	t.Run("WithConfig", func(t *testing.T) {
		cfg := &config.DatabaseConfig{
			Host: "test-host",
			Port: 3307,
		}

		m := &MySQL{}
		opt := WithConfig(cfg)
		opt(m)

		if m.config.Host != "test-host" {
			t.Errorf("Host = %s, want test-host", m.config.Host)
		}
		if m.config.Port != 3307 {
			t.Errorf("Port = %d, want 3307", m.config.Port)
		}
	})

	t.Run("WithLogger", func(t *testing.T) {
		mockLog := &mockLogger{}
		m := &MySQL{}
		opt := WithLogger(mockLog)
		opt(m)

		if m.logger == nil {
			t.Error("logger should be set")
		}
	})
}

func TestMySQLStruct(t *testing.T) {
	m := &MySQL{
		config: &config.DatabaseConfig{
			Driver:             "mysql",
			Host:               "localhost",
			Port:               3306,
			Database:           "test",
			Username:           "root",
			Password:           "password",
			Charset:            "utf8mb4",
			MaxIdleConns:       10,
			MaxOpenConns:       100,
			ConnMaxLifetime:    time.Hour,
			SlowQueryThreshold: 200 * time.Millisecond,
			SlowQueryLog:       true,
		},
	}

	if m.config.Driver != "mysql" {
		t.Errorf("Driver = %s, want mysql", m.config.Driver)
	}
	if m.config.Host != "localhost" {
		t.Errorf("Host = %s, want localhost", m.config.Host)
	}
	if m.config.Port != 3306 {
		t.Errorf("Port = %d, want 3306", m.config.Port)
	}
}

func TestGormLoggerLevels(t *testing.T) {
	tests := []struct {
		level string
	}{
		{"silent"},
		{"error"},
		{"warn"},
		{"info"},
		{"unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.level, func(t *testing.T) {
			logger := newGormLogger(nil, tt.level)
			if logger == nil {
				t.Error("newGormLogger should return non-nil")
			}
		})
	}
}

func TestGetDB(t *testing.T) {
	originalDB := db
	defer func() { db = originalDB }()

	db = nil
	result := GetDB()
	if result != nil {
		t.Error("GetDB should return nil when db is nil")
	}
}

func TestSetDB(t *testing.T) {
	originalDB := db
	defer func() { db = originalDB }()

	SetDB(nil)
	if db != nil {
		t.Error("db should be nil after SetDB(nil)")
	}
}
