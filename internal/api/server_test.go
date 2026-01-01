package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func setupTestRouter(t *testing.T) (string, http.Handler) {
	t.Helper()
	dir := t.TempDir()
	return dir, NewRouter(dir)
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
}

func doRequest(t *testing.T, router http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var reader *bytes.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		reader = bytes.NewReader(payload)
	} else {
		reader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, reader)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func decodeJSONBody[T any](t *testing.T, rec *httptest.ResponseRecorder, dest *T) {
	t.Helper()
	if err := json.NewDecoder(rec.Body).Decode(dest); err != nil {
		t.Fatalf("decode json: %v", err)
	}
}

func TestHealth(t *testing.T) {
	_, router := setupTestRouter(t)
	rec := doRequest(t, router, http.MethodGet, "/health", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	var payload map[string]string
	decodeJSONBody(t, rec, &payload)
	if payload["status"] != "ok" {
		t.Fatalf("expected status ok, got %q", payload["status"])
	}
}

func TestTreeEndpoint(t *testing.T) {
	dir, router := setupTestRouter(t)
	writeFile(t, filepath.Join(dir, "root.md"), "root")
	writeFile(t, filepath.Join(dir, "sub", "child.md"), "child")
	writeFile(t, filepath.Join(dir, "ignore.txt"), "ignore")

	rec := doRequest(t, router, http.MethodGet, "/tree", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	var tree TreeNode
	decodeJSONBody(t, rec, &tree)
	if tree.Type != "folder" {
		t.Fatalf("expected root type folder, got %q", tree.Type)
	}
	if tree.Name != "Notes" {
		t.Fatalf("expected root name Notes, got %q", tree.Name)
	}
	foundRoot := false
	foundSub := false
	for _, child := range tree.Children {
		if child.Type == "file" && child.Name == "root.md" {
			foundRoot = true
		}
		if child.Type == "folder" && child.Name == "sub" {
			foundSub = true
			if len(child.Children) != 1 || child.Children[0].Name != "child.md" {
				t.Fatalf("expected sub/child.md in tree")
			}
		}
	}
	if !foundRoot || !foundSub {
		t.Fatalf("expected root.md and sub folder in tree")
	}
}

func TestTreeShowTemplatesSetting(t *testing.T) {
	dir, router := setupTestRouter(t)
	writeFile(t, filepath.Join(dir, "Templates", "default.template"), "Template")

	settings := []byte(`{"version":2,"showTemplates":true}`)
	if err := os.WriteFile(filepath.Join(dir, "settings.json"), settings, 0o644); err != nil {
		t.Fatalf("write settings.json: %v", err)
	}

	rec := doRequest(t, router, http.MethodGet, "/tree", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	var tree TreeNode
	decodeJSONBody(t, rec, &tree)

	foundTemplate := false
	var visit func(node TreeNode)
	visit = func(node TreeNode) {
		if node.Path == "Templates/default.template" {
			foundTemplate = true
			return
		}
		for _, child := range node.Children {
			visit(child)
		}
	}
	visit(tree)
	if !foundTemplate {
		t.Fatalf("expected template to be present in tree")
	}

	settings = []byte(`{"version":2,"showTemplates":false}`)
	if err := os.WriteFile(filepath.Join(dir, "settings.json"), settings, 0o644); err != nil {
		t.Fatalf("write settings.json: %v", err)
	}

	rec = doRequest(t, router, http.MethodGet, "/tree", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	var hiddenTree TreeNode
	decodeJSONBody(t, rec, &hiddenTree)

	foundTemplate = false
	visit = func(node TreeNode) {
		if node.Path == "Templates/default.template" {
			foundTemplate = true
			return
		}
		for _, child := range node.Children {
			visit(child)
		}
	}
	visit(hiddenTree)
	if foundTemplate {
		t.Fatalf("expected template to be hidden in tree")
	}
}

func TestCreateTemplateNote(t *testing.T) {
	dir, router := setupTestRouter(t)
	payload := map[string]string{
		"path":    "Templates/default.template",
		"content": "Template content",
	}
	rec := doRequest(t, router, http.MethodPost, "/notes", payload)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", rec.Code)
	}

	templatePath := filepath.Join(dir, "Templates", "default.template")
	data, err := os.ReadFile(templatePath)
	if err != nil {
		t.Fatalf("read template: %v", err)
	}
	if string(data) != "Template content" {
		t.Fatalf("expected template content, got %q", string(data))
	}
}

func TestTreeCreatesDailyNoteFromTemplate(t *testing.T) {
	dir, router := setupTestRouter(t)
	dailyDir := filepath.Join(dir, "Daily")
	if err := os.MkdirAll(dailyDir, 0o755); err != nil {
		t.Fatalf("mkdir Daily: %v", err)
	}
	settings := []byte(`{"dailyFolder":"Daily"}`)
	if err := os.WriteFile(filepath.Join(dir, "settings.json"), settings, 0o644); err != nil {
		t.Fatalf("write settings.json: %v", err)
	}
	templateContent := "Daily template\n{{date:YYYY-MM-DD}}"
	if err := os.WriteFile(filepath.Join(dailyDir, "default.template"), []byte(templateContent), 0o644); err != nil {
		t.Fatalf("write default.template: %v", err)
	}

	originalNow := timeNow
	timeNow = func() time.Time { return time.Date(2025, 1, 5, 10, 0, 0, 0, time.UTC) }
	t.Cleanup(func() { timeNow = originalNow })

	rec := doRequest(t, router, http.MethodGet, "/tree", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	dailyNotePath := filepath.Join(dailyDir, "2025-01-05.md")
	data, err := os.ReadFile(dailyNotePath)
	if err != nil {
		t.Fatalf("read daily note: %v", err)
	}
	expected := "Daily template\n2025-01-05"
	if string(data) != expected {
		t.Fatalf("expected daily note to match template, got %q", string(data))
	}
}

func TestCreateNoteUsesFolderTemplate(t *testing.T) {
	dir, router := setupTestRouter(t)
	projectDir := filepath.Join(dir, "Project")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("mkdir Project: %v", err)
	}
	templateContent := "Template content {{date:YYYY-MM-DD}} {{time:HH:mm}} {{datetime:YYYY-MM-DD HH:mm}} {{day:ddd}} {{year:YYYY}} {{month:YYYY-MM}} {{title}} {{path}} {{folder}}"
	if err := os.WriteFile(filepath.Join(projectDir, "default.template"), []byte(templateContent), 0o644); err != nil {
		t.Fatalf("write default.template: %v", err)
	}

	originalNow := timeNow
	timeNow = func() time.Time { return time.Date(2025, 2, 10, 9, 5, 6, 0, time.Local) }
	t.Cleanup(func() { timeNow = originalNow })

	payload := map[string]string{
		"path":    "Project/Custom",
		"content": "User content",
	}
	rec := doRequest(t, router, http.MethodPost, "/notes", payload)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", rec.Code)
	}

	notePath := filepath.Join(projectDir, "Custom.md")
	data, err := os.ReadFile(notePath)
	if err != nil {
		t.Fatalf("read note: %v", err)
	}
	expected := "Template content 2025-02-10 09:05 2025-02-10 09:05 Mon 2025 2025-02 Custom Project/Custom.md Project"
	if string(data) != expected {
		t.Fatalf("expected note to match template, got %q", string(data))
	}
}

func TestIgnoreDotUnderscoreFiles(t *testing.T) {
	dir, router := setupTestRouter(t)
	writeFile(t, filepath.Join(dir, "visible.md"), "Hello #Visible")
	writeFile(t, filepath.Join(dir, "._hidden.md"), "Hello #Hidden")
	writeFile(t, filepath.Join(dir, "sub", "._nested.md"), "Nested #Hidden")

	rec := doRequest(t, router, http.MethodGet, "/tree", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	var tree TreeNode
	decodeJSONBody(t, rec, &tree)

	var names []string
	var visit func(node TreeNode)
	visit = func(node TreeNode) {
		names = append(names, node.Name)
		for _, child := range node.Children {
			visit(child)
		}
	}
	visit(tree)
	for _, name := range names {
		if name == "._hidden.md" || name == "._nested.md" {
			t.Fatalf("expected ignored file %q to be excluded from tree", name)
		}
	}

	rec = doRequest(t, router, http.MethodGet, "/search?query=hidden", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	var matches []SearchResult
	decodeJSONBody(t, rec, &matches)
	if len(matches) != 0 {
		t.Fatalf("expected no hidden matches, got %#v", matches)
	}

	rec = doRequest(t, router, http.MethodGet, "/tags", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	var groups []TagGroup
	decodeJSONBody(t, rec, &groups)
	groupMap := make(map[string]TagGroup)
	for _, group := range groups {
		groupMap[group.Tag] = group
	}
	if _, ok := groupMap["Hidden"]; ok {
		t.Fatalf("expected hidden tag to be excluded")
	}
	if _, ok := groupMap["Visible"]; !ok {
		t.Fatalf("expected Visible tag to be included")
	}
}

func TestNotesCRUD(t *testing.T) {
	_, router := setupTestRouter(t)

	rec := doRequest(t, router, http.MethodPost, "/notes", map[string]string{
		"path":    "new-note",
		"content": "first",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", rec.Code)
	}
	var created map[string]string
	decodeJSONBody(t, rec, &created)
	if created["path"] != "new-note.md" {
		t.Fatalf("expected new-note.md, got %q", created["path"])
	}

	rec = doRequest(t, router, http.MethodGet, "/notes?path=new-note.md", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	var note NoteResponse
	decodeJSONBody(t, rec, &note)
	if note.Content != "first" {
		t.Fatalf("expected content first, got %q", note.Content)
	}

	rec = doRequest(t, router, http.MethodPatch, "/notes", map[string]string{
		"path":    "new-note.md",
		"content": "updated",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	rec = doRequest(t, router, http.MethodPatch, "/notes/rename", map[string]string{
		"path":    "new-note.md",
		"newPath": "renamed",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	var renameResp map[string]string
	decodeJSONBody(t, rec, &renameResp)
	if renameResp["newPath"] != "renamed.md" {
		t.Fatalf("expected renamed.md, got %q", renameResp["newPath"])
	}

	rec = doRequest(t, router, http.MethodGet, "/notes?path=renamed.md", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	var renamed NoteResponse
	decodeJSONBody(t, rec, &renamed)
	if renamed.Content != "updated" {
		t.Fatalf("expected updated content, got %q", renamed.Content)
	}

	rec = doRequest(t, router, http.MethodDelete, "/notes?path=renamed.md", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
}

func TestTasksFromNotes(t *testing.T) {
	dir, router := setupTestRouter(t)

	content := strings.Join([]string{
		"Intro",
		"- [ ] Call Mom +Home #Family @Alice >2025-01-31 ^2",
		"  - [x] Done thing +Work >2025-02-01 ^5",
		"- [ ] Bad due date >tomorrow",
	}, "\n")
	writeFile(t, filepath.Join(dir, "tasks-note.md"), content)

	rec := doRequest(t, router, http.MethodGet, "/tasks", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	var list TaskListResponse
	decodeJSONBody(t, rec, &list)
	if len(list.Tasks) != 3 {
		t.Fatalf("expected 3 tasks, got %d", len(list.Tasks))
	}
	if list.Notice == "" {
		t.Fatalf("expected notice for invalid due dates")
	}

	var first TaskItem
	for _, task := range list.Tasks {
		if task.LineNumber == 2 {
			first = task
			break
		}
	}
	if first.Text != "Call Mom" {
		t.Fatalf("expected task text Call Mom, got %q", first.Text)
	}
	if first.Project != "home" {
		t.Fatalf("expected project home, got %q", first.Project)
	}
	if len(first.Tags) != 1 || first.Tags[0] != "family" {
		t.Fatalf("expected tags to include family, got %#v", first.Tags)
	}
	if len(first.Mentions) != 1 || first.Mentions[0] != "alice" {
		t.Fatalf("expected mentions to include alice, got %#v", first.Mentions)
	}
	if first.DueDate != "2025-01-31" || first.DueDateISO != "2025-01-31" {
		t.Fatalf("expected due date 2025-01-31, got %q/%q", first.DueDate, first.DueDateISO)
	}
	if first.Priority != 2 {
		t.Fatalf("expected priority 2, got %d", first.Priority)
	}
	if first.Completed {
		t.Fatalf("expected task to be incomplete")
	}
}

func TestTasksToggle(t *testing.T) {
	dir, router := setupTestRouter(t)

	content := strings.Join([]string{
		"- [ ] Task One",
		"- [x] Task Two",
	}, "\n")
	writeFile(t, filepath.Join(dir, "toggle.md"), content)

	rec := doRequest(t, router, http.MethodGet, "/tasks", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	var list TaskListResponse
	decodeJSONBody(t, rec, &list)

	var target TaskItem
	for _, task := range list.Tasks {
		if task.Path == "toggle.md" && task.LineNumber == 1 {
			target = task
			break
		}
	}
	if target.LineHash == "" {
		t.Fatalf("expected line hash for task")
	}

	rec = doRequest(t, router, http.MethodPatch, "/tasks/toggle", map[string]any{
		"path":       "toggle.md",
		"lineNumber": target.LineNumber,
		"lineHash":   target.LineHash,
		"completed":  true,
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	data, err := os.ReadFile(filepath.Join(dir, "toggle.md"))
	if err != nil {
		t.Fatalf("read updated note: %v", err)
	}
	lines := strings.Split(string(data), "\n")
	if !strings.HasPrefix(lines[0], "- [x] ") {
		t.Fatalf("expected task to be marked complete, got %q", lines[0])
	}
}

func TestFoldersCRUD(t *testing.T) {
	_, router := setupTestRouter(t)

	rec := doRequest(t, router, http.MethodPost, "/folders", map[string]string{
		"path": "folder-a",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", rec.Code)
	}

	rec = doRequest(t, router, http.MethodPatch, "/folders", map[string]string{
		"path":    "folder-a",
		"newPath": "folder-b",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	rec = doRequest(t, router, http.MethodDelete, "/folders?path=folder-b", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
}

func TestSettingsCRUD(t *testing.T) {
	dir, router := setupTestRouter(t)

	rec := doRequest(t, router, http.MethodGet, "/settings", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	var settingsResp SettingsResponse
	decodeJSONBody(t, rec, &settingsResp)
	if settingsResp.Notice == "" {
		t.Fatalf("expected notice when creating settings.json")
	}
	if settingsResp.Settings.DefaultView != "split" {
		t.Fatalf("expected defaultView split, got %q", settingsResp.Settings.DefaultView)
	}
	if settingsResp.Settings.AutosaveIntervalSeconds != 30 {
		t.Fatalf("expected autosaveIntervalSeconds 30, got %d", settingsResp.Settings.AutosaveIntervalSeconds)
	}
	if settingsResp.Settings.SidebarWidth != 300 {
		t.Fatalf("expected sidebarWidth 300, got %d", settingsResp.Settings.SidebarWidth)
	}
	if settingsResp.Settings.DefaultFolder != "" {
		t.Fatalf("expected defaultFolder empty, got %q", settingsResp.Settings.DefaultFolder)
	}
	if _, err := os.Stat(filepath.Join(dir, "settings.json")); err != nil {
		t.Fatalf("expected settings.json to exist")
	}

	rec = doRequest(t, router, http.MethodPatch, "/settings", map[string]any{
		"darkMode":                true,
		"defaultView":             "preview",
		"autosaveEnabled":         true,
		"autosaveIntervalSeconds": 10,
		"sidebarWidth":            280,
		"defaultFolder":           "Projects",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	var updated Settings
	decodeJSONBody(t, rec, &updated)
	if !updated.DarkMode {
		t.Fatalf("expected darkMode true")
	}
	if updated.DefaultView != "preview" {
		t.Fatalf("expected defaultView preview, got %q", updated.DefaultView)
	}
	if !updated.AutosaveEnabled {
		t.Fatalf("expected autosaveEnabled true")
	}
	if updated.AutosaveIntervalSeconds != 10 {
		t.Fatalf("expected autosaveIntervalSeconds 10, got %d", updated.AutosaveIntervalSeconds)
	}
	if updated.SidebarWidth != 280 {
		t.Fatalf("expected sidebarWidth 280, got %d", updated.SidebarWidth)
	}
	if updated.DefaultFolder != "Projects" {
		t.Fatalf("expected defaultFolder Projects, got %q", updated.DefaultFolder)
	}

	rec = doRequest(t, router, http.MethodGet, "/settings", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	settingsResp = SettingsResponse{}
	decodeJSONBody(t, rec, &settingsResp)
	if !settingsResp.Settings.DarkMode {
		t.Fatalf("expected darkMode true from settings")
	}
	if settingsResp.Settings.DefaultView != "preview" {
		t.Fatalf("expected defaultView preview from settings")
	}
	if !settingsResp.Settings.AutosaveEnabled {
		t.Fatalf("expected autosaveEnabled true from settings")
	}
	if settingsResp.Settings.SidebarWidth != 280 {
		t.Fatalf("expected sidebarWidth 280 from settings")
	}
	if settingsResp.Settings.DefaultFolder != "Projects" {
		t.Fatalf("expected defaultFolder Projects from settings")
	}
}

func TestSearchEndpoint(t *testing.T) {
	dir, router := setupTestRouter(t)
	writeFile(t, filepath.Join(dir, "alpha.md"), "hello world")
	writeFile(t, filepath.Join(dir, "beta.md"), "queryterm")

	rec := doRequest(t, router, http.MethodGet, "/search?query=alpha", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	var matches []SearchResult
	decodeJSONBody(t, rec, &matches)
	if len(matches) != 1 || matches[0].Path != "alpha.md" || matches[0].Type != "note" {
		t.Fatalf("expected alpha.md match, got %#v", matches)
	}

	rec = doRequest(t, router, http.MethodGet, "/search?query=queryterm", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	matches = nil
	decodeJSONBody(t, rec, &matches)
	if len(matches) != 1 || matches[0].Path != "beta.md" || matches[0].Type != "note" {
		t.Fatalf("expected beta.md match, got %#v", matches)
	}
}

func TestTagsEndpoint(t *testing.T) {
	dir, router := setupTestRouter(t)
	writeFile(t, filepath.Join(dir, "alpha.md"), "Hello #TagOne\n##NoTag\nword#NoTag\n#TagTwo and #tagtwo")
	writeFile(t, filepath.Join(dir, "sub", "beta.md"), "Another #TagTwo")

	rec := doRequest(t, router, http.MethodGet, "/tags", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	var groups []TagGroup
	decodeJSONBody(t, rec, &groups)

	groupMap := make(map[string]TagGroup)
	for _, group := range groups {
		groupMap[group.Tag] = group
	}
	if len(groupMap) != 3 {
		t.Fatalf("expected 3 tags, got %d", len(groupMap))
	}
	tagOne, ok := groupMap["TagOne"]
	if !ok || len(tagOne.Notes) != 1 || tagOne.Notes[0].Path != "alpha.md" {
		t.Fatalf("expected TagOne in alpha.md")
	}
	tagTwo, ok := groupMap["TagTwo"]
	if !ok {
		t.Fatalf("expected TagTwo tag")
	}
	paths := make(map[string]bool)
	for _, note := range tagTwo.Notes {
		paths[note.Path] = true
	}
	if !paths["alpha.md"] || !paths[filepath.ToSlash(filepath.Join("sub", "beta.md"))] {
		t.Fatalf("expected TagTwo in alpha.md and sub/beta.md")
	}
	if _, ok := groupMap["tagtwo"]; !ok {
		t.Fatalf("expected tagtwo tag")
	}
}

func TestFilesEndpoint(t *testing.T) {
	dir, router := setupTestRouter(t)
	writeFile(t, filepath.Join(dir, "asset.png"), "binary")

	rec := doRequest(t, router, http.MethodGet, "/files?path=asset.png", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "binary") {
		t.Fatalf("expected file contents")
	}
}
