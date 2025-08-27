package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/alantheprice/ledit/pkg/llm"
	"github.com/alantheprice/ledit/pkg/prompts"
	"github.com/alantheprice/ledit/pkg/utils"
)

// executeDirectEditTodo handles simple documentation edits directly
func executeDirectEditTodo(ctx *SimplifiedAgentContext, todo *TodoItem) error {
	ctx.Logger.LogProcessStep("✏️ Performing direct edit (simple changes)")

	editRequest, err := generateEditRequest(ctx, todo)
	if err != nil {
		return fmt.Errorf("failed to generate edit request: %w", err)
	}

	return applyDirectEdit(editRequest.FilePath, editRequest.Content, ctx.Logger, ctx)
}

// editRequest represents the structure for edit operations
type editRequest struct {
	FilePath string `json:"file_path"`
	Changes  string `json:"changes"`
	Content  string `json:"content"`
}

// generateEditRequest creates an edit request from LLM
func generateEditRequest(ctx *SimplifiedAgentContext, todo *TodoItem) (*editRequest, error) {
	prompt := buildEditPrompt(ctx, todo)
	messages := []prompts.Message{
		{Role: "system", Content: "You are an expert at making simple, targeted edits. Provide specific file paths and exact content changes."},
		{Role: "user", Content: prompt},
	}

	smartTimeout := GetSmartTimeout(ctx.Config, ctx.Config.EditingModel, "editing")
	response, tokenUsage, err := llm.GetLLMResponse(ctx.Config.EditingModel, messages, "", ctx.Config, smartTimeout)
	if err != nil {
		return nil, fmt.Errorf("LLM request failed: %w", err)
	}

	trackTokenUsage(ctx, tokenUsage, ctx.Config.EditingModel)
	return parseEditResponse(response)
}

// buildEditPrompt creates the prompt for direct editing
func buildEditPrompt(ctx *SimplifiedAgentContext, todo *TodoItem) string {
	return fmt.Sprintf(`You need to make a simple edit based on this todo:

Todo: %s
Description: %s
Overall Task: %s

Please provide the specific file path and the exact changes needed. Respond in JSON format:
{
  "file_path": "path/to/file",
  "changes": "description of what to change",
  "content": "the new content to use"
}`, todo.Content, todo.Description, ctx.UserIntent)
}

// parseEditResponse parses the LLM response for edit instructions
func parseEditResponse(response string) (*editRequest, error) {
	clean, err := utils.ExtractJSON(response)
	if err != nil {
		return nil, fmt.Errorf("failed to extract JSON from response: %w", err)
	}

	var req editRequest
	if err := json.Unmarshal([]byte(clean), &req); err != nil {
		return nil, fmt.Errorf("failed to parse edit request JSON: %w", err)
	}

	return &req, nil
}

// applyDirectEdit applies a direct edit to a file
func applyDirectEdit(filePath, newContent string, logger *utils.Logger, ctx *SimplifiedAgentContext) error {
	if filePath == "" || newContent == "" {
		return fmt.Errorf("file path and content are required")
	}

	if err := validateFilePath(filePath); err != nil {
		return fmt.Errorf("invalid file path: %w", err)
	}

	return writeContentToFile(filePath, newContent, logger, ctx)
}

// validateFilePath performs basic validation on file path
func validateFilePath(filePath string) error {
	if strings.Contains(filePath, "..") {
		return fmt.Errorf("path traversal not allowed")
	}
	if strings.HasPrefix(filePath, "/") && !strings.HasPrefix(filePath, "/tmp/") {
		return fmt.Errorf("absolute paths not allowed outside /tmp")
	}
	return nil
}

// writeContentToFile writes content to the specified file
func writeContentToFile(filePath, content string, logger *utils.Logger, ctx *SimplifiedAgentContext) error {
	logger.LogProcessStep(fmt.Sprintf("📝 Writing to file: %s", filePath))

	// Ensure directory exists
	if err := ensureDirectoryExists(filePath); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write the file
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	// Mark files as modified
	ctx.FilesModified = true

	logger.LogProcessStep(fmt.Sprintf("✅ Successfully wrote to %s", filePath))
	return nil
}

// ensureDirectoryExists creates directory if it doesn't exist
func ensureDirectoryExists(filePath string) error {
	dir := ""
	if idx := strings.LastIndex(filePath, "/"); idx != -1 {
		dir = filePath[:idx]
	}
	
	if dir != "" {
		return os.MkdirAll(dir, 0755)
	}
	
	return nil
}