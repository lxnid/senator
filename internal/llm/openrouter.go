package llm

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	openai "github.com/sashabaranov/go-openai"
)

const (
	// OpenRouterBaseURL is the base URL for OpenRouter API
	OpenRouterBaseURL = "https://openrouter.ai/api/v1"

	// DefaultModel is the default model to use
	DefaultModel = "google/gemini-2.0-flash-exp:free"
)

// Client wraps the OpenAI client for OpenRouter
type Client struct {
	client *openai.Client
	model  string
}

// ClientConfig holds configuration for the LLM client
type ClientConfig struct {
	APIKey  string
	Model   string
	BaseURL string // Optional: defaults to OpenRouter, set to "http://localhost:11434/v1" for Ollama
}

// NewClient creates a new LLM client (supports OpenRouter, Ollama, or any OpenAI-compatible API)
func NewClient(cfg ClientConfig) (*Client, error) {
	// API key is optional for local providers like Ollama
	apiKey := cfg.APIKey
	if apiKey == "" {
		apiKey = "ollama" // Placeholder for Ollama (it ignores this)
	}

	model := cfg.Model
	if model == "" {
		model = DefaultModel
	}

	// Configure client with custom or default base URL
	config := openai.DefaultConfig(apiKey)
	if cfg.BaseURL != "" {
		config.BaseURL = cfg.BaseURL
	} else {
		config.BaseURL = OpenRouterBaseURL
	}

	return &Client{
		client: openai.NewClientWithConfig(config),
		model:  model,
	}, nil
}

// Message represents a chat message
type Message struct {
	Role    string
	Content string
}

// ChatRequest represents a chat completion request
type ChatRequest struct {
	Messages    []Message
	Model       string // Override default model
	MaxTokens   int
	Temperature float32
	Stream      bool
}

// ChatResponse represents a chat completion response
type ChatResponse struct {
	Content      string
	Model        string
	FinishReason string
	Usage        Usage
}

// Usage tracks token usage
type Usage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

// Chat sends a chat completion request with retry logic
func (c *Client) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	// Set 120s timeout for the entire operation if not already set
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 120*time.Second)
		defer cancel()
	}

	var lastErr error
	backoff := 1 * time.Second

	// Retry loop (3 attempts)
	for attempt := 0; attempt < 3; attempt++ {
		model := req.Model
		if model == "" {
			model = c.model
		}

		messages := make([]openai.ChatCompletionMessage, len(req.Messages))
		for i, msg := range req.Messages {
			messages[i] = openai.ChatCompletionMessage{
				Role:    msg.Role,
				Content: msg.Content,
			}
		}

		maxTokens := req.MaxTokens
		if maxTokens == 0 {
			maxTokens = 4096
		}

		temperature := req.Temperature
		if temperature == 0 {
			temperature = 0.7
		}

		resp, err := c.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
			Model:       model,
			Messages:    messages,
			MaxTokens:   maxTokens,
			Temperature: temperature,
		})
		
		if err == nil {
			if len(resp.Choices) == 0 {
				return nil, errors.New("no response choices returned")
			}
			return &ChatResponse{
				Content:      resp.Choices[0].Message.Content,
				Model:        resp.Model,
				FinishReason: string(resp.Choices[0].FinishReason),
				Usage: Usage{
					PromptTokens:     resp.Usage.PromptTokens,
					CompletionTokens: resp.Usage.CompletionTokens,
					TotalTokens:      resp.Usage.TotalTokens,
				},
			}, nil
		}

		lastErr = err
		
		// Check for specific errors to retry
		// OpenRouter returns 429 for rate limits
		errMsg := err.Error()
		isRetryable := false
		if strings.Contains(errMsg, "429") || strings.Contains(errMsg, "500") || strings.Contains(errMsg, "502") || strings.Contains(errMsg, "503") {
			isRetryable = true
		}

		if !isRetryable {
			break
		}

		// Wait before retrying
		fmt.Printf("⚠️  API error (attempt %d/3): %v. Retrying in %v...\n", attempt+1, err, backoff)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(backoff):
			backoff *= 2 // Exponential backoff
		}
	}

	return nil, fmt.Errorf("chat completion failed after retries: %w", lastErr)
}

// StreamCallback is called for each chunk during streaming
type StreamCallback func(chunk string) error

// ChatStream sends a streaming chat completion request
func (c *Client) ChatStream(ctx context.Context, req ChatRequest, callback StreamCallback) (*ChatResponse, error) {
	// Set 120s timeout for the entire operation if not already set
	// Note: For streaming, we might want a longer timeout or just per-chunk timeout, 
	// but 120s is reasonable for code review generation.
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 120*time.Second)
		defer cancel()
	}

	var lastErr error
	backoff := 1 * time.Second

	for attempt := 0; attempt < 3; attempt++ {
		model := req.Model
		if model == "" {
			model = c.model
		}

		messages := make([]openai.ChatCompletionMessage, len(req.Messages))
		for i, msg := range req.Messages {
			messages[i] = openai.ChatCompletionMessage{
				Role:    msg.Role,
				Content: msg.Content,
			}
		}

		maxTokens := req.MaxTokens
		if maxTokens == 0 {
			maxTokens = 4096
		}

		temperature := req.Temperature
		if temperature == 0 {
			temperature = 0.7
		}

		stream, err := c.client.CreateChatCompletionStream(ctx, openai.ChatCompletionRequest{
			Model:       model,
			Messages:    messages,
			MaxTokens:   maxTokens,
			Temperature: temperature,
			Stream:      true,
		})
		
		if err == nil {
			defer stream.Close()

			var fullContent string
			var finishReason string

			for {
				response, err := stream.Recv()
				if errors.Is(err, io.EOF) {
					// Stream completed successfully
					return &ChatResponse{
						Content:      fullContent,
						Model:        model,
						FinishReason: finishReason,
					}, nil
				}
				if err != nil {
					// Stream interrupted
					lastErr = fmt.Errorf("stream error: %w", err)
					break // break inner loop to check for retry
				}

				if len(response.Choices) > 0 {
					chunk := response.Choices[0].Delta.Content
					fullContent += chunk

					if response.Choices[0].FinishReason != "" {
						finishReason = string(response.Choices[0].FinishReason)
					}

					if callback != nil && chunk != "" {
						if err := callback(chunk); err != nil {
							return nil, fmt.Errorf("callback error: %w", err)
						}
					}
				}
			}
			
			// If we get here, stream failed mid-way. Check if retryable.
		} else {
			lastErr = err
		}

		// Check for specific errors to retry
		errMsg := lastErr.Error()
		isRetryable := false
		if strings.Contains(errMsg, "429") || strings.Contains(errMsg, "500") || strings.Contains(errMsg, "502") || strings.Contains(errMsg, "503") || strings.Contains(errMsg, "stream error") {
			isRetryable = true
		}

		if !isRetryable {
			break
		}

		// Wait before retrying
		// Note: For streaming, retrying means restarting the whole stream. 
		// Ideally we'd resume, but simple retry is better than failure.
		fmt.Printf("⚠️  API error (attempt %d/3): %v. Retrying stream in %v...\n", attempt+1, lastErr, backoff)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(backoff):
			backoff *= 2
		}
	}

	return nil, fmt.Errorf("stream failed after retries: %w", lastErr)
}

// Model returns the current model
func (c *Client) Model() string {
	return c.model
}

// SetModel changes the default model
func (c *Client) SetModel(model string) {
	c.model = model
}
