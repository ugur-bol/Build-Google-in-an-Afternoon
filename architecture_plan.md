# ANTIGRAVITY Architecture Plan

## 1. Purpose

This document defines the architecture for **ANTIGRAVITY**, a localhost-runnable web crawler and real-time search engine built for the “Google in a Day” assignment. The design is intentionally optimized for both **project delivery** and **manual verifiability**. In other words, the system is not only expected to crawl, index, and search correctly, but also to expose its internal decisions in a way that can be **manually verified** from raw storage and API responses.

The architecture prioritizes the following:

- **Language-native implementation** instead of high-level crawling/parsing frameworks.
- **Single-machine scalability** with safe concurrency and controlled load.
- **Live indexing + live search** so search can run while crawling is still active.
- **Human-verifiable scoring** so relevance ranking can be validated directly from persisted data.
- **Simple observability** through a dashboard or CLI that clearly shows crawl progress, queue depth, and throttling state.

---

## 2. Assignment-to-Architecture Mapping

The assignment requires two core capabilities: `index` and `search`.

### 2.1 Index Requirement

`index(origin, k)` must:

- Start from an origin URL.
- Recursively crawl links until maximum depth `k`.
- Never crawl the same page twice.
- Operate safely at large scale on a single machine.
- Enforce back pressure through a bounded queue, worker limit, or crawl rate.

### 2.2 Search Requirement

`search(query)` must:

- Return relevant results while indexing is still running.
- Return triples in the form:
  - `(relevant_url, origin_url, depth)`
- Reflect new results as pages are discovered.
- Apply a simple and explainable relevance heuristic.

### 2.3 UI / CLI Requirement

The system must support:

- Triggering indexing.
- Triggering search.
- Viewing real-time state.

Required visibility:

- URLs processed vs queued.
- Queue depth.
- Back-pressure / throttling status.

### 2.4 Manual Verification Requirement

This assignment benefits from “Human-in-the-Loop” verification. The system should make it easy to confirm correctness by inspection:

1. Crawled data is visible in the repository.
2. The raw storage file `data/storage/p.data` can be inspected.
3. A word appearing on multiple URLs can be found.
4. The API can search that word.
5. The top result can be compared to a manually calculated score.
6. The score formula matches API ranking.

Because of that, this architecture **must explicitly preserve term-level entries** in a raw, line-oriented file format that is human-readable and GitHub-visible.

---

## 3. Recommended Tech Stack

To stay aligned with the instruction to use **language-native functionality**, the recommended implementation stack is:

- **Language:** Go
- **HTTP server:** `net/http`
- **HTTP client / crawling:** `net/http`
- **HTML parsing:** `golang.org/x/net/html` or tokenizer-level parsing logic
- **Concurrency:** goroutines, channels, mutexes, RWMutex, atomic counters
- **Persistence:** plain files in `data/storage/`
- **Frontend/UI:** lightweight server-rendered HTML + JSON endpoints, or CLI fallback

### Why Go?

Go is the best fit for this assignment because it naturally supports:

- bounded worker pools,
- channels for crawl scheduling,
- mutex-protected shared state,
- lightweight HTTP services,
- simple deployment on localhost,
- strong performance without framework dependency.

This choice also maps cleanly to the assignment’s expectation of **architectural sensibility** and native concurrency control.

---

## 4. High-Level System Design

ANTIGRAVITY is composed of **five cooperating subsystems**:

1. **Crawler Coordinator**
2. **Fetcher + Parser Workers**
3. **Indexer / Storage Engine**
4. **Search Engine**
5. **Observability Layer (Dashboard / CLI + Metrics API)**

### 4.1 Data Flow Overview

1. User submits `/index` with `origin` and `maxDepth`.
2. Coordinator seeds a bounded crawl queue with the origin URL.
3. Workers pull tasks from the queue.
4. Each worker:
   - fetches the page,
   - extracts visible text,
   - extracts outgoing links,
   - tokenizes words,
   - computes frequency information,
   - emits index entries,
   - enqueues new links if depth < maxDepth and URL is unseen.
5. Index entries are persisted immediately.
6. Search requests query the in-memory inverted index while crawling is active.
7. Dashboard polls state endpoints to show queue depth, processed count, throttle state, and recent results.

