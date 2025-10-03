package aggregator

import (
	"testing"
)

func TestAggregator_AddResult(t *testing.T) {
	agg := New(false)

	// Test successful result
	result1 := ProcessingResult{
		URL: "https://example.com/1",
		WordCounts: map[string]int{
			"technology": 5,
			"innovation": 3,
			"computer":   2,
		},
	}

	agg.AddResult(result1)

	processed, totalWords, uniqueWords, _ := agg.GetStats()
	if processed != 1 {
		t.Errorf("Expected 1 processed article, got %d", processed)
	}
	if totalWords != 10 {
		t.Errorf("Expected 10 total words, got %d", totalWords)
	}
	if uniqueWords != 3 {
		t.Errorf("Expected 3 unique words, got %d", uniqueWords)
	}
}

func TestAggregator_GetTopWords(t *testing.T) {
	agg := New(false)

	// Add multiple results
	results := []ProcessingResult{
		{
			URL: "https://example.com/1",
			WordCounts: map[string]int{
				"technology": 10,
				"innovation": 5,
				"computer":   3,
			},
		},
		{
			URL: "https://example.com/2",
			WordCounts: map[string]int{
				"technology": 8,
				"science":    7,
				"computer":   2,
			},
		},
	}

	for _, result := range results {
		agg.AddResult(result)
	}

	topWords := agg.GetTopWords(3)

	// Expected order: technology (18), science (7), computer (5), innovation (5)
	// Since computer and innovation have same count, computer comes first alphabetically
	expected := []WordCount{
		{"technology", 18},
		{"science", 7},
		{"computer", 5},
	}

	if len(topWords) != 3 {
		t.Fatalf("Expected 3 top words, got %d", len(topWords))
	}

	for i, expected := range expected {
		if topWords[i].Word != expected.Word || topWords[i].Count != expected.Count {
			t.Errorf("Expected top word %d to be %+v, got %+v", i, expected, topWords[i])
		}
	}
}

func TestAggregator_ConcurrentAccess(t *testing.T) {
	agg := New(false)

	// Test concurrent access
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			result := ProcessingResult{
				URL: "https://example.com/" + string(rune(id)),
				WordCounts: map[string]int{
					"word": 1,
				},
			}
			agg.AddResult(result)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	processed, totalWords, _, _ := agg.GetStats()
	if processed != 10 {
		t.Errorf("Expected 10 processed articles, got %d", processed)
	}
	if totalWords != 10 {
		t.Errorf("Expected 10 total words, got %d", totalWords)
	}

	topWords := agg.GetTopWords(1)
	if len(topWords) != 1 || topWords[0].Word != "word" || topWords[0].Count != 10 {
		t.Errorf("Expected top word to be {word: 10}, got %+v", topWords[0])
	}
}
