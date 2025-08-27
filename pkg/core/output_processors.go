package core

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// CodeGenerationOutputProcessor processes code generation outputs
type CodeGenerationOutputProcessor struct {
	allowedExtensions []string
	maxFileSize       int
}

func NewCodeGenerationOutputProcessor() *CodeGenerationOutputProcessor {
	return &CodeGenerationOutputProcessor{
		allowedExtensions: []string{".go", ".py", ".js", ".ts", ".java", ".cpp", ".c", ".rs", ".md", ".txt", ".json", ".yaml", ".yml"},
		maxFileSize:       100000, // 100KB max file size
	}
}

func (cp *CodeGenerationOutputProcessor) GetProcessorID() string {
	return "code_generation_processor_v1"
}

func (cp *CodeGenerationOutputProcessor) ProcessOutput(output *RawOutput) (*ProcessedOutput, error) {
	result := &ProcessedOutput{
		TaskType: TaskTypeCodeGeneration,
		Success:  false,
		Data:     make(map[string]interface{}),
		Files:    []FileResult{},
		Actions:  []Action{},
		Errors:   []string{},
		Warnings: []string{},
		Metadata: make(map[string]interface{}),
	}

	// Extract code blocks from the output
	codeBlocks := cp.extractCodeBlocks(output.Content)
	if len(codeBlocks) == 0 {
		// Try to extract inline code or direct file content
		if fileResult := cp.extractDirectContent(output.Content); fileResult != nil {
			result.Files = []FileResult{*fileResult}
		} else {
			result.Errors = append(result.Errors, "No code blocks or file content found in output")
			return result, nil
		}
	} else {
		result.Files = codeBlocks
	}

	// Validate generated files
	for i, file := range result.Files {
		if err := cp.validateFile(&file); err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("File %s: %v", file.Path, err))
			// Mark file as potentially problematic but don't fail completely
			result.Files[i].Operation = "create_with_warning"
		}
	}

	// Extract any actions or commands mentioned
	actions := cp.extractActions(output.Content)
	result.Actions = actions

	// Set success if we have valid output
	result.Success = len(result.Files) > 0 || len(result.Actions) > 0

	// Add metadata
	result.Metadata["code_blocks_found"] = len(codeBlocks)
	result.Metadata["total_lines"] = cp.countTotalLines(result.Files)

	return result, nil
}

func (cp *CodeGenerationOutputProcessor) ValidateOutput(output *ProcessedOutput) error {
	if !output.Success {
		return fmt.Errorf("output processing was not successful")
	}

	if len(output.Files) == 0 && len(output.Actions) == 0 {
		return fmt.Errorf("no files or actions produced")
	}

	// Validate each file
	for _, file := range output.Files {
		if err := cp.validateFile(&file); err != nil {
			return fmt.Errorf("file validation failed for %s: %w", file.Path, err)
		}
	}

	return nil
}

func (cp *CodeGenerationOutputProcessor) GetExpectedFormat() OutputFormat {
	return OutputFormat{
		Type: "mixed",
		Schema: map[string]interface{}{
			"files": []map[string]interface{}{
				{
					"path":      "string",
					"operation": "string",
					"content":   "string",
				},
			},
			"actions": []map[string]interface{}{
				{
					"type":        "string",
					"description": "string",
				},
			},
		},
		Validators: []string{"file_syntax", "file_size", "extension_check"},
		Required:   []string{"files"},
	}
}

func (cp *CodeGenerationOutputProcessor) extractCodeBlocks(content string) []FileResult {
	var files []FileResult
	
	// Pattern to match code blocks with optional filename
	codeBlockPattern := regexp.MustCompile(`(?s)` + "`" + `{3}(\w+)?\s*(?:#\s*(.+?))?\n(.*?)` + "`" + `{3}`)
	matches := codeBlockPattern.FindAllStringSubmatch(content, -1)

	for i, match := range matches {
		language := match[1]
		filename := strings.TrimSpace(match[2])
		code := strings.TrimSpace(match[3])

		if code == "" {
			continue
		}

		// Generate filename if not provided
		if filename == "" {
			filename = fmt.Sprintf("generated_file_%d%s", i+1, cp.getExtensionForLanguage(language))
		}

		files = append(files, FileResult{
			Path:      filename,
			Operation: "create",
			Content:   code,
			Diff:      "", // Would be populated if updating existing file
		})
	}

	return files
}

