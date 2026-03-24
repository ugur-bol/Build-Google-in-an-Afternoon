package search

import (
	"antigravity/internal/indexer"
	"antigravity/internal/models"
	"antigravity/internal/normalize"
)

// Engine wraps the indexer for search operations.
type Engine struct {
	idx *indexer.Indexer
}

// New creates a search engine backed by the given indexer.
func New(idx *indexer.Indexer) *Engine {
	return &Engine{idx: idx}
}

// Query tokenizes the input query and returns sorted results.
// For single-word queries, exact match. For multi-word, union of all terms.
func (e *Engine) Query(rawQuery string) models.SearchResponse {
	tokens := normalize.Tokenize(rawQuery)
	if len(tokens) == 0 {
		return models.SearchResponse{
			Query:   rawQuery,
			Count:   0,
			Results: []models.Posting{},
		}
	}

	var results []models.Posting
	if len(tokens) == 1 {
		results = e.idx.Search(tokens[0])
	} else {
		results = e.idx.SearchMulti(tokens)
	}

	if results == nil {
		results = []models.Posting{}
	}

	return models.SearchResponse{
		Query:   rawQuery,
		Count:   len(results),
		Results: results,
	}
}
