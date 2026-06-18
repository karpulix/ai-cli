package llm

import (
	"context"
	"fmt"
	"strings"

	openai "github.com/sashabaranov/go-openai"
	"github.com/karpulix/ai-cli/internal/config"
	"github.com/karpulix/ai-cli/internal/prompt"
)

type Client struct {
	client         *openai.Client
	model          string
	promptTemplate string
}

func NewFromProfile(p config.Profile, promptTemplate string) (*Client, error) {
	model := p.Model
	if model == "" {
		model = openai.GPT4oMini
	}

	apiKey := p.APIKey
	if apiKey == "" {
		apiKey = "ollama"
	}

	cfg := openai.DefaultConfig(apiKey)
	if p.BaseURL != "" {
		cfg.BaseURL = strings.TrimRight(p.BaseURL, "/")
	}

	if strings.TrimSpace(promptTemplate) == "" {
		promptTemplate = prompt.Default()
	}

	return &Client{
		client:         openai.NewClientWithConfig(cfg),
		model:          model,
		promptTemplate: promptTemplate,
	}, nil
}

func New() (*Client, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	p, _, err := cfg.Active()
	if err != nil {
		return nil, err
	}
	return NewFromProfile(p, cfg.PromptTemplate)
}

func (c *Client) Complete(ctx context.Context, userPrompt string) (string, error) {
	systemPrompt := prompt.RenderNow(c.promptTemplate)

	resp, err := c.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: c.model,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: systemPrompt},
			{Role: openai.ChatMessageRoleUser, Content: userPrompt},
		},
	})
	if err != nil {
		return "", err
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("empty response from model")
	}

	return sanitize(resp.Choices[0].Message.Content), nil
}

func sanitize(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```") {
		lines := strings.Split(s, "\n")
		if len(lines) >= 2 {
			lines = lines[1:]
			if len(lines) > 0 && strings.HasPrefix(lines[len(lines)-1], "```") {
				lines = lines[:len(lines)-1]
			}
			s = strings.Join(lines, "\n")
		}
	}
	return strings.TrimSpace(s)
}
