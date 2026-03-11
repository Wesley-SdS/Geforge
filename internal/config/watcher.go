package config

import (
	"fmt"
	"log/slog"
	"sync"

	"github.com/fsnotify/fsnotify"
)

// Watcher watches a config file for changes and triggers a callback with the new config.
type Watcher struct {
	path     string
	callback func(*GatewayConfig)
	logger   *slog.Logger
	watcher  *fsnotify.Watcher
	mu       sync.Mutex
	closed   bool
}

// NewWatcher creates a new config file watcher.
// The callback is invoked with the new valid config whenever the file changes.
// Invalid configs are logged and ignored (the previous config remains active).
func NewWatcher(path string, callback func(*GatewayConfig), logger *slog.Logger) (*Watcher, error) {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("creating fsnotify watcher: %w", err)
	}

	w := &Watcher{
		path:     path,
		callback: callback,
		logger:   logger,
		watcher:  fsw,
	}

	go w.watch()

	if err := fsw.Add(path); err != nil {
		fsw.Close()
		return nil, fmt.Errorf("watching config file: %w", err)
	}

	logger.Info("config watcher started", slog.String("path", path))
	return w, nil
}

func (w *Watcher) watch() {
	for {
		select {
		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}
			if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) {
				w.reload()
			}
		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			w.logger.Error("config watcher error", slog.String("error", err.Error()))
		}
	}
}

func (w *Watcher) reload() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return
	}

	cfg, err := Load(w.path)
	if err != nil {
		w.logger.Warn("config reload failed, keeping current config",
			slog.String("error", err.Error()),
			slog.String("path", w.path),
		)
		return
	}

	w.logger.Info("config reloaded successfully", slog.String("path", w.path))
	w.callback(cfg)
}

// Close stops the watcher and releases resources.
func (w *Watcher) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.closed = true
	return w.watcher.Close()
}
