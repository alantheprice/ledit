package editor

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/alantheprice/ledit/pkg/config"
	"github.com/alantheprice/ledit/pkg/llm"
	"github.com/alantheprice/ledit/pkg/prompts"
	"github.com/alantheprice/ledit/pkg/types"
	"github.com/alantheprice/ledit/pkg/utils"
)

type RequirementProcessor struct {
	cfg *config.Config
}

func NewRequirementProcessor(cfg *config.Config) *RequirementProcessor {
	return &RequirementProcessor{cfg: cfg}
}

func (p *RequirementProcessor) Process(plan *types.OrchestrationPlan) error {
	logger := utils.GetLogger(true) // Get the logger instance
	execCfg := *p.cfg
	execCfg.SkipPrompt = true

	maxAttempts := p.cfg.OrchestrationMaxAttempts
	// No fallback needed here, as config.Config now initializes OrchestrationMaxAttempts with a default.
	// If a user has an old config file without this field, it will default to 0,
	// which means the user would need to update their config or the default in config.go
	// should be set to a non-zero value. It's already set to 4 in createConfig.

	for i := range plan.Requirements {
		req := &plan.Requirements[i]

		fullInstruction := p.getFullInstructionForRequirement(req)

		if req.Status == "completed" {
			logger.LogProcessStep(prompts.SkippingCompletedStep(req.Instruction)) // Use prompt
			continue
		}

		if req.Status == "failed" {
			logger.LogProcessStep(prompts.RetryingFailedStep(req.Instruction)) // Use prompt
		} else {
			logger.LogProcessStep(prompts.ExecutingStep(req.Instruction)) // Use prompt
		}
		logger.LogProcessStep(prompts.ProcessingFile(req.Filepath)) // Use prompt

		var lastValidationErr error

		for attempt := 1; attempt <= maxAttempts; attempt++ {
			if attempt > 1 {
				logger.LogProcessStep(prompts.RetryAttempt(attempt, maxAttempts, req.Instruction)) // Use prompt
			}

			currentInstruction := p.getCurrentInstructionForAttempt(req, fullInstruction)

			processedInstruction, err := processInstructions(currentInstruction, p.cfg)
			if err != nil {
				req.Status = "failed"
				req.LastLLMResponse = ""
				_ = saveOrchestrationPlan(plan)                                            // Save plan even on processInstructions error
				logger.LogProcessStep(prompts.ProcessInstructionFailed(req.Filepath, err)) // Use prompt
				return fmt.Errorf("failed to process instruction for file %s: %v", req.Filepath, err)
			}

			diffForTargetFile, err := ProcessCodeGeneration(req.Filepath, processedInstruction, &execCfg)
			if err != nil {
				req.Status = "failed"
				req.LastLLMResponse = diffForTargetFile
				_ = saveOrchestrationPlan(plan)                                            // Save plan even on ProcessCodeGeneration error
				logger.LogProcessStep(prompts.ProcessRequirementFailed(req.Filepath, err)) // Use prompt
				return fmt.Errorf("failed to process requirement for file %s: %w", req.Filepath, err)
			}

			if strings.Contains(req.Filepath, "setup.sh") {
				logger.LogProcessStep(prompts.SetupStepCompleted(req.Instruction)) // Use prompt
				req.Status = "completed"
				req.ValidationFailureContext = ""
				req.LastLLMResponse = ""
				if err := saveOrchestrationPlan(plan); err != nil { // Save plan on completion
					logger.LogProcessStep(prompts.SaveProgressFailed(req.Filepath, err)) // Use prompt
					return fmt.Errorf("step for %s completed, but failed to save progress: %w", req.Filepath, err)
				}
				break
			}

			setupErr := createAndRunSetupScript(req, &execCfg)
			if setupErr != nil {
				logger.LogProcessStep(prompts.SetupFailedAttempt(attempt, setupErr)) // Use prompt
				req.Status = "failed"
				req.ValidationFailureContext = prompts.ValidationFailureContextSetupScriptFailed(setupErr) // Use prompt
				req.LastLLMResponse = diffForTargetFile
				_ = saveOrchestrationPlan(plan) // Save plan on setup script failure
				lastValidationErr = setupErr
				continue
			}

			lastValidationErr = createAndRunValidationScript(req, &execCfg)
			if lastValidationErr != nil {
				req.Status = "failed"
				req.ValidationFailureContext = lastValidationErr.Error() // Store the error message
				req.LastLLMResponse = diffForTargetFile
				_ = saveOrchestrationPlan(plan) // Save plan on validation script failure
				continue
			}

			req.Status = "completed"
			req.ValidationFailureContext = ""
			req.LastLLMResponse = ""
			if err := saveOrchestrationPlan(plan); err != nil {
				logger.LogProcessStep(prompts.SaveProgressFailed(req.Filepath, err)) // Use prompt
				return fmt.Errorf("step for %s completed, but failed to save progress: %w", req.Filepath, err)
			}
			logger.LogProcessStep(prompts.StepCompleted(req.Instruction)) // Use prompt
			break
		}

		if lastValidationErr != nil {
			logger.LogProcessStep(prompts.StepFailedAfterAttempts(req.Instruction, maxAttempts, lastValidationErr)) // Use prompt
			return fmt.Errorf("step '%s' failed after %d attempts: %w", req.Instruction, maxAttempts, lastValidationErr)
		}
	}

	logger.LogProcessStep(prompts.AllOrchestrationStepsCompleted()) // Use prompt
	return nil
}

