package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client 定义 LLM 客户端接口
type Client interface {
	Chat(ctx context.Context, messages []Message, options ...Option) (string, error)
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OpenAIClient struct {
	endpoint   string
	apiKey     string
	httpClient *http.Client
	model      string
}

type Option func(*OpenAIClient)

func WithModel(model string) Option {
	return func(c *OpenAIClient) {
		c.model = model
	}
}

func NewOpenAIClient(endpoint, apiKey string, model string) *OpenAIClient {
	return &OpenAIClient{
		endpoint: endpoint,
		apiKey:   apiKey,
		model:    model,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

type chatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

type chatResponse struct {
	Choices []struct {
		Message Message `json:"message"`
	} `json:"choices"`
}

func (c *OpenAIClient) Chat(ctx context.Context, messages []Message, options ...Option) (string, error) {
	// Apply options if needed (e.g. override model per request)
	// For simplicity, we use the client's default model unless changed here, 
	// but the current structure applies options to the client instance which is not thread safe for per-request options.
	// For MVP, we'll just use the configured model.

	reqBody := chatRequest{
		Model:    c.model,
		Messages: messages,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// 直接使用配置的 endpoint，不再硬编码路径
	req, err := http.NewRequestWithContext(ctx, "POST", c.endpoint, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("llm request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("llm api error (status %d): %s", resp.StatusCode, string(body))
	}

	var chatResp chatResponse
	if err := json.Unmarshal(body, &chatResp); err != nil {
		return "", fmt.Errorf("failed to parse llm response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no choices returned from llm")
	}

	return chatResp.Choices[0].Message.Content, nil
}
