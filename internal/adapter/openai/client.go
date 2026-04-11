// Package openai provides an AI client adapter using the OpenAI chat completions API.
package openai

import (
	"context"
	"errors"
	"fmt"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/packages/param"
	"github.com/polpetta/patrizio/internal/domain"
)

// Client implements the domain.AIClient interface using the OpenAI SDK.
type Client struct {
	client *openai.Client
	model  string
}

// New creates a new OpenAI client adapter.
func New(apiKey, baseURL, model string) *Client {
	opts := []option.RequestOption{
		option.WithAPIKey(apiKey),
	}
	if baseURL != "" {
		opts = append(opts, option.WithBaseURL(baseURL))
	}

	client := openai.NewClient(opts...)

	return &Client{
		client: &client,
		model:  model,
	}
}

// ChatCompletion sends a chat completion request and returns the assistant's response text.
func (c *Client) ChatCompletion(ctx context.Context, messages []domain.ChatMessage) (string, error) {
	if len(messages) == 0 {
		return "", errors.New("no messages provided")
	}

	params := openai.ChatCompletionNewParams{
		Model:    c.model,
		Messages: toSDKMessages(messages),
	}

	completion, err := c.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return "", fmt.Errorf("chat completion request failed: %w", err)
	}

	if len(completion.Choices) == 0 {
		return "", errors.New("no response choices returned from API")
	}

	content := completion.Choices[0].Message.Content
	if content == "" {
		return "", errors.New("empty response content from API")
	}

	return content, nil
}

// toSDKMessages converts domain ChatMessage slice to the OpenAI SDK message format.
func toSDKMessages(messages []domain.ChatMessage) []openai.ChatCompletionMessageParamUnion {
	sdkMsgs := make([]openai.ChatCompletionMessageParamUnion, 0, len(messages))

	for _, msg := range messages {
		switch msg.Role {
		case "system":
			sdkMsgs = append(sdkMsgs, openai.SystemMessage(msg.Content))
		case "user":
			if msg.Name != "" {
				sdkMsgs = append(sdkMsgs, openai.ChatCompletionMessageParamUnion{
					OfUser: &openai.ChatCompletionUserMessageParam{
						Content: openai.ChatCompletionUserMessageParamContentUnion{
							OfString: param.NewOpt(msg.Content),
						},
						Name: param.NewOpt(msg.Name),
					},
				})
			} else {
				sdkMsgs = append(sdkMsgs, openai.UserMessage(msg.Content))
			}
		case "assistant":
			sdkMsgs = append(sdkMsgs, openai.AssistantMessage(msg.Content))
		}
	}

	return sdkMsgs
}
