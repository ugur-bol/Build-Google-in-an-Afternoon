package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"antigravity/internal/crawler"
	"antigravity/internal/indexer"
	"antigravity/internal/models"
	"antigravity/internal/search"
	"antigravity/internal/state"
)

// Handlers holds references to all subsystems needed by the HTTP handlers.
type Handlers struct {
	Crawler *crawler.Crawler
	Indexer *indexer.Indexer
	Search  *search.Engine
	State   *state.Engine
	Storage *indexer.Storage
}

// HandleIndex starts a crawl job.
func (h *Handlers) HandleIndex(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.IndexRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}

	if req.Origin == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "origin is required"})
		return
	}
	if req.MaxDepth < 0 {
		req.MaxDepth = 0
	}
	if req.MaxDepth > 10 {
		req.MaxDepth = 10 // safety cap
	}

	// Create a fresh crawler for this job
	c := crawler.New(h.Indexer, h.Storage, h.State)
	h.Crawler = c

	// Register the job in history
	h.State.AddJob(req.Origin, req.MaxDepth)

	go c.Start(req.Origin, req.MaxDepth)

	log.Printf("[api] index started: origin=%s maxDepth=%d", req.Origin, req.MaxDepth)

	resp := models.IndexResponse{
		Status:  "accepted",
		Message: "Crawl job started",
		Origin:  req.Origin,
		Depth:   req.MaxDepth,
		Metrics: h.State.Snapshot(),
	}
	writeJSON(w, http.StatusAccepted, resp)
}

// HandlePause pauses the active crawler.
func (h *Handlers) HandlePause(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if h.Crawler == nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "no active crawl"})
		return
	}
	h.Crawler.Pause()
	writeJSON(w, http.StatusOK, map[string]string{"status": "paused"})
}

// HandleResume resumes a paused crawler.
func (h *Handlers) HandleResume(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if h.Crawler == nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "no active crawl"})
		return
	}
	h.Crawler.Resume()
	writeJSON(w, http.StatusOK, map[string]string{"status": "resumed"})
}

// HandleSearch handles search queries.
func (h *Handlers) HandleSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	query := r.URL.Query().Get("query")
	if strings.TrimSpace(query) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "query parameter is required"})
		return
	}

	result := h.Search.Query(query)
	writeJSON(w, http.StatusOK, result)
}

// HandleState returns current system metrics.
func (h *Handlers) HandleState(w http.ResponseWriter, r *http.Request) {
	metrics := h.State.Snapshot()

	// Add index stats
	words, postings := h.Indexer.Stats()
	type stateResp struct {
		models.Metrics
		IndexedWords  int              `json:"indexed_words"`
		TotalPostings int              `json:"total_postings"`
		Jobs          []models.CrawlJob `json:"jobs"`
	}

	resp := stateResp{
		Metrics:       metrics,
		IndexedWords:  words,
		TotalPostings: postings,
		Jobs:          h.State.Jobs(),
	}
	writeJSON(w, http.StatusOK, resp)
}

// HandleHealth returns basic health check.
func (h *Handlers) HandleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
