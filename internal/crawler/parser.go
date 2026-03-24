package crawler

import (
	"strings"

	"golang.org/x/net/html"

	"antigravity/internal/normalize"
)

// ParseResult contains text content, links, and title extracted from HTML.
type ParseResult struct {
	Text  string
	Links []string
	Title string
}

// Parse extracts visible text, links, and title from raw HTML.
func Parse(rawHTML string, baseURL string) *ParseResult {
	result := &ParseResult{}
	tokenizer := html.NewTokenizer(strings.NewReader(rawHTML))

	var textParts []string
	inScript := false
	inStyle := false
	inTitle := false
	var titleParts []string

	for {
		tt := tokenizer.Next()
		switch tt {
		case html.ErrorToken:
			result.Text = strings.Join(textParts, " ")
			result.Title = strings.TrimSpace(strings.Join(titleParts, " "))
			return result

		case html.StartTagToken, html.SelfClosingTagToken:
			tn, hasAttr := tokenizer.TagName()
			tagName := string(tn)

			switch tagName {
			case "script":
				inScript = true
			case "style":
				inStyle = true
			case "title":
				inTitle = true
			}

			if tagName == "a" && hasAttr {
				for {
					key, val, more := tokenizer.TagAttr()
					if string(key) == "href" {
						resolved := normalize.ResolveURL(baseURL, string(val))
						if resolved != "" {
							result.Links = append(result.Links, resolved)
						}
					}
					if !more {
						break
					}
				}
			}

		case html.EndTagToken:
			tn, _ := tokenizer.TagName()
			tagName := string(tn)
			switch tagName {
			case "script":
				inScript = false
			case "style":
				inStyle = false
			case "title":
				inTitle = false
			}

		case html.TextToken:
			if !inScript && !inStyle {
				text := strings.TrimSpace(string(tokenizer.Text()))
				if text != "" {
					textParts = append(textParts, text)
					if inTitle {
						titleParts = append(titleParts, text)
					}
				}
			}
		}
	}
}
