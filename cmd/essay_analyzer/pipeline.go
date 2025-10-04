package main

import (
	"context"
	"fmt"
	"sync"

	"github.com/firefly/essay-analyzer/internal/aggregator"
	"github.com/firefly/essay-analyzer/internal/config"
	"github.com/firefly/essay-analyzer/internal/fetcher"
	"github.com/firefly/essay-analyzer/internal/parser"
	"github.com/firefly/essay-analyzer/internal/processor"
)

// runPipeline orchestrates the concurrent processing pipeline
func runPipeline(
	ctx context.Context,
	cfg *config.Config,
	fetch *fetcher.Fetcher,
	htmlParser *parser.Parser,
	textProcessor *processor.Processor,
	agg *aggregator.Aggregator,
	workerCfg WorkerConfig,
) error {
	// Create channels with appropriate buffer sizes
	urlCh := make(chan URLJob, 100)
	htmlCh := make(chan HTMLResult, 50)
	textCh := make(chan TextResult, 50)
	resultsCh := make(chan aggregator.ProcessingResult, 100)
	errorCh := make(chan error, 100)

	// Wait group for coordinating shutdown
	var wg sync.WaitGroup

	// Start URL reader
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(urlCh)
		if err := readURLs(ctx, cfg.URLsFile, urlCh, cfg.Verbose); err != nil {
			select {
			case errorCh <- fmt.Errorf("reading URLs: %w", err):
			case <-ctx.Done():
			}
		}
	}()

	// Create separate wait groups for each stage to enable cascading channel closes
	fetcherWg := &sync.WaitGroup{}
	parserWg := &sync.WaitGroup{}
	processorWg := &sync.WaitGroup{}

	// Start fetcher workers
	for i := 0; i < workerCfg.Fetchers; i++ {
		wg.Add(1)
		fetcherWg.Add(1)
		go func(id int) {
			defer wg.Done()
			defer fetcherWg.Done()
			fetcherWorker(ctx, id, fetch, urlCh, htmlCh, errorCh, cfg.Verbose)
		}(i)
	}

	// Start parser workers
	for i := 0; i < workerCfg.Parsers; i++ {
		wg.Add(1)
		parserWg.Add(1)
		go func(id int) {
			defer wg.Done()
			defer parserWg.Done()
			parserWorker(ctx, id, htmlParser, htmlCh, textCh, errorCh, cfg.Verbose)
		}(i)
	}

	// Start processor workers
	for i := 0; i < workerCfg.Processors; i++ {
		wg.Add(1)
		processorWg.Add(1)
		go func(id int) {
			defer wg.Done()
			defer processorWg.Done()
			processorWorker(ctx, id, textProcessor, textCh, resultsCh, errorCh, cfg.Verbose)
		}(i)
	}

	// Close channels in cascade as each stage completes
	go func() {
		fetcherWg.Wait()
		close(htmlCh)
	}()

	go func() {
		parserWg.Wait()
		close(textCh)
	}()

	go func() {
		processorWg.Wait()
		close(resultsCh)
	}()

	// Start aggregator
	wg.Add(1)
	go func() {
		defer wg.Done()
		aggregatorWorker(ctx, agg, resultsCh, cfg.Verbose)
	}()

	// Start error collector
	var errorCount int
	errorWg := sync.WaitGroup{}
	errorWg.Add(1)
	go func() {
		defer errorWg.Done()
		for err := range errorCh {
			errorCount++
			if cfg.Verbose {
				fmt.Printf("❌ Error #%d: %v\n", errorCount, err)
			}
		}
	}()

	// Wait for all workers to complete
	wg.Wait()

	// Close error channel and wait for error collector to finish
	close(errorCh)
	errorWg.Wait()

	if cfg.Verbose && errorCount > 0 {
		fmt.Printf("⚠️  Total errors encountered: %d\n", errorCount)
	}

	return nil
}

// calculateWorkerDistribution distributes workers across pipeline stages
func calculateWorkerDistribution(totalWorkers int) WorkerConfig {
	// Distribution strategy:
	// 60% fetchers (I/O bound)
	// 20% parsers (CPU bound)
	// 20% processors (CPU bound)

	fetchers := max(1, (totalWorkers*60)/100)
	parsers := max(1, (totalWorkers*20)/100)
	processors := max(1, totalWorkers-fetchers-parsers) // Remainder goes to processors

	return WorkerConfig{
		Fetchers:   fetchers,
		Parsers:    parsers,
		Processors: processors,
	}
}

// max returns the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
