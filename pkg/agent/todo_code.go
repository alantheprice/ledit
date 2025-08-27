package agent

import (
	"fmt"
)

// executeCodeCommandTodo handles complex code changes via granular editing workflow
func executeCodeCommandTodo(ctx *SimplifiedAgentContext, todo *TodoItem) error {
	ctx.Logger.LogProcessStep("🛠️ Using granular editing workflow (complex changes)")

	phases := []func(*SimplifiedAgentContext, *TodoItem) error{
		executeExplorationPhase,
		executePlanningPhase,
		executeGranularEditingPhase,
		executeVerificationPhase,
	}

	phaseNames := []string{"exploration", "planning", "editing", "verification"}

	for i, phase := range phases {
		if err := phase(ctx, todo); err != nil {
			return fmt.Errorf("%s phase failed: %w", phaseNames[i], err)
		}
	}

	return nil
}

// executeOptimizedCodeEditingTodo handles code editing using optimized editing service with rollback
func executeOptimizedCodeEditingTodo(ctx *SimplifiedAgentContext, todo *TodoItem) error {
	ctx.Logger.LogProcessStep("⚡ Performing optimized code edit with rollback support")

	editingService := NewOptimizedEditingService(ctx.Config, ctx.Logger)
	result, err := editingService.ExecuteOptimizedEditWithRollback(todo, ctx)
	if err != nil {
		ctx.Logger.LogProcessStep(fmt.Sprintf("❌ Optimized edit failed: %v", err))
		return err
	}

	trackEditingMetrics(ctx, result, todo)
	ctx.Logger.LogProcessStep(fmt.Sprintf("✅ Optimized edit completed using %s strategy", result.Strategy))
	return nil
}

// trackEditingMetrics tracks token usage and costs from editing operations
func trackEditingMetrics(ctx *SimplifiedAgentContext, result *EditingResult, todo *TodoItem) {
	metrics := result.Metrics
	if metrics.TotalTokens <= 0 {
		return
	}

	// Update overall metrics
	ctx.TotalTokensUsed += metrics.TotalTokens
	ctx.TotalCost += metrics.TotalCost

	// Estimate token breakdown (typical editing: ~60% prompt, 40% completion)
	estimatedPromptTokens := int(float64(metrics.TotalTokens) * 0.6)
	estimatedCompletionTokens := metrics.TotalTokens - estimatedPromptTokens
	ctx.TotalPromptTokens += estimatedPromptTokens
	ctx.TotalCompletionTokens += estimatedCompletionTokens

	ctx.Logger.LogProcessStep(fmt.Sprintf("📊 Optimized edit used %d tokens ($%.4f)",
		metrics.TotalTokens, metrics.TotalCost))

	// Store results for analysis and potential rollback
	storeEditingResults(ctx, result, todo)
}

// storeEditingResults stores editing results and revision information
func storeEditingResults(ctx *SimplifiedAgentContext, result *EditingResult, todo *TodoItem) {
	if result.Diff != "" {
		ctx.AnalysisResults[todo.ID+"_edit_result"] = result.Diff
		ctx.FilesModified = true
	}

	if len(result.RevisionIDs) > 0 {
		ctx.AnalysisResults[todo.ID+"_revision_ids"] = fmt.Sprintf("%v", result.RevisionIDs)
		ctx.Logger.LogProcessStep(fmt.Sprintf("🔄 Rollback available with revision IDs: %v", result.RevisionIDs))

		for _, revisionID := range result.RevisionIDs {
			ctx.Logger.LogProcessStep(fmt.Sprintf("💾 Revision stored for rollback: %s", revisionID))
		}
	}
}

// Note: EditingResult, EditingMetrics, and related functions are defined in editing_optimizer.go
// Note: Phase execution functions are defined in granular_editing.go