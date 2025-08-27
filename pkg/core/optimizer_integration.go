package core

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// PromptOptimizer integrates with the prompt optimization system
type PromptOptimizer struct {
	optimizerPath string
	cacheDir      string
	enabled       bool
}

// NewPromptOptimizer creates a new prompt optimizer instance
func NewPromptOptimizer() *PromptOptimizer {
	// Check if prompt optimizer is available
	optimizerPath := "./prompt_optimizer_advanced"
	if _, err := os.Stat(optimizerPath); os.IsNotExist(err) {
		// Try alternative paths
		if path, err := exec.LookPath("prompt_optimizer_advanced"); err == nil {
			optimizerPath = path
		} else {
			// Optimizer not available
			return &PromptOptimizer{enabled: false}
		}
	}

	cacheDir := filepath.Join(".ledit", "optimized_prompts")
	os.MkdirAll(cacheDir, 0755)

	return &PromptOptimizer{
		optimizerPath: optimizerPath,
		cacheDir:      cacheDir,
		enabled:       true,
	}
}

// OptimizePrompt optimizes a prompt for specific models and tasks
func (po *PromptOptimizer) OptimizePrompt(prompt *Prompt, taskType TaskType, models []string) (*Prompt, error) {
	if !po.enabled {
		// Return original prompt if optimizer is not available
		return prompt, nil
	}

	// Check cache first
	cacheKey := po.getCacheKey(prompt.ID, taskType, models)
	if cached := po.getFromCache(cacheKey); cached != nil {
		return cached, nil
	}

	// Prepare optimization request
	request := OptimizationRequest{
		PromptID:     prompt.ID,
		TaskType:     string(taskType),
		Models:       models,
		SystemPrompt: prompt.SystemPrompt,
		UserPrompt:   prompt.UserPrompt,
		Temperature:  prompt.Temperature,
		MaxTokens:    prompt.MaxTokens,
		Hints:        po.getOptimizationHints(taskType),
	}

	// Run optimization
	optimizedPrompt, err := po.runOptimization(request)
	if err != nil {
		// Log error but return original prompt
		fmt.Printf("Prompt optimization failed: %v\n", err)
		return prompt, nil
	}

	// Cache the result
	po.saveToCache(cacheKey, optimizedPrompt)

	return optimizedPrompt, nil
}

// OptimizationRequest represents a request to optimize a prompt
type OptimizationRequest struct {
	PromptID     string                 `json:"prompt_id"`
	TaskType     string                 `json:"task_type"`
	Models       []string               `json:"models"`
	SystemPrompt string                 `json:"system_prompt"`
	UserPrompt   string                 `json:"user_prompt"`
	Temperature  float64                `json:"temperature"`
	MaxTokens    int                    `json:"max_tokens"`
	Hints        map[string]interface{} `json:"hints"`
}

// OptimizationResponse represents the response from optimization
type OptimizationResponse struct {
	Success          bool                   `json:"success"`
	OptimizedPrompt  *Prompt                `json:"optimized_prompt"`
	Improvements     []string               `json:"improvements"`
	Metadata         map[string]interface{} `json:"metadata"`
	Error           string                 `json:"error,omitempty"`
}

func (po *PromptOptimizer) runOptimization(request OptimizationRequest) (*Prompt, error) {
	// Create temporary config file for optimization
	configFile, err := po.createOptimizationConfig(request)
	if err != nil {
		return nil, fmt.Errorf("failed to create optimization config: %w", err)
	}
	defer os.Remove(configFile)

	// Run the prompt optimizer
	cmd := exec.Command(po.optimizerPath,
		"--config", configFile,
		"--type", "text_replacement",
		"--iterations", "2", // Limit iterations for performance
		"--model-mapping",
		"--models", strings.Join(request.Models, ","),
	)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("optimizer execution failed: %w", err)
	}

	// Parse optimization result
	var response OptimizationResponse
	if err := json.Unmarshal(output, &response); err != nil {
		// If JSON parsing fails, try to extract improved prompt from output
		return po.parseTextualResponse(string(output), request)
	}

	if !response.Success {
		return nil, fmt.Errorf("optimization failed: %s", response.Error)
	}

	return response.OptimizedPrompt, nil
}

