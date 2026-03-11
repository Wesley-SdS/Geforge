package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMetrics_RecordsRequest(t *testing.T) {
	t.Parallel()

	handler := Metrics("/test")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}
