package config

import (
	"log/slog"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

func TestWatcher_ReloadsOnChange(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping watcher test in short mode")
	}

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	initial := []byte(`
routes:
  - path: /api/v1
    targets:
      - url: http://localhost:3001
`)
	if err := os.WriteFile(cfgPath, initial, 0644); err != nil {
		t.Fatal(err)
	}

	var reloadCount atomic.Int32
	var lastCfg atomic.Value

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	w, err := NewWatcher(cfgPath, func(cfg *GatewayConfig) {
		reloadCount.Add(1)
		lastCfg.Store(cfg)
	}, logger)
	if err != nil {
		t.Fatalf("NewWatcher error: %v", err)
	}
	defer w.Close()

	// Wait for watcher to be ready
	time.Sleep(100 * time.Millisecond)

	// Modify the config file
	updated := []byte(`
routes:
  - path: /api/v2
    targets:
      - url: http://localhost:3002
`)
	if err := os.WriteFile(cfgPath, updated, 0644); err != nil {
		t.Fatal(err)
	}

	// Wait for reload
	deadline := time.After(3 * time.Second)
	for {
		if reloadCount.Load() > 0 {
			break
		}
		select {
		case <-deadline:
			t.Fatal("timeout waiting for config reload")
		default:
			time.Sleep(50 * time.Millisecond)
		}
	}

	cfg := lastCfg.Load().(*GatewayConfig)
	if cfg.Routes[0].Path != "/api/v2" {
		t.Errorf("expected reloaded path /api/v2, got %s", cfg.Routes[0].Path)
	}
}

func TestWatcher_InvalidConfigKeepsCurrent(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping watcher test in short mode")
	}

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")

	initial := []byte(`
routes:
  - path: /api/v1
    targets:
      - url: http://localhost:3001
`)
	if err := os.WriteFile(cfgPath, initial, 0644); err != nil {
		t.Fatal(err)
	}

	var reloadCount atomic.Int32
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	w, err := NewWatcher(cfgPath, func(cfg *GatewayConfig) {
		reloadCount.Add(1)
	}, logger)
	if err != nil {
		t.Fatalf("NewWatcher error: %v", err)
	}
	defer w.Close()

	time.Sleep(100 * time.Millisecond)

	// Write invalid config
	invalid := []byte(`routes: []`)
	if err := os.WriteFile(cfgPath, invalid, 0644); err != nil {
		t.Fatal(err)
	}

	// Wait briefly — callback should NOT have been called
	time.Sleep(500 * time.Millisecond)

	if reloadCount.Load() != 0 {
		t.Errorf("expected no reload on invalid config, got %d reloads", reloadCount.Load())
	}
}

func TestWatcher_Close(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(cfgPath, []byte(`
routes:
  - path: /test
    targets:
      - url: http://localhost:3001
`), 0644); err != nil {
		t.Fatal(err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	w, err := NewWatcher(cfgPath, func(cfg *GatewayConfig) {}, logger)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}
}
