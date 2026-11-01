package council

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/lxnid/senator/internal/llm"
)

// Reviewer represents an AI critic
type Reviewer struct {
	Config ReviewerConfig
	Client *llm.Client
}

// NewReviewer creates a new reviewer instance
func NewReviewer(cfg ReviewerConfig) (*Reviewer, error) {
	client, err := llm.NewClient(llm.ClientConfig{
		APIKey:  cfg.APIKey,
		Model:   cfg.Model,
		BaseURL: cfg.BaseURL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create reviewer client for %s: %w", cfg.Name, err)
	}

	return &Reviewer{
		Config: cfg,
		Client: client,
	}, nil
}

// Review performs a code review
func (r *Reviewer) Review(ctx context.Context, code string) (*ReviewResult, error) {
	// Build messages
	messages := llm.BuildReviewMessages(r.Config.Role, code)

	// Call LLM
	var warning string
	if len(code) > 32000 {
		warning = fmt.Sprintf("⚠️ Input code is very large (%d chars). Review might be truncated or slow.", len(code))
	}

	resp, err := r.Client.Chat(ctx, llm.ChatRequest{
		Messages: messages,
	})
	if err != nil {
		return nil, fmt.Errorf("review failed: %w", err)
	}

	// Parse JSON output
	// LLMs often wrap JSON in markdown blocks, let's strip those
	cleanJSON := stripMarkdown(resp.Content)
	
	var issues struct {
		Issues []Issue `json:"issues"`
	}
	
	if err := json.Unmarshal([]byte(cleanJSON), &issues); err != nil {
		// Store error in result for display/logging, don't print to stdout
		return &ReviewResult{
			ReviewerType: r.Config.Role,
			Model:        r.Config.Model,
			RawOutput:    resp.Content,
			Error:        fmt.Errorf("JSON parse error: %w", err),
			Warning:      warning,
		}, nil
	}

	return &ReviewResult{
		ReviewerType: r.Config.Role,
		Model:        r.Config.Model,
		RawOutput:    resp.Content,
		Issues:       issues.Issues,
		Warning:      warning,
	}, nil
}

// stripMarkdown removes ```json ... ``` wrappers from LLM output
func stripMarkdown(content string) string {
	content = strings.TrimSpace(content)
	if strings.HasPrefix(content, "```") {
		lines := strings.Split(content, "\n")
		if len(lines) >= 2 {
			// Remove first and last line
			return strings.Join(lines[1:len(lines)-1], "\n")
		}
	}
	return content
}
