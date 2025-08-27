package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/alantheprice/ledit/pkg/config"
	"github.com/alantheprice/ledit/pkg/editor"
	"github.com/alantheprice/ledit/pkg/prompts"
	"github.com/alantheprice/ledit/pkg/providers"
	tuiPkg "github.com/alantheprice/ledit/pkg/tui"
	ui "github.com/alantheprice/ledit/pkg/ui"
	"github.com/alantheprice/ledit/pkg/utils"
)

// codeCmd represents the code command using the new unified framework
var codeCmd = createCodeCommand()

func createCodeCommand() *BaseCommand {
	cmd := NewBaseCommand(
		"code [instructions]",
		"Generate updated code based on instructions",
		`Processes a file or generates new files based on natural language instructions using an LLM.

The code command can run in two modes:

1. **Interactive Mode** (default when no instructions provided):
   - Run "ledit code" to start interactive TUI mode
   - Enter code generation requests in real-time
   - Watch progress and see live updates
   - Perfect for iterative code development

2. **Direct Mode** (when instructions provided):
   - Run "ledit code \"your instructions\"" for one-shot execution
   - Ideal for scripting, automation, and command-line tools
   - No UI interface, direct output to console

When using the --image flag, ensure your model supports vision input. Vision-capable models include:
  - openai:gpt-4o, openai:gpt-4-turbo, openai:gpt-4-vision-preview
  - deepinfra:google/gemini-2.5-flash, deepinfra:google/gemini-2.5-pro

Examples:
  # Interactive mode (default)
  ledit code
  
  # Direct mode (for automation/scripting)
  ledit code "Add error handling to the main function"
  ledit code --filename main.go "Refactor this function to be more efficient"
  ledit code --model gpt-4 --skip-prompt "Generate a REST API endpoint"
  ledit code --image screenshot.png "Create a UI component based on this design"`,
	)

	// Add custom flags specific to code command
	cmd.AddCustomFlag("filename", "f", "", "The filename to process (optional)")
	cmd.AddCustomFlag("image", "i", "", "Path to an image file to use as UI reference")
	cmd.AddCustomFlag("ui", "", "", "Start interactive TUI mode for real-time code generation")

	// Set the command execution function
	cmd.SetRunFunc(executeCodeCommand)

	return cmd
}

func executeCodeCommand(cfg *CommandConfig, args []string) error {
	// Extract instructions from arguments (filter out --ui flag if present)
	instructions := extractInstructions(args)
	
	// If no instructions provided, default to interactive TUI mode
	if instructions == "" {
		return startInteractiveCodeTUI()
	}

	// Log the original user prompt for direct mode
	utils.LogUserPrompt(instructions)

	// Get custom flag values
	filename := getFilenameFlag(args)
	imagePath := getImageFlag(args)

	// Configure UI output routing
	if cfg.Config != nil {
		cfg.Config.SkipPrompt = cfg.SkipPrompt
	}

	// Publish model information to UI if active
	if ui.IsUIActive() && cfg.Config != nil {
		ui.PublishModel(cfg.Config.EditingModel)
	}

	// Publish status update
	ui.PublishStatus("Processing code generation request...")

	// Show processing message only in console mode - UI shows progress differently
	ui.PrintContext(prompts.ProcessingCodeGeneration()+"\n", false)
	startTime := time.Now()

	// Execute code generation with enhanced UI integration
	result, err := executeCodeGenerationWithUI(filename, instructions, cfg.Config, imagePath)
	if err != nil {
		ui.PublishStatus("Code generation failed")
		return fmt.Errorf("code generation failed: %w", err)
	}

	duration := time.Since(startTime)

	// Update UI with completion status
	ui.PublishStatus(fmt.Sprintf("Code generation completed in %v", duration))

	// Display completion message with timing
	ui.PrintContext(prompts.CodeGenerationFinished(duration), false)

	// Display and publish token usage if available
	if result != nil && result.TokenUsage != nil {
		if err := displayTokenUsage(cfg, result.TokenUsage); err != nil {
			// Log error but don't fail the command
			if cfg.Logger != nil {
				cfg.Logger.LogError(fmt.Errorf("failed to display token usage: %w", err))
			}
		}
	}

	return nil
}

// CodeGenerationResult represents the result of code generation
type CodeGenerationResult struct {
	TokenUsage *TokenUsageInfo
	Files      []string
	Duration   time.Duration
}

// TokenUsageInfo represents token usage information
type TokenUsageInfo struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
	Model           string
}

// startInteractiveCodeTUI starts the TUI in interactive code mode
func startInteractiveCodeTUI() error {
	// Set TUI as output sink
	ui.SetDefaultSink(ui.TuiSink{})

	// Start TUI with interactive code mode
	if err := tuiPkg.RunInteractiveCode(); err != nil {
		return fmt.Errorf("failed to start interactive code TUI: %w", err)
	}
	return nil
}

