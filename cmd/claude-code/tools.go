package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/packages/param"
)

type Tool interface {
	// Name of the tool
	Name() string

	// Definition returns the OpenAI tool definition to
	// register it to the model
	Definition() openai.ChatCompletionToolUnionParam

	// Execute runs the tool with raw JSON arguments
	// and returns a result string or error
	//
	// The tool with args given as json string
	// Each tool might interpret arguments differently,
	// hence the generic argument.
	Execute(args json.RawMessage) (output string, err error)
}

type ReadTool struct{}

func (r ReadTool) Name() string {
	return "Read"
}

func (r ReadTool) Definition() openai.ChatCompletionToolUnionParam {
	return openai.ChatCompletionFunctionTool(
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
		})
}

func (r ReadTool) Execute(args json.RawMessage) (output string, err error) {
	var arguments map[string]string
	err = json.Unmarshal(args, &arguments)
	if err != nil {
		return "", fmt.Errorf("parsing failed for Read tool_call arguments: %w", err)
	}
	fmt.Fprintf(os.Stderr, "tool_call: Read, file_path: %v\n", arguments["file_path"])
	content, err := os.ReadFile(arguments["file_path"])
	return string(content), err
}

type WriteTool struct{}

func (w WriteTool) Name() string {
	return "Write"
}

func (w WriteTool) Definition() openai.ChatCompletionToolUnionParam {
	return openai.ChatCompletionFunctionTool(
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
		})
}

func (w WriteTool) Execute(args json.RawMessage) (output string, err error) {
	var arguments map[string]string
	err = json.Unmarshal([]byte(args), &arguments)
	if err != nil {
		return "", fmt.Errorf("parsing failed for Write tool_call arguments: %w", err)
	}
	path := arguments["file_path"]
	content := arguments["content"]
	fmt.Fprintf(os.Stderr, "tool_call: Write, file_path: %v, content: %v\n", path, content)

	err = os.WriteFile(path, []byte(content), 0644)
	return "write file successfully", err
}

type BashTool struct{}

func (b BashTool) Name() string {
	return "Bash"
}

func (b BashTool) Definition() openai.ChatCompletionToolUnionParam {
	return openai.ChatCompletionFunctionTool(
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
		})
}

func (b BashTool) Execute(args json.RawMessage) (output string, err error) {
	var arguments map[string]string
	err = json.Unmarshal(args, &arguments)
	if err != nil {
		return "", fmt.Errorf("parsing failed for Bash tool_call arguments: %w", err)
	}
	command := arguments["command"]
	fmt.Fprintf(os.Stderr, "tool_call: Bash, command: %v\n", command)
	out, err := exec.Command("sh", "-c", command).CombinedOutput()
	return string(out), err
}
