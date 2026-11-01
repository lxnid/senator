package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lxnid/senator/internal/council"
	"github.com/lxnid/senator/internal/mcp"
	"github.com/lxnid/senator/internal/tui"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	reviewFile  string
	reviewPR    int
	reviewRepo  string
	interactive bool
)

var reviewCmd = &cobra.Command{
	Use:   "review",
	Short: "Conduct a Senate review on a file or PR",
	Long: `Orchestrate a multi-agent review of a code file or GitHub Pull Request.

Examples:
  senator review --file internal/mcp/client.go
  senator review --pr 123 --repo owner/name --interactive`,
	RunE: runReview,
}

func init() {
	rootCmd.AddCommand(reviewCmd)

	reviewCmd.Flags().StringVarP(&reviewFile, "file", "f", "", "path to local file to review")
	reviewCmd.Flags().IntVar(&reviewPR, "pr", 0, "GitHub PR number")
	reviewCmd.Flags().StringVar(&reviewRepo, "repo", "", "GitHub repository (owner/name)")
	reviewCmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "run in interactive TUI mode")
}

func runReview(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// 1. Fetch Code
	var code, contextInfo string
	var err error

	if reviewFile != "" {
		code, err = readLocalFile(reviewFile)
		if err != nil {
			return err
		}
		contextInfo = fmt.Sprintf("File: %s", reviewFile)
	} else if reviewPR != 0 {
		if reviewRepo == "" {
			return fmt.Errorf("--repo owner/name is required with --pr")
		}
		
		// Parse owner/repo
		// Simplified...
		owner := "lxnid" // placeholder/extract from repo string
		repo := "senator"
		
		fmt.Printf("🔌 Connecting to GitHub for PR #%d...\n", reviewPR)
		ghClient, err := mcp.NewGitHubClient(ctx)
		if err != nil {
			return fmt.Errorf("failed to create GitHub client: %w", err)
		}
		defer ghClient.Close()
		
		prDetails, err := ghClient.FetchPR(ctx, owner, repo, reviewPR)
		if err != nil {
			// Fallback or error
			fmt.Printf("⚠️  Failed to fetch PR via MCP: %v\n", err)
			return err
		}
		
		code = prDetails.Description // or Diff if available
		contextInfo = prDetails.Title
	} else {
		return fmt.Errorf("please specify --file or --pr")
	}

	// 2. Configure Senate - read from config with Ollama support
	apiKey := viper.GetString("openrouter.api_key")
	if apiKey == "" {
		apiKey = os.Getenv("SENATOR_OPENROUTER_API_KEY")
	}
	baseURL := viper.GetString("openrouter.base_url")
	
	// API key only required for OpenRouter
	if apiKey == "" && baseURL == "" {
		return fmt.Errorf("API key required (or set base_url for Ollama)")
	}

	// Read model from config or use sensible defaults
	defaultModel := viper.GetString("openrouter.model")
	if defaultModel == "" {
		defaultModel = "qwen2.5-coder:7b" // Ollama default
	}

	config := council.ReviewConfig{
		Code:    code,
		Context: contextInfo,
		Reviewers: []council.ReviewerConfig{
			{Name: "Security Expert", Role: "security", Model: defaultModel, APIKey: apiKey, BaseURL: baseURL},
			{Name: "Logic Analyzer", Role: "logic", Model: defaultModel, APIKey: apiKey, BaseURL: baseURL},
			{Name: "Style Checker", Role: "style", Model: defaultModel, APIKey: apiKey, BaseURL: baseURL},
		},
		Chairman: council.ReviewerConfig{
			Name: "The Chairman", Role: "chairman", Model: defaultModel, APIKey: apiKey, BaseURL: baseURL,
		},
	}

	fmt.Printf("🤖 Using model: %s\n", defaultModel)
	if baseURL != "" {
		fmt.Printf("🦙 Provider: Ollama (%s)\n", baseURL)
	} else {
		fmt.Println("☁️  Provider: OpenRouter")
	}

	if interactive {
		// Run TUI
		p := tea.NewProgram(tui.NewModel(code, contextInfo))
		
		// Run orchestration in a goroutine and send messages to TUI
		go func() {
			orchestrator := council.NewOrchestrator(config)
			orchestrator.OnResult = func(res council.ReviewResult) {
				p.Send(res)
			}
			
			report, err := orchestrator.ConductReview(ctx)
			if err != nil {
				// p.Send(errMsg(err))
				return
			}
			p.Send(*report)
		}()
		
		if _, err := p.Run(); err != nil {
			return fmt.Errorf("TUI error: %w", err)
		}
		return nil
	}

	// Non-interactive Mode (CLI Output)
	fmt.Println("🏛️  The Senate is now in session...")
	fmt.Printf("   Reviewing: %s\n", contextInfo)
	fmt.Println("   Witnesses called: Security, Logic, Style")

	orchestrator := council.NewOrchestrator(config)
	report, err := orchestrator.ConductReview(ctx)
	if err != nil {
		return fmt.Errorf("senate error: %w", err)
	}

	fmt.Println("\n⚖️  The Senate has spoken:")
	fmt.Println("==========================")
	fmt.Println(report.Summary)
	fmt.Println("==========================")

	return nil
}

func readLocalFile(path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}

	content, err := os.ReadFile(absPath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	return string(content), nil
}
