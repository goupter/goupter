package log

import (
	"testing"
)

func TestSensitiveType(t *testing.T) {
	// Test SensitiveType constants are defined
	tests := []struct {
		sType SensitiveType
		name  string
	}{
		{SensitivePhone, "SensitivePhone"},
		{SensitiveIDCard, "SensitiveIDCard"},
		{SensitiveEmail, "SensitiveEmail"},
		{SensitiveBankCard, "SensitiveBankCard"},
		{SensitivePassword, "SensitivePassword"},
		{SensitiveToken, "SensitiveToken"},
		{SensitiveCustom, "SensitiveCustom"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just verify the constant is defined and has a value
			_ = tt.sType
		})
	}
}

func TestNewDefaultMasker(t *testing.T) {
	masker := NewDefaultMasker()
	if masker == nil {
		t.Fatal("NewDefaultMasker should return non-nil")
	}
}

func TestDefaultMasker_MaskPhone(t *testing.T) {
	masker := NewDefaultMasker()

	tests := []struct {
		input    string
		expected string
	}{
		{"13812345678", "138****5678"},
		{"15900001111", "159****1111"},
		{"12345", "12345"}, // 不符合手机号格式，不脱敏
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := masker.Mask(tt.input)
			if got != tt.expected {
				t.Errorf("Mask(%s) = %s, want %s", tt.input, got, tt.expected)
			}
		})
	}
}

func TestDefaultMasker_MaskIDCard(t *testing.T) {
	masker := NewDefaultMasker()

	tests := []struct {
		input    string
		expected string
	}{
		{"110101199001011234", "110101199****11234"},
		{"11010119900101123X", "110101199****1123X"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := masker.Mask(tt.input)
			if got != tt.expected {
				t.Errorf("Mask(%s) = %s, want %s", tt.input, got, tt.expected)
			}
		})
	}
}

func TestDefaultMasker_MaskEmail(t *testing.T) {
	masker := NewDefaultMasker()

	tests := []struct {
		input    string
		expected string
	}{
		{"test@example.com", "t***@example.com"},
		{"user123@gmail.com", "u******@gmail.com"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := masker.Mask(tt.input)
			if got != tt.expected {
				t.Errorf("Mask(%s) = %s, want %s", tt.input, got, tt.expected)
			}
		})
	}
}

func TestDefaultMasker_MaskBankCard(t *testing.T) {
	masker := NewDefaultMasker()

	tests := []struct {
		input    string
		expected string
	}{
		{"6222021234567890123", "622202********90123"},
		{"6222021234567890", "622202*****67890"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := masker.Mask(tt.input)
			if got != tt.expected {
				t.Errorf("Mask(%s) = %s, want %s", tt.input, got, tt.expected)
			}
		})
	}
}

func TestDefaultMasker_MaskField(t *testing.T) {
	masker := NewDefaultMasker()

	tests := []struct {
		key      string
		value    string
		expected string
	}{
		{"phone", "13812345678", "138****5678"},
		{"mobile", "13812345678", "138****5678"},
		{"id_card", "110101199001011234", "110101********1234"},
		{"email", "test@example.com", "t***@example.com"},
		{"password", "secret123", "********"},
		{"token", "abc123xyz", "********"},
		{"api_key", "key123", "********"},
		{"secret", "mysecret", "********"},
		{"regular_field", "normal value", "normal value"}, // 普通字段不脱敏
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			got := masker.MaskField(tt.key, tt.value)
			if got != tt.expected {
				t.Errorf("MaskField(%s, %s) = %v, want %s", tt.key, tt.value, got, tt.expected)
			}
		})
	}
}

func TestDefaultMasker_AddRule(t *testing.T) {
	masker := NewDefaultMasker()

	// 添加自定义规则
	err := masker.AddRule(`custom_\d+`, func(value string) string {
		return "MASKED"
	})
	if err != nil {
		t.Fatalf("AddRule failed: %v", err)
	}

	got := masker.Mask("custom_12345")
	if got != "MASKED" {
		t.Errorf("Mask with custom rule = %s, want MASKED", got)
	}
}

func TestDefaultMasker_AddRule_InvalidPattern(t *testing.T) {
	masker := NewDefaultMasker()

	// 无效的正则表达式
	err := masker.AddRule(`[invalid`, func(value string) string {
		return "MASKED"
	})
	if err == nil {
		t.Error("AddRule with invalid pattern should return error")
	}
}

func TestDefaultMasker_AddSensitiveKey(t *testing.T) {
	masker := NewDefaultMasker()

	masker.AddSensitiveKey("custom_field", SensitivePassword)

	got := masker.MaskField("custom_field", "sensitive_value")
	if got != "********" {
		t.Errorf("MaskField with custom key = %v, want ********", got)
	}
}

