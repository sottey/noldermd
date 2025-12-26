package ui

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestUIIndexRoute(t *testing.T) {
	router := NewRouter()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Header().Get("Content-Type"), "text/html") {
		t.Fatalf("expected HTML content type, got %q", rec.Header().Get("Content-Type"))
	}
}

func TestUIStaticAssets(t *testing.T) {
	router := NewRouter()

	req := httptest.NewRequest(http.MethodGet, "/styles.css", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected styles.css 200, got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/missing.txt", nil)
	rec = httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected missing.txt 404, got %d", rec.Code)
	}
}
