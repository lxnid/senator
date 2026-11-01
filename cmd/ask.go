package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/charmbracelet/glamour"
	"github.com/lxnid/senator/internal/llm"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	askModel  string
	askStream bool
)

var askCmd = &cobra.Command{
	Use:   "ask [question]",
	Short: "Ask a question to an AI model via OpenRouter",
	Long: `Send a question to an AI model through OpenRouter and get a response.
The response can be streamed in real-time or returned all at once.

Examples:
  # Simple question
  senator ask "What is the capital of France?"

  # With specific model
  senator ask --model "deepseek/deepseek-r1" "Explain goroutines in Go"

  # Streaming response
  senator ask --stream "Tell me a short story"`,
	Args: cobra.MinimumNArgs(1),
	RunE: runAsk,
}

func init() {
	rootCmd.AddCommand(askCmd)

	askCmd.Flags().StringVarP(&askModel, "model", "m", "", "model to use (default: qwen/qwen-2.5-coder-32b-instruct)")
	askCmd.Flags().BoolVarP(&askStream, "stream", "s", true, "stream the response in real-time")

	// Bind to viper for config file support
	viper.BindPFlag("openrouter.model", askCmd.Flags().Lookup("model"))
}

func runAsk(cmd *cobra.Command, args []string) error {
	question := strings.Join(args, " ")

	// Get base URL (for Ollama or other providers)
	baseURL := viper.GetString("openrouter.base_url")

	// Get API key from env or config (optional for Ollama)
	apiKey := viper.GetString("openrouter.api_key")
	if apiKey == "" {
		apiKey = os.Getenv("SENATOR_OPENROUTER_API_KEY")
	}
	if apiKey == "" {
		apiKey = os.Getenv("OPENROUTER_API_KEY")
	}
	// API key is only required for OpenRouter (not for Ollama)
	if apiKey == "" && baseURL == "" {
		return fmt.Errorf("OpenRouter API key required (or set base_url for Ollama). Set SENATOR_OPENROUTER_API_KEY or configure in configs/senator.yaml")
	}

	// Determine model
	model := askModel
	if model == "" {
		model = viper.GetString("openrouter.model")
	}

	// Create LLM client
	client, err := llm.NewClient(llm.ClientConfig{
		APIKey:  apiKey,
		Model:   model,
		BaseURL: baseURL,
	})
	if err != nil {
		return fmt.Errorf("failed to create LLM client: %w", err)
	}

	// Setup context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle Ctrl+C
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\n⚠️  Interrupted")
		cancel()
	}()

	// Build messages
	messages := llm.BuildMessages(
		llm.GetSystemPrompt("simple_qa"),
		question,
	)

	// Always show which model is being used
	fmt.Printf("🤖 Model: %s\n\n", client.Model())

	// Create markdown renderer with dark style for better terminal visibility
	renderer, _ := glamour.NewTermRenderer(
		glamour.WithStylePath("dark"),
		glamour.WithWordWrap(120),
	)

	if askStream {
		// Streaming response - collect chunks silently, then render formatted
		var fullContent strings.Builder
		_, err = client.ChatStream(ctx, llm.ChatRequest{
			Messages: messages,
		}, func(chunk string) error {
			fullContent.WriteString(chunk)
			return nil
		})
		
		// Render formatted output only
		if renderer != nil && fullContent.Len() > 0 {
			rendered, err := renderer.Render(fullContent.String())
			if err == nil {
				fmt.Print(rendered)
			} else {
				fmt.Println(fullContent.String())
			}
		} else {
			fmt.Println(fullContent.String())
		}
	} else {
		// Non-streaming response - render directly
		resp, err := client.Chat(ctx, llm.ChatRequest{
			Messages: messages,
		})
		if err != nil {
			return fmt.Errorf("chat failed: %w", err)
		}
		
		if renderer != nil {
			rendered, renderErr := renderer.Render(resp.Content)
			if renderErr == nil {
				fmt.Print(rendered)
			} else {
				fmt.Println(resp.Content)
			}
		} else {
			fmt.Println(resp.Content)
		}

		if verbose {
			fmt.Println("---")
			fmt.Printf("📊 Tokens: %d prompt + %d completion = %d total\n",
				resp.Usage.PromptTokens, resp.Usage.CompletionTokens, resp.Usage.TotalTokens)
		}
	}

	if err != nil {
		return fmt.Errorf("chat failed: %w", err)
	}

	return nil
}
