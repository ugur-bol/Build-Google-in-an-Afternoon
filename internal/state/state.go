package state

import (
	"antigravity/internal/models"
	"sync"
	"sync/atomic"
	"time"
)

// Engine holds all shared, thread-safe counters and status flags.
type Engine struct {
	queued         atomic.Int64
	processed      atomic.Int64
	activeWorkers  atomic.Int64
	failed         atomic.Int64
	skippedVisited atomic.Int64
	maxQueueDepth  atomic.Int64
	throttled      atomic.Int64 // 0=false, 1=true
	status         atomic.Value // string: "idle", "running", "paused", "done"

	// Crawl history
	jobsMu  sync.RWMutex
	jobs    []models.CrawlJob
	nextID  int
}

// New creates a new Engine in idle state.
func New() *Engine {
	e := &Engine{}
	e.status.Store("idle")
	return e
}

func (e *Engine) IncQueued()         { e.queued.Add(1) }
func (e *Engine) DecQueued()         { e.queued.Add(-1) }
func (e *Engine) IncProcessed()      { e.processed.Add(1) }
func (e *Engine) IncActiveWorkers()  { e.activeWorkers.Add(1) }
func (e *Engine) DecActiveWorkers()  { e.activeWorkers.Add(-1) }
func (e *Engine) IncFailed()         { e.failed.Add(1) }
func (e *Engine) IncSkippedVisited() { e.skippedVisited.Add(1) }

func (e *Engine) SetThrottled(v bool) {
	if v {
		e.throttled.Store(1)
	} else {
		e.throttled.Store(0)
	}
}

func (e *Engine) SetStatus(s string) {
	e.status.Store(s)
	// Update the latest job status
	e.jobsMu.Lock()
	if len(e.jobs) > 0 {
		last := &e.jobs[len(e.jobs)-1]
		if s == "done" || s == "paused" || s == "running" {
			last.Status = s
			last.Pages = e.processed.Load()
			last.Failed = e.failed.Load()
		}
	}
	e.jobsMu.Unlock()
}

func (e *Engine) UpdateMaxQueueDepth(current int64) {
	for {
		old := e.maxQueueDepth.Load()
		if current <= old {
			return
		}
		if e.maxQueueDepth.CompareAndSwap(old, current) {
			return
		}
	}
}

// AddJob registers a new crawl job.
func (e *Engine) AddJob(origin string, maxDepth int) int {
	e.jobsMu.Lock()
	defer e.jobsMu.Unlock()
	e.nextID++
	job := models.CrawlJob{
		ID:        e.nextID,
		Origin:    origin,
		MaxDepth:  maxDepth,
		Status:    "running",
		StartedAt: time.Now(),
	}
	e.jobs = append(e.jobs, job)
	return e.nextID
}

// Jobs returns a copy of all crawl jobs.
func (e *Engine) Jobs() []models.CrawlJob {
	e.jobsMu.RLock()
	defer e.jobsMu.RUnlock()

	// Update latest job with live counters
	if len(e.jobs) > 0 {
		last := e.jobs[len(e.jobs)-1]
		status, _ := e.status.Load().(string)
		if status == "running" || status == "paused" {
			last.Pages = e.processed.Load()
			last.Failed = e.failed.Load()
			last.Status = status
			result := make([]models.CrawlJob, len(e.jobs))
			copy(result, e.jobs)
			result[len(result)-1] = last
			return result
		}
	}

	result := make([]models.CrawlJob, len(e.jobs))
	copy(result, e.jobs)
	return result
}

// Snapshot returns a copy of all current metrics.
func (e *Engine) Snapshot() models.Metrics {
	status, _ := e.status.Load().(string)
	if status == "" {
		status = "idle"
	}
	return models.Metrics{
		Queued:         e.queued.Load(),
		Processed:      e.processed.Load(),
		ActiveWorkers:  e.activeWorkers.Load(),
		Failed:         e.failed.Load(),
		SkippedVisited: e.skippedVisited.Load(),
		Throttled:      e.throttled.Load() == 1,
		MaxQueueDepth:  e.maxQueueDepth.Load(),
		Status:         status,
	}
}
