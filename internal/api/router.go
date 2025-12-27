package api

import (
	"github.com/go-chi/chi/v5"
)

func NewRouter(notesDir string) chi.Router {
	s := &Server{notesDir: notesDir}

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
	r.Post("/folders", s.handleCreateFolder)
	r.Patch("/folders", s.handleRenameFolder)
	r.Delete("/folders", s.handleDeleteFolder)
	r.Get("/tasks", s.handleTasksList)
	r.Post("/tasks", s.handleTasksCreate)
	r.Get("/tasks/{id}", s.handleTasksGet)
	r.Patch("/tasks/{id}", s.handleTasksUpdate)
	r.Delete("/tasks/{id}", s.handleTasksDelete)

	return r
}
