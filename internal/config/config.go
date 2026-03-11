package config

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

// GatewayConfig is the root configuration for the gateway.
type GatewayConfig struct {
	Server  ServerConfig  `yaml:"server"`
	Routes  []RouteConfig `yaml:"routes"`
	Metrics MetricsConfig `yaml:"metrics"`
	Logging LoggingConfig `yaml:"logging"`
	CORS    CORSConfig    `yaml:"cors"`
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Port         int           `yaml:"port"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
	IdleTimeout  time.Duration `yaml:"idle_timeout"`
}

// RouteConfig defines a single route and its upstreams.
type RouteConfig struct {
	Path            string               `yaml:"path"`
	Methods         []string             `yaml:"methods"`
	Targets         []TargetConfig       `yaml:"targets"`
	BalanceStrategy string               `yaml:"balance_strategy"`
	StripPrefix     bool                 `yaml:"strip_prefix"`
	Timeout         time.Duration        `yaml:"timeout"`
	RateLimit       *RateLimitConfig     `yaml:"rate_limit"`
	CircuitBreaker  *CircuitBreakerConfig `yaml:"circuit_breaker"`
}

// TargetConfig represents a single upstream target.
type TargetConfig struct {
	URL    string `yaml:"url"`
	Weight int    `yaml:"weight"`
}

// RateLimitConfig controls per-route rate limiting.
type RateLimitConfig struct {
	RequestsPerSecond float64 `yaml:"requests_per_second"`
	Burst             int     `yaml:"burst"`
}

// CircuitBreakerConfig controls per-route circuit breaking.
type CircuitBreakerConfig struct {
	FailureThreshold int           `yaml:"failure_threshold"`
	ResetTimeout     time.Duration `yaml:"reset_timeout"`
	HalfOpenMaxReqs  int           `yaml:"half_open_max_requests"`
}

// MetricsConfig controls Prometheus metrics exposure.
type MetricsConfig struct {
	Enabled bool   `yaml:"enabled"`
	Path    string `yaml:"path"`
}

// LoggingConfig controls structured logging.
type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

// CORSConfig controls Cross-Origin Resource Sharing headers.
type CORSConfig struct {
	AllowedOrigins []string `yaml:"allowed_origins"`
	AllowedMethods []string `yaml:"allowed_methods"`
	AllowedHeaders []string `yaml:"allowed_headers"`
	MaxAge         int      `yaml:"max_age"`
}

// ApplyDefaults sets default values for unset configuration fields.
func (c *GatewayConfig) ApplyDefaults() {
	if c.Server.Port == 0 {
		c.Server.Port = 8080
	}
	if c.Server.ReadTimeout == 0 {
		c.Server.ReadTimeout = 30 * time.Second
	}
	if c.Server.WriteTimeout == 0 {
		c.Server.WriteTimeout = 30 * time.Second
	}
	if c.Server.IdleTimeout == 0 {
		c.Server.IdleTimeout = 120 * time.Second
	}
	if c.Logging.Level == "" {
		c.Logging.Level = "info"
	}
	if c.Logging.Format == "" {
		c.Logging.Format = "json"
	}
	if !c.Metrics.Enabled && c.Metrics.Path == "" {
		c.Metrics.Enabled = true
	}
	if c.Metrics.Path == "" {
		c.Metrics.Path = "/metrics"
	}
	if len(c.CORS.AllowedOrigins) == 0 {
		c.CORS.AllowedOrigins = []string{"*"}
	}
	if len(c.CORS.AllowedMethods) == 0 {
		c.CORS.AllowedMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	}
	if c.CORS.MaxAge == 0 {
		c.CORS.MaxAge = 86400
	}

	for i := range c.Routes {
		if c.Routes[i].Timeout == 0 {
			c.Routes[i].Timeout = 30 * time.Second
		}
		if c.Routes[i].BalanceStrategy == "" {
			c.Routes[i].BalanceStrategy = "round-robin"
		}
		for j := range c.Routes[i].Targets {
			if c.Routes[i].Targets[j].Weight == 0 {
				c.Routes[i].Targets[j].Weight = 1
			}
		}
		if cb := c.Routes[i].CircuitBreaker; cb != nil {
			if cb.FailureThreshold == 0 {
				cb.FailureThreshold = 5
			}
			if cb.ResetTimeout == 0 {
				cb.ResetTimeout = 30 * time.Second
			}
			if cb.HalfOpenMaxReqs == 0 {
				cb.HalfOpenMaxReqs = 3
			}
		}
	}
}

// Validate returns an error if the configuration is invalid.
func (c *GatewayConfig) Validate() error {
	var errs []string

	if c.Server.Port < 1 || c.Server.Port > 65535 {
		errs = append(errs, fmt.Sprintf("invalid port: %d", c.Server.Port))
	}
	if len(c.Routes) == 0 {
		errs = append(errs, "at least one route required")
	}
	for i, route := range c.Routes {
		if route.Path == "" {
			errs = append(errs, fmt.Sprintf("route[%d]: path is required", i))
		}
		if route.Path != "" && !strings.HasPrefix(route.Path, "/") {
			errs = append(errs, fmt.Sprintf("route[%d]: path must start with /", i))
		}
		if strings.Contains(route.Path, "..") {
			errs = append(errs, fmt.Sprintf("route[%d]: path must not contain '..' (path traversal)", i))
		}
		if len(route.Targets) == 0 {
			errs = append(errs, fmt.Sprintf("route[%d]: at least one target required", i))
		}
		for j, target := range route.Targets {
			if _, err := url.Parse(target.URL); err != nil {
				errs = append(errs, fmt.Sprintf("route[%d].target[%d]: invalid URL: %s", i, j, err))
			}
			if target.URL == "" {
				errs = append(errs, fmt.Sprintf("route[%d].target[%d]: URL is required", i, j))
			}
		}
		if route.BalanceStrategy != "" &&
			route.BalanceStrategy != "round-robin" &&
			route.BalanceStrategy != "weighted" {
			errs = append(errs, fmt.Sprintf("route[%d]: unsupported balance strategy: %s", i, route.BalanceStrategy))
		}
		if route.RateLimit != nil {
			if route.RateLimit.RequestsPerSecond <= 0 {
				errs = append(errs, fmt.Sprintf("route[%d]: requests_per_second must be > 0", i))
			}
			if route.RateLimit.Burst <= 0 {
				errs = append(errs, fmt.Sprintf("route[%d]: burst must be > 0", i))
			}
		}
	}

	switch c.Logging.Level {
	case "debug", "info", "warn", "error":
	default:
		errs = append(errs, fmt.Sprintf("invalid log level: %s", c.Logging.Level))
	}
	switch c.Logging.Format {
	case "json", "text":
	default:
		errs = append(errs, fmt.Sprintf("invalid log format: %s", c.Logging.Format))
	}

	if len(errs) > 0 {
		return fmt.Errorf("config validation failed:\n  %s", strings.Join(errs, "\n  "))
	}
	return nil
}
