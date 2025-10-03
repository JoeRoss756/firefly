# Essay Analyzer


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

## Implementation Specifics

### Parsing Strategy

Our HTML parsing strategy prioritizes **content quality over robustness** by using Engadget-specific CSS selectors:

1. **Primary selector**: `article header, [data-article-body='true']` - Extracts both article title/author and body content
2. **Fallback selector**: `[data-article-body='true']` - Extracts only the article body content

#### Why This Approach?

**Quality Word Counts**: By targeting specific content elements, we avoid counting:
- Navigation menu items ("Home", "About", "Contact")
- Advertisement text ("Click here", "Subscribe now")
- HTML/CSS class names and technical terms
- Footer and sidebar content

**Content-Only Focus**: The selectors specifically target editorial content, ensuring our word frequency analysis reflects the actual essays rather than website infrastructure.

**Single-Domain Optimization**: Since all 40,000 URLs are from Engadget.com, we can optimize for their specific HTML structure rather than building a generic parser.

#### Trade-offs

- **Prioritizes Quality**: Better word frequency data from clean content extraction
- **Sacrifices Robustness**: Won't work well on other websites without selector updates
- **Future Work**: Could implement generic content detection algorithms (readability scoring, content-to-noise ratio analysis) at the cost of complexity and performance

### robots.txt Compliance

Our robots.txt implementation is optimized for **single-domain processing**:

**Current Approach**:
- Fetch robots.txt once at startup for engadget.com
- Cache and reuse the parsed rules for all 40,000 URLs
- Validate each URL against the cached rules before fetching

**Benefits**:
- **Performance**: Only one robots.txt request instead of 40,000
- **Efficiency**: Cached rule parsing and pattern matching
- **Compliance**: Full respect for crawl delays and disallow patterns

**Limitations**:
- **Single Domain**: Assumes all URLs are from the same domain
- **Static Rules**: Doesn't handle robots.txt updates during processing

#### Future Generalization

For multi-domain support, the system could:
- Extract domain from each URL and maintain a robots.txt cache per domain
- Implement TTL-based cache expiration for robots.txt rules
- Add domain-specific rate limiting and crawl delay handling

**Trade-off**: This would add complexity and reduce performance (more network requests, cache management overhead) for the current single-domain use case.
