# Senator 🏛️

> A universal MCP-powered orchestration layer for multi-model code reviews

Senator is a CLI tool that orchestrates AI code review tasks using the Model Context Protocol (MCP). It convenes a "Senate" of AI agents to debate code changes before they are committed.

## Features

- 🔌 **Universal MCP Host** - Connect to any MCP server (GitHub, Postgres, Filesystem, Docker)
- 🔀 **Multi-Agent Reviews** - Orchestrate security, logic, and style reviewers
- ⚡ **Parallel Execution** - Concurrent critic reviews with Go's goroutines
- 📦 **Single Binary** - No Python, no Docker, no dependencies

## Quick Start

```bash
# Build
make build

# Test MCP connection
./senator ping --server "npx -y @modelcontextprotocol/server-filesystem /tmp" --list-tools

# View help
./senator --help
```

## Installation

```bash
# Clone and build
git clone https://github.com/lxnid/senator.git
cd senator
make build

# Or install globally
make install
```

## Commands

| Command           | Description                      |
| ----------------- | -------------------------------- |
| `senator ping`    | Test connection to an MCP server |
| `senator version` | Show version information         |

## Configuration

Create `~/.senator.yaml`:

```yaml
openrouter:
  api_key: "sk-or-v1-..."

models:
  author: "qwen/qwen-2.5-coder-32b-instruct"
  critics:
    - "deepseek/deepseek-r1"
    - "meta-llama/llama-3.1-8b-instruct"
```

## Development

```bash
make build      # Build binary
make test       # Run tests
make fmt        # Format code
make lint       # Run linter
make build-all  # Cross-platform builds
```

## Architecture

Senator is built as an MCP Host that spawns and communicates with MCP servers via stdio:

```
┌─────────────────┐     stdio     ┌──────────────────┐
│  Senator CLI    │◄────────────►│  MCP Server      │
│  (MCP Host)     │               │  (subprocess)    │
└─────────────────┘               └──────────────────┘
         │
         │ OpenRouter API
         ▼
┌─────────────────┐
│  LLM Models     │
│  (Author+Critics)│
└─────────────────┘
```

## Tech Stack

- **Go 1.23+** - Systems-level control and single binary distribution
- **Cobra** - CLI framework
- **mcp-go** - MCP SDK for stdio transport
- **Viper** - Configuration management

## License

MIT
