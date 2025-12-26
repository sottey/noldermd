package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

type Server struct {
	notesDir string
}

type TreeNode struct {
	Name     string     `json:"name"`
	Path     string     `json:"path"`
	Type     string     `json:"type"`
	Children []TreeNode `json:"children,omitempty"`
}

type NoteResponse struct {
	Path     string    `json:"path"`
	Content  string    `json:"content"`
	Modified time.Time `json:"modified"`
}

type NotePayload struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

type NoteRenamePayload struct {
	Path    string `json:"path"`
	NewPath string `json:"newPath"`
}

type FolderPayload struct {
	Path    string `json:"path"`
	NewPath string `json:"newPath"`
}

type SearchResult struct {
	Path string `json:"path"`
	Name string `json:"name"`
}

type TagGroup struct {
	Tag   string         `json:"tag"`
	Notes []SearchResult `json:"notes"`
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleTree(w http.ResponseWriter, r *http.Request) {
	pathParam := r.URL.Query().Get("path")
	absPath, relPath, err := s.resolvePath(pathParam)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			writeError(w, http.StatusNotFound, "path not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "unable to read path")
		return
	}
	if !info.IsDir() {
		writeError(w, http.StatusBadRequest, "path must be a folder")
		return
	}

	rootName := filepath.Base(absPath)
	root := TreeNode{
		Name: rootName,
		Path: relPath,
		Type: "folder",
	}

	children, err := s.buildTree(absPath, relPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "unable to build tree")
		return
	}
	root.Children = children

	writeJSON(w, http.StatusOK, root)
}

