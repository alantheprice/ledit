package llm

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// containsToolCall checks if the response contains tool calls
func containsToolCall(response string) bool {
	// Fast-path: anywhere in the response
	if strings.Contains(response, `"tool_calls"`) {
		return true
	}

	return checkVariousFormats(response)
}

// checkVariousFormats checks different formats where tool calls might appear
func checkVariousFormats(response string) bool {
	trimmed := strings.TrimSpace(response)

	// JSON starting with tool_calls
	if strings.HasPrefix(trimmed, "{") && strings.Contains(response, `"tool_calls"`) {
		return true
	}

	// JSON code block
	if checkJSONCodeBlock(response) {
		return true
	}

	// Generic fenced block
	if checkFencedBlock(response) {
		return true
	}

	logDebugInfo(response, trimmed)
	return false
}

// checkJSONCodeBlock checks for tool calls in JSON code blocks
func checkJSONCodeBlock(response string) bool {
	if !strings.Contains(response, "```json") {
		return false
	}

	start := strings.Index(response, "```json")
	if start < 0 {
		return false
	}

	start += 7
	end := strings.Index(response[start:], "```")
	if end <= 0 {
		return false
	}

	jsonContent := response[start : start+end]
	return strings.Contains(jsonContent, `"tool_calls"`)
}

// checkFencedBlock checks for tool calls in generic fenced blocks
func checkFencedBlock(response string) bool {
	if !strings.Contains(response, "```") {
		return false
	}

	start := strings.Index(response, "```")
	if start < 0 {
		return false
	}

	start += 3
	end := strings.Index(response[start:], "```")
	if end <= 0 {
		return false
	}

	block := response[start : start+end]
	return strings.Contains(block, `"tool_calls"`)
}

// logDebugInfo logs debug information for troubleshooting
func logDebugInfo(response, trimmed string) {
	if os.Getenv("LEDIT_DEBUG_TOOL_CALLS") != "1" {
		return
	}

	fmt.Printf("DEBUG: containsToolCall check failed for response: %s\n", response)
	fmt.Printf("DEBUG: trimmed: %s\n", trimmed)
	fmt.Printf("DEBUG: starts with {: %v\n", strings.HasPrefix(trimmed, "{"))
	fmt.Printf("DEBUG: contains tool_calls: %v\n", strings.Contains(response, `"tool_calls"`))
}

// parseToolCalls extracts tool calls from LLM response
func parseToolCalls(response string) ([]ToolCall, error) {
	if os.Getenv("LEDIT_DEBUG_TOOL_CALLS") == "1" {
		fmt.Printf("DEBUG: parseToolCalls called with response: %s\n", response)
	}

	// Try robust extraction first
	if calls := extractToolCallsRobust(response); len(calls) > 0 {
		return calls, nil
	}

	// Try standard JSON parsing
	return parseToolCallsFromJSON(response)
}

// parseToolCallsFromJSON attempts standard JSON parsing of tool calls
func parseToolCallsFromJSON(response string) ([]ToolCall, error) {
	cleanResponse := extractCleanJSON(response)
	
	var responseObj map[string]any
	if err := json.Unmarshal([]byte(cleanResponse), &responseObj); err != nil {
		return nil, fmt.Errorf("failed to parse response as JSON: %w", err)
	}

	return extractToolCallsFromObject(responseObj)
}

// extractCleanJSON extracts clean JSON from various response formats
func extractCleanJSON(response string) string {
	// Try to extract JSON from code blocks first
	if jsonContent := extractFromCodeBlock(response, "json"); jsonContent != "" {
		return jsonContent
	}

	// Try generic fenced blocks
	if jsonContent := extractFromFencedBlock(response); jsonContent != "" {
		return jsonContent
	}

	// Use the response as-is
	return strings.TrimSpace(response)
}

// extractFromCodeBlock extracts content from code blocks with specific language
func extractFromCodeBlock(response, language string) string {
	marker := "```" + language
	start := strings.Index(response, marker)
	if start < 0 {
		return ""
	}

	start += len(marker)
	end := strings.Index(response[start:], "```")
	if end <= 0 {
		return ""
	}

	return strings.TrimSpace(response[start : start+end])
}

// extractFromFencedBlock extracts content from generic fenced blocks
func extractFromFencedBlock(response string) string {
	start := strings.Index(response, "```")
	if start < 0 {
		return ""
	}

	// Skip the opening ```
	start += 3
	
	// Find the end of the first line (language specifier)
	lineEnd := strings.Index(response[start:], "\n")
	if lineEnd >= 0 {
		start += lineEnd + 1
	}

	end := strings.Index(response[start:], "```")
	if end <= 0 {
		return ""
	}

	return strings.TrimSpace(response[start : start+end])
}

// extractToolCallsFromObject extracts tool calls from parsed JSON object
func extractToolCallsFromObject(responseObj map[string]any) ([]ToolCall, error) {
	toolCallsAny, ok := responseObj["tool_calls"]
	if !ok {
		return nil, fmt.Errorf("no tool_calls found in response")
	}

	toolCallsList, ok := toolCallsAny.([]any)
	if !ok {
		return nil, fmt.Errorf("tool_calls is not an array")
	}

	var toolCalls []ToolCall
	for i, callAny := range toolCallsList {
		call, ok := callAny.(map[string]any)
		if !ok {
			continue
		}

		toolCall, valid := normalizeToolCall(call)
		if valid {
			toolCalls = append(toolCalls, toolCall)
		} else if os.Getenv("LEDIT_DEBUG_TOOL_CALLS") == "1" {
			fmt.Printf("DEBUG: Failed to normalize tool call %d: %+v\n", i, call)
		}
	}

	return toolCalls, nil
}

