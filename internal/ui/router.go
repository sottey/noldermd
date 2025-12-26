package ui

import (
	"bytes"
	"embed"
	"io/fs"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

//go:embed web/*
var assets embed.FS

func NewRouter() chi.Router {
	r := chi.NewRouter()

	fsys, err := fs.Sub(assets, "web")
	if err != nil {
		panic(err)
	}

	fileServer := http.FileServer(http.FS(fsys))

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		data, err := fs.ReadFile(fsys, "index.html")
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		http.ServeContent(w, r, "index.html", time.Now(), bytes.NewReader(data))
	})

	r.Handle("/*", fileServer)

	return r
}