func (s *Server) handleGetNote(w http.ResponseWriter, r *http.Request) {
	pathParam := r.URL.Query().Get("path")
	if strings.TrimSpace(pathParam) == "" {
		writeError(w, http.StatusBadRequest, "path is required")
		return
	}

	absPath, relPath, err := s.resolvePath(pathParam)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			writeError(w, http.StatusNotFound, "note not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "unable to read note")
		return
	}
	if info.IsDir() {
		writeError(w, http.StatusBadRequest, "path is a folder")
		return
	}
	if !isMarkdown(absPath) {
		writeError(w, http.StatusBadRequest, "not a markdown file")
		return
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "unable to read note")
		return
	}

	resp := NoteResponse{
		Path:     relPath,
		Content:  string(data),
		Modified: info.ModTime(),
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleCreateNote(w http.ResponseWriter, r *http.Request) {
	payload, err := decodeJSON[NotePayload](r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if strings.TrimSpace(payload.Path) == "" {
		writeError(w, http.StatusBadRequest, "path is required")
		return
	}

	pathParam := ensureMarkdown(strings.TrimSpace(payload.Path))
	absPath, relPath, err := s.resolvePath(pathParam)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if _, err := os.Stat(absPath); err == nil {
		writeError(w, http.StatusConflict, "note already exists")
		return
	} else if !os.IsNotExist(err) {
		writeError(w, http.StatusInternalServerError, "unable to check note")
		return
	}

	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		writeError(w, http.StatusInternalServerError, "unable to create parent folders")
		return
	}

	if err := os.WriteFile(absPath, []byte(payload.Content), 0o644); err != nil {
		writeError(w, http.StatusInternalServerError, "unable to create note")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{"path": relPath})
}

func (s *Server) handleUpdateNote(w http.ResponseWriter, r *http.Request) {
	payload, err := decodeJSON[NotePayload](r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if strings.TrimSpace(payload.Path) == "" {
		writeError(w, http.StatusBadRequest, "path is required")
		return
	}

	absPath, relPath, err := s.resolvePath(payload.Path)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			writeError(w, http.StatusNotFound, "note not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "unable to read note")
		return
	}
	if info.IsDir() {
		writeError(w, http.StatusBadRequest, "path is a folder")
		return
	}
	if !isMarkdown(absPath) {
		writeError(w, http.StatusBadRequest, "not a markdown file")
		return
	}

	if err := os.WriteFile(absPath, []byte(payload.Content), 0o644); err != nil {
		writeError(w, http.StatusInternalServerError, "unable to update note")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"path": relPath})
}

func (s *Server) handleDeleteNote(w http.ResponseWriter, r *http.Request) {
	pathParam := r.URL.Query().Get("path")
	if strings.TrimSpace(pathParam) == "" {
		writeError(w, http.StatusBadRequest, "path is required")
		return
	}

	absPath, _, err := s.resolvePath(pathParam)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			writeError(w, http.StatusNotFound, "note not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "unable to read note")
		return
	}
	if info.IsDir() {
		writeError(w, http.StatusBadRequest, "path is a folder")
		return
	}
	if !isMarkdown(absPath) {
		writeError(w, http.StatusBadRequest, "not a markdown file")
		return
	}

	if err := os.Remove(absPath); err != nil {
		writeError(w, http.StatusInternalServerError, "unable to delete note")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) handleGetFile(w http.ResponseWriter, r *http.Request) {
	pathParam := r.URL.Query().Get("path")
	if strings.TrimSpace(pathParam) == "" {
		writeError(w, http.StatusBadRequest, "path is required")
		return
	}

	absPath, _, err := s.resolvePath(pathParam)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			writeError(w, http.StatusNotFound, "file not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "unable to read file")
		return
	}
	if info.IsDir() {
		writeError(w, http.StatusBadRequest, "path is a folder")
		return
	}

	http.ServeFile(w, r, absPath)
}

func (s *Server) handleCreateFolder(w http.ResponseWriter, r *http.Request) {
	payload, err := decodeJSON[FolderPayload](r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if strings.TrimSpace(payload.Path) == "" {
		writeError(w, http.StatusBadRequest, "path is required")
		return
	}

	absPath, relPath, err := s.resolvePath(payload.Path)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if _, err := os.Stat(absPath); err == nil {
		writeError(w, http.StatusConflict, "folder already exists")
		return
	} else if !os.IsNotExist(err) {
		writeError(w, http.StatusInternalServerError, "unable to check folder")
		return
	}

	if err := os.MkdirAll(absPath, 0o755); err != nil {
		writeError(w, http.StatusInternalServerError, "unable to create folder")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{"path": relPath})
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("query"))
	if query == "" {
		writeError(w, http.StatusBadRequest, "query is required")
		return
	}

	lowerQuery := strings.ToLower(query)
	var results []SearchResult

	err := filepath.WalkDir(s.notesDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !isMarkdown(d.Name()) {
			return nil
		}

		rel, err := filepath.Rel(s.notesDir, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		nameLower := strings.ToLower(d.Name())
		if strings.Contains(nameLower, lowerQuery) {
			results = append(results, SearchResult{
				Path: rel,
				Name: d.Name(),
			})
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		if strings.Contains(strings.ToLower(string(data)), lowerQuery) {
			results = append(results, SearchResult{
				Path: rel,
				Name: d.Name(),
			})
		}

		return nil
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "unable to search notes")
		return
	}

	writeJSON(w, http.StatusOK, results)
}

func (s *Server) handleTags(w http.ResponseWriter, r *http.Request) {
	tagPattern := regexp.MustCompile(`(^|\s)#([A-Za-z]+)\b`)
	tagMap := make(map[string]map[string]string)

	err := filepath.WalkDir(s.notesDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !isMarkdown(d.Name()) {
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
		matches := tagPattern.FindAllStringSubmatch(string(data), -1)
		if len(matches) == 0 {
			return nil
		}

		baseName := filepath.Base(rel)
		for _, match := range matches {
			tag := match[2]
			if tag == "" {
				continue
			}
			if tagMap[tag] == nil {
				tagMap[tag] = make(map[string]string)
			}
			tagMap[tag][rel] = baseName
		}

		return nil
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "unable to list tags")
		return
	}

	tags := make([]string, 0, len(tagMap))
	for tag := range tagMap {
		tags = append(tags, tag)
	}
	sort.Slice(tags, func(i, j int) bool {
		return strings.ToLower(tags[i]) < strings.ToLower(tags[j])
	})

	groups := make([]TagGroup, 0, len(tags))
	for _, tag := range tags {
		notesMap := tagMap[tag]
		notes := make([]SearchResult, 0, len(notesMap))
		for path, name := range notesMap {
			notes = append(notes, SearchResult{Path: path, Name: name})
		}
		sort.Slice(notes, func(i, j int) bool {
			nameA := strings.ToLower(notes[i].Name)
			nameB := strings.ToLower(notes[j].Name)
			if nameA == nameB {
				return notes[i].Path < notes[j].Path
			}
			return nameA < nameB
		})
		groups = append(groups, TagGroup{Tag: tag, Notes: notes})
	}

	writeJSON(w, http.StatusOK, groups)
}

func (s *Server) handleRenameNote(w http.ResponseWriter, r *http.Request) {
	payload, err := decodeJSON[NoteRenamePayload](r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if strings.TrimSpace(payload.Path) == "" || strings.TrimSpace(payload.NewPath) == "" {
		writeError(w, http.StatusBadRequest, "path and newPath are required")
		return
	}

	absPath, relPath, err := s.resolvePath(payload.Path)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	absNewPath, relNewPath, err := s.resolvePath(ensureMarkdown(payload.NewPath))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			writeError(w, http.StatusNotFound, "note not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "unable to read note")
		return
	}
	if info.IsDir() {
		writeError(w, http.StatusBadRequest, "path is a folder")
		return
	}
	if !isMarkdown(absPath) {
		writeError(w, http.StatusBadRequest, "not a markdown file")
		return
	}

	if _, err := os.Stat(absNewPath); err == nil {
		writeError(w, http.StatusConflict, "destination already exists")
		return
	} else if !os.IsNotExist(err) {
		writeError(w, http.StatusInternalServerError, "unable to check destination")
		return
	}

	if err := os.MkdirAll(filepath.Dir(absNewPath), 0o755); err != nil {
		writeError(w, http.StatusInternalServerError, "unable to prepare destination")
		return
	}

	if err := os.Rename(absPath, absNewPath); err != nil {
		writeError(w, http.StatusInternalServerError, "unable to rename note")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"path": relPath, "newPath": relNewPath})
}

func (s *Server) handleRenameFolder(w http.ResponseWriter, r *http.Request) {
	payload, err := decodeJSON[FolderPayload](r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if strings.TrimSpace(payload.Path) == "" || strings.TrimSpace(payload.NewPath) == "" {
		writeError(w, http.StatusBadRequest, "path and newPath are required")
		return
	}

	absPath, relPath, err := s.resolvePath(payload.Path)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	absNewPath, relNewPath, err := s.resolvePath(payload.NewPath)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			writeError(w, http.StatusNotFound, "folder not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "unable to read folder")
		return
	}
	if !info.IsDir() {
		writeError(w, http.StatusBadRequest, "path is not a folder")
		return
	}

	if _, err := os.Stat(absNewPath); err == nil {
		writeError(w, http.StatusConflict, "destination already exists")
		return
	} else if !os.IsNotExist(err) {
		writeError(w, http.StatusInternalServerError, "unable to check destination")
		return
	}

	if err := os.MkdirAll(filepath.Dir(absNewPath), 0o755); err != nil {
		writeError(w, http.StatusInternalServerError, "unable to prepare destination")
		return
	}

	if err := os.Rename(absPath, absNewPath); err != nil {
		writeError(w, http.StatusInternalServerError, "unable to rename folder")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"path": relPath, "newPath": relNewPath})
}

func (s *Server) handleDeleteFolder(w http.ResponseWriter, r *http.Request) {
	pathParam := r.URL.Query().Get("path")
	if strings.TrimSpace(pathParam) == "" {
		writeError(w, http.StatusBadRequest, "path is required")
		return
	}

	absPath, _, err := s.resolvePath(pathParam)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			writeError(w, http.StatusNotFound, "folder not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "unable to read folder")
		return
	}
	if !info.IsDir() {
		writeError(w, http.StatusBadRequest, "path is not a folder")
		return
	}

	if err := os.RemoveAll(absPath); err != nil {
		writeError(w, http.StatusInternalServerError, "unable to delete folder")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) buildTree(absPath, relPath string) ([]TreeNode, error) {
	entries, err := os.ReadDir(absPath)
	if err != nil {
		return nil, err
	}

	var nodes []TreeNode
	for _, entry := range entries {
		name := entry.Name()
		childRel := filepath.Join(relPath, name)
		childAbs := filepath.Join(absPath, name)

		if entry.IsDir() {
			children, err := s.buildTree(childAbs, childRel)
			if err != nil {
				return nil, err
			}
			nodes = append(nodes, TreeNode{
				Name:     name,
				Path:     filepath.ToSlash(childRel),
				Type:     "folder",
				Children: children,
			})
			continue
		}

		if !isMarkdown(name) {
			if isImage(name) {
				nodes = append(nodes, TreeNode{
					Name: name,
					Path: filepath.ToSlash(childRel),
					Type: "asset",
				})
				continue
			}
			if isPDF(name) {
				nodes = append(nodes, TreeNode{
					Name: name,
					Path: filepath.ToSlash(childRel),
					Type: "pdf",
				})
				continue
			}
			if isCSV(name) {
				nodes = append(nodes, TreeNode{
					Name: name,
					Path: filepath.ToSlash(childRel),
					Type: "csv",
				})
				continue
			}
			continue
		}

		nodes = append(nodes, TreeNode{
			Name: name,
			Path: filepath.ToSlash(childRel),
			Type: "file",
		})
	}

	sort.Slice(nodes, func(i, j int) bool {
		typeOrder := map[string]int{
			"folder": 0,
			"file":   1,
			"asset":  2,
			"pdf":    3,
			"csv":    4,
		}
		if nodes[i].Type == nodes[j].Type {
			return strings.ToLower(nodes[i].Name) < strings.ToLower(nodes[j].Name)
		}
		return typeOrder[nodes[i].Type] < typeOrder[nodes[j].Type]
	})

	return nodes, nil
}

func (s *Server) resolvePath(input string) (string, string, error) {
	clean, err := cleanRelPath(input)
	if err != nil {
		return "", "", err
	}

	absPath := filepath.Join(s.notesDir, clean)
	relCheck, err := filepath.Rel(s.notesDir, absPath)
	if err != nil {
		return "", "", err
	}
	if relCheck == ".." || strings.HasPrefix(relCheck, ".."+string(os.PathSeparator)) {
		return "", "", errors.New("path escapes notes directory")
	}

	return absPath, filepath.ToSlash(clean), nil
}

func cleanRelPath(input string) (string, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return "", nil
	}
	clean := filepath.Clean(filepath.FromSlash(trimmed))
	if clean == "." {
		return "", nil
	}
	if filepath.IsAbs(clean) {
		return "", errors.New("absolute paths are not allowed")
	}
	if clean == ".." || strings.HasPrefix(clean, ".."+string(os.PathSeparator)) {
		return "", errors.New("path escapes notes directory")
	}

	return clean, nil
}

func ensureMarkdown(path string) string {
	if strings.HasSuffix(strings.ToLower(path), ".md") {
		return path
	}
	return path + ".md"
}

func isMarkdown(name string) bool {
	return strings.HasSuffix(strings.ToLower(name), ".md")
}

func isImage(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".svg", ".bmp", ".tif", ".tiff", ".avif", ".heic":
		return true
	default:
		return false
	}
}

func isPDF(name string) bool {
	return strings.EqualFold(filepath.Ext(name), ".pdf")
}

func isCSV(name string) bool {
	return strings.EqualFold(filepath.Ext(name), ".csv")
}

func decodeJSON[T any](reader io.Reader) (T, error) {
	var payload T
	dec := json.NewDecoder(reader)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&payload); err != nil {
		return payload, err
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return payload, errors.New("unexpected extra data in request body")
	}
	return payload, nil
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
