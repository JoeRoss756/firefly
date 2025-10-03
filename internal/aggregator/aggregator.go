package aggregator

import (
	"sort"
	"sync"
	"time"
)

// WordCount represents a word and its frequency
type WordCount struct {
	Word  string `json:"word"`
	Count int    `json:"count"`
}

// ProcessingResult represents the result from processing a single article
type ProcessingResult struct {
	URL        string
	WordCounts map[string]int
}

// Aggregator collects and aggregates word frequency results
type Aggregator struct {
	mu                   sync.RWMutex
	globalWordCounts     map[string]int
	totalWordsProcessed  int
	totalEssaysProcessed int
	startTime            time.Time
	verbose              bool
}

// New creates a new Aggregator
func New(verbose bool) *Aggregator {
	return &Aggregator{
		globalWordCounts: make(map[string]int),
		startTime:        time.Now(),
		verbose:          verbose,
	}
}

// AddResult adds a processing result to the aggregator
func (a *Aggregator) AddResult(result ProcessingResult) {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Aggregate word counts
	articleWordCount := 0
	for word, count := range result.WordCounts {
		a.globalWordCounts[word] += count
		articleWordCount += count
	}

	a.totalWordsProcessed += articleWordCount
	a.totalEssaysProcessed++

	if a.verbose && a.totalEssaysProcessed%100 == 0 {
		elapsed := time.Since(a.startTime).Seconds()
		rate := float64(a.totalEssaysProcessed) / elapsed
		println("✅ Processed", a.totalEssaysProcessed, "articles,", a.totalWordsProcessed, "words",
			"(", int(rate), "articles/sec )")
	}
}

// GetTopWords returns the top N words by frequency
func (a *Aggregator) GetTopWords(n int) []WordCount {
	a.mu.RLock()
	defer a.mu.RUnlock()

	// Convert map to slice for sorting
	words := make([]WordCount, 0, len(a.globalWordCounts))
	for word, count := range a.globalWordCounts {
		words = append(words, WordCount{Word: word, Count: count})
	}

	// Sort by count (descending), then by word (ascending) for stable results
	sort.Slice(words, func(i, j int) bool {
		if words[i].Count == words[j].Count {
			return words[i].Word < words[j].Word
		}
		return words[i].Count > words[j].Count
	})

	// Return top N words
	if n > len(words) {
		n = len(words)
	}
	return words[:n]
}

// GetStats returns current processing statistics
func (a *Aggregator) GetStats() (processed int, totalWords int, uniqueWords int, elapsed float64) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return a.totalEssaysProcessed, a.totalWordsProcessed,
		len(a.globalWordCounts), time.Since(a.startTime).Seconds()
}

// PrintFinalStats prints final processing statistics
func (a *Aggregator) PrintFinalStats() {
	processed, totalWords, uniqueWords, elapsed := a.GetStats()

	println("\n📊 Final Statistics:")
	println("  Articles processed:", processed)
	println("  Total words processed:", totalWords)
	println("  Unique words found:", uniqueWords)
	println("  Processing time:", int(elapsed), "seconds")

	if processed > 0 {
		rate := float64(processed) / elapsed
		println("  Processing rate:", int(rate), "articles/sec")
	}
}
