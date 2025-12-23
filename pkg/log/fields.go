package log

import (
	"time"
)

// Uint creates a uint field
func Uint(key string, val uint) Field {
	return Field{Key: key, Value: val}
}

// Uint64 creates a uint64 field
func Uint64(key string, val uint64) Field {
	return Field{Key: key, Value: val}
}

// Int32 creates an int32 field
func Int32(key string, val int32) Field {
	return Field{Key: key, Value: val}
}

// Float32 creates a float32 field
func Float32(key string, val float32) Field {
	return Field{Key: key, Value: val}
}

// Duration creates a duration field
func Duration(key string, val time.Duration) Field {
	return Field{Key: key, Value: val.String()}
}

// DurationMs creates a duration field in milliseconds
func DurationMs(key string, val time.Duration) Field {
	return Field{Key: key, Value: val.Milliseconds()}
}

// Time creates a time field
func Time(key string, val time.Time) Field {
	return Field{Key: key, Value: val.Format(time.RFC3339Nano)}
}

// Bytes creates a byte slice field
func Bytes(key string, val []byte) Field {
	return Field{Key: key, Value: val}
}

// Strings creates a string slice field
func Strings(key string, val []string) Field {
	return Field{Key: key, Value: val}
}

// Ints creates an int slice field
func Ints(key string, val []int) Field {
	return Field{Key: key, Value: val}
}

// UserID creates a user_id field
func UserID(val string) Field {
	return String("user_id", val)
}

// RequestID creates a request_id field
func RequestID(val string) Field {
	return String("request_id", val)
}

// TraceID creates a trace_id field
func TraceID(val string) Field {
	return String("trace_id", val)
}

// Method creates a method field
func Method(val string) Field {
	return String("method", val)
}

// Path creates a path field
func Path(val string) Field {
	return String("path", val)
}

// StatusCode creates a status_code field
func StatusCode(val int) Field {
	return Int("status_code", val)
}

// Latency creates a latency field
func Latency(val time.Duration) Field {
	return Duration("latency", val)
}

// LatencyMs creates a latency_ms field
func LatencyMs(val time.Duration) Field {
	return DurationMs("latency_ms", val)
}

// IP creates an ip field
func IP(val string) Field {
	return String("ip", val)
}

// UserAgent creates a user_agent field
func UserAgent(val string) Field {
	return String("user_agent", val)
}
