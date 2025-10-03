package config

import (
	"flag"
	"fmt"
	"os"
	"regexp"
)

// Config holds all configuration for the essay analyzer
type Config struct {
	URLsFile     string
	WordBankFile string
	Verbose      bool
	Workers      int
	RateLimit    float64 // 0 means no limit (unless robots.txt specifies crawl-delay)
}

// WordFilterConfig holds word filtering configuration
type WordFilterConfig struct {
	// Pattern for valid words (3+ chars, alphabetic only)
	Pattern *regexp.Regexp
}

// GetWordFilterConfig returns the word filtering configuration
func GetWordFilterConfig() *WordFilterConfig {
	return &WordFilterConfig{
		Pattern: regexp.MustCompile(`^[a-zA-Z]{3,}$`),
	}
}

const (
	// DefaultTopWords is the default number of top words to return
	DefaultTopWords = 10
)

// GetTopWordsCount returns the number of top words to include in results
func GetTopWordsCount() int {
	return DefaultTopWords
}

// ParseFlags parses command line flags and returns configuration
func ParseFlags() (*Config, error) {
	config := &Config{}

	flag.StringVar(&config.URLsFile, "urls-file", "", "Path to file containing URLs (required)")
	flag.StringVar(&config.WordBankFile, "wordbank-file", "", "Path to word bank file (required)")
	flag.BoolVar(&config.Verbose, "verbose", false, "Enable verbose logging")
	flag.IntVar(&config.Workers, "workers", 50, "Number of concurrent workers")
	flag.Float64Var(&config.RateLimit, "rate-limit", 0, "Requests per second (0 = no limit unless robots.txt specifies)")

	flag.Parse()

	if config.URLsFile == "" {
		return nil, fmt.Errorf("--urls-file is required")
	}

	if config.WordBankFile == "" {
		return nil, fmt.Errorf("--wordbank-file is required")
	}

	if config.Workers <= 0 {
		return nil, fmt.Errorf("--workers must be positive")
	}

	if config.RateLimit < 0 {
		return nil, fmt.Errorf("--rate-limit must be non-negative (0 = no limit)")
	}

	return config, nil
}

// ValidateFiles checks if required files exist
func (c *Config) ValidateFiles() error {
	if _, err := os.Stat(c.URLsFile); os.IsNotExist(err) {
		return fmt.Errorf("URLs file does not exist: %s", c.URLsFile)
	}

	if _, err := os.Stat(c.WordBankFile); os.IsNotExist(err) {
		return fmt.Errorf("word bank file does not exist: %s", c.WordBankFile)
	}

	return nil
}