func (po *PromptOptimizer) createOptimizationConfig(request OptimizationRequest) (string, error) {
	config := map[string]interface{}{
		"task_type":      request.TaskType,
		"target_models":  request.Models,
		"system_prompt":  request.SystemPrompt,
		"user_template":  request.UserPrompt,
		"temperature":    request.Temperature,
		"max_tokens":     request.MaxTokens,
		"optimization_hints": request.Hints,
		"test_cases": po.generateTestCases(request.TaskType),
	}

	configData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return "", err
	}

	configFile := filepath.Join(po.cacheDir, fmt.Sprintf("opt_config_%s.json", request.PromptID))
	if err := os.WriteFile(configFile, configData, 0644); err != nil {
		return "", err
	}

	return configFile, nil
}

func (po *PromptOptimizer) parseTextualResponse(output string, request OptimizationRequest) (*Prompt, error) {
	// Simple parsing for text-based optimization output
	// Look for improved prompts in the output

	lines := strings.Split(output, "\n")
	var systemPrompt, userPrompt string
	var inSystemPrompt, inUserPrompt bool

	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		if strings.Contains(line, "OPTIMIZED SYSTEM PROMPT:") {
			inSystemPrompt = true
			inUserPrompt = false
			continue
		}
		
		if strings.Contains(line, "OPTIMIZED USER PROMPT:") {
			inUserPrompt = true
			inSystemPrompt = false
			continue
		}
		
		if strings.HasPrefix(line, "---") || strings.HasPrefix(line, "===") {
			inSystemPrompt = false
			inUserPrompt = false
			continue
		}
		
		if inSystemPrompt && line != "" {
			if systemPrompt != "" {
				systemPrompt += "\n"
			}
			systemPrompt += line
		}
		
		if inUserPrompt && line != "" {
			if userPrompt != "" {
				userPrompt += "\n"
			}
			userPrompt += line
		}
	}

	// If we found optimized prompts, use them
	if systemPrompt != "" || userPrompt != "" {
		optimized := &Prompt{
			ID:           request.PromptID + "_optimized",
			SystemPrompt: systemPrompt,
			UserPrompt:   userPrompt,
			Temperature:  request.Temperature,
			MaxTokens:    request.MaxTokens,
		}
		
		// Use original prompts as fallback
		if optimized.SystemPrompt == "" {
			optimized.SystemPrompt = request.SystemPrompt
		}
		if optimized.UserPrompt == "" {
			optimized.UserPrompt = request.UserPrompt
		}
		
		return optimized, nil
	}

	return nil, fmt.Errorf("no optimized prompts found in output")
}

func (po *PromptOptimizer) generateTestCases(taskType string) []map[string]interface{} {
	switch TaskType(taskType) {
	case TaskTypeCodeGeneration:
		return []map[string]interface{}{
			{
				"input":    "Create a function to calculate fibonacci numbers",
				"expected": "function implementation with proper error handling",
			},
			{
				"input":    "Add error handling to this function",
				"expected": "enhanced function with try-catch or error checks",
			},
		}
	case TaskTypeAgentAnalysis:
		return []map[string]interface{}{
			{
				"input":    "Build a REST API for user management",
				"expected": "structured todos for API development",
			},
			{
				"input":    "Fix the login bug in the authentication system",
				"expected": "specific todos for debugging and fixing",
			},
		}
	default:
		return []map[string]interface{}{
			{
				"input":    "generic test case",
				"expected": "appropriate response for task type",
			},
		}
	}
}

func (po *PromptOptimizer) getOptimizationHints(taskType TaskType) map[string]interface{} {
	hints := make(map[string]interface{})
	
	switch taskType {
	case TaskTypeCodeGeneration:
		hints["focus"] = "code quality and completeness"
		hints["output_format"] = "structured code with comments"
		hints["context_importance"] = "high"
	case TaskTypeAgentAnalysis:
		hints["focus"] = "structured task breakdown"
		hints["output_format"] = "JSON array of todos"
		hints["context_importance"] = "medium"
	case TaskTypeAgentExecution:
		hints["focus"] = "precise execution and tool usage"
		hints["output_format"] = "clear action descriptions"
		hints["context_importance"] = "low"
	}
	
	return hints
}

