package api

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
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
	logger   *slog.Logger
}

var timeNow = time.Now

type TemplateContext struct {
	Title  string
	Path   string
	Folder string
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
	Type string `json:"type,omitempty"`
	ID   string `json:"id,omitempty"`
}

type TagGroup struct {
	Tag   string         `json:"tag"`
	Notes []SearchResult `json:"notes"`
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleTree(w http.ResponseWriter, r *http.Request) {
	if err := s.ensureDailyNote(); err != nil {
		writeError(w, http.StatusInternalServerError, "unable to ensure daily note")
		return
	}
	settings, _, err := s.loadSettings()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "unable to load settings")
		return
	}
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

	root := TreeNode{
		Name: "Notes",
		Path: relPath,
		Type: "folder",
	}

	children, err := s.buildTree(absPath, relPath, settings.ShowTemplates)
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
	if !isNoteFile(absPath) {
		writeError(w, http.StatusBadRequest, "not a note file")
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

	pathParam := strings.TrimSpace(payload.Path)
	if isTemplate(pathParam) {
		pathParam = ensureTemplate(pathParam)
	} else {
		pathParam = ensureMarkdown(pathParam)
	}
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

	content := payload.Content
	templateContent, ok, err := s.folderTemplateContent(filepath.Dir(absPath))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "unable to load folder template")
		return
	}
	if ok {
		content = applyTemplatePlaceholders(string(templateContent), timeNow(), templateContext(relPath))
	}

	if err := os.WriteFile(absPath, []byte(content), 0o644); err != nil {
		writeError(w, http.StatusInternalServerError, "unable to create note")
		return
	}

	// Task sync from note content is intentionally disabled for now.

	s.logger.Info("note created", "path", relPath, "bytes", len(content), "template", ok)
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
	if !isNoteFile(absPath) {
		writeError(w, http.StatusBadRequest, "not a note file")
		return
	}

	if err := os.WriteFile(absPath, []byte(payload.Content), 0o644); err != nil {
		s.logger.Error("unable to update note", "path", relPath, "absPath", absPath, "error", err)
		writeError(w, http.StatusInternalServerError, "unable to update note")
		return
	}

	// Task sync from note content is intentionally disabled for now.

	s.logger.Info("note updated", "path", relPath, "bytes", len(payload.Content))
	writeJSON(w, http.StatusOK, map[string]string{"path": relPath})
}

func (s *Server) handleDeleteNote(w http.ResponseWriter, r *http.Request) {
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
	if !isNoteFile(absPath) {
		writeError(w, http.StatusBadRequest, "not a note file")
		return
	}

	if err := os.Remove(absPath); err != nil {
		writeError(w, http.StatusInternalServerError, "unable to delete note")
		return
	}

	s.logger.Info("note deleted", "path", relPath)
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

	s.logger.Info("folder created", "path", relPath)
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
		if isIgnoredFile(d.Name()) {
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
				Type: "note",
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
				Type: "note",
			})
		}

		return nil
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "unable to search notes")
		return
	}

	// Task search is intentionally disabled for now.

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
		if isIgnoredFile(d.Name()) {
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
	if !isNoteFile(absPath) {
		writeError(w, http.StatusBadRequest, "not a note file")
		return
	}

	newPathInput := strings.TrimSpace(payload.NewPath)
	if isTemplate(absPath) {
		newPathInput = ensureTemplate(newPathInput)
	} else {
		newPathInput = ensureMarkdown(newPathInput)
	}
	absNewPath, relNewPath, err := s.resolvePath(newPathInput)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
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

	s.logger.Info("note renamed", "path", relPath, "newPath", relNewPath)
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

	s.logger.Info("folder renamed", "path", relPath, "newPath", relNewPath)
	writeJSON(w, http.StatusOK, map[string]string{"path": relPath, "newPath": relNewPath})
}

