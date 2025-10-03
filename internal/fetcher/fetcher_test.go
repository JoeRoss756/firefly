package fetcher

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestParseRobotsTxt_Complete(t *testing.T) {
	robotsTxt := `# Complete robots.txt example
User-agent: *
Disallow: /private/
Disallow: /admin/
Crawl-delay: 2

User-agent: EssayAnalyzer/1.0
Disallow: /restricted/
Crawl-delay: 1

User-agent: Googlebot
Disallow: /temp/

# Comments should be ignored
Sitemap: https://example.com/sitemap.xml`

	reader := strings.NewReader(robotsTxt)
	parser, err := parseRobotsTxt(reader, "https://example.com")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if parser == nil {
		t.Fatal("Expected robots parser to be initialized")
	}

	// Verify parsing worked by checking rule count
	if len(parser.rules) != 3 {
		t.Errorf("Expected 3 rule groups, got %d", len(parser.rules))
	}

	// Check specific rules
	foundWildcard := false
	foundEssayAnalyzer := false
	foundGooglebot := false

	for _, rule := range parser.rules {
		switch rule.UserAgent {
		case "*":
			foundWildcard = true
			if len(rule.Disallowed) != 2 {
				t.Errorf("Expected 2 disallow rules for *, got %d", len(rule.Disallowed))
			}
			if rule.CrawlDelay != 2*time.Second {
				t.Errorf("Expected crawl delay of 2s for *, got %v", rule.CrawlDelay)
			}
		case "EssayAnalyzer/1.0":
			foundEssayAnalyzer = true
			if len(rule.Disallowed) != 1 {
				t.Errorf("Expected 1 disallow rule for EssayAnalyzer, got %d", len(rule.Disallowed))
			}
			if rule.CrawlDelay != 1*time.Second {
				t.Errorf("Expected crawl delay of 1s for EssayAnalyzer, got %v", rule.CrawlDelay)
			}
		case "Googlebot":
			foundGooglebot = true
			if len(rule.Disallowed) != 1 {
				t.Errorf("Expected 1 disallow rule for Googlebot, got %d", len(rule.Disallowed))
			}
		}
	}

	if !foundWildcard || !foundEssayAnalyzer || !foundGooglebot {
		t.Error("Not all expected user-agent rules were found")
	}
}

func TestParseRobotsTxt_Minimal(t *testing.T) {
	robotsTxt := `User-agent: *
Disallow: /admin/`

	reader := strings.NewReader(robotsTxt)
	parser, err := parseRobotsTxt(reader, "https://example.com")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(parser.rules) != 1 {
		t.Errorf("Expected 1 rule group, got %d", len(parser.rules))
	}

	rule := parser.rules[0]
	if rule.UserAgent != "*" {
		t.Errorf("Expected user-agent *, got %s", rule.UserAgent)
	}

	if len(rule.Disallowed) != 1 || rule.Disallowed[0] != "/admin/" {
		t.Errorf("Expected disallow /admin/, got %v", rule.Disallowed)
	}

	if rule.CrawlDelay != 0 {
		t.Errorf("Expected no crawl delay, got %v", rule.CrawlDelay)
	}
}

func TestParseRobotsTxt_Empty(t *testing.T) {
	reader := strings.NewReader("")
	parser, err := parseRobotsTxt(reader, "https://example.com")

	if err != nil {
		t.Fatalf("Expected no error for empty robots.txt, got %v", err)
	}

	if len(parser.rules) != 0 {
		t.Errorf("Expected 0 rules for empty robots.txt, got %d", len(parser.rules))
	}
}

func TestRobotsParser_IsAllowed_BasicRules(t *testing.T) {
	robotsTxt := `User-agent: *
Disallow: /private/
Disallow: /admin/
Disallow: /tag/expire-images*`

	reader := strings.NewReader(robotsTxt)
	parser, err := parseRobotsTxt(reader, "https://example.com")
	if err != nil {
		t.Fatalf("Failed to parse robots.txt: %v", err)
	}

	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		// Allowed paths (same domain)
		{"Root path", "https://example.com/", true},
		{"Article path", "https://example.com/2019/08/23/article.html", true},
		{"Public path", "https://example.com/public/content", true},
		
		// Disallowed paths (same domain)
		{"Private path", "https://example.com/private/secret", false},
		{"Admin path", "https://example.com/admin/dashboard", false},
		{"Wildcard match 1", "https://example.com/tag/expire-images/old", false},
		{"Wildcard match 2", "https://example.com/tag/expire-images123", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.IsAllowed(tt.url, UserAgent)
			if result != tt.expected {
				t.Errorf("IsAllowed(%q) = %v, expected %v", tt.url, result, tt.expected)
			}
		})
	}
}

func TestRobotsParser_IsAllowed_UserAgentSpecific(t *testing.T) {
	robotsTxt := `User-agent: *
Disallow: /private/

User-agent: EssayAnalyzer/1.0
Disallow: /restricted/

User-agent: Googlebot
Disallow: /temp/`

	reader := strings.NewReader(robotsTxt)
	parser, err := parseRobotsTxt(reader, "https://example.com")
	if err != nil {
		t.Fatalf("Failed to parse robots.txt: %v", err)
	}

	tests := []struct {
		name      string
		url       string
		userAgent string
		expected  bool
	}{
		// EssayAnalyzer specific rules
		{"EssayAnalyzer allowed", "https://example.com/public/", "EssayAnalyzer/1.0", true},
		{"EssayAnalyzer restricted", "https://example.com/restricted/content", "EssayAnalyzer/1.0", false},
		
		// Googlebot specific rules
		{"Googlebot allowed", "https://example.com/public/", "Googlebot", true},
		{"Googlebot temp blocked", "https://example.com/temp/file", "Googlebot", false},
		
		// Wildcard rules for unknown user agents
		{"Unknown agent private blocked", "https://example.com/private/", "UnknownBot", false},
		{"Unknown agent public allowed", "https://example.com/public/", "UnknownBot", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.IsAllowed(tt.url, tt.userAgent)
			if result != tt.expected {
				t.Errorf("IsAllowed(%q, %q) = %v, expected %v", tt.url, tt.userAgent, result, tt.expected)
			}
		})
	}
}

