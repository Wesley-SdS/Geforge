package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		file    string
		wantErr bool
	}{
		{"valid config", "testdata/valid.yaml", false},
		{"minimal config", "testdata/minimal.yaml", false},
		{"invalid config", "testdata/invalid.yaml", true},
		{"nonexistent file", "testdata/nonexistent.yaml", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg, err := Load(tt.file)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Load(%s) error = %v, wantErr %v", tt.file, err, tt.wantErr)
			}
			if !tt.wantErr && cfg == nil {
				t.Fatal("expected non-nil config")
			}
		})
	}
}

func TestLoad_Defaults(t *testing.T) {
	t.Parallel()
	cfg, err := Load("testdata/minimal.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Server.Port != 8080 {
		t.Errorf("expected default port 8080, got %d", cfg.Server.Port)
	}
	if cfg.Logging.Level != "info" {
		t.Errorf("expected default log level info, got %s", cfg.Logging.Level)
	}
	if cfg.Logging.Format != "json" {
		t.Errorf("expected default log format json, got %s", cfg.Logging.Format)
	}
	if !cfg.Metrics.Enabled {
		t.Error("expected metrics enabled by default")
	}
}

func TestLoad_EnvOverride(t *testing.T) {
	t.Setenv("GATEFORGE_PORT", "9090")
	t.Setenv("GATEFORGE_LOG_LEVEL", "debug")

	cfg, err := Load("testdata/minimal.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Server.Port != 9090 {
		t.Errorf("expected port 9090 from env, got %d", cfg.Server.Port)
	}
	if cfg.Logging.Level != "debug" {
		t.Errorf("expected log level debug from env, got %s", cfg.Logging.Level)
	}
}

func TestLoad_ExpandEnv(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("TEST_TARGET_URL", "http://localhost:9999")
	content := []byte(`
routes:
  - path: /api/test
    targets:
      - url: ${TEST_TARGET_URL}
`)
	path := filepath.Join(dir, "env.yaml")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Routes[0].Targets[0].URL != "http://localhost:9999" {
		t.Errorf("expected env-expanded URL, got %s", cfg.Routes[0].Targets[0].URL)
	}
}
