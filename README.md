# ANTIGRAVITY (Project 1: “Google in a Day”)

Real-time web crawler + search engine that runs on **localhost** as a single Go binary. Designed to demonstrate **architecture**, **concurrency**, **back pressure**, and **Human-in-the-Loop verification** (via an inspectable raw storage file).

## What this project delivers (assignment mapping)

- **Indexer / Recursive crawling**: crawl from an `origin` URL up to max depth `k`
- **Uniqueness**: never crawl the same URL twice (visited set)
- **Back pressure**: bounded queue + worker pool + throttle indicator
- **Native focus**: built with Go’s standard library (`net/http`) + tokenizer parsing via `golang.org/x/net/html`
- **Searcher**: query returns relevant results as **triples** \((relevant\_url, origin\_url, depth)\) plus scoring fields
- **Live indexing**: search reads the in-memory index while crawling is still active
- **System visibility**: dashboard + `GET /api/state` for live metrics
- **Human-verifiable output**: raw postings persisted to `data/storage/p.data`

## Architecture (high level)

```
┌──────────────────────────────────────────────┐
│                HTTP Server (:3600)           │
│   /index  /search  /api/state  /health  /    │
└─────┬──────┬──────────┬──────────┬──────────┘
      │      │          │          │
      ▼      │          │          │
┌──────────┐ │    ┌──────────┐  ┌──────────┐
│ Crawler  │ │    │ Search   │  │  State   │
│ (workers │ │    │ Engine   │  │ (atomic) │
│ + queue) │ │    └────┬─────┘  └──────────┘
└────┬─────┘ │         │
     │       │    ┌────▼─────┐
     │       └───▶│ Indexer  │  (thread-safe inverted index)
     │            └────┬─────┘
     │                 │
     │            ┌────▼─────┐
     └───────────▶│ Storage  │  `p.data`, `pages.jsonl`
                  └──────────┘
```

## Quick start

```bash
# Build
go build -o antigravity ./cmd/server

# Run
./antigravity
```

- **Dashboard**: `http://localhost:3600`
- **Health**: `http://localhost:3600/health`

## API

### Start indexing (crawl)

```bash
curl -X POST http://localhost:3600/index \
  -H "Content-Type: application/json" \
  -d '{"origin":"https://example.com","maxDepth":2}'
```

### Search (works during indexing)

```bash
curl "http://localhost:3600/search?query=example&sortBy=relevance"
```

### Live state / metrics

```bash
curl http://localhost:3600/api/state
```

### Health check

```bash
curl http://localhost:3600/health
```

## Storage (Human-in-the-Loop verification)

### `data/storage/p.data`

Append-only, line-oriented, human-readable postings ledger:

```text
word url origin depth frequency
```

Example:

```text
python https://example.com/tutorial https://example.com 1 7
programming https://example.com/about https://example.com 0 3
```

### `data/storage/pages.jsonl`

One JSON object per line with page metadata (URL, title, status, word count, etc.).

## Relevance scoring (human-verifiable)

\[
relevance\_score = (frequency \times 10) + 1000 - (depth \times 5)
\]

Implemented in `internal/models/models.go` as `RelevanceScore(frequency, depth)` and returned directly by the search API.

## Manual verification steps

1. **Start a crawl** (dashboard or `POST /index`).
2. **Open** `data/storage/p.data` and find a word that appears on multiple URLs.
3. **Compute** the score for a few lines using the formula above.
4. **Search** the same word:

```bash
curl "http://localhost:3600/search?query=<word>&sortBy=relevance"
```

5. **Verify** the top result’s `relevance_score` matches the highest manual calculation.

## Assumptions & limitations (v1)

- Single-machine, localhost-focused (not distributed)
- HTML only (no JS rendering, no PDFs)
- Simple tokenizer (no stemming/NLP)
- No robots.txt policy engine (politeness via delay/rate controls)