func TestSetGlobalMasker(t *testing.T) {
	original := GetGlobalMasker()

	newMasker := NewDefaultMasker()
	SetGlobalMasker(newMasker)

	got := GetGlobalMasker()
	if got != newMasker {
		t.Error("SetGlobalMasker did not update global masker")
	}

	// 恢复原始
	SetGlobalMasker(original)
}

func TestGetGlobalMasker(t *testing.T) {
	masker := GetGlobalMasker()
	if masker == nil {
		t.Fatal("GetGlobalMasker should return non-nil")
	}
}

func TestMaskString(t *testing.T) {
	// 测试全局 MaskString 函数
	got := MaskString("13812345678")
	// 应该脱敏手机号
	if got == "13812345678" {
		t.Error("MaskString should mask phone number")
	}
}

func TestMaskValue(t *testing.T) {
	// 测试全局 MaskValue 函数
	got := MaskValue("password", "secret123")
	if got != "********" {
		t.Errorf("MaskValue(password) = %v, want ********", got)
	}
}

func TestNewMaskedLogger(t *testing.T) {
	logger := Default()
	masker := NewDefaultMasker()

	maskedLogger := NewMaskedLogger(logger, masker)
	if maskedLogger == nil {
		t.Fatal("NewMaskedLogger should return non-nil")
	}
}

func TestNewMaskedLogger_NilMasker(t *testing.T) {
	logger := Default()

	maskedLogger := NewMaskedLogger(logger, nil)
	if maskedLogger == nil {
		t.Fatal("NewMaskedLogger with nil masker should return non-nil")
	}
}

func TestMaskedLogger_Methods(t *testing.T) {
	logger := Default()
	masker := NewDefaultMasker()
	maskedLogger := NewMaskedLogger(logger, masker)

	// 验证不会 panic
	maskedLogger.Debug("debug msg")
	maskedLogger.Info("info msg")
	maskedLogger.Warn("warn msg")
	maskedLogger.Error("error msg")
}

func TestMaskedLogger_With(t *testing.T) {
	logger := Default()
	masker := NewDefaultMasker()
	maskedLogger := NewMaskedLogger(logger, masker)

	newLogger := maskedLogger.With(String("phone", "13812345678"))
	if newLogger == nil {
		t.Fatal("With should return non-nil logger")
	}
}

func TestMaskedLogger_MasksFields(t *testing.T) {
	logger := Default()
	masker := NewDefaultMasker()
	maskedLogger := NewMaskedLogger(logger, masker)

	// 验证不会 panic
	maskedLogger.Info("user login",
		String("phone", "13812345678"),
		String("password", "secret123"),
		String("email", "test@example.com"),
	)
}

func TestMaskedLogger_Sync(t *testing.T) {
	logger := Default()
	masker := NewDefaultMasker()
	maskedLogger := NewMaskedLogger(logger, masker)

	err := maskedLogger.Sync()
	if err != nil {
		// Sync 可能因为 stdout 而失败
		t.Logf("Sync returned error (may be expected): %v", err)
	}
}

func TestMaskedLogger_SetGetLevel(t *testing.T) {
	logger := Default()
	masker := NewDefaultMasker()
	maskedLogger := NewMaskedLogger(logger, masker)

	maskedLogger.SetLevel(WarnLevel)
	level := maskedLogger.GetLevel()

	if level != WarnLevel {
		t.Errorf("Expected WarnLevel, got %v", level)
	}
}

func TestMaskedLogger_Unwrap(t *testing.T) {
	logger := Default()
	masker := NewDefaultMasker()
	maskedLogger := NewMaskedLogger(logger, masker)

	unwrapped := maskedLogger.Unwrap()
	if unwrapped == nil {
		t.Fatal("Unwrap should return non-nil logger")
	}
}

func TestDefaultMasker_WithOptions(t *testing.T) {
	masker := NewDefaultMasker(
		WithMaskChar("#"),
		WithSensitiveKeys(map[string]SensitiveType{
			"custom_key": SensitivePassword,
		}),
	)

	if masker == nil {
		t.Fatal("NewDefaultMasker with options should return non-nil")
	}

	// 验证自定义脱敏字符
	got := masker.MaskField("password", "secret")
	if got != "########" {
		t.Errorf("MaskField with custom mask char = %v, want ########", got)
	}

	// 验证自定义敏感字段
	got = masker.MaskField("custom_key", "value")
	if got != "########" {
		t.Errorf("MaskField with custom key = %v, want ########", got)
	}
}
