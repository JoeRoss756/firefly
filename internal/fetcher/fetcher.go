package fetcher

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/time/rate"
)

const (
	// UserAgent identifies our crawler to servers
	UserAgent = "EssayAnalyzer/1.0"

	// DefaultTimeout for HTTP requests
	DefaultTimeout = 30 * time.Second

	// MaxRetries for failed requests
	MaxRetries = 3

	// BackoffBase for exponential backoff
	BackoffBase = time.Second
)

// RobotsRule represents a robots.txt rule
type RobotsRule struct {
	UserAgent  string
	Disallowed []string
	CrawlDelay time.Duration
}

// RobotsParser handles robots.txt parsing and compliance
type RobotsParser struct {
	rules   []RobotsRule
	baseURL string
}

// Fetcher handles HTTP requests with rate limiting and retries
type Fetcher struct {
	client        *http.Client
	rateLimiter   *rate.Limiter
	robots        *RobotsParser
	verbose       bool
	userRateLimit float64 // User-specified rate limit (0 = no limit)
}

// New creates a new Fetcher with rate limiting and robots.txt compliance
func New(requestsPerSecond float64, verbose bool) *Fetcher {
	var limiter *rate.Limiter
	if requestsPerSecond > 0 {
		limiter = rate.NewLimiter(rate.Limit(requestsPerSecond), int(requestsPerSecond)+1)
	} else {
		// No rate limit by default - use infinite rate
		limiter = rate.NewLimiter(rate.Inf, 0)
	}

	return &Fetcher{
		client: &http.Client{
			Timeout: DefaultTimeout,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 100, // Increased for higher concurrency
				IdleConnTimeout:     90 * time.Second,
			},
		},
		rateLimiter:   limiter,
		verbose:       verbose,
		userRateLimit: requestsPerSecond,
	}
}

// LoadRobotsTxt fetches and parses robots.txt for the given domain
func (f *Fetcher) LoadRobotsTxt(ctx context.Context, baseURL string) error {
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return fmt.Errorf("parsing base URL: %w", err)
	}
	robotsURL := parsedURL.Scheme + "://" + parsedURL.Host + "/robots.txt"

	if f.verbose {
		fmt.Printf("Fetching robots.txt from: %s\n", robotsURL)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", robotsURL, nil)
	if err != nil {
		return fmt.Errorf("creating robots.txt request: %w", err)
	}

	req.Header.Set("User-Agent", UserAgent)

	resp, err := f.client.Do(req)
	if err != nil {
		return fmt.Errorf("fetching robots.txt: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		// No robots.txt means everything is allowed
		if f.verbose {
			fmt.Println("No robots.txt found - all URLs allowed")
		}
		f.robots = &RobotsParser{baseURL: baseURL}
		return nil
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("robots.txt returned status %d", resp.StatusCode)
	}

	parser, err := parseRobotsTxt(resp.Body, baseURL)
	if err != nil {
		return fmt.Errorf("parsing robots.txt: %w", err)
	}

	f.robots = parser

	// Apply crawl-delay from robots.txt if user didn't specify a rate limit
	if f.userRateLimit == 0 {
		crawlDelay := parser.GetCrawlDelay(UserAgent)
		if crawlDelay > 0 {
			// Convert crawl delay to requests per second
			reqPerSec := 1.0 / crawlDelay.Seconds()
			f.rateLimiter = rate.NewLimiter(rate.Limit(reqPerSec), 1)
			if f.verbose {
				fmt.Printf("Applying robots.txt Crawl-Delay: %v (%.2f req/sec)\n", crawlDelay, reqPerSec)
			}
		} else if f.verbose {
			fmt.Println("No Crawl-Delay in robots.txt - using unlimited rate")
		}
	} else if f.verbose {
		fmt.Printf("Using user-specified rate limit: %.1f req/sec\n", f.userRateLimit)
	}

	if f.verbose {
		fmt.Printf("Loaded robots.txt with %d rule groups\n", len(parser.rules))
	}

	return nil
}

// IsAllowed checks if a URL is allowed by robots.txt
func (f *Fetcher) IsAllowed(urlStr string) bool {
	if f.robots == nil {
		// No robots.txt loaded, assume allowed
		return true
	}

	return f.robots.IsAllowed(urlStr, UserAgent)
}

// FetchURL fetches content from a URL with rate limiting, retries, and robots.txt compliance
func (f *Fetcher) FetchURL(ctx context.Context, urlStr string) (io.ReadCloser, error) {
	// Check robots.txt compliance first
	if !f.IsAllowed(urlStr) {
		return nil, fmt.Errorf("URL disallowed by robots.txt: %s", urlStr)
	}

	var lastErr error

	for attempt := 0; attempt < MaxRetries; attempt++ {
		// Wait for rate limiter
		if err := f.rateLimiter.Wait(ctx); err != nil {
			return nil, fmt.Errorf("rate limiter error: %w", err)
		}

		if f.verbose && attempt > 0 {
			fmt.Printf("Retrying %s (attempt %d/%d)\n", urlStr, attempt+1, MaxRetries)
		}

		req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
		if err != nil {
			return nil, fmt.Errorf("creating request: %w", err)
		}

		req.Header.Set("User-Agent", UserAgent)
		req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
		req.Header.Set("Accept-Language", "en-US,en;q=0.5")
		// Note: Don't set Accept-Encoding manually - Go's HTTP client automatically
		// handles gzip/deflate compression AND decompression when we don't set it
		req.Header.Set("Connection", "keep-alive")

		resp, err := f.client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("HTTP request failed: %w", err)
			f.backoff(attempt)
			continue
		}

		// Check for HTTP errors
		if resp.StatusCode >= 400 {
			resp.Body.Close()
			lastErr = fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)

			// Don't retry client errors (4xx), but do retry server errors (5xx)
			if resp.StatusCode < 500 {
				return nil, lastErr
			}

			f.backoff(attempt)
			continue
		}

		if f.verbose {
			fmt.Printf("Successfully fetched %s (%s)\n", urlStr, resp.Header.Get("Content-Type"))
		}

		return resp.Body, nil
	}

	return nil, fmt.Errorf("failed after %d attempts: %w", MaxRetries, lastErr)
}

