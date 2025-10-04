# Essay Analyzer

A web scraper and word frequency analyzer designed to process large collections of articles. Optimized for Engadget.com with intelligent rate limiting, robots.txt compliance, and concurrent processing.

## Quick Start

### Build
```bash
go build -o essay_analyzer ./cmd/essay_analyzer
```

### Run
```bash
# Basic usage (unlimited rate, respects robots.txt), quick test url file
./essay_analyzer --urls-file files/test-5-urls --wordbank-file files/words.txt

# Basic usage (unlimited rate, respects robots.txt)
./essay_analyzer --urls-file files/endg-urls --wordbank-file files/words.txt

# With verbose logging
./essay_analyzer --urls-file files/endg-urls --wordbank-file files/words.txt --verbose

# Custom rate limiting and workers
./essay_analyzer --urls-file files/endg-urls --wordbank-file files/words.txt \
  --workers 100 --rate-limit 50.0
```

## Command Line Options

| Option | Description | Default | Example |
|--------|-------------|---------|---------|
| `--urls-file` | Path to file containing URLs (one per line) | *required* | `files/endg-urls` |
| `--wordbank-file` | Path to word bank file (one word per line) | *required* | `files/words.txt` |
| `--workers` | Number of concurrent workers | `50` | `--workers 100` |
| `--rate-limit` | Requests per second (0 = unlimited) | `0` | `--rate-limit 50.0` |
| `--verbose` | Enable verbose logging | `false` | `--verbose` |

### Rate Limiting Behavior

The analyzer uses intelligent rate limiting that respects robots.txt:

1. **Default (--rate-limit 0)**: No rate limiting unless robots.txt specifies a `Crawl-Delay`
2. **Explicit Rate**: `--rate-limit 50.0` forces 50 requests/second, overriding robots.txt
3. **robots.txt Priority**: If no rate limit is specified, respects `Crawl-Delay` from robots.txt

**Performance**: With default settings (50 workers, no rate limit), processes ~16 URLs/second (~42 minutes for 40,000 URLs)


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

Our robots.txt implementation is optimized for **single-domain processing** with automatic rate limiting:

**Current Approach**:
- Fetch robots.txt once at startup for engadget.com
- Cache and reuse the parsed rules for all 40,000 URLs
- Validate each URL against the cached rules before fetching
- **Automatically apply `Crawl-Delay`** if present and no rate limit is specified
- User can override with explicit `--rate-limit` flag

**Benefits**:
- **Performance**: Only one robots.txt request instead of 40,000
- **Efficiency**: Cached rule parsing and pattern matching
- **Automatic Compliance**: Respects crawl delays without manual configuration
- **Flexible**: User can override rate limiting when appropriate
- **Fast by Default**: No rate limiting unless required

**Limitations**:
- **Single Domain**: Assumes all URLs are from the same domain
- **Static Rules**: Doesn't handle robots.txt updates during processing

#### Future Generalization

For multi-domain support, the system could:
- Extract domain from each URL and maintain a robots.txt cache per domain
- Implement TTL-based cache expiration for robots.txt rules
- Add domain-specific rate limiting and crawl delay handling

**Trade-off**: This would add complexity and reduce performance (more network requests, cache management overhead) for the current single-domain use case.

### Word Parsing and Extraction

Our word parsing strategy is designed to **maximize word capture** while maintaining **clean frequency analysis** for essay content.

#### Parsing Logic

**Regex Pattern**: `[a-zA-Z]+`
- Extracts **consecutive alphabetic characters** as words
- **Automatically handles punctuation** by treating it as word boundaries
- **Preserves word significance** by extracting alphabetic portions from mixed content

#### Why This Approach?

**Problem**: Adjacent punctuation can cause words to be missed entirely
- `"technology!"` should count as "technology", not be ignored
- `"iPhone5"` should count as "iPhone", preserving the significant term
- `"state-of-the-art"` should extract meaningful components

**Solution**: Extract alphabetic sequences, let wordbank filter for validity

#### Examples

| Input Text | Extracted Words | Counted Words* |
|------------|----------------|----------------|
| `"Technology!"` | `["Technology"]` | `["technology"]` |
| `"iPhone5 launch"` | `["iPhone", "launch"]` | `["iphone", "launch"]` |
| `"state-of-the-art"` | `["state", "of", "the", "art"]` | `["state", "art"]`** |
| `"don't miss this"` | `["don", "t", "miss", "this"]` | `["miss", "this"]`** |
| `"HTML5 and CSS3"` | `["HTML", "and", "CSS"]` | `["html", "css"]`** |

*Assuming words exist in wordbank  
**Assuming "of", "the", "don", "t" are not in wordbank (filtered out)

#### Benefits

1. **No Lost Words**: Punctuation doesn't cause word loss
2. **Preserves Significance**: "iPhone5" → "iPhone" (keeps the important term)
3. **Clean Separation**: Hyphens and punctuation create natural word boundaries
4. **Wordbank Filtering**: Invalid fragments (like "t" from "don't") are filtered out
5. **Case Normalization**: All words converted to lowercase for consistent counting

#### Trade-offs

- **Compound Terms**: "state-of-the-art" becomes separate words rather than a phrase
- **Version Numbers**: "iPhone15" loses version context, becomes just "iPhone"
- **Contractions**: "don't" becomes "don" + "t" (but "t" filtered by wordbank)

**Rationale**: For essay frequency analysis, we prioritize **capturing core concepts** (iPhone, HTML, CSS) over **preserving specific versions or compound phrases**. This approach maximizes word capture while maintaining clean, meaningful frequency data.

## Future Work

### 1. Enhanced Logging and Observability
- **Structured Logging**: Replace `fmt.Printf` with structured logging library (e.g., `zerolog`, `zap`)
- **Progress Tracking**: Real-time progress bar showing URLs processed, success rate, and ETA
- **Metrics Export**: Export metrics to Prometheus or similar monitoring systems
- **Error Classification**: Categorize errors (network, parsing, rate limiting) with detailed statistics
- **Verbose Levels**: Multiple verbosity levels (debug, info, warn, error) instead of binary verbose flag

### 2. Worker Pool Optimization
- **Dynamic Worker Adjustment**: Automatically tune worker counts based on CPU usage and network conditions
- **Adaptive Rate Limiting**: Adjust rate limits based on server response times and error rates
- **Benchmarking Suite**: Automated benchmarks to determine optimal worker distribution for different scenarios

### 3. Generalized HTML Parsing
- **Multi-Site Support**: Generic content extraction
- **Content Validation**: Detect and handle paywalls, CAPTCHAs, and access restrictions

### 4. Containerization and Deployment
- **Docker Support**:
  - Pre-built images on Docker Hub/GitHub Container Registry
- 
### 5. Improved CLI Experience
- **Configuration Files**: YAML/TOML config files for persistent settings
- **Output Formats**: Support for CSV, TSV, Excel, SQLite in addition to JSON
- **Dry Run Mode**: Preview what would be processed without fetching

### 6. Testing and Quality
- **Integration Tests**: End-to-end tests with mock HTTP servers
- **CI/CD Pipeline**: Automated testing and release workflow
