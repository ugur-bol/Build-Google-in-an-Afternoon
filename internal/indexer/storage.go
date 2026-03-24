package indexer

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"antigravity/internal/models"
)

// Storage handles file-based persistence: p.data and pages.jsonl
type Storage struct {
	mu         sync.Mutex
	pDataPath  string
	pagesPath  string
	pDataFile  *os.File
	pagesFile  *os.File
}

// NewStorage creates the storage writer. Opens files for appending.
func NewStorage(pDataPath, pagesPath string) (*Storage, error) {
	// Ensure directory exists
	if err := os.MkdirAll("data/storage", 0755); err != nil {
		return nil, fmt.Errorf("create storage dir: %w", err)
	}

	pf, err := os.OpenFile(pDataPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("open p.data: %w", err)
	}

	pgf, err := os.OpenFile(pagesPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		pf.Close()
		return nil, fmt.Errorf("open pages.jsonl: %w", err)
	}

	return &Storage{
		pDataPath: pDataPath,
		pagesPath: pagesPath,
		pDataFile: pf,
		pagesFile: pgf,
	}, nil
}

// WritePostings appends term posting lines to p.data.
// Format per line: word url origin depth frequency
func (s *Storage) WritePostings(freqs map[string]int, pageURL, origin string, depth int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for word, freq := range freqs {
		line := fmt.Sprintf("%s %s %s %d %d\n", word, pageURL, origin, depth, freq)
		if _, err := s.pDataFile.WriteString(line); err != nil {
			return err
		}
	}
	return s.pDataFile.Sync()
}

// WritePageMeta appends a page metadata JSON line to pages.jsonl.
func (s *Storage) WritePageMeta(meta models.PageMeta) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.Marshal(meta)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(s.pagesFile, "%s\n", data)
	if err != nil {
		return err
	}
	return s.pagesFile.Sync()
}

// Close closes the underlying files.
func (s *Storage) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	var firstErr error
	if err := s.pDataFile.Close(); err != nil && firstErr == nil {
		firstErr = err
	}
	if err := s.pagesFile.Close(); err != nil && firstErr == nil {
		firstErr = err
	}
	return firstErr
}
