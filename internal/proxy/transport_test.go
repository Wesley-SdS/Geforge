package proxy

import (
	"testing"
	"time"
)

func TestDefaultTransportConfig(t *testing.T) {
	t.Parallel()

	cfg := DefaultTransportConfig()
	if cfg.MaxIdleConns != 100 {
		t.Errorf("expected 100 max idle conns, got %d", cfg.MaxIdleConns)
	}
	if cfg.InsecureSkipVerify {
		t.Error("insecure skip verify should be false by default")
	}
}

func TestNewTransport(t *testing.T) {
	t.Parallel()

	cfg := TransportConfig{
		MaxIdleConns:        50,
		MaxIdleConnsPerHost: 5,
		IdleConnTimeout:     60 * time.Second,
		TLSHandshakeTimeout: 5 * time.Second,
		ResponseTimeout:     15 * time.Second,
		InsecureSkipVerify:  true,
	}

	transport := NewTransport(cfg)

	if transport.MaxIdleConns != 50 {
		t.Errorf("expected 50 max idle conns, got %d", transport.MaxIdleConns)
	}
	if transport.MaxIdleConnsPerHost != 5 {
		t.Errorf("expected 5 max idle conns per host, got %d", transport.MaxIdleConnsPerHost)
	}
	if !transport.TLSClientConfig.InsecureSkipVerify {
		t.Error("expected insecure skip verify to be true")
	}
}
