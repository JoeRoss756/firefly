package parser

import (
	"strings"
	"testing"
)

// TestExtractText_IdealSelector tests extraction using the ideal selector (header + body)
func TestExtractText_IdealSelector(t *testing.T) {
	parser := New(false)

	// Create HTML with both header and body content that meets 200+ char requirement
	html := `
	<html>
		<body>
			<article>
				<header>
					<h1>This is a test article title that provides good context</h1>
					<p>By Test Author on January 1, 2024</p>
				</header>
				<div data-article-body="true">
					<p>This is the main article content that contains the body text we want to extract. It should be long enough to meet the minimum length requirement for the ideal selector which is 200 characters minimum.</p>
				</div>
			</article>
		</body>
	</html>`

	reader := strings.NewReader(html)
	result, err := parser.ExtractText(reader)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should contain both header and body content
	if !strings.Contains(result, "This is a test article title") {
		t.Error("Expected result to contain header content")
	}

	if !strings.Contains(result, "This is the main article content") {
		t.Error("Expected result to contain body content")
	}

	// Should have some content
	if len(result) == 0 {
		t.Error("Expected result to have content")
	}

	// Should not increment failure count on success
	if parser.GetFailedCount() != 0 {
		t.Errorf("Expected failure count to be 0, got %d", parser.GetFailedCount())
	}
}

// TestExtractText_FallbackSelector tests fallback to body-only selector
func TestExtractText_FallbackSelector(t *testing.T) {
	parser := New(false)

	// Create HTML with only body content (no header), meeting 100+ char requirement
	html := `
	<html>
		<body>
			<div data-article-body="true">
				<p>This is body-only content that should be extracted using the fallback selector. It needs to be at least 100 characters long to meet the minimum requirement for the body-only selector.</p>
			</div>
		</body>
	</html>`

	reader := strings.NewReader(html)
	result, err := parser.ExtractText(reader)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Should contain body content
	if !strings.Contains(result, "This is body-only content") {
		t.Error("Expected result to contain body content")
	}

	// Should have some content
	if len(result) == 0 {
		t.Error("Expected result to have content")
	}

	// Should not increment failure count on success
	if parser.GetFailedCount() != 0 {
		t.Errorf("Expected failure count to be 0, got %d", parser.GetFailedCount())
	}
}

// TestExtractText_InvalidHTML tests handling of invalid HTML
func TestExtractText_InvalidHTML(t *testing.T) {
	parser := New(false)

	// Malformed HTML that should cause goquery to fail
	invalidHTML := `<html><body><div><p>Unclosed tags and malformed content`

	reader := strings.NewReader(invalidHTML)
	result, err := parser.ExtractText(reader)

	// Note: goquery is quite forgiving, so this test checks if we handle
	// the case where no suitable content is found rather than HTML parsing errors
	if err == nil {
		// If no error, result should be empty and failure count should increment
		if result != "" {
			t.Error("Expected empty result for invalid HTML")
		}
	} else {
		// If there is an error, it should be a parsing error
		if !strings.Contains(err.Error(), "failed to extract clean content") {
			t.Errorf("Expected parsing error, got %v", err)
		}
	}

	// Should increment failure count
	if parser.GetFailedCount() == 0 {
		t.Error("Expected failure count to increment for invalid HTML")
	}
}

// TestExtractText_NoSuitableContent tests when no selectors match
func TestExtractText_NoSuitableContent(t *testing.T) {
	parser := New(false)

	// HTML with no matching selectors
	html := `
	<html>
		<body>
			<div class="some-other-content">
				<p>This content doesn't match our selectors</p>
			</div>
		</body>
	</html>`

	reader := strings.NewReader(html)
	result, err := parser.ExtractText(reader)

	if err == nil {
		t.Fatal("Expected error for no suitable content")
	}

	if result != "" {
		t.Error("Expected empty result when no suitable content found")
	}

	if !strings.Contains(err.Error(), "failed to extract clean content") {
		t.Errorf("Expected specific error message, got %v", err)
	}

	// Should increment failure count
	if parser.GetFailedCount() != 1 {
		t.Errorf("Expected failure count to be 1, got %d", parser.GetFailedCount())
	}
}

// TestExtractText_ShortContent tests that short content is still extracted
func TestExtractText_ShortContent(t *testing.T) {
	parser := New(false)

	// HTML with short content that should still be extracted
	html := `
	<html>
		<body>
			<article>
				<header><h1>Short</h1></header>
				<div data-article-body="true"><p>Brief</p></div>
			</article>
		</body>
	</html>`

	reader := strings.NewReader(html)
	result, err := parser.ExtractText(reader)

	if err != nil {
		t.Fatalf("Expected no error for short content, got %v", err)
	}

	if result == "" {
		t.Error("Expected result even for short content")
	}

	// Should contain the short content
	if !strings.Contains(result, "Short") || !strings.Contains(result, "Brief") {
		t.Error("Expected result to contain short content")
	}

	// Should not increment failure count
	if parser.GetFailedCount() != 0 {
		t.Errorf("Expected failure count to be 0, got %d", parser.GetFailedCount())
	}
}

// TestFailureCount_MultipleFailures tests that failure count increments correctly
func TestFailureCount_MultipleFailures(t *testing.T) {
	parser := New(false)

	// Test multiple failures
	testCases := []string{
		`<html><body><div>No matching selectors</div></body></html>`,
		`<html><body><div class="other">No matching selectors</div></body></html>`,
	}

	for i, html := range testCases {
		reader := strings.NewReader(html)
		result, err := parser.ExtractText(reader)

		if err == nil {
			t.Errorf("Test case %d: Expected error", i)
		}

		if result != "" {
			t.Errorf("Test case %d: Expected empty result", i)
		}

		expectedCount := int64(i + 1)
		if parser.GetFailedCount() != expectedCount {
			t.Errorf("Test case %d: Expected failure count %d, got %d", i, expectedCount, parser.GetFailedCount())
		}
	}
}

// TestNew tests parser creation
func TestNew(t *testing.T) {
	// Test non-verbose parser
	parser := New(false)
	if parser == nil {
		t.Fatal("Expected parser to be created")
	}
	if parser.verbose {
		t.Error("Expected verbose to be false")
	}
	if parser.GetFailedCount() != 0 {
		t.Error("Expected initial failure count to be 0")
	}

	// Test verbose parser
	verboseParser := New(true)
	if !verboseParser.verbose {
		t.Error("Expected verbose to be true")
	}
}

// TestGetFailedCount tests the failure count getter
func TestGetFailedCount(t *testing.T) {
	parser := New(false)

	// Initial count should be 0
	if parser.GetFailedCount() != 0 {
		t.Error("Expected initial failure count to be 0")
	}

	// Trigger a failure
	html := `<html><body><div>No matching content</div></body></html>`
	reader := strings.NewReader(html)
	parser.ExtractText(reader)

	// Count should be 1
	if parser.GetFailedCount() != 1 {
		t.Errorf("Expected failure count to be 1, got %d", parser.GetFailedCount())
	}
}
