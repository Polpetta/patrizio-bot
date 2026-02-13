package domain

import (
	"errors"
	"regexp"
	"strings"
)

var (
	// Matches Unicode letters, digits, and spaces only
	triggerValidationRegex = regexp.MustCompile(`^[\p{L}\p{N}\s]+$`)
	// Matches anything that's NOT Unicode letters, digits, or whitespace
	messageNormalizationRegex = regexp.MustCompile(`[^\p{L}\p{N}\s]+`)
)

// ErrInvalidTrigger indicates a trigger contains disallowed characters
var ErrInvalidTrigger = errors.New("trigger can only contain Unicode letters, digits, and spaces")

// NormalizeMessage normalizes incoming message text by lowercasing and
// replacing all non-Unicode-alphanumeric characters with spaces
func NormalizeMessage(msg string) string {
	// Replace all non-letter/digit/whitespace with space
	normalized := messageNormalizationRegex.ReplaceAllString(msg, " ")
	// Lowercase using Unicode-aware lowercasing
	return strings.ToLower(normalized)
}

// ValidateTrigger checks if trigger text contains only Unicode letters, digits, and spaces
func ValidateTrigger(text string) error {
	if !triggerValidationRegex.MatchString(text) {
		return ErrInvalidTrigger
	}
	return nil
}

// NormalizeTrigger lowercases trigger text for storage
func NormalizeTrigger(text string) string {
	return strings.ToLower(text)
}