// normalizeToolCall converts various tool call formats to standard ToolCall
func normalizeToolCall(m map[string]any) (ToolCall, bool) {
	id, _ := m["id"].(string)
	typ, _ := m["type"].(string)

	fn := extractFunctionInfo(m)
	if fn == nil {
		return ToolCall{}, false
	}

	name, _ := fn["name"].(string)
	if name == "" {
		return ToolCall{}, false
	}

	argsStr := extractArguments(fn)

	return ToolCall{
		ID:        id,
		Type:      typ,
		Function:  ToolCallFunction{Name: name, Arguments: argsStr},
	}, true
}

// extractFunctionInfo extracts function information from tool call
func extractFunctionInfo(m map[string]any) map[string]any {
	if fn, ok := m["function"].(map[string]any); ok {
		return fn
	}
	if fn, ok := m["arguments"].(map[string]any); ok { // Kimi variant
		return fn
	}
	return nil
}

// extractArguments extracts and normalizes function arguments
func extractArguments(fn map[string]any) string {
	var rawArgs any
	if v, ok := fn["arguments"]; ok {
		rawArgs = v
	} else if v, ok := fn["parameters"]; ok {
		rawArgs = v
	}

	return normalizeArguments(rawArgs)
}

// normalizeArguments converts arguments to canonical JSON string format
func normalizeArguments(rawArgs any) string {
	switch a := rawArgs.(type) {
	case string:
		return normalizeStringArguments(a)
	case map[string]any:
		if b, err := json.Marshal(a); err == nil {
			return string(b)
		}
		return "{}"
	default:
		if b, err := json.Marshal(a); err == nil {
			return string(b)
		}
		return "{}"
	}
}

// normalizeStringArguments normalizes string-encoded arguments
func normalizeStringArguments(args string) string {
	candidate := strings.TrimSpace(args)
	var tmp map[string]any
	if json.Unmarshal([]byte(candidate), &tmp) == nil {
		if b, err := json.Marshal(tmp); err == nil {
			return string(b)
		}
		return candidate
	}
	return candidate
}

// extractToolCallsRobust provides robust tool call extraction for various formats
func extractToolCallsRobust(response string) []ToolCall {
	var calls []ToolCall

	// Strategy 1: Find JSON objects with tool call patterns
	calls = append(calls, findJSONToolCalls(response)...)

	// Strategy 2: Find individual tool call objects
	calls = append(calls, findIndividualToolCalls(response)...)

	return deduplicateToolCalls(calls)
}

// findJSONToolCalls finds tool calls in JSON format
func findJSONToolCalls(response string) []ToolCall {
	// Look for JSON objects that might contain tool calls
	jsonRegex := regexp.MustCompile(`\{[^{}]*"tool_calls"[^{}]*\}`)
	matches := jsonRegex.FindAllString(response, -1)

	var calls []ToolCall
	for _, match := range matches {
		if extracted := parseToolCallsFromMatch(match); len(extracted) > 0 {
			calls = append(calls, extracted...)
		}
	}

	return calls
}

// findIndividualToolCalls finds individual tool call objects
func findIndividualToolCalls(response string) []ToolCall {
	// Look for individual function call objects
	callRegex := regexp.MustCompile(`\{[^{}]*"name"[^{}]*"arguments"[^{}]*\}`)
	matches := callRegex.FindAllString(response, -1)

	var calls []ToolCall
	for _, match := range matches {
		if call := parseSingleToolCall(match); call != nil {
			calls = append(calls, *call)
		}
	}

	return calls
}

// parseToolCallsFromMatch parses tool calls from a regex match
func parseToolCallsFromMatch(match string) []ToolCall {
	var obj map[string]any
	if err := json.Unmarshal([]byte(match), &obj); err != nil {
		return nil
	}

	calls, _ := extractToolCallsFromObject(obj)
	return calls
}

// parseSingleToolCall parses a single tool call from JSON string
func parseSingleToolCall(objStr string) *ToolCall {
	var obj map[string]any
	if err := json.Unmarshal([]byte(objStr), &obj); err != nil {
		return nil
	}

	name, _ := obj["name"].(string)
	if name == "" {
		return nil
	}

	argsStr := normalizeArguments(obj["arguments"])

	return &ToolCall{
		ID:       generateToolCallID(),
		Type:     "function",
		Function: ToolCallFunction{Name: name, Arguments: argsStr},
	}
}

// deduplicateToolCalls removes duplicate tool calls
func deduplicateToolCalls(calls []ToolCall) []ToolCall {
	seen := make(map[string]bool)
	var unique []ToolCall

	for _, call := range calls {
		key := call.Function.Name + call.Function.Arguments
		if !seen[key] {
			seen[key] = true
			unique = append(unique, call)
		}
	}

	return unique
}

// generateToolCallID generates a unique ID for tool calls
func generateToolCallID() string {
	return fmt.Sprintf("call_%d", len(fmt.Sprintf("%p", &struct{}{})))
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// extractStringValue extracts a string value from text using key and quote markers
func extractStringValue(text, key, quote string) string {
	pattern := fmt.Sprintf(`%s%s:\s*%s([^%s]*)%s`, quote, key, quote, quote, quote)
	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(text)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}