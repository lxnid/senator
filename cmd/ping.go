package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/lxnid/senator/internal/mcp"
	"github.com/spf13/cobra"
)

var (
	serverCmd  string
	listTools  bool
	timeout    int
)

var pingCmd = &cobra.Command{
	Use:   "ping",
	Short: "Test connection to an MCP server",
	Long: `Ping spawns an MCP server subprocess, performs a handshake,
and optionally lists the available tools. This is useful for testing
that MCP servers are correctly configured.

Examples:
  # Test Docker MCP server
  senator ping --server "docker run -i --rm mcp/docker"

  # Test with tool listing
  senator ping --server "docker run -i --rm mcp/docker" --list-tools

  # Test filesystem MCP server
  senator ping --server "npx -y @modelcontextprotocol/server-filesystem /tmp"`,
	RunE: runPing,
}

func init() {
	rootCmd.AddCommand(pingCmd)

	pingCmd.Flags().StringVarP(&serverCmd, "server", "s", "", "MCP server command to spawn (required)")
	pingCmd.Flags().BoolVarP(&listTools, "list-tools", "l", false, "list available tools from the server")
	pingCmd.Flags().IntVarP(&timeout, "timeout", "t", 30, "connection timeout in seconds")

	pingCmd.MarkFlagRequired("server")
}

func runPing(cmd *cobra.Command, args []string) error {
	// Parse the server command
	parts := strings.Fields(serverCmd)
	if len(parts) == 0 {
		return fmt.Errorf("invalid server command")
	}

	serverCommand := parts[0]
	serverArgs := parts[1:]

	fmt.Printf("🔌 Connecting to MCP server...\n")
	fmt.Printf("   Command: %s\n", serverCmd)

	// Create context with timeout and signal handling
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Second)
	defer cancel()

	// Handle Ctrl+C gracefully
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\n⚠️  Interrupted, shutting down...")
		cancel()
	}()

	// Create MCP client
	cfg := mcp.ClientConfig{
		ServerCommand: serverCommand,
		ServerArgs:    serverArgs,
		ServerName:    "ping-test",
	}

	startTime := time.Now()
	client, err := mcp.NewClient(ctx, cfg)
	if err != nil {
		return fmt.Errorf("❌ Failed to connect: %w", err)
	}
	defer client.Close()

	elapsed := time.Since(startTime)
	fmt.Printf("✅ Connected successfully! (took %v)\n", elapsed.Round(time.Millisecond))

	// List tools if requested
	if listTools {
		fmt.Println("\n📋 Available Tools:")
		tools, err := client.ListTools(ctx)
		if err != nil {
			return fmt.Errorf("failed to list tools: %w", err)
		}

		if len(tools) == 0 {
			fmt.Println("   (no tools available)")
		} else {
			for _, tool := range tools {
				fmt.Printf("   • %s\n", tool.Name)
				if tool.Description != "" {
					fmt.Printf("     %s\n", tool.Description)
				}
			}
		}
		fmt.Printf("\n   Total: %d tools\n", len(tools))
	}

	// List resources
	fmt.Println("\n📁 Available Resources:")
	resources, err := client.ListResources(ctx)
	if err != nil {
		fmt.Printf("   (could not list resources: %v)\n", err)
	} else if len(resources) == 0 {
		fmt.Println("   (no resources available)")
	} else {
		for _, res := range resources {
			fmt.Printf("   • %s\n", res.Name)
		}
		fmt.Printf("\n   Total: %d resources\n", len(resources))
	}

	fmt.Println("\n🏛️  Senator MCP ping complete!")
	return nil
}
