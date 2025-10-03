package io

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/firefly/essay-analyzer/internal/aggregator"
)

// Result represents the final analysis result for JSON output
type Result struct {
	TopWords              []aggregator.WordCount `json:"top_words"`
	TotalWordsProcessed   int                    `json:"total_words_processed"`
	TotalEssaysProcessed  int                    `json:"total_essays_processed"`
	ProcessingTimeSeconds float64                `json:"processing_time_seconds"`
}

// OutputResult outputs the final result as JSON to stdout
func OutputResult(agg *aggregator.Aggregator, topN int) error {
	processed, totalWords, _, elapsed := agg.GetStats()
	topWords := agg.GetTopWords(topN)

	result := Result{
		TopWords:              topWords,
		TotalWordsProcessed:   totalWords,
		TotalEssaysProcessed:  processed,
		ProcessingTimeSeconds: elapsed,
	}

	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling result to JSON: %w", err)
	}

	fmt.Println(string(jsonData))
	return nil
}

// OutputResultToFile outputs the final result as JSON to a file
func OutputResultToFile(agg *aggregator.Aggregator, topN int, filename string) error {
	processed, totalWords, _, elapsed := agg.GetStats()
	topWords := agg.GetTopWords(topN)

	result := Result{
		TopWords:              topWords,
		TotalWordsProcessed:   totalWords,
		TotalEssaysProcessed:  processed,
		ProcessingTimeSeconds: elapsed,
	}

	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling result to JSON: %w", err)
	}

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("creating output file: %w", err)
	}
	defer file.Close()

	_, err = file.Write(jsonData)
	if err != nil {
		return fmt.Errorf("writing to output file: %w", err)
	}

	return nil
}
