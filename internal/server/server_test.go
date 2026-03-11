package server

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"testing"
	"time"
)

func TestServer_New(t *testing.T) {
	t.Parallel()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	srv := New(handler, 0, 30*time.Second, 30*time.Second, 120*time.Second, logger)
	if srv == nil {
		t.Fatal("expected non-nil server")
	}
}

func TestServer_Shutdown(t *testing.T) {
	t.Parallel()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	srv := New(handler, 0, 30*time.Second, 30*time.Second, 120*time.Second, logger)

	go func() {
		srv.Start()
	}()

	time.Sleep(50 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		t.Errorf("shutdown error: %v", err)
	}
}
