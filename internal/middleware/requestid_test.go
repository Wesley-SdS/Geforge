package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/wesleybatista/gateforge/internal/observability"
)

func TestRequestID_GeneratesWhenMissing(t *testing.T) {
	t.Parallel()

	var gotID string
	handler := RequestID()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotID = observability.GetRequestID(r.Context())
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if gotID == "" {
		t.Error("expected request ID to be generated")
	}
	if rec.Header().Get(requestIDHeader) == "" {
		t.Error("expected X-Request-ID in response headers")
	}
	if rec.Header().Get(requestIDHeader) != gotID {
		t.Error("response header should match context value")
	}
}

func TestRequestID_PreservesExisting(t *testing.T) {
	t.Parallel()

	existingID := "existing-request-id-123"
	var gotID string

	handler := RequestID()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotID = observability.GetRequestID(r.Context())
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(requestIDHeader, existingID)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if gotID != existingID {
		t.Errorf("expected preserved ID %s, got %s", existingID, gotID)
	}
	if rec.Header().Get(requestIDHeader) != existingID {
		t.Errorf("expected response header %s, got %s", existingID, rec.Header().Get(requestIDHeader))
	}
}
