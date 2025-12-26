package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunRejectsInvalidPort(t *testing.T) {
	err := Run(Config{NotesDir: t.TempDir(), Port: 0})
	if err == nil {
		t.Fatalf("expected error for invalid port")
	}
}

func TestRunMountsRouters(t *testing.T) {
	tmpDir := t.TempDir()
	notesDir := filepath.Join(tmpDir, "notes")

	originalListen := listenAndServe
	var gotAddr string
	var gotHandler http.Handler
	listenAndServe = func(addr string, handler http.Handler) error {
		gotAddr = addr
		gotHandler = handler
		return nil
	}
	t.Cleanup(func() { listenAndServe = originalListen })

	if err := Run(Config{NotesDir: notesDir, Port: 9999}); err != nil {
		t.Fatalf("Run error: %v", err)
	}

	if gotAddr != ":9999" {
		t.Fatalf("expected addr :9999, got %q", gotAddr)
	}
	if gotHandler == nil {
		t.Fatalf("expected handler to be set")
	}

	if _, err := os.Stat(notesDir); err != nil {
		t.Fatalf("expected notes dir to exist: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	rec := httptest.NewRecorder()
	gotHandler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected /api/v1/health 200, got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/", nil)
	rec = httptest.NewRecorder()
	gotHandler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected / 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Header().Get("Content-Type"), "text/html") {
		t.Fatalf("expected HTML content type, got %q", rec.Header().Get("Content-Type"))
	}
}
