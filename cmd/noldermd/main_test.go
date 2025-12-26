package main

import (
	"bytes"
	"io"
	"os"
	"testing"

	"noldermd/internal/server"
)

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	original := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w
	t.Cleanup(func() { os.Stdout = original })

	fn()

	_ = w.Close()
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	return buf.String()
}

func TestServeCommandDefaults(t *testing.T) {
	var gotCfg server.Config
	cmd := newRootCmd(func(cfg server.Config) error {
		gotCfg = cfg
		return nil
	})
	cmd.SetArgs([]string{"serve"})

	output := captureStdout(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("execute: %v", err)
		}
	})

	if gotCfg.NotesDir != "./Notes" {
		t.Fatalf("expected default notes dir ./Notes, got %q", gotCfg.NotesDir)
	}
	if gotCfg.Port != 8080 {
		t.Fatalf("expected default port 8080, got %d", gotCfg.Port)
	}
	if output == "" {
		t.Fatalf("expected output to be written")
	}
}

func TestServeCommandFlags(t *testing.T) {
	var gotCfg server.Config
	cmd := newRootCmd(func(cfg server.Config) error {
		gotCfg = cfg
		return nil
	})
	cmd.SetArgs([]string{"serve", "--notes-dir", "CustomNotes", "--port", "9090"})

	output := captureStdout(t, func() {
		if err := cmd.Execute(); err != nil {
			t.Fatalf("execute: %v", err)
		}
	})

	if gotCfg.NotesDir != "CustomNotes" {
		t.Fatalf("expected notes dir CustomNotes, got %q", gotCfg.NotesDir)
	}
	if gotCfg.Port != 9090 {
		t.Fatalf("expected port 9090, got %d", gotCfg.Port)
	}
	if !bytes.Contains([]byte(output), []byte("http://localhost:9090")) {
		t.Fatalf("expected output to include port, got %q", output)
	}
}
