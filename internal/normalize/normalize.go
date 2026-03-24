package normalize

import (
	"net/url"
	"regexp"
	"strings"
)

var nonAlphaNum = regexp.MustCompile(`[^a-z0-9]+`)

// Tokenize splits visible text into lowercase alphabetical tokens.
// Removes very short tokens (len < 2) and common noise.
func Tokenize(text string) []string {
	text = strings.ToLower(text)
	parts := nonAlphaNum.Split(text, -1)
	tokens := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if len(p) >= 2 {
			tokens = append(tokens, p)
		}
	}
	return tokens
}

// WordFrequencies counts how often each word appears.
func WordFrequencies(tokens []string) map[string]int {
	freq := make(map[string]int, len(tokens))
	for _, t := range tokens {
		freq[t]++
	}
	return freq
}

// ResolveURL resolves a relative href against a base URL.
func ResolveURL(base, href string) string {
	href = strings.TrimSpace(href)
	if href == "" || strings.HasPrefix(href, "#") || strings.HasPrefix(href, "javascript:") || strings.HasPrefix(href, "mailto:") {
		return ""
	}

	baseURL, err := url.Parse(base)
	if err != nil {
		return ""
	}

	ref, err := url.Parse(href)
	if err != nil {
		return ""
	}

	resolved := baseURL.ResolveReference(ref)

	// Only allow http/https
	if resolved.Scheme != "http" && resolved.Scheme != "https" {
		return ""
	}

	// Strip fragment
	resolved.Fragment = ""

	return resolved.String()
}

// NormalizeURL strips trailing slash and fragment for consistency.
func NormalizeURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	u.Fragment = ""
	result := u.String()
	result = strings.TrimRight(result, "/")
	return result
}