func (s *Server) handleDeleteFolder(w http.ResponseWriter, r *http.Request) {
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

	s.logger.Info("folder deleted", "path", relPath)
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) buildTree(absPath, relPath string, showTemplates bool) ([]TreeNode, error) {
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
			children, err := s.buildTree(childAbs, childRel, showTemplates)
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

		if isIgnoredFile(name) {
			continue
		}

		if !isMarkdown(name) {
			if showTemplates && isTemplate(name) {
				nodes = append(nodes, TreeNode{
					Name: name,
					Path: filepath.ToSlash(childRel),
					Type: "file",
				})
				continue
			}
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

func (s *Server) ensureDailyNote() error {
	settings, _, err := s.loadSettings()
	if err != nil {
		return err
	}
	dailyFolder := strings.TrimSpace(settings.DailyFolder)
	if dailyFolder == "" {
		return nil
	}
	cleaned, err := cleanRelPath(dailyFolder)
	if err != nil {
		return err
	}

	dailyDir := filepath.Join(s.notesDir, cleaned)
	info, err := os.Stat(dailyDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if !info.IsDir() {
		return nil
	}

	today := timeNow().Format("2006-01-02")
	notePath := filepath.Join(dailyDir, today+".md")
	if _, err := os.Stat(notePath); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}

	content, ok, err := s.folderTemplateContent(dailyDir)
	if err != nil {
		return err
	}
	if !ok {
		content = nil
	}
	finalContent := string(content)
	if ok {
		relPath := filepath.ToSlash(filepath.Join(cleaned, today+".md"))
		finalContent = applyTemplatePlaceholders(finalContent, timeNow(), templateContext(relPath))
	}
	return os.WriteFile(notePath, []byte(finalContent), 0o644)
}

func (s *Server) folderTemplateContent(dir string) ([]byte, bool, error) {
	templatePath := filepath.Join(dir, "default.template")
	content, err := os.ReadFile(templatePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return content, true, nil
}

func applyTemplatePlaceholders(input string, now time.Time, ctx TemplateContext) string {
	const tokenPrefix = "{{"
	const tokenSuffix = "}}"
	start := strings.Index(input, tokenPrefix)
	if start == -1 {
		return input
	}

	var out strings.Builder
	out.Grow(len(input))
	remaining := input
	for {
		start = strings.Index(remaining, tokenPrefix)
		if start == -1 {
			out.WriteString(remaining)
			break
		}
		out.WriteString(remaining[:start])
		remaining = remaining[start+len(tokenPrefix):]
		end := strings.Index(remaining, tokenSuffix)
		if end == -1 {
			out.WriteString(tokenPrefix)
			out.WriteString(remaining)
			break
		}
		token := remaining[:end]
		out.WriteString(resolveTemplateToken(token, now, ctx))
		remaining = remaining[end+len(tokenSuffix):]
	}
	return out.String()
}

func resolveTemplateToken(token string, now time.Time, ctx TemplateContext) string {
	if token == "title" {
		return ctx.Title
	}
	if token == "path" {
		return ctx.Path
	}
	if token == "folder" {
		return ctx.Folder
	}
	parts := strings.SplitN(token, ":", 2)
	if len(parts) != 2 {
		return "{{" + token + "}}"
	}
	key := parts[0]
	format := parts[1]
	layout := layoutFromTemplate(format)
	switch key {
	case "date", "time", "datetime", "day", "year", "month":
		return now.Format(layout)
	default:
		return "{{" + token + "}}"
	}
}

func layoutFromTemplate(format string) string {
	replacer := strings.NewReplacer(
		"YYYY", "2006",
		"MM", "01",
		"DD", "02",
		"HH", "15",
		"mm", "04",
		"ss", "05",
		"dddd", "Monday",
		"ddd", "Mon",
	)
	return replacer.Replace(format)
}

func templateContext(relPath string) TemplateContext {
	path := filepath.ToSlash(relPath)
	folder := filepath.ToSlash(filepath.Dir(relPath))
	if folder == "." {
		folder = ""
	}
	return TemplateContext{
		Title:  strings.TrimSuffix(filepath.Base(relPath), filepath.Ext(relPath)),
		Path:   path,
		Folder: folder,
	}
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

func ensureTemplate(path string) string {
	if isTemplate(path) {
		return path
	}
	return path + ".template"
}

func isMarkdown(name string) bool {
	return strings.HasSuffix(strings.ToLower(name), ".md")
}

func isTemplate(name string) bool {
	return strings.HasSuffix(strings.ToLower(name), ".template")
}

func isNoteFile(name string) bool {
	return isMarkdown(name) || isTemplate(name)
}

func isIgnoredFile(name string) bool {
	return strings.HasPrefix(name, "._")
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

func tagsContain(tags []string, query string) bool {
	if query == "" {
		return false
	}
	for _, tag := range tags {
		if strings.Contains(strings.ToLower(tag), query) {
			return true
		}
	}
	return false
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
	logger := slog.Default().With("component", "api", "status", status)
	switch {
	case status >= http.StatusInternalServerError:
		logger.Error("request error", "message", message)
	case status >= http.StatusBadRequest:
		logger.Warn("request error", "message", message)
	default:
		logger.Info("request error", "message", message)
	}
	writeJSON(w, status, map[string]string{"error": message})
}