This architecture allows **search and indexing to proceed concurrently**, which is a direct answer to the assignment’s “how would search work while indexing is active?” requirement.

---

## 5. Core Architectural Principle: Dual Storage Model

To satisfy both runtime performance and manual inspectability, the system uses **two storage layers at the same time**:

### 5.1 Runtime In-Memory Index

Used for fast live search while crawling is in progress.

Structure:

- `visited set`: prevents duplicate crawls
- `inverted index`: `word -> []Posting`
- `page metadata`: `url -> metadata`
- `crawl job registry`: current state of active crawl sessions

### 5.2 Raw Persistent Storage

Used for:

- GitHub submission visibility,
- manual inspection,
- manual score calculation,
- optional resume/rebuild flows.

Critical file:

- `data/storage/p.data`

Each line should follow this exact format:

```text
word url origin depth frequency
```

Example:

```text
python https://example.com/tutorial https://example.com 1 7
page https://example.com/about https://example.com 1 3
program https://example.com/docs/install https://example.com 2 5
```

This format is intentionally chosen because the verification process requires a reviewer to:

- open `data/storage/p.data`,
- find a repeated word,
- copy three entries,
- manually compute scores,
- compare those scores to API ranking.

That means the storage format is not just an implementation detail; it is part of the deliverable strategy.

---

## 6. Data Model

### 6.1 Crawl Task

```text
CrawlTask {
  url: string
  origin: string
  depth: int
}
```

### 6.2 Posting

```text
Posting {
  word: string
  url: string
  origin: string
  depth: int
  frequency: int
  titleMatch: bool
  relevanceScore: int
}
```

### 6.3 Page Metadata

```text
PageMeta {
  url: string
  origin: string
  depth: int
  statusCode: int
  fetchedAt: time
  title: string
  wordCount: int
  outgoingLinks: int
}
```

### 6.4 Crawl Metrics

```text
Metrics {
  queued: int
  activeWorkers: int
  processed: int
  failed: int
  skippedVisited: int
  throttled: bool
  maxQueueDepth: int
}
```

---

## 7. Crawling Architecture

### 7.1 Crawl Coordinator

The Crawl Coordinator owns the crawl lifecycle.

Responsibilities:

- accept a new `index(origin, k)` request,
- normalize origin URL,
- initialize session state,
- push the first task into the queue,
- maintain queue bounds,
- manage worker pool,
- expose crawl metrics.

### 7.2 Visited Set

A thread-safe visited registry is mandatory.

Recommended structure:

```text
map[string]bool + RWMutex
```

Before scheduling a discovered URL:

1. normalize URL,
2. remove fragments,
3. optionally canonicalize trailing slashes,
4. check visited set,
5. insert if not already present.

This ensures the same page is never crawled twice.

### 7.3 Worker Pool

Use a fixed-size worker pool instead of unbounded goroutines.

Recommended starting config:

- 5–20 workers depending on machine capacity
- bounded queue capacity, e.g. 500 or 1000 tasks
- per-request timeout, e.g. 5–10 seconds

This satisfies the “large scale on one machine” requirement without creating uncontrolled concurrency.

### 7.4 Link Extraction

Workers should:

- fetch HTML,
- parse anchor tags,
- resolve relative URLs,
- keep only `http` / `https` URLs,
- optionally restrict to same-domain or configurable-domain mode.

Because the assignment does not require full internet-scale crawling, a same-domain default is a reasonable assumption and reduces noise.

### 7.5 Depth Handling

Depth is defined as the number of hops from the origin.

Rules:

- origin page is depth `0`
- direct links from origin are depth `1`
- children of depth `1` pages are depth `2`
- stop enqueueing when `depth == maxDepth`

This aligns precisely with the assignment statement.

---

## 8. Back Pressure Strategy

Back pressure is a required grading criterion, so it must be visible in both the code and the dashboard.

### 8.1 Bounded Queue

The crawl queue must have a fixed maximum capacity.

Example:

- `queue capacity = 1000`

If the queue nears saturation:

- new discovered links are temporarily dropped or deferred,
- throttle status becomes `ON`,
- metrics endpoint reports back-pressure activation.

### 8.2 Rate Limiting

In addition to queue bounds, workers should obey a small fetch rate limit.

Example options:

- token bucket per crawler,
- fixed inter-request delay,
- host-based polite crawling delay.