func (po *PromptOptimizer) getCacheKey(promptID string, taskType TaskType, models []string) string {
	modelsStr := strings.Join(models, "_")
	return fmt.Sprintf("%s_%s_%s", promptID, taskType, modelsStr)
}

func (po *PromptOptimizer) getFromCache(cacheKey string) *Prompt {
	cachePath := filepath.Join(po.cacheDir, cacheKey+".json")
	
	data, err := os.ReadFile(cachePath)
	if err != nil {
		return nil
	}
	
	var prompt Prompt
	if err := json.Unmarshal(data, &prompt); err != nil {
		return nil
	}
	
	// Check if cache is still valid (24 hours)
	if info, err := os.Stat(cachePath); err == nil {
		if time.Since(info.ModTime()) > 24*time.Hour {
			os.Remove(cachePath) // Remove stale cache
			return nil
		}
	}
	
	return &prompt
}

func (po *PromptOptimizer) saveToCache(cacheKey string, prompt *Prompt) {
	cachePath := filepath.Join(po.cacheDir, cacheKey+".json")
	
	data, err := json.MarshalIndent(prompt, "", "  ")
	if err != nil {
		return
	}
	
	os.WriteFile(cachePath, data, 0644)
}

// Enhanced ExecutionEngine with optimization
type OptimizedExecutionEngine struct {
	*DefaultExecutionEngine
	optimizer *PromptOptimizer
}

func NewOptimizedExecutionEngine() *OptimizedExecutionEngine {
	return &OptimizedExecutionEngine{
		DefaultExecutionEngine: NewExecutionEngine(),
		optimizer:              NewPromptOptimizer(),
	}
}

func (e *OptimizedExecutionEngine) ExecuteWithOptimization(ctx context.Context, request *ExecutionRequest) (*ExecutionResult, error) {
	// Build the initial prompt
	promptCtx := &PromptContext{
		TaskType:     request.TaskType,
		UserInput:    fmt.Sprintf("%v", request.Input),
		Instructions: fmt.Sprintf("%v", request.Input),
		Config:       request.Config,
	}

	prompt, err := request.PromptBuilder.BuildPrompt(promptCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to build prompt: %w", err)
	}

	// Optimize the prompt if optimization is enabled
	if request.Options.UseOptimization {
		models := prompt.ModelHints.Preferred
		if len(models) == 0 {
			models = []string{"openai:gpt-4o"} // Default model
		}

		optimizedPrompt, err := e.optimizer.OptimizePrompt(prompt, request.TaskType, models)
		if err != nil {
			// Log warning but continue with original prompt
			fmt.Printf("Prompt optimization failed, using original: %v\n", err)
		} else {
			prompt = optimizedPrompt
		}
	}

	// Create a new execution request with the optimized prompt
	optimizedRequest := *request
	optimizedRequest.PromptBuilder = &OptimizedPromptBuilder{
		optimizedPrompt: prompt,
		originalBuilder: request.PromptBuilder,
	}

	// Continue with normal execution using the optimized prompt
	result, err := e.DefaultExecutionEngine.Execute(ctx, &optimizedRequest)
	if err != nil {
		return result, err
	}

	// Mark as optimized if we used optimization
	if result != nil && request.Options.UseOptimization {
		result.OptimizedUsed = true
	}

	return result, nil
}

// OptimizedPromptBuilder wraps an optimized prompt
type OptimizedPromptBuilder struct {
	optimizedPrompt *Prompt
	originalBuilder PromptBuilder
}

func (opb *OptimizedPromptBuilder) GetPromptID() string {
	return opb.optimizedPrompt.ID
}

func (opb *OptimizedPromptBuilder) BuildPrompt(ctx *PromptContext) (*Prompt, error) {
	// Return the already optimized prompt
	return opb.optimizedPrompt, nil
}

func (opb *OptimizedPromptBuilder) GetModelPreferences() ModelPreferences {
	return opb.originalBuilder.GetModelPreferences()
}

func (opb *OptimizedPromptBuilder) GetOptimizationHints() OptimizationHints {
	return opb.originalBuilder.GetOptimizationHints()
}