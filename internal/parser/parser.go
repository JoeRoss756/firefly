package parser

import (
	"fmt"
	"io"
	"strings"
	"sync/atomic"

	"github.com/PuerkitoBio/goquery"
)

// Parser extracts text content from HTML with selective content filtering
type Parser struct {
	verbose     bool
	failedCount int64 // Atomic counter for failed parsing attempts
}

// New creates a new Parser
func New(verbose bool) *Parser {
	return &Parser{
		verbose: verbose,
	}
}

// ExtractText extracts clean text content from HTML using selective parsing
func (p *Parser) ExtractText(reader io.Reader) (string, error) {
	doc, err := goquery.NewDocumentFromReader(reader)
	if err != nil {
		return "", fmt.Errorf("parsing HTML: %w", err)
	}

	// Selective content extraction - prioritize clean content over noisy fallbacks
	contentSelectors := []struct {
		selector string
		desc     string
	}{
		{"article header, [data-article-body='true']", "header + body (ideal)"},
		{"[data-article-body='true']", "body only (good)"},
	}

	for _, sel := range contentSelectors {
		content := doc.Find(sel.selector)
		if content.Length() > 0 {
			text := strings.TrimSpace(content.Text())
			if len(text) > 0 {
				if p.verbose {
					fmt.Printf("✅ Extracted text using: %s (%d chars)\n", sel.desc, len(text))
				}
				return text, nil
			}
		}
	}

	// If we reach here, parsing failed - increment counter
	atomic.AddInt64(&p.failedCount, 1)

	if p.verbose {
		fmt.Printf("❌ Failed to extract clean content - no suitable selectors found\n")
	}

	return "", fmt.Errorf("failed to extract clean content: no suitable selectors matched")
}

// GetFailedCount returns the number of articles that failed to parse
func (p *Parser) GetFailedCount() int64 {
	return atomic.LoadInt64(&p.failedCount)
}

// PrintStats prints parsing statistics (call this at the end of processing)
func (p *Parser) PrintStats(totalArticles int64) {
	failedCount := p.GetFailedCount()
	successCount := totalArticles - failedCount

	if p.verbose {
		fmt.Printf("\n=== PARSING STATISTICS ===\n")
		fmt.Printf("Successfully parsed articles: %d\n", successCount)
		fmt.Printf("Failed to parse articles: %d\n", failedCount)
		if totalArticles > 0 {
			successRate := float64(successCount) / float64(totalArticles) * 100
			fmt.Printf("Success rate: %.1f%%\n", successRate)
		}

		if failedCount > 0 {
			fmt.Printf("\n⚠️  WARNING: %d articles failed to parse.\n", failedCount)
			fmt.Printf("This may indicate that Engadget has changed their HTML structure.\n")
			fmt.Printf("Consider updating the content selectors in the parser.\n")
		}
		fmt.Printf("===========================\n\n")
	}
}
