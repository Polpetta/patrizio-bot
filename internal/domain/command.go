// Package domain contains the core business logic and domain models for the Patrizio bot.
// It defines pure domain functions, port interfaces, and command parsing utilities.
package domain

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var (
	// ErrInvalidCommand indicates a malformed command
	ErrInvalidCommand = errors.New("invalid command syntax")
)

// Command types
const (
	CommandFilter  = "/filter"
	CommandStop    = "/stop"
	CommandStopAll = "/stopall"
	CommandFilters = "/filters"
)

// FilterCommand represents a parsed /filter command
type FilterCommand struct {
	Triggers     []string
	ResponseType string // "text", "media", "reaction"
	ResponseText string // for text responses
	Reaction     string // for reaction responses (the emoji)
}

// StopCommand represents a parsed /stop command
type StopCommand struct {
	Trigger string
}

// ParseFilterCommand parses a /filter command.
// Supported formats:
// - /filter word response text
// - /filter "multi word" response text
// - /filter (word1, word2, "multi word") response text
// - /filter word react:😂
func ParseFilterCommand(text string) (*FilterCommand, error) {
	// Remove /filter prefix
	text = strings.TrimPrefix(text, CommandFilter)
	text = strings.TrimSpace(text)

	if text == "" {
		return nil, fmt.Errorf("%w: no arguments provided", ErrInvalidCommand)
	}

	var triggers []string
	var remaining string

	// Check for parenthesized multi-trigger syntax: (trigger1, trigger2, ...)
	if strings.HasPrefix(text, "(") {
		closeIdx := strings.Index(text, ")")
		if closeIdx == -1 {
			return nil, fmt.Errorf("%w: unclosed parenthesis", ErrInvalidCommand)
		}

		triggerList := text[1:closeIdx]
		remaining = strings.TrimSpace(text[closeIdx+1:])

		// Parse comma-separated triggers (handle quoted phrases)
		parsedTriggers, err := parseCommaSeparatedTriggers(triggerList)
		if err != nil {
			return nil, err
		}
		triggers = parsedTriggers
	} else {
		// Single trigger (quoted or unquoted)
		trigger, rest, err := parseNextToken(text)
		if err != nil {
			return nil, err
		}
		triggers = []string{trigger}
		remaining = rest
	}

	if remaining == "" {
		return nil, fmt.Errorf("%w: no response provided", ErrInvalidCommand)
	}

	// Check if response is a reaction (react:emoji)
	if strings.HasPrefix(remaining, "react:") {
		reaction := strings.TrimPrefix(remaining, "react:")
		reaction = strings.TrimSpace(reaction)
		if reaction == "" {
			return nil, fmt.Errorf("%w: empty reaction", ErrInvalidCommand)
		}

		return &FilterCommand{
			Triggers:     triggers,
			ResponseType: ResponseTypeReaction,
			Reaction:     reaction,
		}, nil
	}

	// Otherwise it's a text response
	return &FilterCommand{
		Triggers:     triggers,
		ResponseType: ResponseTypeText,
		ResponseText: remaining,
	}, nil
}

// ParseStopCommand parses a /stop command.
// Formats:
// - /stop word
// - /stop "multi word"
func ParseStopCommand(text string) (*StopCommand, error) {
	text = strings.TrimPrefix(text, CommandStop)
	text = strings.TrimSpace(text)

	if text == "" {
		return nil, fmt.Errorf("%w: no trigger provided", ErrInvalidCommand)
	}

	trigger, remaining, err := parseNextToken(text)
	if err != nil {
		return nil, err
	}

	if remaining != "" {
		return nil, fmt.Errorf("%w: unexpected text after trigger", ErrInvalidCommand)
	}

	return &StopCommand{
		Trigger: trigger,
	}, nil
}

// parseNextToken extracts the next token (quoted or unquoted) and returns the rest
func parseNextToken(text string) (token string, remaining string, err error) {
	text = strings.TrimSpace(text)

	if text == "" {
		return "", "", fmt.Errorf("%w: expected token", ErrInvalidCommand)
	}

	// Handle quoted token
	if strings.HasPrefix(text, "\"") {
		closeIdx := strings.Index(text[1:], "\"")
		if closeIdx == -1 {
			return "", "", fmt.Errorf("%w: unclosed quote", ErrInvalidCommand)
		}
		token = text[1 : closeIdx+1]
		remaining = strings.TrimSpace(text[closeIdx+2:])
		return token, remaining, nil
	}

	// Handle unquoted token (up to next space)
	parts := strings.SplitN(text, " ", 2)
	token = parts[0]
	if len(parts) > 1 {
		remaining = parts[1]
	}
	return token, remaining, nil
}

// parseCommaSeparatedTriggers parses a comma-separated list of triggers (quoted or unquoted)
func parseCommaSeparatedTriggers(text string) ([]string, error) {
	var triggers []string
	text = strings.TrimSpace(text)

	for text != "" {
		// Extract next trigger
		var trigger string
		var afterQuoted bool

		if strings.HasPrefix(text, "\"") {
			// Quoted trigger
			closeIdx := strings.Index(text[1:], "\"")
			if closeIdx == -1 {
				return nil, fmt.Errorf("%w: unclosed quote in trigger list", ErrInvalidCommand)
			}
			trigger = text[1 : closeIdx+1]
			text = strings.TrimSpace(text[closeIdx+2:])
			afterQuoted = true
		} else {
			// Unquoted trigger (up to comma or end)
			commaIdx := strings.Index(text, ",")
			if commaIdx == -1 {
				// Last trigger
				trigger = strings.TrimSpace(text)
				text = ""
			} else {
				trigger = strings.TrimSpace(text[:commaIdx])
				text = strings.TrimSpace(text[commaIdx+1:])
			}
		}

		if trigger == "" {
			return nil, fmt.Errorf("%w: empty trigger in list", ErrInvalidCommand)
		}

		triggers = append(triggers, trigger)

		// For quoted triggers, skip the comma separator
		if afterQuoted && text != "" {
			if strings.HasPrefix(text, ",") {
				text = strings.TrimSpace(text[1:])
			}
		}
	}

	if len(triggers) == 0 {
		return nil, fmt.Errorf("%w: no triggers in list", ErrInvalidCommand)
	}

	return triggers, nil
}

// Helper to detect command type from message
var commandPattern = regexp.MustCompile(`^(/filter|/stop|/stopall|/filters)\b`)

// GetCommandType returns the command type if the message is a command, empty string otherwise
func GetCommandType(text string) string {
	matches := commandPattern.FindStringSubmatch(text)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}
