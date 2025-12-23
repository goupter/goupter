package server

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

	"google.golang.org/grpc/credentials"
)

// TLSConfig TLS配置
type TLSConfig struct {
	// 证书文件路径
	CertFile string
	// 私钥文件路径
	KeyFile string
	// CA证书文件路径（用于mTLS）
	CAFile string
	// 是否启用mTLS（双向认证）
	EnableMTLS bool
	// 最低TLS版本
	MinVersion uint16
	// 最高TLS版本
	MaxVersion uint16
	// 密码套件
	CipherSuites []uint16
	// 是否跳过证书验证（仅用于测试）
	InsecureSkipVerify bool
	// 服务器名称（用于验证）
	ServerName string
}

// DefaultTLSConfig 默认TLS配置
func DefaultTLSConfig() *TLSConfig {
	return &TLSConfig{
		MinVersion: tls.VersionTLS12,
		MaxVersion: tls.VersionTLS13,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
		},
	}
}

// BuildTLSConfig 构建Go标准库TLS配置
func (c *TLSConfig) BuildTLSConfig() (*tls.Config, error) {
	tlsConfig := &tls.Config{
		MinVersion:         c.MinVersion,
		MaxVersion:         c.MaxVersion,
		InsecureSkipVerify: c.InsecureSkipVerify,
	}

	// 设置密码套件
	if len(c.CipherSuites) > 0 {
		tlsConfig.CipherSuites = c.CipherSuites
	}

	// 设置服务器名称
	if c.ServerName != "" {
		tlsConfig.ServerName = c.ServerName
	}

	// 加载证书
	if c.CertFile != "" && c.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(c.CertFile, c.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load certificate: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	// 加载CA证书（用于mTLS）
	if c.CAFile != "" {
		caCert, err := os.ReadFile(c.CAFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA certificate: %w", err)
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to parse CA certificate")
		}

		tlsConfig.RootCAs = caCertPool
		tlsConfig.ClientCAs = caCertPool

		if c.EnableMTLS {
			tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
		}
	}

	return tlsConfig, nil
}

// BuildServerTLSConfig 构建服务端TLS配置
func (c *TLSConfig) BuildServerTLSConfig() (*tls.Config, error) {
	if c.CertFile == "" || c.KeyFile == "" {
		return nil, fmt.Errorf("certificate and key files are required for server TLS")
	}

	tlsConfig, err := c.BuildTLSConfig()
	if err != nil {
		return nil, err
	}

	// 服务端特定配置
	tlsConfig.PreferServerCipherSuites = true

	return tlsConfig, nil
}

// BuildClientTLSConfig 构建客户端TLS配置
func (c *TLSConfig) BuildClientTLSConfig() (*tls.Config, error) {
	tlsConfig, err := c.BuildTLSConfig()
	if err != nil {
		return nil, err
	}

	return tlsConfig, nil
}

// BuildGRPCServerCredentials 构建gRPC服务端凭证
func (c *TLSConfig) BuildGRPCServerCredentials() (credentials.TransportCredentials, error) {
	tlsConfig, err := c.BuildServerTLSConfig()
	if err != nil {
		return nil, err
	}

	return credentials.NewTLS(tlsConfig), nil
}

// BuildGRPCClientCredentials 构建gRPC客户端凭证
func (c *TLSConfig) BuildGRPCClientCredentials() (credentials.TransportCredentials, error) {
	tlsConfig, err := c.BuildClientTLSConfig()
	if err != nil {
		return nil, err
	}

	return credentials.NewTLS(tlsConfig), nil
}

// === 便捷函数 ===

// NewServerTLSConfig 创建服务端TLS配置
func NewServerTLSConfig(certFile, keyFile string) (*tls.Config, error) {
	cfg := DefaultTLSConfig()
	cfg.CertFile = certFile
	cfg.KeyFile = keyFile
	return cfg.BuildServerTLSConfig()
}

// NewServerMTLSConfig 创建服务端mTLS配置
func NewServerMTLSConfig(certFile, keyFile, caFile string) (*tls.Config, error) {
	cfg := DefaultTLSConfig()
	cfg.CertFile = certFile
	cfg.KeyFile = keyFile
	cfg.CAFile = caFile
	cfg.EnableMTLS = true
	return cfg.BuildServerTLSConfig()
}

// NewClientTLSConfig 创建客户端TLS配置
func NewClientTLSConfig(caFile, serverName string) (*tls.Config, error) {
	cfg := DefaultTLSConfig()
	cfg.CAFile = caFile
	cfg.ServerName = serverName
	return cfg.BuildClientTLSConfig()
}

// NewClientMTLSConfig 创建客户端mTLS配置
func NewClientMTLSConfig(certFile, keyFile, caFile, serverName string) (*tls.Config, error) {
	cfg := DefaultTLSConfig()
	cfg.CertFile = certFile
	cfg.KeyFile = keyFile
	cfg.CAFile = caFile
	cfg.ServerName = serverName
	return cfg.BuildClientTLSConfig()
}

// NewInsecureClientTLSConfig 创建不安全的客户端TLS配置（仅用于测试）
func NewInsecureClientTLSConfig() *tls.Config {
	return &tls.Config{
		InsecureSkipVerify: true,
	}
}

// LoadCertificateFromPEM 从PEM内容加载证书
func LoadCertificateFromPEM(certPEM, keyPEM []byte) (tls.Certificate, error) {
	return tls.X509KeyPair(certPEM, keyPEM)
}

// LoadCACertPool 加载CA证书池
func LoadCACertPool(caCertPEM []byte) (*x509.CertPool, error) {
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caCertPEM) {
		return nil, fmt.Errorf("failed to parse CA certificate")
	}
	return pool, nil
}

// LoadCACertPoolFromFile 从文件加载CA证书池
func LoadCACertPoolFromFile(caFile string) (*x509.CertPool, error) {
	caCert, err := os.ReadFile(caFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA certificate: %w", err)
	}
	return LoadCACertPool(caCert)
}

// === 自签名证书生成（用于测试） ===

// SelfSignedCertConfig 自签名证书配置
type SelfSignedCertConfig struct {
	Organization []string
	CommonName   string
	DNSNames     []string
	IPAddresses  []string
	ValidDays    int
	IsCA         bool
	KeyBits      int
}

// DefaultSelfSignedCertConfig 默认自签名证书配置
func DefaultSelfSignedCertConfig() *SelfSignedCertConfig {
	return &SelfSignedCertConfig{
		Organization: []string{"Test Organization"},
		CommonName:   "localhost",
		DNSNames:     []string{"localhost"},
		ValidDays:    365,
		KeyBits:      2048,
	}
}
