package llm

import (
	"context"
	"fmt"
	"strings"

	"github.com/alantheprice/ledit/pkg/config"
	"github.com/alantheprice/ledit/pkg/tools"
	"github.com/alantheprice/ledit/pkg/types"
)

// ExecuteBasicToolCall executes a tool call with default context
func ExecuteBasicToolCall(toolCall ToolCall, cfg *config.Config) (string, error) {
	return ExecuteBasicToolCallWithContext(context.Background(), toolCall, cfg)
}

// ExecuteBasicToolCallWithContext executes a tool call with provided context
func ExecuteBasicToolCallWithContext(ctx context.Context, toolCall ToolCall, cfg *config.Config) (string, error) {
	// Convert to types.ToolCall for the unified executor
	typesToolCall := convertToTypesToolCall(toolCall)

	// Use the new unified tool executor
	result, err := tools.ExecuteToolCall(ctx, typesToolCall)
	if err != nil {
		return "", err
	}

	if !result.Success {
		return "", fmt.Errorf("tool execution failed: %v", strings.Join(result.Errors, "; "))
	}

	return formatToolResult(result.Output), nil
}

// convertToTypesToolCall converts LLM ToolCall to types.ToolCall
func convertToTypesToolCall(toolCall ToolCall) types.ToolCall {
	return types.ToolCall{
		ID:   toolCall.ID,
		Type: toolCall.Type,
		Function: types.ToolCallFunction{
			Name:      toolCall.Function.Name,
			Arguments: toolCall.Function.Arguments,
		},
	}
}

// formatToolResult converts tool result to string format
func formatToolResult(output interface{}) string {
	if output == nil {
		return ""
	}
	
	if str, ok := output.(string); ok {
		return str
	}
	
	return fmt.Sprintf("%v", output)
}

// isLikelyTextOrCode checks if a file path represents text or code
func isLikelyTextOrCode(path string) bool {
	lower := strings.ToLower(path)
	
	textExts := []string{
		".go", ".ts", ".tsx", ".js", ".jsx", ".py", ".java", ".rb", ".rs",
		".c", ".cc", ".cpp", ".h", ".hpp", ".cs", ".php", ".kt",
		".m", ".mm", ".swift", ".scala", ".sql", ".sh", ".bash", ".zsh", ".fish",
		".yaml", ".yml", ".json", ".toml", ".ini", ".md", ".txt",
	}
	
	for _, ext := range textExts {
		if strings.HasSuffix(lower, ext) {
			return true
		}
	}
	
	return false
}

// sanitizeOutput redacts possible secrets from logs
func sanitizeOutput(s string) string {
	redactions := []string{
		"AWS_SECRET", "AWS_ACCESS_KEY", "OPENAI_API_KEY", "DEEPINFRA_API_KEY",
		"GEMINI_API_KEY", "GROQ_API_KEY", "ANTHROPIC_API_KEY",
	}
	
	out := s
	for _, key := range redactions {
		if strings.Contains(out, key) {
			out = strings.ReplaceAll(out, key, "<REDACTED>")
		}
	}
	
	return out
}

// classifyError categorizes errors for routing and analysis
func classifyError(err error) string {
	if err == nil {
		return "none"
	}
	
	msg := strings.ToLower(err.Error())
	
	switch {
	case strings.Contains(msg, "permission") || strings.Contains(msg, "denied"):
		return "permission"
	case strings.Contains(msg, "not found") || strings.Contains(msg, "no such file"):
		return "not_found"
	case strings.Contains(msg, "timeout") || strings.Contains(msg, "deadline"):
		return "transient"
	case strings.Contains(msg, "invalid") || strings.Contains(msg, "bad request"):
		return "invalid_args"
	case strings.Contains(msg, "network") || strings.Contains(msg, "connection"):
		return "network"
	case strings.Contains(msg, "rate limit") || strings.Contains(msg, "quota"):
		return "rate_limit"
	default:
		return "unknown"
	}
}