### 8.3 Worker Cap

The number of concurrent fetches must stay fixed.

Why this matters:

- protects local machine resources,
- prevents file write storms,
- simplifies reasoning about throughput,
- demonstrates controlled system design.

### 8.4 Observable Throttle State

Dashboard / API should expose:

- `throttled: true|false`
- `queueDepth`
- `queueCapacity`
- `workersBusy`

This makes back pressure demonstrable during grading.

---

## 9. Parsing and Tokenization

### 9.1 Text Extraction

The parser should extract visible body text and optionally title text.

Recommended rule:

- include `<title>` text,
- include body text nodes,
- ignore script/style/noscript,
- lowercase all tokens.

### 9.2 Tokenization

Simple tokenizer is sufficient:

- lowercase,
- split on non-alphanumeric boundaries,
- discard empty strings,
- optionally discard 1-character tokens,
- optionally apply stop-word filtering.

Because manual validation is a goal, do **not** overcomplicate preprocessing. Avoid stemming for v1, since it makes raw line inspection harder.

### 9.3 Frequency Counting

For each page:

- build `map[word]frequency`
- emit one posting per unique word for that page

This directly supports line-by-line manual verification.

---

## 10. Indexing Model

### 10.1 Inverted Index Structure

Recommended in-memory structure:

```text
map[string][]Posting
```

Protected by:

```text
RWMutex
```

Write path:

- crawler workers acquire write lock when appending postings

Read path:

- search requests acquire read lock while scoring/sorting

### 10.2 File Persistence

Every posting should also be appended to `data/storage/p.data`.

Append-only write example:

```text
<word> <url> <origin> <depth> <frequency>
```

This file is the raw index ledger.

### 10.3 Metadata Files

Recommended additional files:

- `data/storage/pages.jsonl` → one JSON object per crawled page
- `data/storage/crawl_state.json` → current metrics / session state
- `data/storage/errors.log` → failed fetches / parse errors

Only `p.data` is essential for manual verification, but these extra files improve traceability.

---

## 11. Search Architecture

### 11.1 Search API Contract

Primary endpoint:

```http
GET /search?query=<word>&sortBy=relevance
```

This endpoint must return results compatible with manual verification.

Recommended response shape:

```json
{
  "query": "python",
  "count": 3,
  "results": [
    {
      "url": "https://example.com/tutorial",
      "origin": "https://example.com",
      "depth": 1,
      "frequency": 7,
      "relevance_score": 1065
    }
  ]
}
```

### 11.2 Relevance Formula

To make manual validation straightforward, the API should use a simple, transparent formula:

```text
score = (frequency * 10) + 1000 - (depth * 5)
```

Interpretation:

- `frequency * 10` rewards repeated word presence on the page
- `+1000` acts as the exact-match bonus
- `depth * 5` penalizes pages farther away from the origin

Because a reviewer should be able to perform the same computation by hand, the implementation should not hide or transform the formula.

### 11.3 Search Algorithm

For query term `q`:

1. lowercase and normalize `q`
2. retrieve `postings = invertedIndex[q]`
3. for each posting, compute score
4. sort descending by score
5. return triples / result objects

For v1, search can be **single-term exact match**. This is enough for the assignment requirements.

### 11.4 Live Search While Indexing

This requirement is satisfied because:

- crawlers append postings incrementally,
- postings become visible immediately in memory,
- search reads the index concurrently using read locks.

Thus, users can search while indexing is still active and receive partial but growing result sets.

---

## 12. API Surface

### 12.1 Start Crawl

```http
POST /index
Content-Type: application/json

{
  "origin": "https://example.com",
  "maxDepth": 2
}
```

Response:

```json
{
  "status": "started",
  "origin": "https://example.com",
  "maxDepth": 2
}
```

### 12.2 Search

```http
GET /search?query=python&sortBy=relevance
```

### 12.3 State / Dashboard Data

The dashboard is backed by the live state endpoint:

```http
GET /api/state
```

Response example (shape may evolve, but these are the key metrics):

```json
{
  "processed": 42,
  "queued": 13,
  "active_workers": 5,
  "failed": 2,
  "skipped_visited": 7,
  "throttled": false,
  "max_queue_depth": 100,
  "status": "running"
}
```

### 12.4 Recent Pages / Debug View