// backoff implements exponential backoff with jitter
func (f *Fetcher) backoff(attempt int) {
	backoff := BackoffBase * time.Duration(1<<uint(attempt))
	if backoff > 30*time.Second {
		backoff = 30 * time.Second
	}

	if f.verbose {
		fmt.Printf("Backing off for %v\n", backoff)
	}

	time.Sleep(backoff)
}

// parseRobotsTxt parses robots.txt content
func parseRobotsTxt(reader io.Reader, baseURL string) (*RobotsParser, error) {
	parser := &RobotsParser{
		baseURL: baseURL,
		rules:   make([]RobotsRule, 0),
	}

	scanner := bufio.NewScanner(reader)
	var currentRule *RobotsRule

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Split on first colon
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.ToLower(strings.TrimSpace(parts[0]))
		value := strings.TrimSpace(parts[1])

		switch key {
		case "user-agent":
			// Start new rule group
			if currentRule != nil {
				parser.rules = append(parser.rules, *currentRule)
			}
			currentRule = &RobotsRule{
				UserAgent:  value,
				Disallowed: make([]string, 0),
			}

		case "disallow":
			if currentRule != nil && value != "" {
				currentRule.Disallowed = append(currentRule.Disallowed, value)
			}

		case "crawl-delay":
			if currentRule != nil {
				if delay, err := strconv.Atoi(value); err == nil {
					currentRule.CrawlDelay = time.Duration(delay) * time.Second
				}
			}
		}
	}

	// Add final rule
	if currentRule != nil {
		parser.rules = append(parser.rules, *currentRule)
	}

	return parser, scanner.Err()
}

// IsAllowed checks if a URL is allowed for the given user agent
func (rp *RobotsParser) IsAllowed(urlStr, userAgent string) bool {
	if len(rp.rules) == 0 {
		return true // No rules means everything is allowed
	}

	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return false // Invalid URL
	}

	path := parsedURL.Path
	if path == "" {
		path = "/"
	}

	// Find applicable rules (specific user-agent first, then *)
	var applicableRules []RobotsRule

	// First, add specific user-agent rules
	for _, rule := range rp.rules {
		if strings.EqualFold(rule.UserAgent, userAgent) {
			applicableRules = append(applicableRules, rule)
		}
	}

	// Then, always add wildcard rules (they apply to everyone)
	for _, rule := range rp.rules {
		if rule.UserAgent == "*" {
			applicableRules = append(applicableRules, rule)
		}
	}

	// Check disallow patterns
	for _, rule := range applicableRules {
		for _, pattern := range rule.Disallowed {
			if matchesPattern(path, pattern) {
				return false
			}
		}
	}

	return true
}

// GetCrawlDelay returns the crawl delay from robots.txt for the given user agent
func (rp *RobotsParser) GetCrawlDelay(userAgent string) time.Duration {
	if len(rp.rules) == 0 {
		return 0
	}

	// Check for specific user-agent rules first
	for _, rule := range rp.rules {
		if strings.EqualFold(rule.UserAgent, userAgent) && rule.CrawlDelay > 0 {
			return rule.CrawlDelay
		}
	}

	// Check wildcard rules
	for _, rule := range rp.rules {
		if rule.UserAgent == "*" && rule.CrawlDelay > 0 {
			return rule.CrawlDelay
		}
	}

	return 0
}

// matchesPattern checks if a path matches a robots.txt pattern
func matchesPattern(path, pattern string) bool {
	if pattern == "" {
		return false
	}

	// Handle wildcard patterns
	if strings.Contains(pattern, "*") {
		// Convert robots.txt pattern to regex
		regexPattern := regexp.QuoteMeta(pattern)
		regexPattern = strings.ReplaceAll(regexPattern, "\\*", ".*")
		regexPattern = "^" + regexPattern

		if matched, _ := regexp.MatchString(regexPattern, path); matched {
			return true
		}
	}

	// Exact prefix match
	return strings.HasPrefix(path, pattern)
}