func (p *RequirementProcessor) getFullInstructionForRequirement(req *types.OrchestrationRequirement) string {
	if strings.Contains(req.Filepath, ".") && testableFileTypes()[filepath.Ext(req.Filepath)] {
		return fmt.Sprintf(
			"As a TDD developer, you should write tests and write the associated code to accomplish the requirements: '%s'",
			req.Instruction,
		)
	} else if strings.Contains(req.Filepath, "setup.sh") {
		return fmt.Sprintf(
			"Update the setup script, but make sure to keep the file idempotent so it can be run multiple times: '%s'",
			req.Instruction,
		)
	}
	return req.Instruction
}

func (p *RequirementProcessor) getCurrentInstructionForAttempt(req *types.OrchestrationRequirement, fullInstruction string) string {
	logger := utils.GetLogger(true) // Get the logger instance for this method too
	if req.ValidationFailureContext != "" {
		re := regexp.MustCompile(`(?i)\s*#WS\s*$`)
		originalInstructionWithoutTag := strings.TrimSpace(re.ReplaceAllString(fullInstruction, ""))

		var contextPrompt string
		var generatedSearchQuery string

		searchQueryMessages := []prompts.Message{
			{Role: "system", Content: "You are an expert at generating concise search queries to resolve software development issues. Your output should ONLY be the search query, enclosed in double quotes."},
			{Role: "user", Content: fmt.Sprintf(
				"Based on the following error and context, generate a concise search query (2-15 words) that would help find relevant information to resolve this issue:\n\nError: %s\n\nContext: %s",
				req.ValidationFailureContext, originalInstructionWithoutTag,
			)},
		}

		_, searchQueryRaw, err := llm.GetLLMResponse(p.cfg.WorkspaceModel, searchQueryMessages, "search_query_generator", p.cfg, 1*time.Minute)
		if err == nil && searchQueryRaw != "" {
			generatedSearchQuery = strings.Trim(searchQueryRaw, `"`)
			generatedSearchQuery = strings.TrimSpace(generatedSearchQuery)
			logger.LogProcessStep(prompts.GeneratedSearchQuery(generatedSearchQuery)) // Use prompt
		} else {
			logger.LogProcessStep(prompts.SearchQueryGenerationWarning(err)) // Use prompt
		}

		if req.LastLLMResponse != "" {
			contextPrompt = prompts.RetryPromptWithDiff(originalInstructionWithoutTag, req.Filepath, req.ValidationFailureContext, req.LastLLMResponse) // Use prompt
		} else {
			contextPrompt = prompts.RetryPromptWithoutDiff(originalInstructionWithoutTag, req.Filepath, req.ValidationFailureContext) // Use prompt
		}

		if generatedSearchQuery != "" {
			contextPrompt = fmt.Sprintf("#SG \"%s\"\n%s", generatedSearchQuery, contextPrompt)
			logger.LogProcessStep(prompts.AddedSearchGrounding(generatedSearchQuery)) // Use prompt
		}

		logger.LogProcessStep(prompts.AddingValidationFailureContext()) // Use prompt
		return contextPrompt
	}
	return fullInstruction
}