```http
GET /pages/recent
```

Optional, but useful for UI visibility.

---

## 13. UI / CLI Plan

The assignment allows either UI or CLI, but the strongest submission is a **minimal web dashboard** plus curl-friendly APIs.

### 13.1 Dashboard Sections

#### A. Crawl Control

- input: origin URL
- input: max depth
- button: Start Indexing

#### B. System State

- URLs processed
- URLs queued
- active workers
- failures
- throttle status
- current queue depth

#### C. Search Panel

- query textbox
- search button
- result table with:
  - URL
  - origin
  - depth
  - frequency
  - relevance_score

#### D. Raw Storage Hint

A small debug/info section can mention:

- raw postings are written to `data/storage/p.data`

This is useful because it implicitly guides the evaluator toward the verifiable storage artifact.

### 13.2 Why a Dashboard Is Better Than CLI Alone

A dashboard better demonstrates:

- real-time visibility,
- back-pressure status,
- search during indexing,
- architectural maturity.

Still, the underlying APIs should remain usable from CLI for grading simplicity.

---

## 14. Thread Safety Design

Concurrency correctness is explicitly graded, so shared state must be clearly protected.

### 14.1 Protected Shared Structures

- `visited set` → RWMutex
- `inverted index` → RWMutex
- `metrics counters` → atomic or mutex
- `file writes` → single writer goroutine or file mutex

### 14.2 Recommended File Write Pattern

To avoid corrupted writes to `p.data`, use one of these patterns:

#### Option A: File Mutex

All workers lock a shared file mutex before append.

#### Option B: Single Storage Writer Goroutine (Preferred)

Workers send posting records over a channel to a dedicated storage writer goroutine.

Advantages:

- ordered writes,
- fewer lock collisions,
- safer persistence,
- cleaner architecture.

This is the better design choice for the final explanation section.

---

## 15. Persistence and Resume (Bonus)

The assignment says resume support is a plus, not mandatory. ANTIGRAVITY should be designed so this bonus can be added cleanly.

### 15.1 Minimal Resume Strategy

Persist:

- visited URLs,
- pending queue,
- crawl configuration,
- metrics snapshot.

Files:

- `data/storage/visited.json`
- `data/storage/queue.json`
- `data/storage/crawl_state.json`

On restart:

- reload pending queue,
- reload visited set,
- continue worker execution.

### 15.2 Why This Is Bonus-Safe

The main grading priority is functionality and architecture. Therefore resume support should be implemented only if time remains after the base crawler/search/dashboard works reliably.

---

## 16. Repository Structure

Recommended public GitHub structure:

```text
ANTIGRAVITY/
├─ cmd/
│  └─ server/
│     └─ main.go
├─ internal/
│  ├─ api/
│  │  ├─ handlers.go
│  │  └─ routes.go
│  ├─ crawler/
│  │  ├─ crawler.go
│  │  ├─ fetcher.go
│  │  └─ parser.go
│  ├─ indexer/
│  │  ├─ indexer.go
│  │  └─ storage.go
│  ├─ models/
│  │  └─ models.go
│  ├─ normalize/
│  │  └─ normalize.go
│  ├─ search/
│  │  └─ search.go
│  └─ state/
│     └─ state.go
├─ web/
│  ├─ templates/
│  │  └─ index.html
│  └─ static/
│     ├─ app.js
│     └─ styles.css
├─ data/
│  └─ storage/
│     ├─ p.data
│     ├─ pages.jsonl
├─ product_prd.md
├─ architecture_plan.md
├─ recommendation.md
├─ README.md
├─ go.mod
└─ go.sum
```

This structure also helps the evaluator quickly locate the required deliverables.

---

## 17. How This Architecture Supports Human-in-the-Loop Verification

Manual verification is not separate from the implementation; it is a deliberate design goal that makes the system easier to audit and explain.

### 17.1 Raw Storage Requirement

Because `p.data` stores lines in the form:

```text
word url origin depth frequency
```

it becomes trivial to:

- find a repeated word,
- copy three entries,
- manually calculate each score.

### 17.2 API Verification Requirement

Because `/search?query=<word>&sortBy=relevance` uses the same underlying posting data and the exact same score formula, the student can verify that:

- the highest manual score matches API rank #1,
- the system is explainable and deterministic.

