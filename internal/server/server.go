package server

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-chi/chi/v5"

	"noldermd/internal/api"
	"noldermd/internal/ui"
)

type Config struct {
	NotesDir string
	Port     int
}

func Run(cfg Config) error {
	if cfg.Port <= 0 {
		return fmt.Errorf("port must be positive")
	}

	notesDir, err := filepath.Abs(cfg.NotesDir)
	if err != nil {
		return fmt.Errorf("resolve notes dir: %w", err)
	}

	if err := os.MkdirAll(notesDir, 0o755); err != nil {
		return fmt.Errorf("ensure notes dir: %w", err)
	}

	r := chi.NewRouter()
	r.Mount("/api/v1", api.NewRouter(notesDir))
	r.Mount("/", ui.NewRouter())

	addr := fmt.Sprintf(":%d", cfg.Port)
	return listenAndServe(addr, r)
}

var listenAndServe = http.ListenAndServe
