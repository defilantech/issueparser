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

type Client struct {
	httpClient *http.Client
	endpoint   string
	model      string
}

type ChatRequest struct {
	Model           string    `json:"model"`
	Messages        []Message `json:"messages"`
	MaxTokens       int       `json:"max_tokens,omitempty"`
	Temperature     float64   `json:"temperature,omitempty"`
	TopP            float64   `json:"top_p,omitempty"`
	RepeatPenalty   float64   `json:"repeat_penalty,omitempty"`
	Stop            []string  `json:"stop,omitempty"`
	PresencePenalty float64   `json:"presence_penalty,omitempty"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatResponse struct {
	ID      string   `json:"id"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

func NewClient(endpoint, model string) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 5 * time.Minute}, // LLM calls can be slow
		endpoint:   endpoint,
		model:      model,
	}
}

func (c *Client) Chat(ctx context.Context, messages []Message, maxTokens int) (*ChatResponse, error) {
	req := ChatRequest{
		Model:           c.model,
		Messages:        messages,
		MaxTokens:       maxTokens,
		Temperature:     0.7,  // Higher temperature to avoid repetition
		TopP:            0.9,  // Nucleus sampling
		RepeatPenalty:   1.15, // Penalize repetition (llama.cpp parameter)
		PresencePenalty: 0.1,  // Slight presence penalty
		Stop:            []string{"```\n\n", "\n\n\n\n"}, // Stop on repeated newlines
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/v1/chat/completions", c.endpoint)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("LLM API error %d: %s", resp.StatusCode, string(respBody))
	}

	var chatResp ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &chatResp, nil
}

func (c *Client) Complete(ctx context.Context, systemPrompt, userPrompt string, maxTokens int) (string, error) {
	messages := []Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}

	resp, err := c.Chat(ctx, messages, maxTokens)
	if err != nil {
		return "", err
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	return resp.Choices[0].Message.Content, nil
}

func (c *Client) HealthCheck(ctx context.Context) error {
	url := fmt.Sprintf("%s/health", c.endpoint)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("unhealthy status: %d", resp.StatusCode)
	}

	return nil
}