### 17.3 Human-in-the-Loop Requirement

This design demonstrates Human-in-the-Loop verification by making the internal ranking pipeline inspectable:

- raw storage is visible,
- score formula is simple,
- API output is transparent,
- dashboard exposes live state.

That directly supports the assignment’s “AI Stewardship” evaluation criterion.

---

## 18. “Chain-of-Thought” Style Enhancement (Explainability)

A strong answer is:

> The system could be enhanced by making ranking more explicitly stepwise and inspectable. Instead of returning only the final relevance score, the search engine could expose a score breakdown per result, such as frequency contribution, exact-match bonus, and depth penalty. This would let a human reviewer follow the ranking logic step by step, verify why one page outranks another, and debug unexpected search results more easily. In a broader Chain-of-Thought style architecture, the crawler and search engine could also log intermediate reasoning states such as why a URL was skipped, why throttling activated, or how normalization changed a query term before scoring.
>
> Additionally, the system could introduce a transparent ranking explanation layer in the UI. For each search result, the interface could display the raw posting line from `p.data`, the parsed fields, and the exact arithmetic used to compute the final score. That would turn the search engine from a black box into an auditable pipeline, improving trust, debugging speed, and maintainability while staying aligned with the assignment’s Human-in-the-Loop philosophy.

---

## 19. Incremental Build Plan

The assignment explicitly prefers an iterative workflow. The implementation should therefore proceed in the following order.

### Phase 1 — Core Crawler

Build first:

- crawl queue
- worker pool
- visited set
- fetch + parse
- depth-limited traversal

Success condition:

- pages crawl without duplicates
- depth logic works

### Phase 2 — Index Persistence

Build next:

- tokenization
- frequency counting
- append to `data/storage/p.data`
- page metadata logging

Success condition:

- raw postings file grows correctly
- repeated words appear across multiple URLs

### Phase 3 — Live Search

Build next:

- in-memory inverted index
- `/search` API
- relevance sorting
- exact formula implementation

Success condition:

- a word found in raw storage can be searched through the API
- top result matches manual score

### Phase 4 — Metrics and Dashboard

Build next:

- `/metrics`
- dashboard UI
- crawl control panel
- search results table

Success condition:

- real-time system state visible during crawling

### Phase 5 — Bonus Persistence / Resume

Build last if time remains:

- reloadable queue
- stored visited set
- session restart support

---

## 20. Non-Goals for v1

To stay within the 3–5 hour constraint, ANTIGRAVITY v1 should **not** attempt:

- distributed crawling,
- advanced NLP relevance,
- PageRank,
- JS-rendered page execution,
- robots.txt policy engine beyond optional notes,
- duplicate content hashing,
- stemming/lemmatization,
- multi-term boolean query language.

These can be listed as future enhancements, but should not distract from a stable, explainable implementation.

---

## 21. Risk Management

### Risk 1: Overengineering

Mitigation:

- keep ranking simple,
- keep parser simple,
- prefer deterministic file-based persistence.

### Risk 2: Concurrency bugs

Mitigation:

- protect shared maps,
- centralize file writing,
- keep worker boundaries explicit.

### Risk 3: Verification mismatch

Mitigation:

- ensure `p.data` exists in repo,
- ensure format is exactly `word url origin depth frequency`,
- ensure `/search?query=...&sortBy=relevance` works,
- ensure score formula exactly matches the documented arithmetic.

### Risk 4: Weak architectural explanation

Mitigation:

- document back pressure clearly,
- document live search clearly,
- justify native library choices,
- explain why the dual storage model exists.

---

## 22. Final Architectural Recommendation

The best version of ANTIGRAVITY is a **single-binary Go application** that runs locally, exposes HTTP endpoints, serves a small dashboard, writes a human-readable raw postings file, and supports concurrent crawling and search through thread-safe in-memory indexing. This design is strong because it is simple enough to finish in the project timeframe, yet sophisticated enough to demonstrate concurrency control, back pressure, observability, and explainable relevance ranking.

Most importantly, this architecture is deliberately shaped around manual verification. By making `data/storage/p.data` a first-class output and by using a transparent formula for `relevance_score`, the system turns internal search behavior into something a reviewer can manually inspect, compute, and defend. That makes the implementation not only functional, but easy to evaluate and explain.
