package openai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/polpetta/patrizio/internal/domain"
)

// fakeCompletionResponse builds a minimal OpenAI chat completion JSON response.
func fakeCompletionResponse(content string) string {
	resp := map[string]interface{}{
		"id":      "chatcmpl-test",
		"object":  "chat.completion",
		"created": 1234567890,
		"model":   "gpt-4o-mini",
		"choices": []map[string]interface{}{
			{
				"index": 0,
				"message": map[string]interface{}{
					"role":    "assistant",
					"content": content,
				},
				"finish_reason": "stop",
			},
		},
	}
	b, _ := json.Marshal(resp)
	return string(b)
}

func TestClient_ChatCompletion_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request basics
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/chat/completions" {
			t.Errorf("expected /chat/completions, got %s", r.URL.Path)
		}

		// Verify request body
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}

		if body["model"] != "test-model" {
			t.Errorf("expected model test-model, got %v", body["model"])
		}

		messages, ok := body["messages"].([]interface{})
		if !ok {
			t.Fatalf("expected messages array, got %T", body["messages"])
		}
		if len(messages) != 2 {
			t.Errorf("expected 2 messages, got %d", len(messages))
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fakeCompletionResponse("Hello! I'm an AI assistant.")))
	}))
	defer server.Close()

	client := New("test-key", server.URL, "test-model")

	messages := []domain.ChatMessage{
		{Role: "system", Content: "You are helpful."},
		{Role: "user", Content: "Hi there!"},
	}

	result, err := client.ChatCompletion(context.Background(), messages)
	if err != nil {
		t.Fatalf("ChatCompletion failed: %v", err)
	}

	if result != "Hello! I'm an AI assistant." {
		t.Errorf("result = %q, want %q", result, "Hello! I'm an AI assistant.")
	}
}

func TestClient_ChatCompletion_EmptyMessages(t *testing.T) {
	client := New("test-key", "http://localhost", "test-model")

	_, err := client.ChatCompletion(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for empty messages, got nil")
	}
}

func TestClient_ChatCompletion_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error": {"message": "Internal server error", "type": "server_error"}}`))
	}))
	defer server.Close()

	client := New("test-key", server.URL, "test-model")

	messages := []domain.ChatMessage{
		{Role: "user", Content: "Hello"},
	}

	_, err := client.ChatCompletion(context.Background(), messages)
	if err == nil {
		t.Fatal("expected error for API error response, got nil")
	}
}

func TestClient_ChatCompletion_EmptyChoices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		resp := map[string]interface{}{
			"id":      "chatcmpl-test",
			"object":  "chat.completion",
			"created": 1234567890,
			"model":   "gpt-4o-mini",
			"choices": []map[string]interface{}{},
		}
		b, _ := json.Marshal(resp)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(b)
	}))
	defer server.Close()

	client := New("test-key", server.URL, "test-model")

	messages := []domain.ChatMessage{
		{Role: "user", Content: "Hello"},
	}

	_, err := client.ChatCompletion(context.Background(), messages)
	if err == nil {
		t.Fatal("expected error for empty choices, got nil")
	}
}

func TestClient_ChatCompletion_EmptyContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		resp := map[string]interface{}{
			"id":      "chatcmpl-test",
			"object":  "chat.completion",
			"created": 1234567890,
			"model":   "gpt-4o-mini",
			"choices": []map[string]interface{}{
				{
					"index": 0,
					"message": map[string]interface{}{
						"role":    "assistant",
						"content": "",
					},
					"finish_reason": "stop",
				},
			},
		}
		b, _ := json.Marshal(resp)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(b)
	}))
	defer server.Close()

	client := New("test-key", server.URL, "test-model")

	messages := []domain.ChatMessage{
		{Role: "user", Content: "Hello"},
	}

	_, err := client.ChatCompletion(context.Background(), messages)
	if err == nil {
		t.Fatal("expected error for empty content, got nil")
	}
}

func TestClient_ChatCompletion_UserMessageWithName(t *testing.T) {
	var receivedMessages []interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		receivedMessages, _ = body["messages"].([]interface{})

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fakeCompletionResponse("OK")))
	}))
	defer server.Close()

	client := New("test-key", server.URL, "test-model")

	messages := []domain.ChatMessage{
		{Role: "user", Name: "Mario Rossi", Content: "[Mario Rossi]: Tell me a joke"},
	}

	_, err := client.ChatCompletion(context.Background(), messages)
	if err != nil {
		t.Fatalf("ChatCompletion failed: %v", err)
	}

	if len(receivedMessages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(receivedMessages))
	}

	m, _ := receivedMessages[0].(map[string]interface{})
	// name field must not be sent to avoid OpenAI API format restrictions
	if _, ok := m["name"]; ok {
		t.Errorf("expected no 'name' field in request, but it was present")
	}
	content, _ := m["content"].(string)
	if content != "[Mario Rossi]: Tell me a joke" {
		t.Errorf("message content = %q, want %q", content, "[Mario Rossi]: Tell me a joke")
	}
}

func TestClient_ChatCompletion_UserMessageWithoutName(t *testing.T) {
	var receivedMessages []interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		receivedMessages, _ = body["messages"].([]interface{})

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fakeCompletionResponse("OK")))
	}))
	defer server.Close()

	client := New("test-key", server.URL, "test-model")

	messages := []domain.ChatMessage{
		{Role: "user", Content: "Hello"},
	}

	_, err := client.ChatCompletion(context.Background(), messages)
	if err != nil {
		t.Fatalf("ChatCompletion failed: %v", err)
	}

	if len(receivedMessages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(receivedMessages))
	}

	m, _ := receivedMessages[0].(map[string]interface{})
	if _, hasName := m["name"]; hasName {
		t.Error("expected no 'name' field for user message without Name set")
	}
}

func TestClient_ChatCompletion_MessageRoles(t *testing.T) {
	var receivedMessages []interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		receivedMessages, _ = body["messages"].([]interface{})

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(fakeCompletionResponse("OK")))
	}))
	defer server.Close()

	client := New("test-key", server.URL, "test-model")

	messages := []domain.ChatMessage{
		{Role: "system", Content: "Be helpful"},
		{Role: "user", Content: "Question"},
		{Role: "assistant", Content: "Answer"},
		{Role: "user", Content: "Follow-up"},
	}

	_, err := client.ChatCompletion(context.Background(), messages)
	if err != nil {
		t.Fatalf("ChatCompletion failed: %v", err)
	}

	if len(receivedMessages) != 4 {
		t.Fatalf("expected 4 messages sent to API, got %d", len(receivedMessages))
	}

	// Verify roles
	expectedRoles := []string{"system", "user", "assistant", "user"}
	for i, msg := range receivedMessages {
		m, _ := msg.(map[string]interface{})
		role, _ := m["role"].(string)
		if role != expectedRoles[i] {
			t.Errorf("message[%d] role = %q, want %q", i, role, expectedRoles[i])
		}
	}
}
