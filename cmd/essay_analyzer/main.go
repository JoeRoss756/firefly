package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/firefly/essay-analyzer/internal/aggregator"
	"github.com/firefly/essay-analyzer/internal/config"
	"github.com/firefly/essay-analyzer/internal/fetcher"
	outputio "github.com/firefly/essay-analyzer/internal/io"
	"github.com/firefly/essay-analyzer/internal/parser"
	"github.com/firefly/essay-analyzer/internal/processor"
	"github.com/firefly/essay-analyzer/internal/wordbank"
)

// Pipeline data structures for passing data between stages
type URLJob struct {
	URL string
}

type HTMLResult struct {
	URL     string
	Content io.Reader
	Error   error
}

type TextResult struct {
	URL   string
	Text  string
	Error error
}

// WorkerConfig holds configuration for worker pool sizes
type WorkerConfig struct {
	Fetchers   int
	Parsers    int
	Processors int
}

func main() {
	// Parse command line flags
	cfg, err := config.ParseFlags()
	if err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	// Validate files exist
	if err := cfg.ValidateFiles(); err != nil {
		log.Fatalf("File validation error: %v", err)
	}

	if cfg.Verbose {
		fmt.Println("🚀 Starting Essay Analyzer...")
		fmt.Printf("  URLs file: %s\n", cfg.URLsFile)
		fmt.Printf("  Wordbank file: %s\n", cfg.WordBankFile)
		fmt.Printf("  Workers: %d\n", cfg.Workers)
		fmt.Printf("  Rate limit: %.1f req/sec\n", cfg.RateLimit)
	}

	// Initialize components
	wordBank, err := wordbank.New(cfg.WordBankFile)
	if err != nil {
		log.Fatalf("Failed to load wordbank: %v", err)
	}

	if cfg.Verbose {
		fmt.Printf("  Loaded wordbank: %d words\n", wordBank.Size())
	}

	// Initialize fetcher
	fetch := fetcher.New(cfg.RateLimit, cfg.Verbose)

	// Initialize parser and processor
	htmlParser := parser.New(cfg.Verbose)
	textProcessor := processor.New(wordBank, cfg.Verbose)

	// Initialize aggregator
	agg := aggregator.New(cfg.Verbose)

	// Calculate worker distribution
	workerCfg := calculateWorkerDistribution(cfg.Workers)

	if cfg.Verbose {
		fmt.Printf("  Worker distribution: %d fetchers, %d parsers, %d processors\n",
			workerCfg.Fetchers, workerCfg.Parsers, workerCfg.Processors)
	}

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// For single domain optimization, we can pre-load robots.txt
	// This assumes all URLs are from the same domain (Engadget)
	if cfg.Verbose {
		fmt.Println("  Loading robots.txt...")
	}

	// Load robots.txt for engadget.com (assuming all URLs are from same domain)
	if err := fetch.LoadRobotsTxt(ctx, "https://www.engadget.com"); err != nil {
		if cfg.Verbose {
			fmt.Printf("  Warning: Failed to load robots.txt: %v\n", err)
		}
	}

	// Handle interrupt signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\n⚠️  Received interrupt signal, shutting down gracefully...")
		cancel()
	}()

	// Run the pipeline
	if err := runPipeline(ctx, cfg, fetch, htmlParser, textProcessor, agg, workerCfg); err != nil {
		log.Fatalf("Pipeline error: %v", err)
	}

	// Output final results
	if cfg.Verbose {
		agg.PrintFinalStats()
	}

	topN := config.GetTopWordsCount()
	if err := outputio.OutputResult(agg, topN); err != nil {
		log.Fatalf("Output error: %v", err)
	}

	if cfg.Verbose {
		fmt.Println("✅ Analysis complete!")
	}
}