func (cp *CodeGenerationOutputProcessor) extractDirectContent(content string) *FileResult {
	// Try to detect if the entire output is code
	lines := strings.Split(strings.TrimSpace(content), "\n")
	if len(lines) < 3 {
		return nil // Too short to be meaningful code
	}

	// Simple heuristics to detect if content looks like code
	codeIndicators := 0
	for _, line := range lines[:min(10, len(lines))] {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "{") || strings.Contains(line, "}") ||
			strings.Contains(line, "func ") || strings.Contains(line, "def ") ||
			strings.Contains(line, "class ") || strings.Contains(line, "import ") ||
			strings.Contains(line, "package ") || strings.HasPrefix(line, "//") ||
			strings.HasPrefix(line, "#") {
			codeIndicators++
		}
	}

	if codeIndicators >= 2 { // Looks like code
		return &FileResult{
			Path:      "generated_code.txt", // Default filename
			Operation: "create",
			Content:   content,
			Diff:      "",
		}
	}

	return nil
}

func (cp *CodeGenerationOutputProcessor) extractActions(content string) []Action {
	var actions []Action

	// Look for common action patterns
	actionPatterns := []struct {
		pattern string
		actionType string
	}{
		{`(?i)run\s+(.+)`, "command"},
		{`(?i)install\s+(.+)`, "install"},
		{`(?i)create\s+(?:directory|folder)\s+(.+)`, "create_directory"},
		{`(?i)delete\s+(.+)`, "delete"},
		{`(?i)move\s+(.+)\s+to\s+(.+)`, "move"},
	}

	for _, ap := range actionPatterns {
		re := regexp.MustCompile(ap.pattern)
		matches := re.FindAllStringSubmatch(content, -1)
		for _, match := range matches {
			action := Action{
				Type:        ap.actionType,
				Description: match[0], // Full match as description
				Parameters:  make(map[string]interface{}),
			}
			
			if len(match) > 1 {
				action.Parameters["target"] = match[1]
			}
			if len(match) > 2 {
				action.Parameters["destination"] = match[2]
			}
			
			actions = append(actions, action)
		}
	}

	return actions
}

func (cp *CodeGenerationOutputProcessor) validateFile(file *FileResult) error {
	if file.Path == "" {
		return fmt.Errorf("file path is empty")
	}

	if len(file.Content) > cp.maxFileSize {
		return fmt.Errorf("file size exceeds maximum allowed size")
	}

	// Check file extension
	hasAllowedExt := false
	for _, ext := range cp.allowedExtensions {
		if strings.HasSuffix(strings.ToLower(file.Path), ext) {
			hasAllowedExt = true
			break
		}
	}
	if !hasAllowedExt {
		return fmt.Errorf("file extension not in allowed list")
	}

	return nil
}

func (cp *CodeGenerationOutputProcessor) getExtensionForLanguage(language string) string {
	switch strings.ToLower(language) {
	case "go", "golang":
		return ".go"
	case "python", "py":
		return ".py"
	case "javascript", "js":
		return ".js"
	case "typescript", "ts":
		return ".ts"
	case "java":
		return ".java"
	case "cpp", "c++":
		return ".cpp"
	case "c":
		return ".c"
	case "rust", "rs":
		return ".rs"
	case "json":
		return ".json"
	case "yaml", "yml":
		return ".yaml"
	default:
		return ".txt"
	}
}

func (cp *CodeGenerationOutputProcessor) countTotalLines(files []FileResult) int {
	total := 0
	for _, file := range files {
		total += len(strings.Split(file.Content, "\n"))
	}
	return total
}

// AgentAnalysisOutputProcessor processes agent analysis outputs
type AgentAnalysisOutputProcessor struct{}

func NewAgentAnalysisOutputProcessor() *AgentAnalysisOutputProcessor {
	return &AgentAnalysisOutputProcessor{}
}

func (ap *AgentAnalysisOutputProcessor) GetProcessorID() string {
	return "agent_analysis_processor_v1"
}

func (ap *AgentAnalysisOutputProcessor) ProcessOutput(output *RawOutput) (*ProcessedOutput, error) {
	result := &ProcessedOutput{
		TaskType: TaskTypeAgentAnalysis,
		Success:  false,
		Data:     make(map[string]interface{}),
		Files:    []FileResult{},
		Actions:  []Action{},
		Errors:   []string{},
		Warnings: []string{},
		Metadata: make(map[string]interface{}),
	}

	// Try to parse as JSON first (structured output)
	todos, err := ap.parseJSONTodos(output.Content)
	if err != nil {
		// Fall back to text parsing
		todos = ap.parseTextTodos(output.Content)
	}

	if len(todos) == 0 {
		// Create a fallback todo from the output content itself
		fallbackTodo := TodoStruct{
			Content:     output.Content[:min(100, len(output.Content))], // First 100 chars
			Description: output.Content,
			Priority:    5,
			FilePath:    "",
		}
		todos = []TodoStruct{fallbackTodo}
	}

	// Convert todos to actions
	for _, todo := range todos {
		action := Action{
			Type:        "todo",
			Description: todo.Content,
			Parameters: map[string]interface{}{
				"content":     todo.Content,
				"description": todo.Description,
				"priority":    todo.Priority,
				"file_path":   todo.FilePath,
			},
		}
		result.Actions = append(result.Actions, action)
	}

	result.Success = len(result.Actions) > 0
	if dataMap, ok := result.Data.(map[string]interface{}); ok {
		dataMap["todos"] = todos
	}
	result.Metadata["todo_count"] = len(todos)

	return result, nil
}

