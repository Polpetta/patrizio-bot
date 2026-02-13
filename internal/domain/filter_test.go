package domain

import (
	"testing"
)

func TestNormalizeMessage(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Latin with punctuation",
			input:    "I love my dog!",
			expected: "i love my dog ",
		},
		{
			name:     "Latin with embedded punctuation",
			input:    "dog, cat, and fish",
			expected: "dog  cat  and fish",
		},
		{
			name:     "No special characters",
			input:    "hello world",
			expected: "hello world",
		},
		{
			name:     "Mixed scripts (Latin + CJK)",
			input:    "Hello 犬!",
			expected: "hello 犬 ",
		},
		{
			name:     "CJK only",
			input:    "私は犬が好き",
			expected: "私は犬が好き",
		},
		{
			name:     "Cyrillic",
			input:    "Привет, мир!",
			expected: "привет  мир ",
		},
		{
			name:     "Arabic",
			input:    "مرحبا!",
			expected: "مرحبا ",
		},
		{
			name:     "Uppercase to lowercase",
			input:    "HELLO WORLD",
			expected: "hello world",
		},
		{
			name:     "Multiple consecutive spaces preserved",
			input:    "hello   world",
			expected: "hello   world",
		},
		{
			name:     "Special characters become spaces",
			input:    "c++",
			expected: "c ",
		},
		{
			name:     "Emoji removed",
			input:    "hello 😂 world",
			expected: "hello   world",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeMessage(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizeMessage(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestValidateTrigger(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expectErr bool
	}{
		{
			name:      "Valid single word",
			input:     "dog",
			expectErr: false,
		},
		{
			name:      "Valid multi-word",
			input:     "i love dogs",
			expectErr: false,
		},
		{
			name:      "Valid CJK",
			input:     "犬",
			expectErr: false,
		},
		{
			name:      "Valid Cyrillic",
			input:     "Привет",
			expectErr: false,
		},
		{
			name:      "Valid Arabic",
			input:     "مرحبا",
			expectErr: false,
		},
		{
			name:      "Valid with digits",
			input:     "test123",
			expectErr: false,
		},
		{
			name:      "Invalid with punctuation",
			input:     "c++",
			expectErr: true,
		},
		{
			name:      "Invalid with exclamation",
			input:     "hello!",
			expectErr: true,
		},
		{
			name:      "Invalid with comma",
			input:     "hello,",
			expectErr: true,
		},
		{
			name:      "Invalid with emoji",
			input:     "hello😂",
			expectErr: true,
		},
		{
			name:      "Invalid with special characters",
			input:     "test@example",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTrigger(tt.input)
			if tt.expectErr && err == nil {
				t.Errorf("ValidateTrigger(%q) expected error, got nil", tt.input)
			}
			if !tt.expectErr && err != nil {
				t.Errorf("ValidateTrigger(%q) expected no error, got %v", tt.input, err)
			}
		})
	}
}

func TestNormalizeTrigger(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Lowercase Latin",
			input:    "hello",
			expected: "hello",
		},
		{
			name:     "Uppercase Latin to lowercase",
			input:    "HELLO",
			expected: "hello",
		},
		{
			name:     "Mixed case to lowercase",
			input:    "Hello World",
			expected: "hello world",
		},
		{
			name:     "Cyrillic uppercase to lowercase",
			input:    "Привет",
			expected: "привет",
		},
		{
			name:     "CJK unchanged (no case)",
			input:    "犬",
			expected: "犬",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeTrigger(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizeTrigger(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
