package council

import (
	"context"
	"fmt"
	"sync"

	"github.com/lxnid/senator/internal/llm"
)

// SenateOrchestrator coordinates the multi-agent review process
type SenateOrchestrator struct {
	Config   ReviewConfig
	OnResult func(ReviewResult) // Optional callback for streaming results
}

// NewOrchestrator creates a new orchestrator
func NewOrchestrator(cfg ReviewConfig) *SenateOrchestrator {
	return &SenateOrchestrator{
		Config: cfg,
	}
}

// ConductReview runs the full Senate review process
func (o *SenateOrchestrator) ConductReview(ctx context.Context) (*SenateReport, error) {
	// 1. Parallel Critic Reviews
	results := o.runCritics(ctx)

	// 2. Chairman Synthesis
	report, err := o.synthesizeReviews(ctx, results)
	if err != nil {
		return nil, fmt.Errorf("synthesis failed: %w", err)
	}

	return report, nil
}

func (o *SenateOrchestrator) runCritics(ctx context.Context) []ReviewResult {
	var wg sync.WaitGroup
	resultsCh := make(chan ReviewResult, len(o.Config.Reviewers))

	for _, reviewerCfg := range o.Config.Reviewers {
		wg.Add(1)
		go func(cfg ReviewerConfig) {
			defer wg.Done()

			reviewer, err := NewReviewer(cfg)
			if err != nil {
				resultsCh <- ReviewResult{Error: err}
				return
			}

			result, err := reviewer.Review(ctx, o.Config.Code)
			if err != nil {
				resultsCh <- ReviewResult{Error: err}
				return
			}
			resultsCh <- *result
		}(reviewerCfg)
	}

	wg.Wait()
	close(resultsCh)

	var results []ReviewResult
	for res := range resultsCh {
		if o.OnResult != nil {
			o.OnResult(res)
		}
		// Don't print errors to stdout, they are contained in res.Error
		results = append(results, res)
	}

	return results
}

func (o *SenateOrchestrator) synthesizeReviews(ctx context.Context, results []ReviewResult) (*SenateReport, error) {
	chairman, err := NewReviewer(o.Config.Chairman)
	if err != nil {
		return nil, err
	}

	// Prepare data for template
	data := llm.TemplateData{}
	for _, res := range results {
		key := ""
		switch res.ReviewerType {
		case "security":
			key = "SecurityReview"
		case "logic":
			key = "LogicReview"
		case "style":
			key = "StyleReview"
		}
		if key != "" {
			data[key] = res.RawOutput
		}
	}

	// Render prompt
	prompt, err := llm.RenderPrompt("chairman_synthesis", data)
	if err != nil {
		return nil, fmt.Errorf("failed to render synthesis prompt: %w", err)
	}

	// Call Chairman
	resp, err := chairman.Client.Chat(ctx, llm.ChatRequest{
		Messages: []llm.Message{
			{Role: llm.RoleSystem, Content: prompt},
		},
	})
	if err != nil {
		return nil, err
	}

	// Return simple report for now (parsing full synthesis is complex)
	// Ideally we'd want structured output here too
	return &SenateReport{
		Summary: resp.Content,
	}, nil
}
