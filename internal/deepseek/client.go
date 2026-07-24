package deepseek

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// Client calls the DeepSeek chat-completions API.
type Client struct {
	cfg        Config
	httpClient *http.Client
}

func NewClient(cfg Config) *Client {
	return &Client{
		cfg:        cfg,
		httpClient: &http.Client{Timeout: cfg.Timeout},
	}
}

// Message is one turn in a chat-completion request, matching the
// OpenAI-compatible {role, content} shape DeepSeek's API expects.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatRequest is the input to Client.Chat. Model may be left empty to use
// Config.DefaultModel.
type ChatRequest struct {
	Model       string
	Messages    []Message
	Temperature *float64
}

// ChatResult is the assistant's reply plus the fields callers commonly need
// alongside it. ReasoningContent is only populated by reasoning models
// (e.g. deepseek-reasoner); it's empty for plain chat models.
type ChatResult struct {
	Content          string
	ReasoningContent string
	Model            string
	FinishReason     string
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

type apiRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Stream      bool      `json:"stream"`
	Temperature *float64  `json:"temperature,omitempty"`
}

type apiResponse struct {
	Model   string `json:"model"`
	Choices []struct {
		FinishReason string `json:"finish_reason"`
		Message      struct {
			Content          string `json:"content"`
			ReasoningContent string `json:"reasoning_content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
}

// Chat sends a chat-completion request and returns the first choice.
// Streaming is intentionally not supported -- an MCP tool call returns one
// result, so there's no consumer for partial chunks here.
func (c *Client) Chat(ctx context.Context, req ChatRequest) (*ChatResult, error) {
	if c.cfg.APIKey == "" {
		return nil, fmt.Errorf("DEEPSEEK_API_KEY is not set")
	}
	if len(req.Messages) == 0 {
		return nil, fmt.Errorf("at least one message is required")
	}

	model := req.Model
	if model == "" {
		model = c.cfg.DefaultModel
	}

	body, err := json.Marshal(apiRequest{
		Model:       model,
		Messages:    req.Messages,
		Stream:      false,
		Temperature: req.Temperature,
	})
	if err != nil {
		return nil, fmt.Errorf("encode request: %w", err)
	}

	endpoint, err := url.JoinPath(c.cfg.BaseURL, "chat/completions")
	if err != nil {
		return nil, fmt.Errorf("build endpoint: %w", err)
	}
	if !strings.HasPrefix(endpoint, "https://") {
		return nil, fmt.Errorf("refusing non-https DEEPSEEK_BASE_URL %q", c.cfg.BaseURL)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.cfg.APIKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, c.cfg.MaxBodyBytes))
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var parsed apiResponse
	if jsonErr := json.Unmarshal(respBody, &parsed); jsonErr != nil {
		return nil, fmt.Errorf("HTTP %d: unparseable response: %s", resp.StatusCode, truncate(respBody, 500))
	}

	if resp.StatusCode != http.StatusOK {
		if parsed.Error != nil {
			return nil, fmt.Errorf("HTTP %d: %s (%s)", resp.StatusCode, parsed.Error.Message, parsed.Error.Type)
		}
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, truncate(respBody, 500))
	}

	if len(parsed.Choices) == 0 {
		return nil, fmt.Errorf("response had no choices")
	}

	choice := parsed.Choices[0]
	return &ChatResult{
		Content:          choice.Message.Content,
		ReasoningContent: choice.Message.ReasoningContent,
		Model:            parsed.Model,
		FinishReason:     choice.FinishReason,
		PromptTokens:     parsed.Usage.PromptTokens,
		CompletionTokens: parsed.Usage.CompletionTokens,
		TotalTokens:      parsed.Usage.TotalTokens,
	}, nil
}

func truncate(b []byte, n int) string {
	s := string(b)
	if len(s) <= n {
		return s
	}
	return s[:n] + "...[truncated]"
}