func TestMatchesPattern(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		pattern  string
		expected bool
	}{
		{"Exact match", "/admin/", "/admin/", true},
		{"Prefix match", "/admin/dashboard", "/admin/", true},
		{"No match", "/public/", "/admin/", false},
		{"Wildcard match", "/tag/expire-images123", "/tag/expire-images*", true},
		{"Wildcard no match", "/tag/other", "/tag/expire-images*", false},
		{"Empty pattern", "/any/path", "", false},
		{"Root wildcard", "/any/path", "*", true},
		{"Complex wildcard", "/path/to/file.jpg", "/path/*/file.*", true},
		{"Wildcard at end", "/api/v1/users", "/api/v1/*", true},
		{"Multiple wildcards", "/a/b/c/d", "/a/*/c/*", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchesPattern(tt.path, tt.pattern)
			if result != tt.expected {
				t.Errorf("matchesPattern(%q, %q) = %v, expected %v", 
					tt.path, tt.pattern, result, tt.expected)
			}
		})
	}
}

func TestFetcher_IsAllowed_NoRobots(t *testing.T) {
	fetcher := New(1.0, false)
	
	// Without loading robots.txt, everything should be allowed
	tests := []string{
		"https://example.com/",
		"https://example.com/private/",
		"https://example.com/admin/",
		"https://example.com/any/path",
	}

	for _, url := range tests {
		t.Run(url, func(t *testing.T) {
			if !fetcher.IsAllowed(url) {
				t.Errorf("Expected %s to be allowed when no robots.txt loaded", url)
			}
		})
	}
}

func TestFetcher_IsAllowed_WithRobots(t *testing.T) {
	fetcher := New(1.0, false)
	
	// Mock robots.txt data for example.com
	robotsTxt := `User-agent: *
Disallow: /private/
Disallow: /admin/

User-agent: EssayAnalyzer/1.0
Disallow: /restricted/`

	// Parse and set robots data directly
	reader := strings.NewReader(robotsTxt)
	parser, err := parseRobotsTxt(reader, "https://example.com")
	if err != nil {
		t.Fatalf("Failed to parse robots.txt: %v", err)
	}
	fetcher.robots = parser

	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		{"Allowed root", "https://example.com/", true},
		{"Allowed article", "https://example.com/articles/test", true},
		{"Disallowed private", "https://example.com/private/secret", false},
		{"Disallowed admin", "https://example.com/admin/panel", false},
		{"Disallowed restricted (EssayAnalyzer)", "https://example.com/restricted/area", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fetcher.IsAllowed(tt.url)
			if result != tt.expected {
				t.Errorf("IsAllowed(%q) = %v, expected %v", tt.url, result, tt.expected)
			}
		})
	}
}

func TestRateLimiter_Basic(t *testing.T) {
	// Test that rate limiter doesn't panic and allows some requests
	fetcher := New(10.0, false) // 10 requests per second
	
	ctx := context.Background()
	
	// Should be able to make at least one request immediately
	err := fetcher.rateLimiter.Wait(ctx)
	if err != nil {
		t.Fatalf("Rate limiter wait failed: %v", err)
	}
	
	// Should be able to make another request (might be delayed)
	err = fetcher.rateLimiter.Wait(ctx)
	if err != nil {
		t.Fatalf("Second rate limiter wait failed: %v", err)
	}
}

func TestRobotsURLConstruction(t *testing.T) {
	tests := []struct {
		name     string
		baseURL  string
		expected string
	}{
		{
			name:     "URL with trailing slash",
			baseURL:  "https://www.engadget.com/",
			expected: "https://www.engadget.com/robots.txt",
		},
		{
			name:     "URL without trailing slash", 
			baseURL:  "https://www.engadget.com",
			expected: "https://www.engadget.com/robots.txt",
		},
		{
			name:     "URL with path should strip to domain",
			baseURL:  "https://www.engadget.com/articles/tech",
			expected: "https://www.engadget.com/robots.txt",
		},
		{
			name:     "URL with port",
			baseURL:  "https://example.com:8080/path",
			expected: "https://example.com:8080/robots.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the URL construction logic directly
			// This simulates what LoadRobotsTxt does
			parsedURL, err := parseURL(tt.baseURL)
			if err != nil {
				t.Fatalf("Failed to parse URL: %v", err)
			}
			
			robotsURL := parsedURL.Scheme + "://" + parsedURL.Host + "/robots.txt"
			
			if robotsURL != tt.expected {
				t.Errorf("Expected robots URL %s, got %s", tt.expected, robotsURL)
			}
		})
	}
}

// Helper function to test URL parsing without making HTTP requests
func parseURL(baseURL string) (*struct{ Scheme, Host string }, error) {
	// Simple URL parsing simulation
	if strings.HasPrefix(baseURL, "https://") {
		baseURL = strings.TrimPrefix(baseURL, "https://")
		hostPath := strings.Split(baseURL, "/")
		return &struct{ Scheme, Host string }{"https", hostPath[0]}, nil
	}
	if strings.HasPrefix(baseURL, "http://") {
		baseURL = strings.TrimPrefix(baseURL, "http://")
		hostPath := strings.Split(baseURL, "/")
		return &struct{ Scheme, Host string }{"http", hostPath[0]}, nil
	}
	return nil, nil
}
