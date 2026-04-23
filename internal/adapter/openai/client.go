// Package openai provides an AI client adapter using the OpenAI chat completions API.
package openai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/shared"
	"github.com/polpetta/patrizio/internal/domain"
)

// Client implements the domain.AIClient interface using the OpenAI SDK.
type Client struct {
	client           *openai.Client
	model            string
	maxToolIteration int
}

// New creates a new OpenAI client adapter.
func New(apiKey, baseURL, model string, maxToolIterations int) *Client {
	opts := []option.RequestOption{
		option.WithAPIKey(apiKey),
	}
	if baseURL != "" {
		opts = append(opts, option.WithBaseURL(baseURL))
	}

	client := openai.NewClient(opts...)

	return &Client{
		client:           &client,
		model:            model,
		maxToolIteration: maxToolIterations,
	}
}

// ChatCompletion runs a single-shot or multi-turn tool-calling completion, capped at maxToolIterations.
func (c *Client) ChatCompletion(ctx context.Context, messages []domain.ChatMessage, tools []domain.AITool, handler domain.AIToolHandler) (domain.ChatResponse, error) {
	if len(messages) == 0 {
		return domain.ChatResponse{}, errors.New("no messages provided")
	}

	sdkTools := toSDKTools(tools)
	sdkMsgs := toSDKMessages(messages)

	var memoryWritten bool

	for i := 0; ; i++ {
		params := openai.ChatCompletionNewParams{
			Model:    c.model,
			Messages: sdkMsgs,
		}
		if len(sdkTools) > 0 {
			params.Tools = sdkTools
		}

		completion, err := c.client.Chat.Completions.New(ctx, params)
		if err != nil {
			return domain.ChatResponse{}, fmt.Errorf("chat completion request failed: %w", err)
		}

		if len(completion.Choices) == 0 {
			return domain.ChatResponse{}, errors.New("no response choices returned from API")
		}

		choice := completion.Choices[0]

		if len(choice.Message.ToolCalls) == 0 {
			content := choice.Message.Content
			if content == "" {
				return domain.ChatResponse{}, errors.New("empty response content from API")
			}
			return domain.ChatResponse{Content: content, MemoryWritten: memoryWritten}, nil
		}

		if i >= c.maxToolIteration {
			return domain.ChatResponse{Content: choice.Message.Content, MemoryWritten: memoryWritten},
				fmt.Errorf("tool-calling loop exceeded %d iterations", c.maxToolIteration)
		}

		sdkMsgs = append(sdkMsgs, choice.Message.ToParam())

		for _, tc := range choice.Message.ToolCalls {
			result, toolErr := handler.Handle(ctx, tc.Function.Name, json.RawMessage(tc.Function.Arguments))
			if toolErr != nil {
				result = fmt.Sprintf("error: %v", toolErr)
			} else if tc.Function.Name == "append_memory" || tc.Function.Name == "update_memory" {
				memoryWritten = true
			}
			sdkMsgs = append(sdkMsgs, openai.ToolMessage(result, tc.ID))
		}
	}
}

func toSDKMessages(messages []domain.ChatMessage) []openai.ChatCompletionMessageParamUnion {
	sdkMsgs := make([]openai.ChatCompletionMessageParamUnion, 0, len(messages))

	for _, msg := range messages {
		switch msg.Role {
		case "system":
			sdkMsgs = append(sdkMsgs, openai.SystemMessage(msg.Content))
		case "user":
			sdkMsgs = append(sdkMsgs, openai.UserMessage(msg.Content))
		case "assistant":
			sdkMsgs = append(sdkMsgs, openai.AssistantMessage(msg.Content))
		}
	}

	return sdkMsgs
}

func toSDKTools(tools []domain.AITool) []openai.ChatCompletionToolUnionParam {
	if len(tools) == 0 {
		return nil
	}

	sdkTools := make([]openai.ChatCompletionToolUnionParam, 0, len(tools))
	for _, t := range tools {
		var params shared.FunctionParameters
		if err := json.Unmarshal(t.Parameters, &params); err != nil {
			continue
		}
		sdkTools = append(sdkTools, openai.ChatCompletionFunctionTool(shared.FunctionDefinitionParam{
			Name:        t.Name,
			Description: openai.String(t.Description),
			Parameters:  params,
		}))
	}
	return sdkTools
}
