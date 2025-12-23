package server

import (
	"crypto/tls"
	"testing"
)

// === 默认配置测试 ===

func TestDefaultTLSConfig(t *testing.T) {
	cfg := DefaultTLSConfig()

	if cfg.MinVersion != tls.VersionTLS12 {
		t.Errorf("MinVersion应该是TLS1.2，实际: %d", cfg.MinVersion)
	}
	if cfg.MaxVersion != tls.VersionTLS13 {
		t.Errorf("MaxVersion应该是TLS1.3，实际: %d", cfg.MaxVersion)
	}
	if len(cfg.CipherSuites) == 0 {
		t.Error("应该有默认的密码套件")
	}
}

// === 构建TLS配置测试 ===

func TestBuildTLSConfig_Basic(t *testing.T) {
	cfg := DefaultTLSConfig()
	cfg.InsecureSkipVerify = true
	cfg.ServerName = "localhost"

	tlsConfig, err := cfg.BuildTLSConfig()
	if err != nil {
		t.Errorf("不应该返回错误: %v", err)
	}

	if !tlsConfig.InsecureSkipVerify {
		t.Error("InsecureSkipVerify应该是true")
	}
	if tlsConfig.ServerName != "localhost" {
		t.Errorf("ServerName不正确: %s", tlsConfig.ServerName)
	}
	if tlsConfig.MinVersion != tls.VersionTLS12 {
		t.Errorf("MinVersion不正确: %d", tlsConfig.MinVersion)
	}
}

func TestBuildTLSConfig_InvalidCertFile(t *testing.T) {
	cfg := DefaultTLSConfig()
	cfg.CertFile = "/nonexistent/cert.pem"
	cfg.KeyFile = "/nonexistent/key.pem"

	_, err := cfg.BuildTLSConfig()
	if err == nil {
		t.Error("应该返回错误")
	}
}

func TestBuildTLSConfig_InvalidCAFile(t *testing.T) {
	cfg := DefaultTLSConfig()
	cfg.CAFile = "/nonexistent/ca.pem"

	_, err := cfg.BuildTLSConfig()
	if err == nil {
		t.Error("应该返回错误")
	}
}

// === 服务端TLS配置测试 ===

func TestBuildServerTLSConfig_NoCert(t *testing.T) {
	cfg := DefaultTLSConfig()

	_, err := cfg.BuildServerTLSConfig()
	if err == nil {
		t.Error("没有证书应该返回错误")
	}
}

// === 客户端TLS配置测试 ===

func TestBuildClientTLSConfig(t *testing.T) {
	cfg := DefaultTLSConfig()
	cfg.InsecureSkipVerify = true

	tlsConfig, err := cfg.BuildClientTLSConfig()
	if err != nil {
		t.Errorf("不应该返回错误: %v", err)
	}
	if !tlsConfig.InsecureSkipVerify {
		t.Error("客户端配置InsecureSkipVerify应该是true")
	}
}

// === gRPC凭证测试 ===

func TestBuildGRPCServerCredentials_NoCert(t *testing.T) {
	cfg := DefaultTLSConfig()

	_, err := cfg.BuildGRPCServerCredentials()
	if err == nil {
		t.Error("没有证书应该返回错误")
	}
}

func TestBuildGRPCClientCredentials(t *testing.T) {
	cfg := DefaultTLSConfig()
	cfg.InsecureSkipVerify = true

	creds, err := cfg.BuildGRPCClientCredentials()
	if err != nil {
		t.Errorf("不应该返回错误: %v", err)
	}
	if creds == nil {
		t.Error("凭证不应该为nil")
	}
}

// === 便捷函数测试 ===

func TestNewServerTLSConfig_InvalidFiles(t *testing.T) {
	_, err := NewServerTLSConfig("/nonexistent/cert.pem", "/nonexistent/key.pem")
	if err == nil {
		t.Error("无效文件应该返回错误")
	}
}

func TestNewServerMTLSConfig_InvalidFiles(t *testing.T) {
	_, err := NewServerMTLSConfig("/nonexistent/cert.pem", "/nonexistent/key.pem", "/nonexistent/ca.pem")
	if err == nil {
		t.Error("无效文件应该返回错误")
	}
}

func TestNewClientTLSConfig_InvalidCAFile(t *testing.T) {
	_, err := NewClientTLSConfig("/nonexistent/ca.pem", "localhost")
	if err == nil {
		t.Error("无效CA文件应该返回错误")
	}
}

func TestNewClientMTLSConfig_InvalidFiles(t *testing.T) {
	_, err := NewClientMTLSConfig("/nonexistent/cert.pem", "/nonexistent/key.pem", "/nonexistent/ca.pem", "localhost")
	if err == nil {
		t.Error("无效文件应该返回错误")
	}
}

func TestNewInsecureClientTLSConfig(t *testing.T) {
	cfg := NewInsecureClientTLSConfig()
	if !cfg.InsecureSkipVerify {
		t.Error("不安全配置应该设置InsecureSkipVerify为true")
	}
}

// === PEM加载测试 ===

func TestLoadCertificateFromPEM_Invalid(t *testing.T) {
	_, err := LoadCertificateFromPEM([]byte("invalid"), []byte("invalid"))
	if err == nil {
		t.Error("无效PEM应该返回错误")
	}
}

func TestLoadCACertPool_Invalid(t *testing.T) {
	_, err := LoadCACertPool([]byte("invalid"))
	if err == nil {
		t.Error("无效CA PEM应该返回错误")
	}
}

func TestLoadCACertPoolFromFile_NotExist(t *testing.T) {
	_, err := LoadCACertPoolFromFile("/nonexistent/ca.pem")
	if err == nil {
		t.Error("不存在的文件应该返回错误")
	}
}

// === 自签名证书配置测试 ===

func TestDefaultSelfSignedCertConfig(t *testing.T) {
	cfg := DefaultSelfSignedCertConfig()

	if len(cfg.Organization) == 0 {
		t.Error("应该有默认Organization")
	}
	if cfg.CommonName != "localhost" {
		t.Errorf("CommonName应该是localhost，实际: %s", cfg.CommonName)
	}
	if cfg.ValidDays != 365 {
		t.Errorf("ValidDays应该是365，实际: %d", cfg.ValidDays)
	}
	if cfg.KeyBits != 2048 {
		t.Errorf("KeyBits应该是2048，实际: %d", cfg.KeyBits)
	}
}
