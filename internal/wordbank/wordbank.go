package wordbank

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/firefly/essay-analyzer/internal/config"
)

// WordBank holds valid words for filtering
type WordBank struct {
	words map[string]bool
}

// New creates a new WordBank from a file
func New(filename string) (*WordBank, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("opening word bank file: %w", err)
	}
	defer file.Close()

	words := make(map[string]bool)
	scanner := bufio.NewScanner(file)

	// Get word filtering configuration
	filterConfig := config.GetWordFilterConfig()

	for scanner.Scan() {
		word := strings.TrimSpace(scanner.Text())
		if word == "" {
			continue
		}

		// Convert to lowercase for case-insensitive matching
		word = strings.ToLower(word)

		// Only include words that match our validation criteria
		if filterConfig.Pattern.MatchString(word) {
			words[word] = true
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading word bank file: %w", err)
	}

	return &WordBank{
		words: words,
	}, nil
}

// IsValid checks if a word is valid according to our criteria
func (wb *WordBank) IsValid(word string) bool {
	if word == "" {
		return false
	}

	// Convert to lowercase for case-insensitive matching
	word = strings.ToLower(word)

	// Check if word exists in our word bank (already filtered during loading)
	return wb.words[word]
}

// Size returns the number of words in the word bank
func (wb *WordBank) Size() int {
	return len(wb.words)
}
