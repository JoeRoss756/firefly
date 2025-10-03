package processor

import (
	"reflect"
	"testing"
)

// MockWordBank implements WordValidator for testing
type MockWordBank struct {
	validWords map[string]bool
}

// NewMockWordBank creates a mock wordbank with predefined valid words
func NewMockWordBank(validWords []string) *MockWordBank {
	wordMap := make(map[string]bool)
	for _, word := range validWords {
		wordMap[word] = true
	}
	return &MockWordBank{validWords: wordMap}
}

// IsValid implements WordValidator interface
func (m *MockWordBank) IsValid(word string) bool {
	return m.validWords[word]
}

// TestNew tests the New constructor
func TestNew(t *testing.T) {
	mockWordBank := NewMockWordBank([]string{"test", "word"})

	// Test non-verbose processor
	processor := New(mockWordBank, false)

	if processor == nil {
		t.Fatal("Expected processor to be created")
	}

	if processor.wordBank != mockWordBank {
		t.Error("Expected wordBank to be set correctly")
	}

	if processor.verbose {
		t.Error("Expected verbose to be false")
	}

	if processor.wordRegex == nil {
		t.Error("Expected wordRegex to be initialized")
	}

	// Test that regex pattern is correct
	testWords := processor.wordRegex.FindAllString("hello world 123 test-word", -1)
	expected := []string{"hello", "world", "test", "word"}
	if !reflect.DeepEqual(testWords, expected) {
		t.Errorf("Expected regex to extract %v, got %v", expected, testWords)
	}
}

// TestNew_Verbose tests the New constructor with verbose flag
func TestNew_Verbose(t *testing.T) {
	mockWordBank := NewMockWordBank([]string{"test"})

	processor := New(mockWordBank, true)

	if !processor.verbose {
		t.Error("Expected verbose to be true")
	}
}

// TestProcessText_BasicCounting tests basic word counting functionality
func TestProcessText_BasicCounting(t *testing.T) {
	validWords := []string{"technology", "innovation", "computer", "software"}
	mockWordBank := NewMockWordBank(validWords)
	processor := New(mockWordBank, false)

	text := "Technology and innovation drive computer software development. Technology is key."

	result := processor.ProcessText(text)

	expected := map[string]int{
		"technology": 2,
		"innovation": 1,
		"computer":   1,
		"software":   1,
	}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}

// TestProcessText_CaseInsensitive tests case-insensitive processing
func TestProcessText_CaseInsensitive(t *testing.T) {
	validWords := []string{"technology", "innovation"}
	mockWordBank := NewMockWordBank(validWords)
	processor := New(mockWordBank, false)

	text := "TECHNOLOGY Technology innovation INNOVATION Innovation"

	result := processor.ProcessText(text)

	expected := map[string]int{
		"technology": 2,
		"innovation": 3,
	}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}

// TestProcessText_FilterInvalidWords tests that invalid words are filtered out
func TestProcessText_FilterInvalidWords(t *testing.T) {
	validWords := []string{"technology", "computer"}
	mockWordBank := NewMockWordBank(validWords)
	processor := New(mockWordBank, false)

	text := "Technology and innovation drive computer software development"

	result := processor.ProcessText(text)

	// Only "technology" and "computer" should be counted (valid words)
	// "and", "innovation", "drive", "software", "development" should be filtered out
	expected := map[string]int{
		"technology": 1,
		"computer":   1,
	}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}

// TestProcessText_EmptyText tests processing empty text
func TestProcessText_EmptyText(t *testing.T) {
	validWords := []string{"technology"}
	mockWordBank := NewMockWordBank(validWords)
	processor := New(mockWordBank, false)

	result := processor.ProcessText("")

	expected := map[string]int{}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Expected empty map, got %v", result)
	}
}

// TestProcessText_NoValidWords tests text with no valid words
func TestProcessText_NoValidWords(t *testing.T) {
	validWords := []string{"technology", "computer"}
	mockWordBank := NewMockWordBank(validWords)
	processor := New(mockWordBank, false)

	text := "hello world this is a test"

	result := processor.ProcessText(text)

	expected := map[string]int{}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Expected empty map, got %v", result)
	}
}

// TestProcessText_SpecialCharacters tests handling of special characters
func TestProcessText_SpecialCharacters(t *testing.T) {
	validWords := []string{"technology", "innovation", "computer"}
	mockWordBank := NewMockWordBank(validWords)
	processor := New(mockWordBank, false)

	text := "Technology! Innovation? Computer... technology-innovation, computer's innovation."

	result := processor.ProcessText(text)

	// Regex should extract only alphabetic parts
	expected := map[string]int{
		"technology": 2,
		"innovation": 3,
		"computer":   2,
	}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}

// TestProcessText_Numbers tests that numbers are ignored
func TestProcessText_Numbers(t *testing.T) {
	validWords := []string{"technology", "version"}
	mockWordBank := NewMockWordBank(validWords)
	processor := New(mockWordBank, false)

	text := "Technology 2.0 version 123 technology version456"

	result := processor.ProcessText(text)

	// Numbers should be ignored by regex
	expected := map[string]int{
		"technology": 2,
		"version":    2,
	}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}

// TestProcessText_RepeatedWords tests counting repeated words correctly
func TestProcessText_RepeatedWords(t *testing.T) {
	validWords := []string{"test"}
	mockWordBank := NewMockWordBank(validWords)
	processor := New(mockWordBank, false)

	text := "test test test test test"

	result := processor.ProcessText(text)

	expected := map[string]int{
		"test": 5,
	}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}

// TestProcessText_LargeText tests processing larger text
func TestProcessText_LargeText(t *testing.T) {
	validWords := []string{"technology", "innovation", "computer", "software", "development"}
	mockWordBank := NewMockWordBank(validWords)
	processor := New(mockWordBank, false)

	// Simulate larger article text
	text := `
	Technology innovation drives modern computer software development.
	Software development requires innovative technology solutions.
	Computer technology enables software innovation and development.
	Innovation in technology transforms software development practices.
	Development of computer software relies on technology innovation.
	`

	result := processor.ProcessText(text)

	expected := map[string]int{
		"technology":  5,
		"innovation":  4,
		"computer":    3, // appears in: "computer software", "Computer technology", "computer software"
		"software":    5,
		"development": 5,
	}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}
