package config

import (
	"testing"
	"time"
)

func TestGatewayConfig_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cfg     GatewayConfig
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: GatewayConfig{
				Server:  ServerConfig{Port: 8080},
				Logging: LoggingConfig{Level: "info", Format: "json"},
				Routes: []RouteConfig{
					{
						Path:    "/api/test",
						Targets: []TargetConfig{{URL: "http://localhost:3001", Weight: 1}},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid port",
			cfg: GatewayConfig{
				Server:  ServerConfig{Port: 0},
				Logging: LoggingConfig{Level: "info", Format: "json"},
				Routes: []RouteConfig{
					{Path: "/test", Targets: []TargetConfig{{URL: "http://localhost:3001"}}},
				},
			},
			wantErr: true,
		},
		{
			name: "no routes",
			cfg: GatewayConfig{
				Server:  ServerConfig{Port: 8080},
				Logging: LoggingConfig{Level: "info", Format: "json"},
			},
			wantErr: true,
		},
		{
			name: "route without path",
			cfg: GatewayConfig{
				Server:  ServerConfig{Port: 8080},
				Logging: LoggingConfig{Level: "info", Format: "json"},
				Routes:  []RouteConfig{{Targets: []TargetConfig{{URL: "http://localhost:3001"}}}},
			},
			wantErr: true,
		},
		{
			name: "route without targets",
			cfg: GatewayConfig{
				Server:  ServerConfig{Port: 8080},
				Logging: LoggingConfig{Level: "info", Format: "json"},
				Routes:  []RouteConfig{{Path: "/test"}},
			},
			wantErr: true,
		},
		{
			name: "path traversal rejected",
			cfg: GatewayConfig{
				Server:  ServerConfig{Port: 8080},
				Logging: LoggingConfig{Level: "info", Format: "json"},
				Routes: []RouteConfig{
					{Path: "/api/../../admin", Targets: []TargetConfig{{URL: "http://localhost:3001"}}},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid log level",
			cfg: GatewayConfig{
				Server:  ServerConfig{Port: 8080},
				Logging: LoggingConfig{Level: "verbose", Format: "json"},
				Routes: []RouteConfig{
					{Path: "/test", Targets: []TargetConfig{{URL: "http://localhost:3001"}}},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGatewayConfig_ApplyDefaults(t *testing.T) {
	t.Parallel()

	cfg := &GatewayConfig{
		Routes: []RouteConfig{
			{
				Path:    "/test",
				Targets: []TargetConfig{{URL: "http://localhost:3001"}},
				CircuitBreaker: &CircuitBreakerConfig{},
			},
		},
	}

	cfg.ApplyDefaults()

	if cfg.Server.Port != 8080 {
		t.Errorf("expected default port 8080, got %d", cfg.Server.Port)
	}
	if cfg.Server.ReadTimeout != 30*time.Second {
		t.Errorf("expected default read timeout 30s, got %v", cfg.Server.ReadTimeout)
	}
	if cfg.Logging.Level != "info" {
		t.Errorf("expected default log level info, got %s", cfg.Logging.Level)
	}
	if cfg.Routes[0].Timeout != 30*time.Second {
		t.Errorf("expected default route timeout 30s, got %v", cfg.Routes[0].Timeout)
	}
	if cfg.Routes[0].Targets[0].Weight != 1 {
		t.Errorf("expected default weight 1, got %d", cfg.Routes[0].Targets[0].Weight)
	}
	if cfg.Routes[0].CircuitBreaker.FailureThreshold != 5 {
		t.Errorf("expected default failure threshold 5, got %d", cfg.Routes[0].CircuitBreaker.FailureThreshold)
	}
	if len(cfg.CORS.AllowedOrigins) == 0 || cfg.CORS.AllowedOrigins[0] != "*" {
		t.Errorf("expected default CORS allowed origins [*], got %v", cfg.CORS.AllowedOrigins)
	}
}
