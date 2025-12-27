package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

const tasksFileName = "tasks.json"

const dueDateLayout = "2006-01-02"

type Task struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Project   string    `json:"project"`
	Tags      []string  `json:"tags"`
	Created   time.Time `json:"created"`
	Updated   time.Time `json:"updated"`
	DueDate   string    `json:"duedate"`
	Priority  int       `json:"priority"`
	Completed bool      `json:"completed"`
	Notes     string    `json:"notes"`
	Recurring any       `json:"recurring"`
}

type TaskStore struct {
	Version int    `json:"version"`
	Tasks   []Task `json:"tasks"`
}

type TaskListResponse struct {
	Tasks  []Task `json:"tasks"`
	Notice string `json:"notice,omitempty"`
}

type TaskPayload struct {
	Title     string   `json:"title"`
	Project   string   `json:"project"`
	Tags      []string `json:"tags"`
	DueDate   string   `json:"duedate"`
	Priority  int      `json:"priority"`
	Completed bool     `json:"completed"`
	Notes     string   `json:"notes"`
}

func (s *Server) handleTasksList(w http.ResponseWriter, r *http.Request) {
	store, notice, err := s.loadTasks()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "unable to load tasks")
		return
	}

	resp := TaskListResponse{Tasks: store.Tasks}
	if notice != "" {
		resp.Notice = notice
	}

	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleTasksGet(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if id == "" {
		writeError(w, http.StatusBadRequest, "task id is required")
		return
	}

	store, _, err := s.loadTasks()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "unable to load tasks")
		return
	}

	for _, task := range store.Tasks {
		if task.ID == id {
			writeJSON(w, http.StatusOK, task)
			return
		}
	}

	writeError(w, http.StatusNotFound, "task not found")
}

func (s *Server) handleTasksCreate(w http.ResponseWriter, r *http.Request) {
	payload, err := decodeJSON[TaskPayload](r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	payload, err = normalizeTaskPayload(payload)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	store, _, err := s.loadTasks()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "unable to load tasks")
		return
	}

	now := time.Now().UTC()
	task := Task{
		ID:        newUUID(),
		Title:     payload.Title,
		Project:   payload.Project,
		Tags:      payload.Tags,
		Created:   now,
		Updated:   now,
		DueDate:   payload.DueDate,
		Priority:  payload.Priority,
		Completed: payload.Completed,
		Notes:     payload.Notes,
		Recurring: nil,
	}

	store.Tasks = append(store.Tasks, task)
	if err := s.saveTasks(store); err != nil {
		writeError(w, http.StatusInternalServerError, "unable to save tasks")
		return
	}

	writeJSON(w, http.StatusCreated, task)
}

func (s *Server) handleTasksUpdate(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if id == "" {
		writeError(w, http.StatusBadRequest, "task id is required")
		return
	}

	payload, err := decodeJSON[TaskPayload](r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	payload, err = normalizeTaskPayload(payload)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	store, _, err := s.loadTasks()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "unable to load tasks")
		return
	}

	for i, task := range store.Tasks {
		if task.ID != id {
			continue
		}
		updated := task
		updated.Title = payload.Title
		updated.Project = payload.Project
		updated.Tags = payload.Tags
		updated.DueDate = payload.DueDate
		updated.Priority = payload.Priority
		updated.Completed = payload.Completed
		updated.Notes = payload.Notes
		updated.Updated = time.Now().UTC()
		store.Tasks[i] = updated
		if err := s.saveTasks(store); err != nil {
			writeError(w, http.StatusInternalServerError, "unable to save tasks")
			return
		}
		writeJSON(w, http.StatusOK, updated)
		return
	}

	writeError(w, http.StatusNotFound, "task not found")
}

func (s *Server) handleTasksDelete(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if id == "" {
		writeError(w, http.StatusBadRequest, "task id is required")
		return
	}

	store, _, err := s.loadTasks()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "unable to load tasks")
		return
	}

	for i, task := range store.Tasks {
		if task.ID != id {
			continue
		}
		store.Tasks = append(store.Tasks[:i], store.Tasks[i+1:]...)
		if err := s.saveTasks(store); err != nil {
			writeError(w, http.StatusInternalServerError, "unable to save tasks")
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
		return
	}

	writeError(w, http.StatusNotFound, "task not found")
}

func (s *Server) tasksFilePath() string {
	return filepath.Join(s.notesDir, tasksFileName)
}

func (s *Server) loadTasks() (TaskStore, string, error) {
	path := s.tasksFilePath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			store := TaskStore{Version: 1, Tasks: []Task{}}
			if err := os.MkdirAll(s.notesDir, 0o755); err != nil {
				return store, "", err
			}
			if err := s.saveTasks(store); err != nil {
				return store, "", err
			}
			return store, "Created tasks.json", nil
		}
		return TaskStore{}, "", err
	}

	var store TaskStore
	if err := json.Unmarshal(data, &store); err != nil {
		return TaskStore{}, "", err
	}
	if store.Version == 0 {
		store.Version = 1
	}
	if store.Tasks == nil {
		store.Tasks = []Task{}
	}

	return store, "", nil
}

func (s *Server) saveTasks(store TaskStore) error {
	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(s.tasksFilePath(), data, 0o644)
}

func normalizeTaskPayload(payload TaskPayload) (TaskPayload, error) {
	payload.Title = strings.TrimSpace(payload.Title)
	if payload.Title == "" {
		return payload, errors.New("title is required")
	}
	payload.Project = strings.TrimSpace(payload.Project)
	payload.Tags = normalizeTags(payload.Tags)
	if payload.Priority < 1 || payload.Priority > 5 {
		return payload, errors.New("priority must be between 1 and 5")
	}
	if payload.DueDate != "" {
		if _, err := time.Parse(dueDateLayout, payload.DueDate); err != nil {
			return payload, errors.New("duedate must be YYYY-MM-DD")
		}
	}
	return payload, nil
}

func normalizeTags(tags []string) []string {
	if len(tags) == 0 {
		return []string{}
	}
	clean := make([]string, 0, len(tags))
	for _, tag := range tags {
		trimmed := strings.TrimSpace(tag)
		if trimmed == "" {
			continue
		}
		clean = append(clean, trimmed)
	}
	return clean
}

func newUUID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("fallback-%d", time.Now().UnixNano())
	}
	buf[6] = (buf[6] & 0x0f) | 0x40
	buf[8] = (buf[8] & 0x3f) | 0x80
	return fmt.Sprintf(
		"%s-%s-%s-%s-%s",
		hex.EncodeToString(buf[0:4]),
		hex.EncodeToString(buf[4:6]),
		hex.EncodeToString(buf[6:8]),
		hex.EncodeToString(buf[8:10]),
		hex.EncodeToString(buf[10:16]),
	)
}
