package models

import "time"

// CrawlTask represents a single unit of work for the crawler.
type CrawlTask struct {
	URL    string `json:"url"`
	Origin string `json:"origin"`
	Depth  int    `json:"depth"`
}

// Posting represents a single term→document entry in the inverted index.
type Posting struct {
	Word           string  `json:"word"`
	URL            string  `json:"relevant_url"`
	Origin         string  `json:"origin_url"`
	Depth          int     `json:"depth"`
	Frequency      int     `json:"frequency"`
	RelevanceScore float64 `json:"relevance_score"`
}

// PageMeta stores metadata about a fetched page.
type PageMeta struct {
	URL           string    `json:"url"`
	Origin        string    `json:"origin"`
	Depth         int       `json:"depth"`
	Title         string    `json:"title"`
	StatusCode    int       `json:"status_code"`
	FetchedAt     time.Time `json:"fetched_at"`
	OutgoingLinks int       `json:"outgoing_links"`
	WordCount     int       `json:"word_count"`
}

// Metrics holds live system state for the dashboard.
type Metrics struct {
	Queued         int64  `json:"queued"`
	Processed      int64  `json:"processed"`
	ActiveWorkers  int64  `json:"active_workers"`
	Failed         int64  `json:"failed"`
	SkippedVisited int64  `json:"skipped_visited"`
	Throttled      bool   `json:"throttled"`
	MaxQueueDepth  int64  `json:"max_queue_depth"`
	Status         string `json:"status"` // "idle", "running", "paused", "done"
}

// CrawlJob records a completed or in-progress crawl.
type CrawlJob struct {
	ID        int       `json:"id"`
	Origin    string    `json:"origin"`
	MaxDepth  int       `json:"max_depth"`
	Status    string    `json:"status"` // "running", "paused", "done"
	StartedAt time.Time `json:"started_at"`
	Pages     int64     `json:"pages"`
	Failed    int64     `json:"failed"`
}

// IndexRequest is the JSON body for POST /index.
type IndexRequest struct {
	Origin   string `json:"origin"`
	MaxDepth int    `json:"maxDepth"`
}

// IndexResponse is the JSON reply for POST /index.
type IndexResponse struct {
	Status  string  `json:"status"`
	Message string  `json:"message"`
	Origin  string  `json:"origin"`
	Depth   int     `json:"maxDepth"`
	Metrics Metrics `json:"metrics"`
}

// SearchResponse wraps search results.
type SearchResponse struct {
	Query   string    `json:"query"`
	Count   int       `json:"count"`
	Results []Posting `json:"results"`
}

// RelevanceScore computes score = (frequency * 10) + 1000 - (depth * 5)
func RelevanceScore(frequency, depth int) float64 {
	return float64(frequency*10) + 1000.0 - float64(depth*5)
}
