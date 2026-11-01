package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var modelsCmd = &cobra.Command{
	Use:   "models",
	Short: "List available models from Ollama or configured provider",
	Long: `Query the configured LLM provider to list available models.
For Ollama, this fetches from http://localhost:11434/api/tags.

Examples:
  # List Ollama models (when base_url is configured)
  senator models

  # List models from specific endpoint
  senator models --endpoint http://localhost:11434`,
	RunE: runModels,
}

var modelsEndpoint string

func init() {
	rootCmd.AddCommand(modelsCmd)
	modelsCmd.Flags().StringVarP(&modelsEndpoint, "endpoint", "e", "", "Ollama API endpoint (default: from config or http://localhost:11434)")
}

// OllamaModelsResponse represents the Ollama /api/tags response
type OllamaModelsResponse struct {
	Models []OllamaModel `json:"models"`
}

type OllamaModel struct {
	Name       string    `json:"name"`
	ModifiedAt time.Time `json:"modified_at"`
	Size       int64     `json:"size"`
}

func runModels(cmd *cobra.Command, args []string) error {
	// Determine endpoint
	endpoint := modelsEndpoint
	if endpoint == "" {
		baseURL := viper.GetString("openrouter.base_url")
		if baseURL != "" {
			// Convert from OpenAI-compatible URL to Ollama native
			// http://localhost:11434/v1 -> http://localhost:11434
			if len(baseURL) > 3 && baseURL[len(baseURL)-3:] == "/v1" {
				endpoint = baseURL[:len(baseURL)-3]
			} else {
				endpoint = baseURL
			}
		} else {
			endpoint = "http://localhost:11434"
		}
	}

	// Fetch models from Ollama API
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(endpoint + "/api/tags")
	if err != nil {
		return fmt.Errorf("failed to connect to Ollama at %s: %w\n\nMake sure Ollama is running: ollama serve", endpoint, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Ollama returned status %d: %s", resp.StatusCode, string(body))
	}

	var modelsResp OllamaModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&modelsResp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if len(modelsResp.Models) == 0 {
		fmt.Println("No models found. Install one with: ollama pull <model>")
		return nil
	}

	fmt.Printf("📦 Available Ollama Models (%s):\n\n", endpoint)
	for _, model := range modelsResp.Models {
		size := float64(model.Size) / (1024 * 1024 * 1024) // Convert to GB
		fmt.Printf("  • %-40s  %.1f GB\n", model.Name, size)
	}
	fmt.Printf("\n💡 Use in config: model: \"%s\"\n", modelsResp.Models[0].Name)

	return nil
}
