package crawler

import (
	"context"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"antigravity/internal/indexer"
	"antigravity/internal/models"
	"antigravity/internal/normalize"
	"antigravity/internal/state"
)

const (
	MaxWorkers   = 10
	MaxQueueSize = 10000
	FetchTimeout = 10 * time.Second
	PolitenessMs = 100 // milliseconds between requests per worker
)

// Crawler manages the crawl job with a bounded worker pool.
type Crawler struct {
	fetcher *Fetcher
	idx     *indexer.Indexer
	storage *indexer.Storage
	state   *state.Engine

	queue   chan models.CrawlTask
	visited sync.Map // map[string]bool — thread-safe visited set

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Pause/Resume support
	paused   atomic.Bool
	pauseMu  sync.Mutex
	pauseCh  chan struct{} // closed when unpaused
}

// New creates a new Crawler.
func New(idx *indexer.Indexer, storage *indexer.Storage, st *state.Engine) *Crawler {
	ctx, cancel := context.WithCancel(context.Background())
	c := &Crawler{
		fetcher: NewFetcher(FetchTimeout),
		idx:     idx,
		storage: storage,
		state:   st,
		queue:   make(chan models.CrawlTask, MaxQueueSize),
		ctx:     ctx,
		cancel:  cancel,
		pauseCh: make(chan struct{}),
	}
	close(c.pauseCh) // start unpaused
	return c
}

// Start begins a crawl from origin URL to maxDepth.
func (c *Crawler) Start(origin string, maxDepth int) {
	origin = normalize.NormalizeURL(origin)

	c.state.SetStatus("running")

	// Seed the queue
	seed := models.CrawlTask{
		URL:    origin,
		Origin: origin,
		Depth:  0,
	}
	c.enqueue(seed)

	// Launch worker pool
	for i := 0; i < MaxWorkers; i++ {
		c.wg.Add(1)
		go c.worker(i, maxDepth)
	}

	// Monitor completion in background
	go func() {
		c.wg.Wait()
		if !c.paused.Load() {
			c.state.SetStatus("done")
		}
		log.Println("[crawler] all workers finished")
	}()
}

// Pause pauses the crawler. Workers finish their current task then wait.
func (c *Crawler) Pause() {
	c.pauseMu.Lock()
	defer c.pauseMu.Unlock()
	if c.paused.Load() {
		return
	}
	c.paused.Store(true)
	c.pauseCh = make(chan struct{}) // new blocking channel
	c.state.SetStatus("paused")
	log.Println("[crawler] paused")
}

// Resume resumes a paused crawler.
func (c *Crawler) Resume() {
	c.pauseMu.Lock()
	defer c.pauseMu.Unlock()
	if !c.paused.Load() {
		return
	}
	c.paused.Store(false)
	close(c.pauseCh) // unblock all waiting workers
	c.state.SetStatus("running")
	log.Println("[crawler] resumed")
}

// IsPaused returns whether the crawler is currently paused.
func (c *Crawler) IsPaused() bool {
	return c.paused.Load()
}

// waitIfPaused blocks the calling goroutine while the crawler is paused.
func (c *Crawler) waitIfPaused() {
	c.pauseMu.Lock()
	ch := c.pauseCh
	c.pauseMu.Unlock()
	<-ch // returns immediately if channel is closed (unpaused)
}

// enqueue adds a task to the queue if not already visited.
func (c *Crawler) enqueue(task models.CrawlTask) {
	normalizedURL := normalize.NormalizeURL(task.URL)
	task.URL = normalizedURL

	if _, loaded := c.visited.LoadOrStore(normalizedURL, true); loaded {
		c.state.IncSkippedVisited()
		return
	}

	// Check if queue is near capacity → set throttled
	qLen := int64(len(c.queue))
	if qLen >= int64(MaxQueueSize)*9/10 {
		c.state.SetThrottled(true)
	} else {
		c.state.SetThrottled(false)
	}
	c.state.UpdateMaxQueueDepth(qLen + 1)

	select {
	case c.queue <- task:
		c.state.IncQueued()
	default:
		// Queue full, mark throttled
		c.state.SetThrottled(true)
		log.Printf("[crawler] queue full, dropping: %s", task.URL)
	}
}

// worker processes tasks from the queue.
func (c *Crawler) worker(id int, maxDepth int) {
	defer c.wg.Done()
	c.state.IncActiveWorkers()
	defer c.state.DecActiveWorkers()

	// Idle timeout: if no tasks arrive in 5 seconds, assume done
	idleTimeout := time.NewTimer(5 * time.Second)
	defer idleTimeout.Stop()

	for {
		// Check pause before picking up work
		c.waitIfPaused()

		idleTimeout.Reset(5 * time.Second)
		select {
		case <-c.ctx.Done():
			return
		case task, ok := <-c.queue:
			if !ok {
				return
			}
			c.state.DecQueued()
			c.processTask(task, maxDepth)

			// Politeness delay
			time.Sleep(time.Duration(PolitenessMs) * time.Millisecond)

		case <-idleTimeout.C:
			// No more tasks, worker exits
			return
		}
	}
}

// processTask fetches, parses, indexes, and persists a single page.
func (c *Crawler) processTask(task models.CrawlTask, maxDepth int) {
	log.Printf("[worker] depth=%d url=%s", task.Depth, task.URL)

	// Fetch
	result, err := c.fetcher.Fetch(c.ctx, task.URL)
	if err != nil {
		log.Printf("[worker] fetch error: %s — %v", task.URL, err)
		c.state.IncFailed()
		return
	}

	if result.StatusCode < 200 || result.StatusCode >= 300 {
		log.Printf("[worker] non-2xx status %d: %s", result.StatusCode, task.URL)
		c.state.IncFailed()
		return
	}

	// Parse
	parsed := Parse(result.Body, task.URL)

	// Tokenize and count
	tokens := normalize.Tokenize(parsed.Text)
	freqs := normalize.WordFrequencies(tokens)

	// Add to in-memory index
	c.idx.Add(freqs, task.URL, task.Origin, task.Depth)

	// Persist to p.data
	if err := c.storage.WritePostings(freqs, task.URL, task.Origin, task.Depth); err != nil {
		log.Printf("[worker] storage write error: %v", err)
	}

	// Persist page metadata
	meta := models.PageMeta{
		URL:           task.URL,
		Origin:        task.Origin,
		Depth:         task.Depth,
		Title:         parsed.Title,
		StatusCode:    result.StatusCode,
		FetchedAt:     time.Now(),
		OutgoingLinks: len(parsed.Links),
		WordCount:     len(tokens),
	}
	if err := c.storage.WritePageMeta(meta); err != nil {
		log.Printf("[worker] page meta write error: %v", err)
	}

	c.state.IncProcessed()

	// Enqueue child links if within depth limit
	nextDepth := task.Depth + 1
	if nextDepth <= maxDepth {
		for _, link := range parsed.Links {
			child := models.CrawlTask{
				URL:    link,
				Origin: task.Origin,
				Depth:  nextDepth,
			}
			c.enqueue(child)
		}
	}
}

// Stop signals the crawler to stop.
func (c *Crawler) Stop() {
	c.cancel()
}