func (ap *AgentAnalysisOutputProcessor) ValidateOutput(output *ProcessedOutput) error {
	if !output.Success {
		return fmt.Errorf("output processing was not successful")
	}

	if len(output.Actions) == 0 {
		return fmt.Errorf("no todos found in output")
	}

	// Validate todos
	for i, action := range output.Actions {
		if action.Type != "todo" {
			continue
		}
		
		content, ok := action.Parameters["content"].(string)
		if !ok || strings.TrimSpace(content) == "" {
			return fmt.Errorf("todo %d has empty content", i+1)
		}
	}

	return nil
}

func (ap *AgentAnalysisOutputProcessor) GetExpectedFormat() OutputFormat {
	return OutputFormat{
		Type: "json",
		Schema: []map[string]interface{}{
			{
				"content":     "string",
				"description": "string",
				"priority":    "number",
				"file_path":   "string",
			},
		},
		Validators: []string{"json_structure", "required_fields"},
		Required:   []string{"content", "description"},
	}
}

type TodoStruct struct {
	Content     string `json:"content"`
	Description string `json:"description"`
	Priority    int    `json:"priority"`
	FilePath    string `json:"file_path"`
}

func (ap *AgentAnalysisOutputProcessor) parseJSONTodos(content string) ([]TodoStruct, error) {
	// Try to extract JSON from the content
	var todos []TodoStruct
	
	// Look for JSON array in the content
	jsonStart := strings.Index(content, "[")
	jsonEnd := strings.LastIndex(content, "]")
	
	if jsonStart == -1 || jsonEnd == -1 || jsonEnd <= jsonStart {
		return nil, fmt.Errorf("no JSON array found")
	}
	
	jsonContent := content[jsonStart : jsonEnd+1]
	
	if err := json.Unmarshal([]byte(jsonContent), &todos); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}
	
	return todos, nil
}

func (ap *AgentAnalysisOutputProcessor) parseTextTodos(content string) []TodoStruct {
	var todos []TodoStruct
	
	// Simple text parsing for fallback
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		
		// Look for lines that look like todos
		if strings.HasPrefix(line, "-") || strings.HasPrefix(line, "*") || 
		   strings.HasPrefix(line, "1.") || strings.HasPrefix(line, "TODO:") ||
		   strings.Contains(strings.ToLower(line), "create") ||
		   strings.Contains(strings.ToLower(line), "write") ||
		   strings.Contains(strings.ToLower(line), "implement") {
			todo := TodoStruct{
				Content:     line,
				Description: line,
				Priority:    5, // Default priority
				FilePath:    "",
			}
			todos = append(todos, todo)
		}
	}
	
	return todos
}

// SimpleExecutionOutputProcessor processes any text output as successful
type SimpleExecutionOutputProcessor struct{}

func NewSimpleExecutionOutputProcessor() *SimpleExecutionOutputProcessor {
	return &SimpleExecutionOutputProcessor{}
}

func (sep *SimpleExecutionOutputProcessor) GetProcessorID() string {
	return "simple_execution_processor_v1"
}

func (sep *SimpleExecutionOutputProcessor) ProcessOutput(output *RawOutput) (*ProcessedOutput, error) {
	result := &ProcessedOutput{
		TaskType: TaskTypeAgentExecution,
		Success:  true, // Always mark as successful for execution
		Data:     map[string]interface{}{"response": output.Content},
		Files:    []FileResult{},
		Actions:  []Action{{Type: "analysis", Description: output.Content}},
		Errors:   []string{},
		Warnings: []string{},
		Metadata: map[string]interface{}{"content_length": len(output.Content)},
	}

	return result, nil
}

func (sep *SimpleExecutionOutputProcessor) ValidateOutput(output *ProcessedOutput) error {
	// Simple execution always passes validation
	return nil
}

func (sep *SimpleExecutionOutputProcessor) GetExpectedFormat() OutputFormat {
	return OutputFormat{
		Type: "text",
		Schema: map[string]interface{}{
			"response": "string",
		},
		Validators: []string{},
		Required:   []string{},
	}
}

// Helper function
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}