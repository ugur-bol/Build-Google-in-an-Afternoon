package crawler

import (
	"context"
	"io"
	"net/http"
	"time"
)

// Fetcher handles HTTP page retrieval with timeout.
type Fetcher struct {
	client *http.Client
}

// NewFetcher creates a Fetcher with a configurable timeout.
func NewFetcher(timeout time.Duration) *Fetcher {
	return &Fetcher{
		client: &http.Client{
			Timeout: timeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 5 {
					return http.ErrUseLastResponse
				}
				return nil
			},
		},
	}
}

// FetchResult holds the raw HTML body and HTTP status code.
type FetchResult struct {
	Body       string
	StatusCode int
	FinalURL   string
}

// Fetch retrieves the page at url. Returns the body, status code, and any error.
func (f *Fetcher) Fetch(ctx context.Context, rawURL string) (*FetchResult, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "ANTIGRAVITY-Crawler/1.0")
	req.Header.Set("Accept", "text/html,application/xhtml+xml")

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Limit read to 5MB to avoid huge pages
	limited := io.LimitReader(resp.Body, 5*1024*1024)
	body, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}

	return &FetchResult{
		Body:       string(body),
		StatusCode: resp.StatusCode,
		FinalURL:   resp.Request.URL.String(),
	}, nil
}
