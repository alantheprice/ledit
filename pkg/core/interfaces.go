package core

import (
	"context"
	"time"

	"github.com/alantheprice/ledit/pkg/config"
	"github.com/alantheprice/ledit/pkg/llm"
)

// Message represents a message for LLM communication (local definition to avoid import issues)
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// PromptBuilder defines the interface for building prompts for specific tasks
type PromptBuilder interface {
	// GetPromptID returns a unique identifier for this prompt type
	GetPromptID() string
	
	// BuildPrompt creates a prompt with the given context and parameters
	BuildPrompt(ctx *PromptContext) (*Prompt, error)
	
	// GetModelPreferences returns preferred models and fallbacks for this prompt type
	GetModelPreferences() ModelPreferences
	
	// GetOptimizationHints provides hints for prompt optimization
	GetOptimizationHints() OptimizationHints
}

// ContextBuilder defines the interface for building context for LLM requests
type ContextBuilder interface {
	// GetContextID returns a unique identifier for this context type
	GetContextID() string
	
	// BuildContext creates context from the given input
	BuildContext(input *ContextInput) (*LLMContext, error)
	
	// GetTokenBudget returns the token budget for this context type
	GetTokenBudget() TokenBudget
	
	// ShouldInclude determines if a piece of content should be included
	ShouldInclude(content string, metadata map[string]interface{}) bool
}

// OutputProcessor defines the interface for processing LLM outputs
type OutputProcessor interface {
	// GetProcessorID returns a unique identifier for this processor type
	GetProcessorID() string
	
	// ProcessOutput processes the raw LLM output and extracts structured data
	ProcessOutput(output *RawOutput) (*ProcessedOutput, error)
	
	// ValidateOutput checks if the output meets the requirements
	ValidateOutput(output *ProcessedOutput) error
	
	// GetExpectedFormat returns the expected output format
	GetExpectedFormat() OutputFormat
}

// ExecutionEngine coordinates prompt building, context creation, and output processing
type ExecutionEngine interface {
	// Execute runs the complete pipeline: prompt -> context -> LLM -> processing
	Execute(ctx context.Context, request *ExecutionRequest) (*ExecutionResult, error)
	
	// ExecuteWithOptimization uses model-specific optimized prompts
	ExecuteWithOptimization(ctx context.Context, request *ExecutionRequest) (*ExecutionResult, error)
}

// Core data structures

// PromptContext contains all data needed to build a prompt
type PromptContext struct {
	TaskType     TaskType
	UserInput    string
	Filename     string
	FileContent  string
	Instructions string
	Metadata     map[string]interface{}
	Config       *config.Config
}

// Prompt represents a complete prompt ready for LLM execution
type Prompt struct {
	ID           string
	SystemPrompt string
	UserPrompt   string
	Messages     []Message
	Temperature  float64
	MaxTokens    int
	ModelHints   ModelPreferences
}

// ContextInput contains raw inputs for context building
type ContextInput struct {
	UserIntent     string
	WorkspacePath  string
	TargetFiles    []string
	IncludeTypes   []string
	ExcludeTypes   []string
	MaxTokens      int
	Config         *config.Config
}

// LLMContext contains processed context ready for LLM
type LLMContext struct {
	WorkspaceInfo   string
	RelevantFiles   []FileContext
	Dependencies    []string
	ProjectStructure string
	TokenCount      int
}

// FileContext represents context for a specific file
type FileContext struct {
	Path         string
	Content      string
	Language     string
	Summary      string
	TokenCount   int
	Relevance    float64
}

// RawOutput contains the unprocessed LLM response
type RawOutput struct {
	Content     string
	Model       string
	TokenUsage  *llm.TokenUsage
	Metadata    map[string]interface{}
	Timestamp   time.Time
}

// ProcessedOutput contains the structured result after processing
type ProcessedOutput struct {
	TaskType     TaskType
	Success      bool
	Data         interface{}
	Files        []FileResult
	Actions      []Action
	Errors       []string
	Warnings     []string
	Metadata     map[string]interface{}
}

// FileResult represents a file operation result
type FileResult struct {
	Path      string
	Operation string // "create", "update", "delete"
	Content   string
	Diff      string
}

// Action represents an action to be taken
type Action struct {
	Type        string
	Description string
	Parameters  map[string]interface{}
}

// ExecutionRequest represents a complete execution request
type ExecutionRequest struct {
	TaskType        TaskType
	PromptBuilder   PromptBuilder
	ContextBuilder  ContextBuilder
	OutputProcessor OutputProcessor
	Input           interface{}
	Config          *config.Config
	Options         ExecutionOptions
}

// ExecutionResult contains the complete execution result
type ExecutionResult struct {
	Success        bool
	Output         *ProcessedOutput
	TokenUsage     *llm.TokenUsage
	Duration       time.Duration
	ModelUsed      string
	OptimizedUsed  bool
	Errors         []error
}

// ExecutionOptions contains execution configuration
type ExecutionOptions struct {
	UseOptimization  bool
	Timeout          time.Duration
	RetryCount       int
	ValidateOutput   bool
	Debug            bool
}

// Supporting types

type TaskType string

const (
	TaskTypeCodeGeneration TaskType = "code_generation"
	TaskTypeCodeReview     TaskType = "code_review"
	TaskTypeAgentAnalysis  TaskType = "agent_analysis"
	TaskTypeAgentExecution TaskType = "agent_execution"
	TaskTypeQuestion       TaskType = "question"
	TaskTypeRefactoring    TaskType = "refactoring"
	TaskTypeTesting        TaskType = "testing"
)

// ModelPreferences contains model preferences for a task
type ModelPreferences struct {
	Preferred  []string
	Fallbacks  []string
	Avoid      []string
	MinTokens  int
	MaxTokens  int
}

// OptimizationHints provides hints for prompt optimization
type OptimizationHints struct {
	TaskComplexity   string // "simple", "moderate", "complex"
	OutputStructure  string // "structured", "freeform", "code"
	ContextImportance string // "low", "medium", "high"
	ErrorTolerance   string // "strict", "moderate", "lenient"
}

// TokenBudget defines token allocation strategy
type TokenBudget struct {
	Total       int
	System      int
	Context     int
	UserPrompt  int
	Reserved    int
}

// OutputFormat defines the expected output structure
type OutputFormat struct {
	Type        string // "json", "markdown", "code", "mixed"
	Schema      interface{}
	Validators  []string
	Required    []string
}