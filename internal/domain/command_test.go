package domain

import (
	"testing"
)

func TestParseFilterCommand_SingleWord(t *testing.T) {
	cmd, err := ParseFilterCommand("/filter hello Hi there!")
	if err != nil {
		t.Fatalf("ParseFilterCommand failed: %v", err)
	}

	if len(cmd.Triggers) != 1 {
		t.Errorf("Expected 1 trigger, got %d", len(cmd.Triggers))
	}
	if cmd.Triggers[0] != "hello" {
		t.Errorf("Trigger = %q, want %q", cmd.Triggers[0], "hello")
	}
	if cmd.ResponseType != ResponseTypeText {
		t.Errorf("ResponseType = %q, want %q", cmd.ResponseType, ResponseTypeText)
	}
	if cmd.ResponseText != "Hi there!" {
		t.Errorf("ResponseText = %q, want %q", cmd.ResponseText, "Hi there!")
	}
}

func TestParseFilterCommand_QuotedPhrase(t *testing.T) {
	cmd, err := ParseFilterCommand("/filter \"I love dogs\" Dogs are great!")
	if err != nil {
		t.Fatalf("ParseFilterCommand failed: %v", err)
	}

	if len(cmd.Triggers) != 1 {
		t.Errorf("Expected 1 trigger, got %d", len(cmd.Triggers))
	}
	if cmd.Triggers[0] != "I love dogs" {
		t.Errorf("Trigger = %q, want %q", cmd.Triggers[0], "I love dogs")
	}
	if cmd.ResponseText != "Dogs are great!" {
		t.Errorf("ResponseText = %q, want %q", cmd.ResponseText, "Dogs are great!")
	}
}

func TestParseFilterCommand_MultipleTriggers(t *testing.T) {
	cmd, err := ParseFilterCommand("/filter (hi, hello, hey) Greetings!")
	if err != nil {
		t.Fatalf("ParseFilterCommand failed: %v", err)
	}

	if len(cmd.Triggers) != 3 {
		t.Fatalf("Expected 3 triggers, got %d", len(cmd.Triggers))
	}
	if cmd.Triggers[0] != "hi" || cmd.Triggers[1] != "hello" || cmd.Triggers[2] != "hey" {
		t.Errorf("Triggers = %v, want [hi, hello, hey]", cmd.Triggers)
	}
	if cmd.ResponseText != "Greetings!" {
		t.Errorf("ResponseText = %q, want %q", cmd.ResponseText, "Greetings!")
	}
}

func TestParseFilterCommand_MultipleTriggersWithQuotes(t *testing.T) {
	cmd, err := ParseFilterCommand("/filter (hi, \"hi there\", hello) Greetings!")
	if err != nil {
		t.Fatalf("ParseFilterCommand failed: %v", err)
	}

	if len(cmd.Triggers) != 3 {
		t.Fatalf("Expected 3 triggers, got %d", len(cmd.Triggers))
	}
	if cmd.Triggers[0] != "hi" {
		t.Errorf("Triggers[0] = %q, want %q", cmd.Triggers[0], "hi")
	}
	if cmd.Triggers[1] != "hi there" {
		t.Errorf("Triggers[1] = %q, want %q", cmd.Triggers[1], "hi there")
	}
	if cmd.Triggers[2] != "hello" {
		t.Errorf("Triggers[2] = %q, want %q", cmd.Triggers[2], "hello")
	}
}

func TestParseFilterCommand_Reaction(t *testing.T) {
	cmd, err := ParseFilterCommand("/filter lol react:😂")
	if err != nil {
		t.Fatalf("ParseFilterCommand failed: %v", err)
	}

	if len(cmd.Triggers) != 1 {
		t.Errorf("Expected 1 trigger, got %d", len(cmd.Triggers))
	}
	if cmd.Triggers[0] != "lol" {
		t.Errorf("Trigger = %q, want %q", cmd.Triggers[0], "lol")
	}
	if cmd.ResponseType != ResponseTypeReaction {
		t.Errorf("ResponseType = %q, want %q", cmd.ResponseType, ResponseTypeReaction)
	}
	if cmd.Reaction != "😂" {
		t.Errorf("Reaction = %q, want %q", cmd.Reaction, "😂")
	}
}

func TestParseFilterCommand_Errors(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"Empty", "/filter"},
		{"No response", "/filter hello"},
		{"Unclosed quote", "/filter \"hello"},
		{"Unclosed parenthesis", "/filter (hello, hi"},
		{"Empty reaction", "/filter lol react:"},
		{"Empty trigger in list", "/filter (hi, , hey) response"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseFilterCommand(tt.input)
			if err == nil {
				t.Errorf("Expected error for %q, got nil", tt.input)
			}
		})
	}
}

func TestParseStopCommand(t *testing.T) {
	cmd, err := ParseStopCommand("/stop hello")
	if err != nil {
		t.Fatalf("ParseStopCommand failed: %v", err)
	}

	if cmd.Trigger != "hello" {
		t.Errorf("Trigger = %q, want %q", cmd.Trigger, "hello")
	}
}

func TestParseStopCommand_Quoted(t *testing.T) {
	cmd, err := ParseStopCommand("/stop \"i love dogs\"")
	if err != nil {
		t.Fatalf("ParseStopCommand failed: %v", err)
	}

	if cmd.Trigger != "i love dogs" {
		t.Errorf("Trigger = %q, want %q", cmd.Trigger, "i love dogs")
	}
}

func TestParseStopCommand_Errors(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"Empty", "/stop"},
		{"Unclosed quote", "/stop \"hello"},
		{"Extra text", "/stop hello world"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseStopCommand(tt.input)
			if err == nil {
				t.Errorf("Expected error for %q, got nil", tt.input)
			}
		})
	}
}

func TestGetCommandType(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"/filter hello Hi!", "/filter"},
		{"/stop hello", "/stop"},
		{"/stopall", "/stopall"},
		{"/filters", "/filters"},
		{"hello /filter", ""},
		{"not a command", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := GetCommandType(tt.input)
			if result != tt.expected {
				t.Errorf("GetCommandType(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
