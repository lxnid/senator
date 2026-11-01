package mcp

import (
	"context"
	"fmt"
	"os"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

// Client wraps the MCP client with additional functionality for Senator
type Client struct {
	mcpClient  *client.Client
	serverName string
	tools      []mcp.Tool
}

// ClientConfig holds configuration for creating an MCP client
type ClientConfig struct {
	// ServerCommand is the command to spawn the MCP server (e.g., "docker", "npx")
	ServerCommand string
	// ServerArgs are the arguments for the server command
	ServerArgs []string
	// ServerName is a friendly name for logging
	ServerName string
	// Env is optional environment variables for the subprocess
	Env []string
}

// NewClient creates a new MCP client that spawns and connects to an MCP server
func NewClient(ctx context.Context, cfg ClientConfig) (*Client, error) {
	// Use current environment if none specified
	env := cfg.Env
	if env == nil {
		env = os.Environ()
	}

	// Create the MCP client using stdio transport
	// NewStdioMCPClient automatically starts the connection
	mcpClient, err := client.NewStdioMCPClient(cfg.ServerCommand, env, cfg.ServerArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to create MCP client: %w", err)
	}

	// Initialize the connection with our client info
	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    "senator",
		Version: "0.1.0",
	}
	initRequest.Params.Capabilities = mcp.ClientCapabilities{}

	_, err = mcpClient.Initialize(ctx, initRequest)
	if err != nil {
		mcpClient.Close()
		return nil, fmt.Errorf("failed to initialize MCP connection: %w", err)
	}

	return &Client{
		mcpClient:  mcpClient,
		serverName: cfg.ServerName,
	}, nil
}

// ListTools retrieves the list of available tools from the MCP server
func (c *Client) ListTools(ctx context.Context) ([]mcp.Tool, error) {
	result, err := c.mcpClient.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to list tools: %w", err)
	}
	c.tools = result.Tools
	return result.Tools, nil
}

// CallTool invokes a tool on the MCP server
func (c *Client) CallTool(ctx context.Context, name string, arguments map[string]interface{}) (*mcp.CallToolResult, error) {
	request := mcp.CallToolRequest{}
	request.Params.Name = name
	request.Params.Arguments = arguments

	result, err := c.mcpClient.CallTool(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("failed to call tool %s: %w", name, err)
	}
	return result, nil
}

// ListResources retrieves the list of available resources from the MCP server
func (c *Client) ListResources(ctx context.Context) ([]mcp.Resource, error) {
	result, err := c.mcpClient.ListResources(ctx, mcp.ListResourcesRequest{})
	if err != nil {
		return nil, fmt.Errorf("failed to list resources: %w", err)
	}
	return result.Resources, nil
}

// ServerName returns the friendly name of the connected server
func (c *Client) ServerName() string {
	return c.serverName
}

// Close terminates the MCP server subprocess and cleans up resources
func (c *Client) Close() error {
	if c.mcpClient != nil {
		return c.mcpClient.Close()
	}
	return nil
}
