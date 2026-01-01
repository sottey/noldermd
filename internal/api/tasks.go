package api

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type TaskItem struct {
	ID         string   `json:"id"`
	Path       string   `json:"path"`
	LineNumber int      `json:"lineNumber"`
	LineHash   string   `json:"lineHash"`
	Text       string   `json:"text"`
	Completed  bool     `json:"completed"`
	Project    string   `json:"project"`
	Tags       []string `json:"tags"`
	Mentions   []string `json:"mentions"`
	DueDate    string   `json:"dueDate,omitempty"`
	DueDateISO string   `json:"dueDateISO,omitempty"`
	Priority   int      `json:"priority,omitempty"`
}

type TaskListResponse struct {
	Tasks  []TaskItem `json:"tasks"`
	Notice string     `json:"notice,omitempty"`
}

type TaskTogglePayload struct {
	Path       string `json:"path"`
	LineNumber int    `json:"lineNumber"`
	LineHash   string `json:"lineHash"`
	Completed  bool   `json:"completed"`
}

type TaskArchiveResponse struct {
	Archived int `json:"archived"`
	Files    int `json:"files"`
}

func (s *Server) handleTasksList(w http.ResponseWriter, r *http.Request) {
	tasks, notice, err := s.listTasks()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "unable to load tasks")
		return
	}

	resp := TaskListResponse{Tasks: tasks}
	if notice != "" {
		resp.Notice = notice
	}

	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleTasksToggle(w http.ResponseWriter, r *http.Request) {
	payload, err := decodeJSON[TaskTogglePayload](r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if strings.TrimSpace(payload.Path) == "" {
		writeError(w, http.StatusBadRequest, "path is required")
		return
	}
	if payload.LineNumber <= 0 {
		writeError(w, http.StatusBadRequest, "lineNumber must be positive")
		return
	}

	absPath, relPath, err := s.resolvePath(payload.Path)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if !isMarkdown(absPath) {
		writeError(w, http.StatusBadRequest, "not a note file")
		return
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			writeError(w, http.StatusNotFound, "note not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "unable to read note")
		return
	}

	lines := strings.Split(string(data), "\n")
	lineIndex := payload.LineNumber - 1
	if lineIndex < 0 || lineIndex >= len(lines) || !lineHashMatches(lines[lineIndex], payload.LineHash) {
		if payload.LineHash == "" {
			writeError(w, http.StatusBadRequest, "task not found")
			return
		}
		found := false
		for i, line := range lines {
			if lineHashMatches(line, payload.LineHash) {
				lineIndex = i
				found = true
				break
			}
		}
		if !found {
			writeError(w, http.StatusBadRequest, "task not found")
			return
		}
	}

	originalLine := lines[lineIndex]
	lineEnding := ""
	if strings.HasSuffix(originalLine, "\r") {
		lineEnding = "\r"
		originalLine = strings.TrimSuffix(originalLine, "\r")
	}

	updatedLine, ok := setTaskLineCompletion(originalLine, payload.Completed)
	if !ok {
		writeError(w, http.StatusBadRequest, "line is not a task")
		return
	}
	lines[lineIndex] = updatedLine + lineEnding

	updated := strings.Join(lines, "\n")
	if err := os.WriteFile(absPath, []byte(updated), 0o644); err != nil {
		s.logger.Error("unable to update task line", "path", relPath, "line", lineIndex+1, "error", err)
		writeError(w, http.StatusInternalServerError, "unable to update note")
		return
	}

	s.logger.Info("task toggled", "path", relPath, "line", lineIndex+1, "completed", payload.Completed)
	writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
}

func (s *Server) handleTasksArchive(w http.ResponseWriter, r *http.Request) {
	archived, files, err := s.archiveCompletedTasks()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "unable to archive tasks")
		return
	}
	writeJSON(w, http.StatusOK, TaskArchiveResponse{Archived: archived, Files: files})
}

func (s *Server) listTasks() ([]TaskItem, string, error) {
	var tasks []TaskItem
	var warnings []string

	err := filepath.WalkDir(s.notesDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if isIgnoredFile(d.Name()) || !isMarkdown(d.Name()) {
			return nil
		}

		rel, err := filepath.Rel(s.notesDir, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		parsed := parseTodoLines(string(data))
		for _, todo := range parsed {
			task := TaskItem{
				ID:         fmt.Sprintf("%s:%d", rel, todo.LineNumber),
				Path:       rel,
				LineNumber: todo.LineNumber,
				LineHash:   todo.LineHash,
				Text:       todo.Text,
				Completed:  todo.Completed,
				Project:    todo.Project,
				Tags:       todo.Tags,
				Mentions:   todo.Mentions,
				DueDate:    todo.DueDateRaw,
				DueDateISO: todo.DueDateISO,
				Priority:   todo.Priority,
			}
			if todo.DueDateRaw != "" && !todo.DueDateValid {
				warnings = append(warnings, fmt.Sprintf("%s:%d (%s)", rel, todo.LineNumber, todo.DueDateRaw))
				s.logger.Warn("unrecognized due date", "path", rel, "line", todo.LineNumber, "value", todo.DueDateRaw)
			}
			tasks = append(tasks, task)
		}
		return nil
	})
	if err != nil {
		return nil, "", err
	}

	notice := ""
	if len(warnings) > 0 {
		limit := warnings
		if len(limit) > 3 {
			limit = warnings[:3]
		}
		notice = fmt.Sprintf(
			"Found %d task(s) with unrecognized due dates. Examples: %s.",
			len(warnings),
			strings.Join(limit, "; "),
		)
	}

	return tasks, notice, nil
}

func lineHashMatches(line, hash string) bool {
	if hash == "" {
		return false
	}
	raw := strings.TrimSuffix(line, "\r")
	return hashLine(raw) == hash
}

func (s *Server) archiveCompletedTasks() (int, int, error) {
	archived := 0
	filesUpdated := 0

	err := filepath.WalkDir(s.notesDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if isIgnoredFile(d.Name()) || !isMarkdown(d.Name()) {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		lines := strings.Split(string(data), "\n")
		changed := false
		for i, line := range lines {
			updated, ok := archiveCompletedTaskLine(line)
			if !ok {
				continue
			}
			lines[i] = updated
			archived += 1
			changed = true
		}
		if !changed {
			return nil
		}
		output := strings.Join(lines, "\n")
		if err := os.WriteFile(path, []byte(output), 0o644); err != nil {
			return err
		}
		filesUpdated += 1
		return nil
	})
	if err != nil {
		return 0, 0, err
	}
	if archived > 0 {
		s.logger.Info("archived completed tasks", "count", archived, "files", filesUpdated)
	}
	return archived, filesUpdated, nil
}
