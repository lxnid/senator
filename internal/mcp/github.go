package mcp

import (
	"context"
	"fmt"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
)

// GitHubClient wraps the MCP client for GitHub operations
type GitHubClient struct {
	client *Client
}

// NewGitHubClient creates a new GitHub MCP client
func NewGitHubClient(ctx context.Context) (*GitHubClient, error) {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("GITHUB_TOKEN environment variable is required")
	}

	// Prepare environment
	env := os.Environ()
	// The official server-github might read GITHUB_TOKEN from env or args
	// We'll pass it in env just in case, though some servers require arg
	
	client, err := NewClient(ctx, ClientConfig{
		ServerCommand: "npx",
		ServerArgs:    []string{"-y", "@modelcontextprotocol/server-github"},
		ServerName:    "github",
		Env:           env,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create GitHub MCP client: %w", err)
	}

	return &GitHubClient{
		client: client,
	}, nil
}

// Close closes the underlying MCP client
func (c *GitHubClient) Close() error {
	return c.client.Close()
}

// PRDetails represents basic PR info
type PRDetails struct {
	Title       string `json:"title"`
	Description string `json:"body"`
	State       string `json:"state"`
	Diff        string `json:"diff,omitempty"` // populated if available
}

// FetchPR fetches Pull Request details
// Note: The specific tool name depends on the server implementation.
// We assume 'get_pull_request' or similar exists.
func (c *GitHubClient) FetchPR(ctx context.Context, owner, repo string, prNumber int) (*PRDetails, error) {
	// First, let's list tools to be sure (optional debug)
	// tools, _ := c.client.ListTools(ctx)
	
	// We'll try to call 'get_pull_request'
	result, err := c.client.CallTool(ctx, "get_pull_request", map[string]interface{}{
		"owner":       owner,
		"repo":        repo,
		"pull_number": prNumber,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch PR: %w", err)
	}

	// Parse result
	// The result content usually contains a list of text/image content
	if len(result.Content) == 0 {
		return nil, fmt.Errorf("empty response from get_pull_request")
	}

	// Simplified: just dumping the text content for now
	// In a real implementation, we'd parse the JSON from the text if it's structured,
	// or use the text directly if it's a summary.
	// For "The Senate", we really need the DIFF.
	// If 'get_pull_request' doesn't return a diff, we might need another approach.
	// Since we can't easily get the diff via standard tools without a specific 'get_diff' tool,
	// we might rely on the PR description and title for now, OR valid file contents.
	
	// Hack: For the prototype, we'll return the text content as "Code" 
	// (which is likely the PR description + metadata)
	// and warn that diff fetching isn't fully supported via this generic MCP server yet.
	
	textContent := ""
	for _, content := range result.Content {
		if tc, ok := content.(mcp.TextContent); ok {
			textContent += tc.Text + "\n"
		}
	}

	return &PRDetails{
		Title:       fmt.Sprintf("PR #%d: %s/%s", prNumber, owner, repo),
		Description: textContent,
		Diff:        "Diff fetching requires 'git' CLI or specialized MCP tool. Reviewing PR description/context only.",
	}, nil
}
