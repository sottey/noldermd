package api

import (
	"log/slog"

	"github.com/go-chi/chi/v5"
)

func NewRouter(notesDir string, logger ...*slog.Logger) chi.Router {
	var baseLogger *slog.Logger
	if len(logger) > 0 && logger[0] != nil {
		baseLogger = logger[0]
	} else {
		baseLogger = slog.Default()
	}
	s := &Server{
		notesDir: notesDir,
		logger:   baseLogger.With("component", "api"),
	}

	r := chi.NewRouter()
	r.Get("/health", s.handleHealth)
	r.Get("/tree", s.handleTree)
	r.Get("/notes", s.handleGetNote)
	r.Post("/notes", s.handleCreateNote)
	r.Patch("/notes", s.handleUpdateNote)
	r.Patch("/notes/rename", s.handleRenameNote)
	r.Delete("/notes", s.handleDeleteNote)
	r.Get("/files", s.handleGetFile)
	r.Get("/search", s.handleSearch)
	r.Get("/tags", s.handleTags)
	r.Get("/settings", s.handleSettingsGet)
	r.Patch("/settings", s.handleSettingsUpdate)
	r.Post("/folders", s.handleCreateFolder)
	r.Patch("/folders", s.handleRenameFolder)
	r.Delete("/folders", s.handleDeleteFolder)
	r.Get("/tasks", s.handleTasksList)
	r.Patch("/tasks/toggle", s.handleTasksToggle)

	return r
}
