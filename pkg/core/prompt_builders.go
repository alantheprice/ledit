package core

import (
	"fmt"
	"strings"
	
	"github.com/alantheprice/ledit/pkg/prompts"
)

// CodeGenerationPromptBuilder builds prompts for code generation tasks
type CodeGenerationPromptBuilder struct {
	promptManager *prompts.PromptManager
}

func NewCodeGenerationPromptBuilder() *CodeGenerationPromptBuilder {
	return &CodeGenerationPromptBuilder{
		promptManager: prompts.NewPromptManager(),
	}
}

func (cb *CodeGenerationPromptBuilder) GetPromptID() string {
	return "code_generation_v1"
}

func (cb *CodeGenerationPromptBuilder) BuildPrompt(ctx *PromptContext) (*Prompt, error) {
	// Load optimized system prompt
	systemPromptText := cb.getDefaultSystemPrompt()
	
	// Build user prompt with context
	userPrompt := cb.buildUserPrompt(ctx)

	return &Prompt{
		ID:           cb.GetPromptID(),
		SystemPrompt: systemPromptText,
		UserPrompt:   userPrompt,
		Messages: []Message{
			{Role: "system", Content: systemPromptText},
			{Role: "user", Content: userPrompt},
		},
		Temperature: 0.1, // Low temperature for consistent code generation
		MaxTokens:   4000,
		ModelHints:  cb.GetModelPreferences(),
	}, nil
}

func (cb *CodeGenerationPromptBuilder) GetModelPreferences() ModelPreferences {
	return ModelPreferences{
		Preferred: []string{
			"deepinfra:Qwen/Qwen2.5-Coder-32B-Instruct",
			"openai:gpt-4o",
			"deepinfra:google/gemini-2.5-flash",
		},
		Fallbacks: []string{
			"openai:gpt-4-turbo",
			"deepinfra:meta-llama/Meta-Llama-3.1-70B-Instruct",
		},
		MinTokens: 1000,
		MaxTokens: 8000,
	}
}

func (cb *CodeGenerationPromptBuilder) GetOptimizationHints() OptimizationHints {
	return OptimizationHints{
		TaskComplexity:    "moderate",
		OutputStructure:   "code",
		ContextImportance: "high",
		ErrorTolerance:    "strict",
	}
}

func (cb *CodeGenerationPromptBuilder) buildUserPrompt(ctx *PromptContext) string {
	var parts []string

	// Add file context if available
	if ctx.Filename != "" && ctx.FileContent != "" {
		parts = append(parts, fmt.Sprintf("File: %s\n```\n%s\n```", ctx.Filename, ctx.FileContent))
	}

	// Add instructions
	parts = append(parts, fmt.Sprintf("Instructions: %s", ctx.Instructions))

	// Add specific requirements based on task type
	if ctx.TaskType == TaskTypeCodeGeneration {
		parts = append(parts, cb.getCodeGenerationRequirements())
	}

	return strings.Join(parts, "\n\n")
}

func (cb *CodeGenerationPromptBuilder) getDefaultSystemPrompt() string {
	content, err := cb.promptManager.LoadPrompt("code_generation_system.txt")
	if err != nil {
		// Fallback to hardcoded prompt if file not found
		return `You are an expert software developer. Generate high-quality, clean, and efficient code based on the given instructions.

Requirements:
- Follow best practices and coding standards
- Include appropriate error handling
- Add clear comments where necessary
- Ensure code is maintainable and readable
- Provide complete, working implementations

Output the code in the appropriate format for the requested changes.`
	}
	return content
}

func (cb *CodeGenerationPromptBuilder) getCodeGenerationRequirements() string {
	return `Requirements:
- Output clean, production-ready code
- Follow established patterns in the existing codebase
- Include proper error handling
- Add documentation where appropriate
- Ensure compatibility with existing code`
}

// AgentAnalysisPromptBuilder builds prompts for agent analysis tasks
type AgentAnalysisPromptBuilder struct {
	promptManager *prompts.PromptManager
}

func NewAgentAnalysisPromptBuilder() *AgentAnalysisPromptBuilder {
	return &AgentAnalysisPromptBuilder{
		promptManager: prompts.NewPromptManager(),
	}
}

func (ab *AgentAnalysisPromptBuilder) GetPromptID() string {
	return "agent_analysis_v1"
}

func (ab *AgentAnalysisPromptBuilder) BuildPrompt(ctx *PromptContext) (*Prompt, error) {
	systemPromptText := ab.getDefaultSystemPrompt()
	userPrompt := ab.buildAnalysisPrompt(ctx)

	return &Prompt{
		ID:           ab.GetPromptID(),
		SystemPrompt: systemPromptText,
		UserPrompt:   userPrompt,
		Messages: []Message{
			{Role: "system", Content: systemPromptText},
			{Role: "user", Content: userPrompt},
		},
		Temperature: 0.2, // Slightly higher for creative problem solving
		MaxTokens:   3000,
		ModelHints:  ab.GetModelPreferences(),
	}, nil
}

