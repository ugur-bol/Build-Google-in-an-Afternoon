package indexer

import (
	"antigravity/internal/models"
	"sort"
	"sync"
)

// Indexer is a thread-safe in-memory inverted index.
type Indexer struct {
	mu    sync.RWMutex
	index map[string][]models.Posting // word → list of postings
}

// New creates an empty Indexer.
func New() *Indexer {
	return &Indexer{
		index: make(map[string][]models.Posting),
	}
}

// Add inserts postings for all words on a given page.
// freqs: word→count, pageURL, origin, depth are from the crawl task.
func (idx *Indexer) Add(freqs map[string]int, pageURL, origin string, depth int) {
	idx.mu.Lock()
	defer idx.mu.Unlock()

	for word, freq := range freqs {
		p := models.Posting{
			Word:           word,
			URL:            pageURL,
			Origin:         origin,
			Depth:          depth,
			Frequency:      freq,
			RelevanceScore: models.RelevanceScore(freq, depth),
		}
		idx.index[word] = append(idx.index[word], p)
	}
}

// Search returns postings matching the query word, sorted by relevance descending.
func (idx *Indexer) Search(word string) []models.Posting {
	idx.mu.RLock()
	postings, ok := idx.index[word]
	if !ok {
		idx.mu.RUnlock()
		return nil
	}
	// Copy to avoid holding the lock during sort
	result := make([]models.Posting, len(postings))
	copy(result, postings)
	idx.mu.RUnlock()

	sort.Slice(result, func(i, j int) bool {
		return result[i].RelevanceScore > result[j].RelevanceScore
	})
	return result
}

// SearchMulti handles multi-word queries by union: returns all postings matching any term,
// then sorts by relevance.
func (idx *Indexer) SearchMulti(words []string) []models.Posting {
	if len(words) == 0 {
		return nil
	}
	if len(words) == 1 {
		return idx.Search(words[0])
	}

	// Union: collect all postings from all query words
	seen := make(map[string]bool) // key = word+url to dedupe
	var all []models.Posting

	idx.mu.RLock()
	for _, w := range words {
		postings, ok := idx.index[w]
		if !ok {
			continue
		}
		for _, p := range postings {
			key := p.Word + "|" + p.URL
			if !seen[key] {
				seen[key] = true
				all = append(all, p)
			}
		}
	}
	idx.mu.RUnlock()

	sort.Slice(all, func(i, j int) bool {
		return all[i].RelevanceScore > all[j].RelevanceScore
	})
	return all
}

// Stats returns total unique words and total postings.
func (idx *Indexer) Stats() (words int, postings int) {
	idx.mu.RLock()
	defer idx.mu.RUnlock()
	words = len(idx.index)
	for _, p := range idx.index {
		postings += len(p)
	}
	return
}
