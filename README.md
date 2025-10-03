# Essay Analyzer

A high-performance Go application for analyzing word frequencies across multiple web essays with concurrent processing and memory-efficient streaming.

## Design Decisions

### Word Processing
- **Case Insensitive**: All words converted to lowercase for counting ("The" and "the" count as same word)
- **Word Validation**: Words must be 3+ characters, alphabetic only, and exist in provided word bank
- **Streaming Processing**: Large HTML files processed in chunks to prevent memory issues

### Concurrency & Performance
- **Rate Limiting**: 10 requests/second to respect server resources with exponential backoff on errors
- **Worker Pool**: Configurable number of concurrent processors
- **Incremental Aggregation**: Results aggregated as files complete processing

### Memory Management
- **Streaming HTML Processing**: Content processed in overlapping buffers
- **Word Boundary Handling**: Prevents word splitting across chunk boundaries with fallback for edge cases
- **Efficient Data Structures**: Memory-optimized word counting and storage

### Edge Case Handling
- **Boundary Fallback**: If chunks don't contain word boundaries, force split at max buffer size to prevent OOM
- **Rate Limiting**: Exponential backoff on HTTP errors with configurable retry limits
- **Graceful Degradation**: Continue processing other URLs if individual URLs fail

### Web Scraping Compliance
- **robots.txt Parsing**: Fetches and parses robots.txt once at startup for compliance checking
- **URL Validation**: Every URL validated against robots.txt rules before fetching
- **Pattern Matching**: Supports exact paths and wildcard patterns (e.g., `/tag/expire-images*`)
- **User-Agent Specific**: Uses `EssayAnalyzer/1.0` user-agent for rule matching
- **Graceful Fallback**: Assumes all URLs allowed if robots.txt not found (404)
- **Single Domain Assumption**: Current implementation assumes all URLs from same domain (Engadget)

## Architecture

```
cmd/essay_analyzer/     # CLI entry point
internal/
├── fetcher/           # HTTP client with rate limiting
├── parser/            # HTML parsing and text extraction  
├── processor/         # Word validation and streaming processing
├── aggregator/        # Concurrent result aggregation
├── wordbank/          # Word bank loading and validation
└── config/            # Configuration management
```

## Usage

```bash
go run cmd/essay_analyzer/main.go \
  --urls-file files/endg-urls \
  --wordbank-file files/words.txt \
  --verbose
```

### Command Line Options

- `--urls-file`: Path to file containing URLs (one per line)
- `--wordbank-file`: Path to word bank file (one word per line)
- `--verbose`: Enable verbose logging
- `--workers`: Number of concurrent workers (default: 10)
- `--rate-limit`: Requests per second (default: 10.0)

## Key Features

- **High Concurrency**: Process multiple essays simultaneously
- **Memory Efficient**: Stream processing prevents OOM on large files
- **Rate Limited**: Respectful HTTP client with backoff
- **Comprehensive Testing**: High test coverage across all packages
- **Pretty JSON Output**: Top 10 word counts formatted for readability

## Output Format

```json
{
  "top_words": [
    {"word": "technology", "count": 1250},
    {"word": "innovation", "count": 987},
    ...
  ],
  "total_words_processed": 125000,
  "total_essays_processed": 40000,
  "processing_time_seconds": 45.2
}
```
