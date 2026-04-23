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

// fakeToolCallResponse builds an OpenAI response that includes a tool call.
func fakeToolCallResponse(toolName, toolID, argsJSON string) string {
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
					"tool_calls": []map[string]interface{}{
						{
							"id":   toolID,
							"type": "function",
							"function": map[string]interface{}{
								"name":      toolName,
								"arguments": argsJSON,
							},
						},
					},
				},
				"finish_reason": "tool_calls",
			},
		},
	}
	b, _ := json.Marshal(resp)
	return string(b)
}

func TestClient_ChatCompletion_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/chat/completions" {
			t.Errorf("expected /chat/completions, got %s", r.URL.Path)
		}

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

	client := New("test-key", server.URL, "test-model", 5)

	messages := []domain.ChatMessage{
		{Role: "system", Content: "You are helpful."},
		{Role: "user", Content: "Hi there!"},
	}

	result, err := client.ChatCompletion(context.Background(), messages, nil, nil)
	if err != nil {
		t.Fatalf("ChatCompletion failed: %v", err)
	}
	if result.Content != "Hello! I'm an AI assistant." {
		t.Errorf("result.Content = %q, want %q", result.Content, "Hello! I'm an AI assistant.")
	}
	if result.MemoryWritten {
		t.Error("expected MemoryWritten=false for no-tool call")
	}
}

func TestClient_ChatCompletion_EmptyMessages(t *testing.T) {
	client := New("test-key", "http://localhost", "test-model", 5)

	_, err := client.ChatCompletion(context.Background(), nil, nil, nil)
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

	client := New("test-key", server.URL, "test-model", 5)

	messages := []domain.ChatMessage{
		{Role: "user", Content: "Hello"},
	}

	_, err := client.ChatCompletion(context.Background(), messages, nil, nil)
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

	client := New("test-key", server.URL, "test-model", 5)

	messages := []domain.ChatMessage{
		{Role: "user", Content: "Hello"},
	}

	_, err := client.ChatCompletion(context.Background(), messages, nil, nil)
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

	client := New("test-key", server.URL, "test-model", 5)

	messages := []domain.ChatMessage{
		{Role: "user", Content: "Hello"},
	}

	_, err := client.ChatCompletion(context.Background(), messages, nil, nil)
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

	client := New("test-key", server.URL, "test-model", 5)

	messages := []domain.ChatMessage{
		{Role: "user", Name: "Mario Rossi", Content: "[Mario Rossi]: Tell me a joke"},
	}

	_, err := client.ChatCompletion(context.Background(), messages, nil, nil)
	if err != nil {
		t.Fatalf("ChatCompletion failed: %v", err)
	}

	if len(receivedMessages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(receivedMessages))
	}

	m, _ := receivedMessages[0].(map[string]interface{})
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

	client := New("test-key", server.URL, "test-model", 5)

	messages := []domain.ChatMessage{
		{Role: "user", Content: "Hello"},
	}

	_, err := client.ChatCompletion(context.Background(), messages, nil, nil)
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

	client := New("test-key", server.URL, "test-model", 5)

	messages := []domain.ChatMessage{
		{Role: "system", Content: "Be helpful"},
		{Role: "user", Content: "Question"},
		{Role: "assistant", Content: "Answer"},
		{Role: "user", Content: "Follow-up"},
	}

	_, err := client.ChatCompletion(context.Background(), messages, nil, nil)
	if err != nil {
		t.Fatalf("ChatCompletion failed: %v", err)
	}

	if len(receivedMessages) != 4 {
		t.Fatalf("expected 4 messages sent to API, got %d", len(receivedMessages))
	}

	expectedRoles := []string{"system", "user", "assistant", "user"}
	for i, msg := range receivedMessages {
		m, _ := msg.(map[string]interface{})
		role, _ := m["role"].(string)
		if role != expectedRoles[i] {
			t.Errorf("message[%d] role = %q, want %q", i, role, expectedRoles[i])
		}
	}
}

// fakeHandler is a test AIToolHandler that records calls and returns preset responses.
type fakeHandler struct {
	calls   []toolCall
	results map[string]string
}

type toolCall struct {
	name string
	args string
}

func (h *fakeHandler) Handle(_ context.Context, name string, args json.RawMessage) (string, error) {
	h.calls = append(h.calls, toolCall{name: name, args: string(args)})
	if res, ok := h.results[name]; ok {
		return res, nil
	}
	return "ok", nil
}

func TestClient_ChatCompletion_ToolLoop(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		if callCount == 1 {
			// First call: return a tool call
			_, _ = w.Write([]byte(fakeToolCallResponse("read_memory", "call-1", `{}`)))
		} else {
			// Second call: return final text
			_, _ = w.Write([]byte(fakeCompletionResponse("Memory is empty.")))
		}
	}))
	defer server.Close()

	handler := &fakeHandler{results: map[string]string{"read_memory": "(memory is empty)"}}
	client := New("test-key", server.URL, "test-model", 5)

	messages := []domain.ChatMessage{{Role: "user", Content: "What do you remember?"}}
	tools := []domain.AITool{{Name: "read_memory", Description: "reads memory", Parameters: json.RawMessage(`{}`)}}

	result, err := client.ChatCompletion(context.Background(), messages, tools, handler)
	if err != nil {
		t.Fatalf("ChatCompletion failed: %v", err)
	}
	if result.Content != "Memory is empty." {
		t.Errorf("result.Content = %q, want %q", result.Content, "Memory is empty.")
	}
	if callCount != 2 {
		t.Errorf("expected 2 API calls (tool call + final), got %d", callCount)
	}
	if len(handler.calls) != 1 || handler.calls[0].name != "read_memory" {
		t.Errorf("expected handler called once with read_memory, got %+v", handler.calls)
	}
}

func TestClient_ChatCompletion_ToolLoop_MemoryWritten(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		if callCount == 1 {
			_, _ = w.Write([]byte(fakeToolCallResponse("append_memory", "call-1", `{"text":"likes espresso"}`)))
		} else {
			_, _ = w.Write([]byte(fakeCompletionResponse("Got it, I'll remember that.")))
		}
	}))
	defer server.Close()

	handler := &fakeHandler{results: map[string]string{"append_memory": "appended"}}
	client := New("test-key", server.URL, "test-model", 5)

	messages := []domain.ChatMessage{{Role: "user", Content: "I love espresso."}}
	tools := domain.BuildMemoryTools()

	result, err := client.ChatCompletion(context.Background(), messages, tools, handler)
	if err != nil {
		t.Fatalf("ChatCompletion failed: %v", err)
	}
	if !result.MemoryWritten {
		t.Error("expected MemoryWritten=true after append_memory call")
	}
}
