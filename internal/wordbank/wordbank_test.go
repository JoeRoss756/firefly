package wordbank

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNew_MissingFile tests that New returns an error when file is missing
func TestNew_MissingFile(t *testing.T) {
	nonExistentFile := "/path/that/does/not/exist/words.txt"

	wordBank, err := New(nonExistentFile)

	if err == nil {
		t.Fatal("Expected error for missing file, got nil")
	}

	if wordBank != nil {
		t.Error("Expected nil wordbank for missing file")
	}

	if !strings.Contains(err.Error(), "opening word bank file") {
		t.Errorf("Expected error message about opening file, got: %v", err)
	}
}

// TestNew_FilteringCorrectly tests that wordbank filters inputs correctly during loading
func TestNew_FilteringCorrectly(t *testing.T) {
	testFile := filepath.Join("testdata", "test_words.txt")

	wordBank, err := New(testFile)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if wordBank == nil {
		t.Fatal("Expected wordbank to be created")
	}

	// Test words that should be included (3+ chars, alphabetic only, lowercase)
	expectedWords := []string{
		"technology", "innovation", "computer", "software", "hardware",
		"development", "programming", "algorithm", "database", "network",
		"artificial", "intelligence", "machine", "learning", "science",
		"engineering", "analysis", "system", "application", "framework",
		"uppercase", "mixedcase", // These should be converted to lowercase
	}

	for _, word := range expectedWords {
		if !wordBank.IsValid(word) {
			t.Errorf("Expected word '%s' to be valid", word)
		}
	}

	// Test words that should be filtered out
	filteredWords := []string{
		"ai",                   // too short (2 chars)
		"go",                   // too short (2 chars)
		"js",                   // too short (2 chars)
		"x",                    // too short (1 char)
		"ab",                   // too short (2 chars)
		"123",                  // not alphabetic
		"test-word",            // contains hyphen
		"word_with_underscore", // contains underscore
	}

	for _, word := range filteredWords {
		if wordBank.IsValid(word) {
			t.Errorf("Expected word '%s' to be filtered out", word)
		}
	}

	// Verify size - should only include valid words
	// From our test file: 22 valid words (10 + 2 + 10, with UPPERCASE/MixedCase converted)
	expectedSize := 22
	if wordBank.Size() != expectedSize {
		t.Errorf("Expected wordbank size to be %d, got %d", expectedSize, wordBank.Size())
	}
}

// TestIsValid_ValidWords tests IsValid with words that should be valid
func TestIsValid_ValidWords(t *testing.T) {
	testFile := filepath.Join("testdata", "test_words.txt")

	wordBank, err := New(testFile)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	validWords := []string{
		"technology",
		"innovation",
		"computer",
		"artificial",
		"intelligence",
		"programming",
	}

	for _, word := range validWords {
		if !wordBank.IsValid(word) {
			t.Errorf("Expected word '%s' to be valid", word)
		}
	}
}

// TestIsValid_InvalidWords tests IsValid with words that should be invalid
func TestIsValid_InvalidWords(t *testing.T) {
	testFile := filepath.Join("testdata", "test_words.txt")

	wordBank, err := New(testFile)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	invalidWords := []string{
		"",          // empty string
		"ai",        // too short
		"go",        // too short
		"notinbank", // not in wordbank (valid format but not in file)
		"missing",   // not in wordbank
		"test123",   // contains numbers
		"test-word", // contains hyphen
	}

	for _, word := range invalidWords {
		if wordBank.IsValid(word) {
			t.Errorf("Expected word '%s' to be invalid", word)
		}
	}
}

// TestIsValid_CaseInsensitive tests that IsValid handles case correctly
func TestIsValid_CaseInsensitive(t *testing.T) {
	testFile := filepath.Join("testdata", "test_words.txt")

	wordBank, err := New(testFile)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Test case variations of words that exist in the wordbank
	testCases := []struct {
		word     string
		expected bool
	}{
		{"technology", true}, // lowercase (original)
		{"TECHNOLOGY", true}, // uppercase (should be converted)
		{"Technology", true}, // title case (should be converted)
		{"tEcHnOlOgY", true}, // mixed case (should be converted)
		{"UPPERCASE", true},  // originally uppercase in file, stored as lowercase
		{"uppercase", true},  // lowercase version
		{"MixedCase", true},  // originally mixed case in file
		{"mixedcase", true},  // lowercase version
	}

	for _, tc := range testCases {
		result := wordBank.IsValid(tc.word)
		if result != tc.expected {
			t.Errorf("IsValid(%s): expected %v, got %v", tc.word, tc.expected, result)
		}
	}
}

// TestSize tests the Size method
func TestSize(t *testing.T) {
	testFile := filepath.Join("testdata", "test_words.txt")

	wordBank, err := New(testFile)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	size := wordBank.Size()
	if size <= 0 {
		t.Error("Expected wordbank size to be greater than 0")
	}

	// Should be exactly 22 valid words from our test file
	expectedSize := 22
	if size != expectedSize {
		t.Errorf("Expected wordbank size to be %d, got %d", expectedSize, size)
	}
}

// TestNew_EmptyFile tests handling of empty file
func TestNew_EmptyFile(t *testing.T) {
	// Create temporary empty file
	tmpFile, err := os.CreateTemp("", "empty_wordbank_*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	wordBank, err := New(tmpFile.Name())
	if err != nil {
		t.Fatalf("Expected no error for empty file, got %v", err)
	}

	if wordBank == nil {
		t.Fatal("Expected wordbank to be created for empty file")
	}

	if wordBank.Size() != 0 {
		t.Errorf("Expected empty wordbank to have size 0, got %d", wordBank.Size())
	}

	// Any word should be invalid in empty wordbank
	if wordBank.IsValid("test") {
		t.Error("Expected all words to be invalid in empty wordbank")
	}
}

// TestNew_OnlyWhitespace tests file with only whitespace
func TestNew_OnlyWhitespace(t *testing.T) {
	// Create temporary file with only whitespace
	tmpFile, err := os.CreateTemp("", "whitespace_only_*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	content := `
	
	   
	
	`

	if err := os.WriteFile(tmpFile.Name(), []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}

	wordBank, err := New(tmpFile.Name())
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if wordBank.Size() != 0 {
		t.Errorf("Expected wordbank with only whitespace to have size 0, got %d", wordBank.Size())
	}
}
