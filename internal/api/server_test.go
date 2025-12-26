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
	if tree.Name == "" {
		t.Fatalf("expected root name")
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
	if len(matches) != 1 || matches[0].Path != "alpha.md" {
		t.Fatalf("expected alpha.md match, got %#v", matches)
	}

	rec = doRequest(t, router, http.MethodGet, "/search?query=queryterm", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	matches = nil
	decodeJSONBody(t, rec, &matches)
	if len(matches) != 1 || matches[0].Path != "beta.md" {
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
