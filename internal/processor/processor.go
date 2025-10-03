package processor

import (
	"regexp"
	"strings"
)

// Processor handles word processing and counting
type Processor struct {
	wordBank WordValidator
	verbose  bool

	// Word extraction regex
	wordRegex *regexp.Regexp
}

// WordValidator interface for checking word validity
type WordValidator interface {
	IsValid(word string) bool
}

// New creates a new Processor
func New(wordBank WordValidator, verbose bool) *Processor {
	return &Processor{
		wordBank:  wordBank,
		verbose:   verbose,
		wordRegex: regexp.MustCompile(`[a-zA-Z]+`),
	}
}

// ProcessText processes text and returns word counts
func (p *Processor) ProcessText(text string) map[string]int {
	wordCounts := make(map[string]int)

	// Extract all words using regex
	words := p.wordRegex.FindAllString(text, -1)

	for _, word := range words {
		// Convert to lowercase for case-insensitive counting
		word = strings.ToLower(word)

		// Validate word using wordbank (already filtered during loading)
		if p.wordBank.IsValid(word) {
			wordCounts[word]++
		}
	}

	return wordCounts
}