func (ab *AgentAnalysisPromptBuilder) GetModelPreferences() ModelPreferences {
	return ModelPreferences{
		Preferred: []string{
			"openai:gpt-4o",
			"deepinfra:google/gemini-2.5-pro",
			"deepinfra:Qwen/Qwen2.5-72B-Instruct",
		},
		Fallbacks: []string{
			"openai:gpt-4-turbo",
			"deepinfra:meta-llama/Meta-Llama-3.1-70B-Instruct",
		},
		MinTokens: 500,
		MaxTokens: 6000,
	}
}

func (ab *AgentAnalysisPromptBuilder) GetOptimizationHints() OptimizationHints {
	return OptimizationHints{
		TaskComplexity:    "complex",
		OutputStructure:   "structured",
		ContextImportance: "high",
		ErrorTolerance:    "moderate",
	}
}

func (ab *AgentAnalysisPromptBuilder) buildAnalysisPrompt(ctx *PromptContext) string {
	return fmt.Sprintf(`User Request: %s

Analyze this request and create a structured plan with actionable todos.

Use available tools to:
- Read files to understand the codebase
- Search for relevant code patterns  
- Validate assumptions about file locations
- Gather context needed for implementation

Return a JSON array of todos with this structure:
[{
  "content": "Brief task description",
  "description": "Detailed description with context",
  "priority": 1-10,
  "file_path": "optional/file/path.ext"
}]`, ctx.UserInput)
}

func (ab *AgentAnalysisPromptBuilder) getDefaultSystemPrompt() string {
	content, err := ab.promptManager.LoadPrompt("agent_analysis_system.txt")
	if err != nil {
		// Fallback to hardcoded prompt if file not found
		return `You are an expert development assistant. Analyze user requests and break them down into actionable todos.

Use tools to understand the codebase structure and validate your assumptions. Create specific, actionable tasks that can be executed systematically.

Always return valid JSON with structured todos.`
	}
	return content
}

// AgentExecutionPromptBuilder builds prompts for agent execution tasks
type AgentExecutionPromptBuilder struct {
	promptManager *prompts.PromptManager
}

func NewAgentExecutionPromptBuilder() *AgentExecutionPromptBuilder {
	return &AgentExecutionPromptBuilder{
		promptManager: prompts.NewPromptManager(),
	}
}

func (ae *AgentExecutionPromptBuilder) GetPromptID() string {
	return "agent_execution_v1"
}

func (ae *AgentExecutionPromptBuilder) BuildPrompt(ctx *PromptContext) (*Prompt, error) {
	systemPrompt := ae.getDefaultSystemPrompt()
	userPrompt := ae.buildExecutionPrompt(ctx)

	return &Prompt{
		ID:           ae.GetPromptID(),
		SystemPrompt: systemPrompt,
		UserPrompt:   userPrompt,
		Messages: []Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Temperature: 0.1, // Low temperature for precise execution
		MaxTokens:   2000,
		ModelHints:  ae.GetModelPreferences(),
	}, nil
}

func (ae *AgentExecutionPromptBuilder) GetModelPreferences() ModelPreferences {
	return ModelPreferences{
		Preferred: []string{
			"openai:gpt-4o",
			"deepinfra:Qwen/Qwen2.5-Coder-32B-Instruct",
			"deepinfra:google/gemini-2.5-flash",
		},
		Fallbacks: []string{
			"openai:gpt-4-turbo",
			"deepinfra:meta-llama/Meta-Llama-3.1-70B-Instruct",
		},
		MinTokens: 200,
		MaxTokens: 4000,
	}
}

func (ae *AgentExecutionPromptBuilder) GetOptimizationHints() OptimizationHints {
	return OptimizationHints{
		TaskComplexity:    "moderate",
		OutputStructure:   "mixed",
		ContextImportance: "medium",
		ErrorTolerance:    "strict",
	}
}

func (ae *AgentExecutionPromptBuilder) buildExecutionPrompt(ctx *PromptContext) string {
	return fmt.Sprintf(`Task: %s

Execute this task using available tools. Be precise and thorough in your execution.

%s`, ctx.Instructions, ae.getExecutionGuidelines())
}

func (ae *AgentExecutionPromptBuilder) getDefaultSystemPrompt() string {
	content, err := ae.promptManager.LoadPrompt("agent_execution_system.txt")
	if err != nil {
		// Fallback to hardcoded prompt if file not found
		return `You are a precise task executor. Use available tools to complete tasks efficiently and accurately.

Follow these principles:
- Use tools appropriately for each task type
- Validate your work when possible
- Provide clear feedback on your progress
- Handle errors gracefully`
	}
	return content
}

func (ae *AgentExecutionPromptBuilder) getExecutionGuidelines() string {
	return `Guidelines:
- Use read_file to examine existing code
- Use grep_search to find relevant patterns
- Use run_shell_command for system operations
- Validate changes before completing
- Report any issues encountered`
}