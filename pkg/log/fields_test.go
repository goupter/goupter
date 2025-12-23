package log

import (
	"testing"
	"time"
)

func TestUint(t *testing.T) {
	f := Uint("count", 42)
	if f.Key != "count" || f.Value != uint(42) {
		t.Errorf("Uint() = {%s, %v}, want {count, 42}", f.Key, f.Value)
	}
}

func TestUint64(t *testing.T) {
	f := Uint64("big", 18446744073709551615)
	if f.Key != "big" {
		t.Errorf("Uint64() key = %s, want big", f.Key)
	}
}

func TestInt32(t *testing.T) {
	f := Int32("num", -2147483648)
	if f.Key != "num" || f.Value != int32(-2147483648) {
		t.Errorf("Int32() = {%s, %v}, want {num, -2147483648}", f.Key, f.Value)
	}
}

func TestFloat32(t *testing.T) {
	f := Float32("price", 19.99)
	if f.Key != "price" {
		t.Errorf("Float32() key = %s, want price", f.Key)
	}
}

func TestDuration(t *testing.T) {
	d := 5 * time.Second
	f := Duration("elapsed", d)
	if f.Key != "elapsed" {
		t.Errorf("Duration() key = %s, want elapsed", f.Key)
	}
	if f.Value != "5s" {
		t.Errorf("Duration() value = %v, want 5s", f.Value)
	}
}

func TestDurationMs(t *testing.T) {
	d := 1500 * time.Millisecond
	f := DurationMs("elapsed_ms", d)
	if f.Key != "elapsed_ms" || f.Value != int64(1500) {
		t.Errorf("DurationMs() = {%s, %v}, want {elapsed_ms, 1500}", f.Key, f.Value)
	}
}

func TestTime(t *testing.T) {
	now := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	f := Time("timestamp", now)
	if f.Key != "timestamp" {
		t.Errorf("Time() key = %s, want timestamp", f.Key)
	}
	expected := now.Format(time.RFC3339Nano)
	if f.Value != expected {
		t.Errorf("Time() value = %v, want %s", f.Value, expected)
	}
}

func TestBytes(t *testing.T) {
	data := []byte{0x48, 0x65, 0x6c, 0x6c, 0x6f}
	f := Bytes("data", data)
	if f.Key != "data" {
		t.Errorf("Bytes() key = %s, want data", f.Key)
	}
}

func TestStrings(t *testing.T) {
	vals := []string{"a", "b", "c"}
	f := Strings("items", vals)
	if f.Key != "items" {
		t.Errorf("Strings() key = %s, want items", f.Key)
	}
}

func TestInts(t *testing.T) {
	vals := []int{1, 2, 3}
	f := Ints("nums", vals)
	if f.Key != "nums" {
		t.Errorf("Ints() key = %s, want nums", f.Key)
	}
}

func TestUserID(t *testing.T) {
	f := UserID("user123")
	if f.Key != "user_id" || f.Value != "user123" {
		t.Errorf("UserID() = {%s, %v}, want {user_id, user123}", f.Key, f.Value)
	}
}

func TestRequestID(t *testing.T) {
	f := RequestID("req-456")
	if f.Key != "request_id" || f.Value != "req-456" {
		t.Errorf("RequestID() = {%s, %v}, want {request_id, req-456}", f.Key, f.Value)
	}
}

func TestTraceID(t *testing.T) {
	f := TraceID("trace-789")
	if f.Key != "trace_id" || f.Value != "trace-789" {
		t.Errorf("TraceID() = {%s, %v}, want {trace_id, trace-789}", f.Key, f.Value)
	}
}

func TestMethod(t *testing.T) {
	f := Method("POST")
	if f.Key != "method" || f.Value != "POST" {
		t.Errorf("Method() = {%s, %v}, want {method, POST}", f.Key, f.Value)
	}
}

func TestPath(t *testing.T) {
	f := Path("/api/users")
	if f.Key != "path" || f.Value != "/api/users" {
		t.Errorf("Path() = {%s, %v}, want {path, /api/users}", f.Key, f.Value)
	}
}

func TestStatusCode(t *testing.T) {
	f := StatusCode(200)
	if f.Key != "status_code" || f.Value != 200 {
		t.Errorf("StatusCode() = {%s, %v}, want {status_code, 200}", f.Key, f.Value)
	}
}

func TestLatency(t *testing.T) {
	d := 150 * time.Millisecond
	f := Latency(d)
	if f.Key != "latency" {
		t.Errorf("Latency() key = %s, want latency", f.Key)
	}
}

func TestLatencyMs(t *testing.T) {
	d := 150 * time.Millisecond
	f := LatencyMs(d)
	if f.Key != "latency_ms" || f.Value != int64(150) {
		t.Errorf("LatencyMs() = {%s, %v}, want {latency_ms, 150}", f.Key, f.Value)
	}
}

func TestIP(t *testing.T) {
	f := IP("192.168.1.1")
	if f.Key != "ip" || f.Value != "192.168.1.1" {
		t.Errorf("IP() = {%s, %v}, want {ip, 192.168.1.1}", f.Key, f.Value)
	}
}

func TestUserAgent(t *testing.T) {
	ua := "Mozilla/5.0"
	f := UserAgent(ua)
	if f.Key != "user_agent" || f.Value != ua {
		t.Errorf("UserAgent() = {%s, %v}, want {user_agent, %s}", f.Key, f.Value, ua)
	}
}