// executeCodeGenerationWithUI wraps code generation with UI progress updates
func executeCodeGenerationWithUI(filename, instructions string, cfg *config.Config, imagePath string) (*CodeGenerationResult, error) {
	startTime := time.Now()
	
	// Publish progress updates
	ui.PublishProgress(0, 1, []ui.ProgressRow{
		{Name: "Code Generation", Status: "processing", Step: "Analyzing request", Tokens: 0, Cost: 0},
	})

	// Execute the actual code generation
	diff, err := editor.ProcessCodeGeneration(filename, instructions, cfg, imagePath)
	if err != nil {
		// Update progress with error
		ui.PublishProgress(0, 1, []ui.ProgressRow{
			{Name: "Code Generation", Status: "failed", Step: "Error: " + err.Error(), Tokens: 0, Cost: 0},
		})
		return nil, err
	}

	duration := time.Since(startTime)

	// Create result with token usage from config
	result := &CodeGenerationResult{
		Files:    []string{filename}, // Single file for now
		Duration: duration,
	}

	// Extract generated files from diff if possible
	if diff != "" {
		// For now, just indicate that we have generated content
		result.Files = []string{"Generated code"}
	}

	if cfg.LastTokenUsage != nil {
		result.TokenUsage = &TokenUsageInfo{
			PromptTokens:     cfg.LastTokenUsage.PromptTokens,
			CompletionTokens: cfg.LastTokenUsage.CompletionTokens,
			TotalTokens:      cfg.LastTokenUsage.TotalTokens,
			Model:           cfg.EditingModel,
		}

		// Calculate cost
		if provider, err := providers.GetProvider(cfg.EditingModel); err == nil {
			cost := provider.CalculateCost(providers.TokenUsage{
				PromptTokens:     cfg.LastTokenUsage.PromptTokens,
				CompletionTokens: cfg.LastTokenUsage.CompletionTokens,
				TotalTokens:      cfg.LastTokenUsage.TotalTokens,
			})

			// Update progress with final results
			ui.PublishProgress(1, 1, []ui.ProgressRow{
				{Name: "Code Generation", Status: "completed", Step: "Generated code changes", 
				 Tokens: result.TokenUsage.TotalTokens, Cost: cost},
			})

			// Update progress with totals
			ui.PublishProgressWithTokens(1, 1, result.TokenUsage.TotalTokens, cost, cfg.EditingModel, []ui.ProgressRow{
				{Name: "Code Generation", Status: "completed", Step: "Generated code changes", 
				 Tokens: result.TokenUsage.TotalTokens, Cost: cost},
			})
		}
	} else {
		// Update progress without token info
		ui.PublishProgress(1, 1, []ui.ProgressRow{
			{Name: "Code Generation", Status: "completed", Step: "Generated code changes", Tokens: 0, Cost: 0},
		})
	}

	return result, nil
}

// displayTokenUsage displays token usage information
func displayTokenUsage(cfg *CommandConfig, tokenUsage *TokenUsageInfo) error {
	if tokenUsage == nil {
		return nil
	}

	// Use provider interface for cost calculation
	provider, err := providers.GetProvider(tokenUsage.Model)
	if err != nil {
		return fmt.Errorf("failed to get provider for model %s: %w", tokenUsage.Model, err)
	}

	cost := provider.CalculateCost(providers.TokenUsage{
		PromptTokens:     tokenUsage.PromptTokens,
		CompletionTokens: tokenUsage.CompletionTokens,
		TotalTokens:      tokenUsage.TotalTokens,
	})

	// Only show token summary in console mode - UI shows this in header
	ui.PrintfContext(false, "Token Usage: %d prompt + %d completion = %d total (Cost: $%.4f)\n",
		tokenUsage.PromptTokens,
		tokenUsage.CompletionTokens,
		tokenUsage.TotalTokens,
		cost)

	// Also log for debugging purposes
	if cfg.Logger != nil {
		cfg.Logger.LogProcessStep(fmt.Sprintf("Token Usage: %d prompt + %d completion = %d total (Cost: $%.4f)",
			tokenUsage.PromptTokens,
			tokenUsage.CompletionTokens,
			tokenUsage.TotalTokens,
			cost))
	}

	return nil
}

// Helper functions for flag parsing (these would be properly implemented via BaseCommand)

func extractInstructions(args []string) string {
	var instructions []string
	for _, arg := range args {
		// Skip flags
		if strings.HasPrefix(arg, "--") {
			continue
		}
		instructions = append(instructions, arg)
	}
	return strings.Join(instructions, " ")
}

func getFilenameFlag(args []string) string {
	for i, arg := range args {
		if arg == "--filename" || arg == "-f" {
			if i+1 < len(args) {
				return args[i+1]
			}
		}
	}
	return ""
}

func getImageFlag(args []string) string {
	for i, arg := range args {
		if arg == "--image" || arg == "-i" {
			if i+1 < len(args) {
				return args[i+1]
			}
		}
	}
	return ""
}
