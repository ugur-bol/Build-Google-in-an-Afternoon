# ANTIGRAVITY — Product Requirements Document (PRD)

## Context / problem statement

Build a functional **web crawler** and **real-time search engine** from scratch that runs on **localhost**. The solution must demonstrate:

- **Architectural sensibility** (single-machine scalability, clear boundaries)
- **Concurrency management** (thread-safe structures, no corruption)
- **Back pressure** (bounded queue/rate of work)
- **Human-in-the-Loop verification** (results can be manually inspected and validated)

## Goals (what “success” means)

- **G1 — Working indexer**: `index(origin, k)` crawls recursively up to depth \(k\), never crawling the same URL twice.
- **G2 — Working searcher**: `search(query)` returns relevant results and includes the required triple \((relevant\_url, origin\_url, depth)\).
- **G3 — Live indexing**: search works while indexing is active and reflects newly discovered results.
- **G4 — Visibility**: a simple dashboard + metrics endpoint show crawl progress, queue depth, and throttling/back-pressure status.
- **G5 — Verifiability**: raw term postings are persisted in a human-readable file (`data/storage/p.data`) so a reviewer can validate ranking manually.

## Non-goals (v1)

- Distributed crawling (multi-machine)
- JavaScript rendering / headless browser crawling
- NLP relevance (stemming, synonyms, embeddings)
- Production hardening (auth, multi-tenant isolation, TLS, global scale)

## Users / reviewers

- **Primary**: course evaluator running the system locally
- **Secondary**: quiz reviewer who manually inspects `data/storage/p.data` and cross-checks API results

## Assumptions & constraints

- Crawl scale can be “large”, but must run on a **single machine**.
- Use **language-native** components for core functionality (Go `net/http`, goroutines, channels, mutexes; HTML tokenization via `golang.org/x/net/html`).
- Ranking heuristic must be **simple and explainable**.

## User journeys

### Journey A — Start a crawl and observe the system

1. User enters `origin` URL + `maxDepth` in the dashboard (or via API).
2. System begins crawling with a bounded queue + worker pool.
3. User monitors progress and back-pressure status via `GET /api/state`.

### Journey B — Search while indexing is still running

1. While crawl is active, user submits a query in the dashboard (or `GET /search`).
2. Results return immediately from the in-memory inverted index.
3. As new pages are indexed, subsequent searches include new results.

### Journey C — Manual (quiz) verification

1. Reviewer opens `data/storage/p.data` and finds a word present on multiple URLs.
2. Reviewer computes expected score from stored `(depth, frequency)`.
3. Reviewer calls `/search?query=<word>&sortBy=relevance` and verifies rank #1 matches the highest manual score.

## Functional requirements

### FR1 — Indexing API (`index(origin, k)`)

- Endpoint: `POST /index` with JSON body `{ "origin": string, "maxDepth": number }`
- Depth semantics: origin is depth `0`; each hop increments depth by `1`; stop enqueueing new links at `maxDepth`
- Never crawl the same URL twice (visited set)
- Must implement back pressure (bounded queue capacity and/or throttle)

### FR2 — Search API (`search(query)`)

- Endpoint: `GET /search?query=<term>&sortBy=relevance`
- Returns results that include the triple:
  - `(relevant_url, origin_url, depth)`
- Must work while indexing is active (concurrent reads of the index)
- Results are ranked by a transparent relevance heuristic

### FR3 — System visibility (UI + metrics)

- Dashboard served at `/` shows:
  - indexing progress (processed vs queued)
  - queue depth
  - throttled/back-pressure status
- Metrics endpoint: `GET /api/state` returns the same key signals for programmatic inspection

### FR4 — Persistence for verification

- `data/storage/p.data` contains term postings with the exact line format:

```text
word url origin depth frequency
```

- `data/storage/pages.jsonl` stores page metadata (JSONL)

## Relevance heuristic (v1)

\[
relevance\_score = (frequency \times 10) + 1000 - (depth \times 5)
\]

This is intentionally simple so it can be manually verified from `p.data`.

## Non-functional requirements

- **Thread safety**: shared structures are protected (mutex/RWMutex/atomic where appropriate)
- **Controlled load**: bounded work queue and fixed worker concurrency
- **Resilience**: fetch/parsing failures do not crash the server; failures are counted in state
- **Inspectability**: clear API, predictable storage format, and deterministic scoring

## API surface (implementation contract)

| Endpoint | Method | Purpose |
|---|---:|---|
| `/` | GET | Dashboard UI |
| `/index` | POST | Start crawl job |
| `/search` | GET | Query index (works during crawl) |
| `/api/state` | GET | Live system state + metrics |
| `/health` | GET | Liveness check |

## Acceptance criteria (grader checklist)

1. Server runs on **`localhost:3600`**
2. `POST /index` starts a crawl job and returns an accepted response
3. System enforces uniqueness (no duplicate crawls)
4. System demonstrates back pressure (bounded queue and throttle/limit visibility)
5. `GET /search` returns results while indexing is still running (live indexing)
6. Search response includes the required triple \((relevant\_url, origin\_url, depth)\)
7. `data/storage/p.data` exists and follows the required format
8. `relevance_score` in API matches manual calculation from `p.data`
9. Dashboard + `GET /api/state` make system state visible in real time

## Risks & trade-offs (v1)

| Risk | Impact | Mitigation |
|---|---|---|
| Memory pressure on large crawls | Index growth | bounded queue, depth cap, controlled concurrency |
| Slow/unreliable sites | crawl stalls / errors | timeouts, failure accounting, graceful error handling |
| Data corruption under concurrency | incorrect results | RWMutex/Mutex/atomic usage, single-writer storage |
| Overengineering | missed deadline | simple ranking, simple parsing, focus on rubric |

## Future enhancements (post-assignment)

- Resume/restart support (persist visited set + frontier)
- robots.txt compliance + per-host scheduling
- Stronger ranking (TF‑IDF/BM25), multi-term queries
- Persistent/sharded index for scale-out deployments
