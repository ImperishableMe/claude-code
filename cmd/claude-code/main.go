package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

func setupClient() openai.Client {
	apiKey := os.Getenv("OPENROUTER_API_KEY")
	baseUrl := os.Getenv("OPENROUTER_BASE_URL")

	if baseUrl == "" {
		baseUrl = "https://openrouter.ai/api/v1"
	}

	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "error: OPENROUTER_API_KEY environment variable is required")
		os.Exit(1)
	}

	return openai.NewClient(option.WithAPIKey(apiKey), option.WithBaseURL(baseUrl))
}

func run(client openai.Client, prompt string, baseModel string) {
	tools := []Tool{
		ReadTool{},
		WriteTool{},
		BashTool{},
	}
	toolMap := map[string]Tool{}
	for _, tool := range tools {
		toolMap[tool.Name()] = tool
	}

	var toolDefinitions []openai.ChatCompletionToolUnionParam
	for _, tool := range toolMap {
		toolDefinitions = append(toolDefinitions, tool.Definition())
	}

	messages := []openai.ChatCompletionMessageParamUnion{
		openai.UserMessage(prompt),
	}

	const MaxIterations = 10
	for i := 0; i < MaxIterations; i++ {
		fmt.Fprintf(os.Stderr, "Sending LLM %d messages\n", len(messages))
		resp, err := client.Chat.Completions.New(
			context.Background(),
			openai.ChatCompletionNewParams{
				Model:    baseModel,
				Tools:    toolDefinitions,
				Messages: messages,
			},
		)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		if len(resp.Choices) == 0 {
			fmt.Fprintln(os.Stderr, "error: no choices in response")
			os.Exit(1)
		}
		messages = append(messages, resp.Choices[0].Message.ToParam())

		if finishReason := resp.Choices[0].FinishReason; finishReason == "tool_calls" {
			toolCalls := resp.Choices[0].Message.ToolCalls
			if len(toolCalls) == 0 {
				fmt.Fprintln(os.Stderr, "error: tool_calls is finishReason, but no toolCalls")
				// TODO: Reply back to LLM
				os.Exit(1)
			}
			fmt.Fprintf(os.Stderr, "tool_calls: %v\n", toolCalls)
			for _, toolCall := range toolCalls {
				toolName := toolCall.Function.Name
				fmt.Fprintf(os.Stderr, "name: %v\n", toolName)
				if tool, ok := toolMap[toolName]; ok {
					out, err := tool.Execute(json.RawMessage(toolCall.Function.Arguments))
					if err != nil {
						messages = append(messages, openai.ToolMessage(err.Error(), toolCall.ID))
					} else {
						messages = append(messages, openai.ToolMessage(out, toolCall.ID))
					}
				} else {
					fmt.Printf("unknown tool call: %v\n", toolName)
					messages = append(
						messages, openai.ToolMessage(
							fmt.Sprintf("unknown tool_call: %v", toolName),
							toolCall.ID))
				}
			}
		} else if finishReason == "stop" {
			fmt.Println(resp.Choices[0].Message.Content)
			return
		}
	}
}

func main() {
	var prompt string
	flag.StringVar(&prompt, "p", "", "Prompt to send to LLM (shorthand)")
	flag.StringVar(&prompt, "prompt", "", "Prompt to send to LLM")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: claude-code -p \"your prompt\"\n\nOptions:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nEnvironment variables:\n")
		fmt.Fprintf(os.Stderr, "  OPENROUTER_API_KEY     API key for OpenRouter (required)\n")
		fmt.Fprintf(os.Stderr, "  OPENROUTER_BASE_URL    Base URL for API (default: https://openrouter.ai/api/v1)\n")
		fmt.Fprintf(os.Stderr, "  OPENROUTER_BASE_MODEL  Model to use (default: anthropic/claude-haiku-4.5)\n")
	}
	flag.Parse()
	if prompt == "" {
		fmt.Fprintln(os.Stderr, "error: prompt is required (-p \"your prompt\")")
		flag.Usage()
		os.Exit(1)
	}

	baseModel := os.Getenv("OPENROUTER_BASE_MODEL") // use `openrouter/free` locally
	if baseModel == "" {
		baseModel = "anthropic/claude-haiku-4.5"
	}

	client := setupClient()

	run(client, prompt, baseModel)
}
