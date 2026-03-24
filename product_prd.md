# ANTIGRAVITY — Product Requirements Document

## Problem Statement

A datathon/assignment requires building a functional single-machine web crawler and search engine that:
1. Crawls web pages from a given origin URL up to a configurable depth
2. Indexes content and provides real-time search with transparent relevance scoring
3. Persists raw term data in a human-inspectable format for quiz verification

## Goals

- **Primary:** Build a working crawler + search engine on localhost
- **Primary:** Score well on assignment requirements AND the quiz that inspects raw storage
- **Primary:** Transparent, verifiable relevance scoring formula

## Non-Goals

- Distributed crawling across multiple machines
- Full-text search with stemming, synonyms, or NLP
- JavaScript rendering / SPA crawling
- Production-grade deployment (TLS, auth, load balancing)

## Users

- Assignment evaluators running on localhost
- Quiz/interview panel inspecting code structure and `p.data`

## Functional Requirements

### FR-1: Indexing
- Accept origin URL and max depth via `POST /index`
- Crawl recursively, never visit same page twice
- Handle large scope via bounded queue and worker pool
- Search remains functional during active crawling

### FR-2: Search
- Accept query via `GET /search?query=<word>&sortBy=relevance`
- Return results with: `relevant_url`, `origin_url`, `depth`, `frequency`, `relevance_score`
- Score formula: `(frequency × 10) + 1000 - (depth × 5)`
- Sort descending by `relevance_score`

### FR-3: System Visibility
- `GET /api/state` returns live metrics
- Dashboard displays real-time: processed, queued, active workers, throttled status, failed counts

### FR-4: Raw Storage
- Persist term data in `data/storage/p.data`
- Format: `word url origin depth frequency` (one per line)
- Human-readable and GitHub-visible

## Non-Functional Requirements

- **Performance:** 10 concurrent workers, 100ms politeness delay
- **Safety:** Thread-safe index and visited set using Go concurrency primitives
- **Reliability:** Graceful handling of fetch failures, non-2xx responses
- **Inspectability:** Clean code, obvious formula, reviewable in quiz

## API Requirements

| Endpoint | Method | Description |
|---|---|---|
| `/index` | POST | Start crawl job |
| `/search` | GET | Search indexed content |
| `/api/state` | GET | System metrics |
| `/health` | GET | Health check |
| `/` | GET | Dashboard UI |

## Storage Requirements

| File | Format | Purpose |
|---|---|---|
| `data/storage/p.data` | `word url origin depth frequency` | Quiz-inspectable term data |
| `data/storage/pages.jsonl` | JSON lines | Page metadata |

## Observability Requirements

- Atomic counters for all system metrics
- 2-second polling from dashboard
- Health endpoint for liveness check

## Acceptance Criteria

1. Server runs on `localhost:3600`
2. POST /index starts crawling and returns accepted status
3. GET /search returns results sorted by relevance formula
4. Search works while indexing is active
5. No duplicate page crawls
6. `p.data` is created with correct format
7. API `relevance_score` matches manual calculation from `p.data`
8. Dashboard shows live queue depth and system status

## Risks & Tradeoffs

| Risk | Mitigation |
|---|---|
| Memory pressure from large crawls | Queue cap (10k), depth cap (10) |
| Slow external sites | 10s fetch timeout, failure counting |
| Concurrent data corruption | RWMutex on index, Mutex on storage |
| Queue saturation | Throttle flag, dropped URLs logged |

## Future Enhancements

- Distributed crawling with persistent job queues
- robots.txt compliance
- Index persistence and reload on startup
- TF-IDF or BM25 ranking
- Sharded index for horizontal scaling
- Full-text search with stemming
