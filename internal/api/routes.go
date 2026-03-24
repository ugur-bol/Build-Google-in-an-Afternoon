package api

import (
	"net/http"
	"os"
	"path/filepath"
)

// RegisterRoutes wires up all HTTP routes on the given mux.
func RegisterRoutes(mux *http.ServeMux, h *Handlers) {
	// API endpoints
	mux.HandleFunc("/index", h.HandleIndex)
	mux.HandleFunc("/search", h.HandleSearch)
	mux.HandleFunc("/api/state", h.HandleState)
	mux.HandleFunc("/api/pause", h.HandlePause)
	mux.HandleFunc("/api/resume", h.HandleResume)
	mux.HandleFunc("/health", h.HandleHealth)

	// Serve dashboard
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" || r.URL.Path == "/index.html" {
			tmplPath := filepath.Join("web", "templates", "index.html")
			if _, err := os.Stat(tmplPath); err != nil {
				http.Error(w, "dashboard not found", http.StatusNotFound)
				return
			}
			http.ServeFile(w, r, tmplPath)
			return
		}
		// Serve static assets
		staticDir := filepath.Join("web", "static")
		http.StripPrefix("/static/", http.FileServer(http.Dir(staticDir))).ServeHTTP(w, r)
	})
}
