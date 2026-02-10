# Claude Code (Go)

A minimal Claude Code clone built in Go — an LLM-powered coding assistant that reads files, writes files, and executes shell commands through an agentic tool-calling loop. Built as a [CodeCrafters "Build Your Own Claude Code"](https://codecrafters.io/challenges/claude-code) challenge solution.

## Installation

```sh
go install github.com/ImperishableMe/claude-code/cmd/claude-code@latest
```

Make sure `$GOPATH/bin` is in your `PATH`:

```sh
export PATH="$PATH:$(go env GOPATH)/bin"
```

## Usage

```sh
# Set your API key
export OPENROUTER_API_KEY="your-api-key"

# Run with a prompt
claude-code -p "your prompt here"
claude-code --prompt "your prompt here"

# Show help
claude-code --help
```

## Architecture

The agent follows a standard LLM tool-calling loop:

```
Prompt → LLM → Tool Calls → Execute Tools → Feed Results Back → LLM → ... → Final Response
```

1. User prompt is sent to the LLM along with tool definitions
2. The LLM responds with either a final message or tool calls
3. Tool calls are executed locally and results are appended to the conversation
4. The loop repeats (up to 10 iterations) until the LLM produces a final response

All tools implement a common `Tool` interface with `Name()`, `Definition()`, and `Execute()` methods, registered in a simple map-based dispatch table.

## Tools

| Tool | Description |
|------|-------------|
| **Read** | Reads file contents from disk (`os.ReadFile`) |
| **Write** | Writes content to a file with `0644` permissions |
| **Bash** | Executes shell commands via `sh -c` and returns combined stdout/stderr |

## Prerequisites

- Go 1.25+
- An [OpenRouter](https://openrouter.ai) API key

## Configuration

| Environment Variable | Required | Default | Description |
|---------------------|----------|---------|-------------|
| `OPENROUTER_API_KEY` | Yes | — | OpenRouter API key for authentication |
| `OPENROUTER_BASE_URL` | No | `https://openrouter.ai/api/v1` | API base URL |
| `OPENROUTER_BASE_MODEL` | No | `anthropic/claude-haiku-4.5` | Model to use for completions |

## Development

```sh
# Build locally
go build -o claude-code ./cmd/claude-code/

# Run locally
./claude-code -p "your prompt here"

# Or use the wrapper script
./your_program.sh -p "your prompt here"
```

## Project Structure

```
cmd/claude-code/
├── main.go    # Entry point, agent loop, OpenRouter client setup
└── tools.go   # Tool interface and implementations (Read, Write, Bash)
```
