package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/packages/param"
)

func setupClient() openai.Client {
	apiKey := os.Getenv("OPENROUTER_API_KEY")
	baseUrl := os.Getenv("OPENROUTER_BASE_URL")

	if baseUrl == "" {
		baseUrl = "https://openrouter.ai/api/v1"
	}

	if apiKey == "" {
		panic("Env variable OPENROUTER_API_KEY not found")
	}

	return openai.NewClient(option.WithAPIKey(apiKey), option.WithBaseURL(baseUrl))
}

func run(client openai.Client, prompt string, baseModel string) {
	tools := []openai.ChatCompletionToolUnionParam{
		openai.ChatCompletionFunctionTool(
			openai.FunctionDefinitionParam{
				Name:        "Read",
				Strict:      param.Opt[bool]{},
				Description: param.Opt[string]{Value: "Read and return the contents of a file"},
				Parameters: openai.FunctionParameters{
					"type": "object",
					"properties": map[string]any{
						"file_path": map[string]any{
							"type":        "string",
							"description": "The path to the file to read",
						},
					},
					"required": []string{"file_path"},
				},
			}),
		openai.ChatCompletionFunctionTool(
			openai.FunctionDefinitionParam{
				Name:        "Write",
				Strict:      param.Opt[bool]{},
				Description: param.Opt[string]{Value: "Write the contents to a file"},
				Parameters: openai.FunctionParameters{
					"type": "object",
					"properties": map[string]any{
						"file_path": map[string]any{
							"type":        "string",
							"description": "The path to which the file to write",
						},
						"content": map[string]any{
							"type":        "string",
							"description": "The content of the file to write",
						},
					},
					"required": []string{"file_path", "content"},
				},
			}),
		openai.ChatCompletionFunctionTool(
			openai.FunctionDefinitionParam{
				Name:        "Bash",
				Strict:      param.Opt[bool]{},
				Description: param.Opt[string]{Value: "Execute a shell command"},
				Parameters: openai.FunctionParameters{
					"type": "object",
					"properties": map[string]any{
						"command": map[string]any{
							"type":        "string",
							"description": "The command to execute",
						},
					},
					"required": []string{"command"},
				},
			}),
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
				Tools:    tools,
				Messages: messages,
			},
		)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		if len(resp.Choices) == 0 {
			panic("No choices in response")
		}

		fmt.Fprintln(os.Stderr, resp.Choices[0].Message.Content)

		messages = append(messages, resp.Choices[0].Message.ToParam())

		if finishReason := resp.Choices[0].FinishReason; finishReason == "tool_calls" {
			// assume at least one tool call
			toolCalls := resp.Choices[0].Message.ToolCalls
			if len(toolCalls) == 0 {
				fmt.Fprintln(os.Stderr, "error: tool_calls is finishReason, but no toolCalls")
				// TODO: Reply back to LLM
				os.Exit(1)
			}
			fmt.Fprintf(os.Stderr, "tool_calls: %v\n", toolCalls)
			for _, toolCall := range toolCalls {
				functionName := toolCall.Function.Name
				fmt.Fprintf(os.Stderr, "name: %v\n", functionName)

				if functionName == "Read" {
					var arguments map[string]string
					err = json.Unmarshal([]byte(toolCall.Function.Arguments), &arguments)
					if err != nil {
						fmt.Fprintln(os.Stderr, "error: parsing failed for Read tool_call arguments")
						panic(err)
					}
					fmt.Fprintf(os.Stderr, "tool_call: Read, file_path: %v\n", arguments["file_path"])

					content, err := os.ReadFile(arguments["file_path"])
					if err != nil {
						fmt.Fprintln(os.Stderr, "error: ", err)
						os.Exit(1)
					}
					messages = append(messages, openai.ToolMessage(string(content), toolCall.ID))
					// fmt.Print(string(content))

				} else if functionName == "Write" {
					var arguments map[string]string
					err = json.Unmarshal([]byte(toolCall.Function.Arguments), &arguments)
					if err != nil {
						fmt.Fprintln(os.Stderr, "error: parsing failed for write tool_call arguments")
						panic(err)
					}
					path := arguments["file_path"]
					content := arguments["content"]
					fmt.Fprintf(os.Stderr, "tool_call: Write, file_path: %v, content: %v\n", path, content)

					err := os.WriteFile(path, []byte(content), 0644)
					if err != nil {
						fmt.Fprintln(os.Stderr, "error: ", err)
						messages = append(messages, openai.ToolMessage(err.Error(), toolCall.ID))
					} else {
						messages = append(messages, openai.ToolMessage("wrote the content successfully", toolCall.ID))
					}
				} else if functionName == "Bash" {
					var arguments map[string]string
					err = json.Unmarshal([]byte(toolCall.Function.Arguments), &arguments)
					if err != nil {
						fmt.Fprintln(os.Stderr, "error: parsing failed for bash tool_call arguments")
						panic(err)
					}
					command := arguments["command"]
					fmt.Fprintf(os.Stderr, "tool_call: Bash, command: %v\n", command)

					out, err := exec.Command("sh", "-c", command).CombinedOutput()
					if err != nil {
						fmt.Fprintln(os.Stderr, "error: ", err)
						messages = append(messages, openai.ToolMessage(err.Error(), toolCall.ID))
					} else {
						messages = append(messages, openai.ToolMessage(string(out), toolCall.ID))
					}
				} else {
					fmt.Printf("unknown tool call: %v\n", functionName)
					messages = append(
						messages, openai.ToolMessage(
							fmt.Sprintf("unknown tool_call: %v", functionName),
							toolCall.ID))
				}
			}
		} else {
			fmt.Println(resp.Choices[0].Message.Content)
			return
		}
	}
}

func main() {
	var prompt string
	flag.StringVar(&prompt, "p", "", "Prompt to send to LLM")
	flag.Parse()
	if prompt == "" {
		panic("Prompt must not be empty")
	}

	baseModel := os.Getenv("OPENROUTER_BASE_MODEL") // use `openrouter/free` locally
	if baseModel == "" {
		baseModel = "anthropic/claude-haiku-4.5"
	}

	client := setupClient()

	run(client, prompt, baseModel)
}
