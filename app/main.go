package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/packages/param"
)

func main() {
	var prompt string
	flag.StringVar(&prompt, "p", "", "Prompt to send to LLM")
	flag.Parse()

	if prompt == "" {
		panic("Prompt must not be empty")
	}

	apiKey := os.Getenv("OPENROUTER_API_KEY")
	baseUrl := os.Getenv("OPENROUTER_BASE_URL")
	baseModel := os.Getenv("OPENROUTER_BASE_MODEL") // use `openrouter/free` locally

	if baseUrl == "" {
		baseUrl = "https://openrouter.ai/api/v1"
	}

	if apiKey == "" {
		panic("Env variable OPENROUTER_API_KEY not found")
	}

	if baseModel == "" {
		baseModel = "anthropic/claude-haiku-4.5"
	}

	client := openai.NewClient(option.WithAPIKey(apiKey), option.WithBaseURL(baseUrl))
	resp, err := client.Chat.Completions.New(
		context.Background(),
		openai.ChatCompletionNewParams{
			//Model: "anthropic/claude-haiku-4.5",
			Model: baseModel,
			Messages: []openai.ChatCompletionMessageParamUnion{
				{
					OfUser: &openai.ChatCompletionUserMessageParam{
						Content: openai.ChatCompletionUserMessageParamContentUnion{
							OfString: openai.String(prompt),
						},
					},
				},
			},
			Tools: []openai.ChatCompletionToolUnionParam{
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
			},
		},
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	if len(resp.Choices) == 0 {
		panic("No choices in response")
	}

	switch resp.Choices[0].FinishReason {
	case "tool_calls":
		// assume at least one tool call
		toolCalls := resp.Choices[0].Message.ToolCalls
		if len(toolCalls) == 0 {
			fmt.Fprintln(os.Stderr, "error: tool_calls is finishReason, but no toolCalls")
			// TODO: Reply back to LLM
			os.Exit(1)
		}

		functionName := toolCalls[0].Function.Name
		if functionName == "Read" {
			var arguments map[string]string
			err = json.Unmarshal([]byte(toolCalls[0].Function.Arguments), &arguments)
			if err != nil {
				fmt.Fprintln(os.Stderr, "error: parsing failed for Read tool_call arguments")
				panic(err)
			}
			fmt.Fprintf(os.Stderr, "tool_call: Read, file_name: %v\n", arguments["file_path"])

			content, err := os.ReadFile(arguments["file_path"])
			if err != nil {
				fmt.Fprintln(os.Stderr, "error: ", err)
				os.Exit(1)
			}
			fmt.Print(string(content))

		} else {
			fmt.Printf("unknown tool call: %v\n", functionName)
		}
	default:
		break
	}
	fmt.Print(resp.Choices[0].Message.Content)
}
