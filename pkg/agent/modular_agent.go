package agent

import (
	"context"
	"fmt"
	"time"

	"github.com/alantheprice/ledit/pkg/config"
	"github.com/alantheprice/ledit/pkg/core"
	ui "github.com/alantheprice/ledit/pkg/ui"
	"github.com/alantheprice/ledit/pkg/utils"
)

// ModularAgentService provides the new architecture-based agent implementation
type ModularAgentService struct {
	coreService *core.CoreService
	config      *config.Config
}

func NewModularAgentService(cfg *config.Config) *ModularAgentService {
	return &ModularAgentService{
		coreService: core.NewCoreService(cfg),
		config:      cfg,
	}
}

// RunModularAgent runs the agent using the new modular architecture
func RunModularAgent(userIntent string, skipPrompt bool, model string) error {
	startTime := time.Now()

	cfg, err := config.LoadOrInitConfig(skipPrompt)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if model != "" {
		cfg.EditingModel = model
	}
	cfg.SkipPrompt = skipPrompt
	cfg.FromAgent = true

	// Initialize the modular agent service
	agentService := NewModularAgentService(cfg)

	// Show mode info
	if ui.IsUIActive() {
		ui.Logf("🎯 %s", userIntent)
	} else {
		ui.Out().Print("🤖 Modular Agent Mode\n")
		ui.Out().Printf("🎯 Intent: %s\n", userIntent)
	}

	// Step 1: Analyze the user request using the modular architecture
	ui.Out().Print("🧭 Analyzing request and creating action plan...\n")

	ctx := context.Background()
	analysisResult, err := agentService.coreService.AnalyzeAgentRequest(ctx, userIntent)
	if err != nil {
		return fmt.Errorf("failed to analyze user request: %w", err)
	}

	if !analysisResult.Success {
		return fmt.Errorf("request analysis failed")
	}

	if len(analysisResult.Todos) == 0 {
		ui.Out().Print("⚠️ No actionable todos were created from the analysis\n")
		return fmt.Errorf("no actionable todos could be created")
	}

	ui.Out().Printf("✅ Created %d todos (optimized: %v)\n", len(analysisResult.Todos), analysisResult.Optimized)

	// Display the todos
	for i, todo := range analysisResult.Todos {
		ui.Out().Printf("📋 %d. %s\n", i+1, todo.Content)
		if todo.Description != "" && todo.Description != todo.Content {
			ui.Out().Printf("    └─ %s\n", todo.Description)
		}
	}

	// Step 2: Execute each todo using the execution engine
	ui.Out().Print("\n🚀 Starting todo execution...\n")

	completedCount := 0
	for i, todo := range analysisResult.Todos {
		ui.Out().Printf("📋 Executing todo %d/%d: %s\n", i+1, len(analysisResult.Todos), todo.Content)

		// Execute todo using existing agent infrastructure
		agentTodo := TodoItem{
			ID:          todo.ID,
			Content:     todo.Content,
			Description: todo.Description,
			Priority:    todo.Priority,
			Status:      "in_progress",
			FilePath:    todo.FilePath,
		}
		
		// Create a simplified agent context for execution
		agentCtx := &SimplifiedAgentContext{
			UserIntent:      userIntent,
			Config:          cfg,
			SkipPrompt:      skipPrompt,
			CurrentTodo:     &agentTodo,
			Logger:          utils.GetLogger(skipPrompt),
			AnalysisResults: make(map[string]string),
			Todos:           []TodoItem{agentTodo},
		}
		
		err := executeTodoWithSmartRetry(agentCtx, &agentTodo)
		if err != nil {
			ui.Out().Printf("❌ Todo %d failed: %v\n", i+1, err)
			// Continue with other todos rather than failing completely
			continue
		}

		// Mark as successful if no error
		ui.Out().Printf("✅ Todo %d completed\n", i+1)
		completedCount++
	}

	// Final summary
	duration := time.Since(startTime)
	ui.Out().Printf("\n🎉 Agent completed! %d/%d todos successful in %v\n", 
		completedCount, len(analysisResult.Todos), duration.Round(time.Second))

	if completedCount == 0 {
		return fmt.Errorf("no todos were completed successfully")
	}

	return nil
}

// LegacyAgentAdapter adapts the new modular system to work with existing agent interface
type LegacyAgentAdapter struct {
	service *ModularAgentService
}

func NewLegacyAgentAdapter(cfg *config.Config) *LegacyAgentAdapter {
	return &LegacyAgentAdapter{
		service: NewModularAgentService(cfg),
	}
}

// CreateTodosUsingModularSystem creates todos using the new modular architecture
func (adapter *LegacyAgentAdapter) CreateTodosUsingModularSystem(ctx *SimplifiedAgentContext) error {
	analysisCtx := context.Background()
	
	// Use the modular system to analyze and create todos
	result, err := adapter.service.coreService.AnalyzeAgentRequest(analysisCtx, ctx.UserIntent)
	if err != nil {
		return fmt.Errorf("modular analysis failed: %w", err)
	}

	if !result.Success {
		return fmt.Errorf("modular analysis was not successful")
	}

	// Convert core.TodoItem to agent.TodoItem format
	ctx.Todos = make([]TodoItem, 0, len(result.Todos))
	for i, coreTodo := range result.Todos {
		agentTodo := TodoItem{
			ID:          coreTodo.ID,
			Content:     coreTodo.Content,
			Description: coreTodo.Description,
			Priority:    coreTodo.Priority,
			Status:      coreTodo.Status,
			FilePath:    coreTodo.FilePath,
		}
		ctx.Todos = append(ctx.Todos, agentTodo)
		
		// Log the created todo
		ctx.Logger.LogProcessStep(fmt.Sprintf("📋 Created todo %d: %s", i+1, agentTodo.Content))
	}

	return nil
}

// Integration helper function to replace existing createTodos in a backward-compatible way
func CreateTodosWithModularArchitecture(ctx *SimplifiedAgentContext) error {
	adapter := NewLegacyAgentAdapter(ctx.Config)
	return adapter.CreateTodosUsingModularSystem(ctx)
}