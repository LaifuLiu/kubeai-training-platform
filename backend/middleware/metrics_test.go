package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestMetricsSkipsPrometheusEndpoint(t *testing.T) {
	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	Metrics(next).ServeHTTP(rec, req)

	if !nextCalled {
		t.Fatal("expected /metrics to be passed through")
	}
}

func TestMetricsRecordsStatusAndRoutePattern(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/training-runs/{name}", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, "created")
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/training-runs/demo", nil)
	rec := httptest.NewRecorder()
	Metrics(mux).ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusCreated)
	}

	path := "GET /api/v1/training-runs/{name}"
	got := testutil.ToFloat64(httpRequestsTotal.WithLabelValues(http.MethodGet, path, "201"))
	if got < 1 {
		t.Fatalf("expected request counter to increment, got %v", got)
	}
}

func TestMetricsDefaultsUnmatchedPathAndImplicitOK(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "ok")
	})

	req := httptest.NewRequest(http.MethodPost, "/unknown", nil)
	rec := httptest.NewRecorder()
	Metrics(next).ServeHTTP(rec, req)

	got := testutil.ToFloat64(httpRequestsTotal.WithLabelValues(http.MethodPost, "unmatched", "200"))
	if got < 1 {
		t.Fatalf("expected unmatched 200 counter to increment, got %v", got)
	}
}
