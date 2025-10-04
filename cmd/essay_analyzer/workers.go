package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/firefly/essay-analyzer/internal/aggregator"
	"github.com/firefly/essay-analyzer/internal/fetcher"
	"github.com/firefly/essay-analyzer/internal/parser"
	"github.com/firefly/essay-analyzer/internal/processor"
)

// readURLs reads URLs from file and sends them to the URL channel
func readURLs(ctx context.Context, filename string, urlCh chan<- URLJob, verbose bool) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("opening URLs file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	urlCount := 0

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		url := strings.TrimSpace(scanner.Text())
		if url == "" || strings.HasPrefix(url, "#") {
			continue // Skip empty lines and comments
		}

		select {
		case urlCh <- URLJob{URL: url}:
			urlCount++
			if verbose && urlCount%1000 == 0 {
				fmt.Printf("📖 Queued %d URLs...\n", urlCount)
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("reading URLs file: %w", err)
	}

	if verbose {
		fmt.Printf("📖 Finished reading %d URLs\n", urlCount)
	}

	return nil
}

// fetcherWorker fetches HTML content for URLs
func fetcherWorker(
	ctx context.Context,
	id int,
	fetch *fetcher.Fetcher,
	urlCh <-chan URLJob,
	htmlCh chan<- HTMLResult,
	errorCh chan<- error,
	verbose bool,
) {
	for {
		select {
		case job, ok := <-urlCh:
			if !ok {
				return // Channel closed
			}

			// Check robots.txt compliance
			allowed := fetch.IsAllowed(job.URL)
			if !allowed {
				select {
				case errorCh <- fmt.Errorf("robots.txt disallows %s", job.URL):
				case <-ctx.Done():
					return
				}
				continue
			}

			// Fetch content
			content, err := fetch.FetchURL(ctx, job.URL)

			select {
			case htmlCh <- HTMLResult{
				URL:     job.URL,
				Content: content,
				Error:   err,
			}:
			case <-ctx.Done():
				return
			}

		case <-ctx.Done():
			return
		}
	}
}

// parserWorker parses HTML content to extract text
func parserWorker(
	ctx context.Context,
	id int,
	htmlParser *parser.Parser,
	htmlCh <-chan HTMLResult,
	textCh chan<- TextResult,
	errorCh chan<- error,
	verbose bool,
) {
	for {
		select {
		case result, ok := <-htmlCh:
			if !ok {
				return // Channel closed
			}

			var text string
			var err error

			if result.Error != nil {
				err = fmt.Errorf("fetch failed: %w", result.Error)
			} else {
				text, err = htmlParser.ExtractText(result.Content)
				if err != nil {
					err = fmt.Errorf("parsing failed: %w", err)
				}
			}

			select {
			case textCh <- TextResult{
				URL:   result.URL,
				Text:  text,
				Error: err,
			}:
			case <-ctx.Done():
				return
			}

		case <-ctx.Done():
			return
		}
	}
}

// processorWorker processes text to extract word counts
func processorWorker(
	ctx context.Context,
	id int,
	textProcessor *processor.Processor,
	textCh <-chan TextResult,
	resultsCh chan<- aggregator.ProcessingResult,
	errorCh chan<- error,
	verbose bool,
) {
	for {
		select {
		case result, ok := <-textCh:
			if !ok {
				return // Channel closed
			}

			if result.Error != nil {
				select {
				case errorCh <- fmt.Errorf("processing %s: %w", result.URL, result.Error):
				case <-ctx.Done():
					return
				}
				continue
			}

			// Process text to get word counts
			wordCounts := textProcessor.ProcessText(result.Text)

			select {
			case resultsCh <- aggregator.ProcessingResult{
				URL:        result.URL,
				WordCounts: wordCounts,
			}:
			case <-ctx.Done():
				return
			}

		case <-ctx.Done():
			return
		}
	}
}

// aggregatorWorker collects and aggregates results
func aggregatorWorker(
	ctx context.Context,
	agg *aggregator.Aggregator,
	resultsCh <-chan aggregator.ProcessingResult,
	verbose bool,
) {
	for {
		select {
		case result, ok := <-resultsCh:
			if !ok {
				return // Channel closed
			}

			agg.AddResult(result)

		case <-ctx.Done():
			return
		}
	}
}
