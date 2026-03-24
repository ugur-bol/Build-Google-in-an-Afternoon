# ANTIGRAVITY

> Real-time web crawler and search engine — single-machine, localhost, built from scratch in Go.

## Overview

ANTIGRAVITY is a web crawler and search engine that runs entirely on your local machine. It crawls web pages from a given origin URL up to a configurable depth, builds an in-memory inverted index, persists raw term data to an inspectable flat file (`data/storage/p.data`), and exposes a search API with a transparent relevance scoring formula.

## Architecture

```
┌─────────────────────────────────────────────┐
│                HTTP Server (:3600)           │
│   /index  /search  /api/state  /health  /   │
└─────┬──────┬──────────┬──────────┬──────────┘
      │      │          │          │
      ▼      │          │          │
┌──────────┐ │    ┌──────────┐  ┌──────────┐
│ Crawler  │ │    │ Search   │  │  State   │
│ ┌──────┐ │ │    │ Engine   │  │ (atomic) │
│ │Queue │ │ │    └────┬─────┘  └──────────┘
│ │(chan) │ │ │         │
│ └──┬───┘ │ │    ┌────▼─────┐
│    │     │ │    │ Indexer  │
│ Workers  │─┼───▶│ (in-mem) │
│ (pool)   │ │    │ inverted │
└──────────┘ │    │ index    │
             │    └────┬─────┘
             │         │
             │    ┌────▼─────┐
             │    │ Storage  │
             │    │ p.data   │
             │    │ pages.jsonl
             │    └──────────┘
```

**Subsystems:**
- **Crawler** — Bounded worker pool with channel-based queue, dedup via `sync.Map`, politeness delay
- **Fetcher** — HTTP client with timeout, redirect following, body size limit
- **Parser** — `golang.org/x/net/html` tokenizer for text extraction and link discovery
- **Indexer** — Thread-safe in-memory `map[word][]Posting` with `sync.RWMutex`
- **Storage** — Mutex-protected append-only file writer for `p.data` and `pages.jsonl`
- **Search** — Tokenizes query, looks up index, sorts by relevance score
- **State** — Atomic counters for real-time system metrics

## Why Go?

- Goroutines and channels make concurrent crawling natural and efficient
- No runtime dependency — single static binary
- Standard library provides HTTP server/client, JSON, and all primitives needed
- Straightforward, readable, quiz-friendly code

## Quick Start

```bash
# Build
go build -o antigravity ./cmd/server

# Run
./antigravity
# Server starts at http://localhost:3600
```

Open the dashboard: http://localhost:3600

## API

### POST /index
Start a crawl job.
```bash
curl -X POST http://localhost:3600/index \
  -H "Content-Type: application/json" \
  -d '{"origin": "https://example.com", "maxDepth": 2}'
```

### GET /search?query=\<word\>&sortBy=relevance
Search indexed content.
```bash
curl "http://localhost:3600/search?query=python&sortBy=relevance"
```

### GET /api/state
Live system metrics.
```bash
curl http://localhost:3600/api/state
```

### GET /health
Health check.
```bash
curl http://localhost:3600/health
```

## Storage Format

### data/storage/p.data
Human-readable, line-oriented. Each line:
```
word url origin depth frequency
```
Example:
```
python https://example.com/tutorial https://example.com 1 7
programming https://example.com/about https://example.com 0 3
```

### data/storage/pages.jsonl
One JSON object per line with page metadata (URL, title, status code, word count, etc.).

## Relevance Formula

```
relevance_score = (frequency × 10) + 1000 - (depth × 5)
```

This is implemented in `internal/models/models.go` as `RelevanceScore(frequency, depth)` and used directly in the indexer when building postings.

## Quiz Verification Guide

To manually verify the relevance scoring:

1. **Start a crawl** via the dashboard or API
2. **Open `data/storage/p.data`** — find a word that appears on multiple pages
3. **Compute the expected score** for each occurrence:
   ```
   score = (frequency × 10) + 1000 - (depth × 5)
   ```
4. **Hit the search API:**
   ```bash
   curl "http://localhost:3600/search?query=<word>&sortBy=relevance"
   ```
5. **Verify** — The first result's `relevance_score` should match the highest manually computed score

**Example walkthrough:**
```
# From p.data:
# go https://go.dev https://go.dev 0 42
# Score = (42 × 10) + 1000 - (0 × 5) = 420 + 1000 - 0 = 1420

# go https://go.dev/doc https://go.dev 1 15
# Score = (15 × 10) + 1000 - (1 × 5) = 150 + 1000 - 5 = 1145

# API should return the first entry (score 1420) at position #1
```

## Assumptions & Limitations

- **Single-machine only** — designed for localhost, not distributed
- **In-memory index** — limited by available RAM; no persistence-based reload on restart
- **HTML only** — does not process PDFs, images, or JavaScript-rendered content
- **No robots.txt** — politeness is via request delay only
- **Word tokenization** — simple alphanumeric split; no stemming or NLP
- **Multi-word search** — uses union (returns pages matching any term)
- **Depth cap** — capped at 10 to prevent runaway crawls
- **Queue cap** — 10,000 entries; excess URLs are dropped with throttle indication
