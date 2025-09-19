package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ExportState exports the current agent state for persistence
func (a *Agent) ExportState() ([]byte, error) {
	// Generate compact summary for next session continuity
	compactSummary := a.GenerateCompactSummary()

	state := AgentState{
		Messages:        a.messages,
		PreviousSummary: a.previousSummary,
		CompactSummary:  compactSummary, // Store 5K-limited summary for continuity
		TaskActions:     a.taskActions,
		SessionID:       a.sessionID,
	}
	return json.Marshal(state)
}

// ImportState imports agent state from JSON data
func (a *Agent) ImportState(data []byte) error {
	var state AgentState
	if err := json.Unmarshal(data, &state); err != nil {
		return err
	}
	a.messages = state.Messages
	// Prefer compact summary for continuity, fallback to legacy summary
	if state.CompactSummary != "" {
		a.previousSummary = state.CompactSummary
	} else {
		a.previousSummary = state.PreviousSummary
	}
	a.taskActions = state.TaskActions
	a.sessionID = state.SessionID
	return nil
}

// SaveStateToFile saves the agent state to a file
func (a *Agent) SaveStateToFile(filename string) error {
	stateData, err := a.ExportState()
	if err != nil {
		return err
	}
	return os.WriteFile(filename, stateData, 0644)
}

// LoadStateFromFile loads agent state from a file
func (a *Agent) LoadStateFromFile(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	return a.ImportState(data)
}

// LoadSummaryFromFile loads ONLY the compact summary from a state file for minimal continuity
func (a *Agent) LoadSummaryFromFile(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	var state AgentState
	if err := json.Unmarshal(data, &state); err != nil {
		return err
	}

	// Only load the compact summary, not the full conversation state
	if state.CompactSummary != "" {
		a.previousSummary = state.CompactSummary
		if a.debug {
			a.debugLog("📄 Loaded compact summary (%d chars)\n", len(state.CompactSummary))
		}
	} else if state.PreviousSummary != "" {
		// Fallback to legacy summary if compact summary not available
		a.previousSummary = state.PreviousSummary
		if a.debug {
			a.debugLog("📄 Loaded legacy summary (%d chars)\n", len(state.PreviousSummary))
		}
	}

	return nil
}

// SaveConversationSummary saves the conversation summary to the state file
func (a *Agent) SaveConversationSummary() error {
	// Generate summary before saving
	_ = a.GenerateConversationSummary() // Generate summary to update state

	// Save state to file
	stateFile := ".coder_state.json"
	if err := a.SaveStateToFile(stateFile); err != nil {
		return fmt.Errorf("failed to save conversation state: %v", err)
	}

	if a.debug {
		a.debugLog("💾 Saved conversation summary to %s\n", stateFile)
	}

	return nil
}

// AddTaskAction records a completed task action for continuity
func (a *Agent) AddTaskAction(actionType, description, details string) {
	a.taskActions = append(a.taskActions, TaskAction{
		Type:        actionType,
		Description: description,
		Details:     details,
	})
}

// GenerateActionSummary creates a summary of completed actions for continuity
func (a *Agent) GenerateActionSummary() string {
	if len(a.taskActions) == 0 {
		return "No actions completed yet."
	}

	var summary strings.Builder
	summary.WriteString("Previous actions completed:\n")

	for i, action := range a.taskActions {
		summary.WriteString(fmt.Sprintf("%d. %s: %s", i+1, action.Type, action.Description))
		if action.Details != "" {
			summary.WriteString(fmt.Sprintf(" (%s)", action.Details))
		}
		summary.WriteString("\n")
	}

	return summary.String()
}

// SetPreviousSummary sets the summary of previous actions for continuity
func (a *Agent) SetPreviousSummary(summary string) {
	a.previousSummary = summary
}

// GetPreviousSummary returns the summary of previous actions
func (a *Agent) GetPreviousSummary() string {
	return a.previousSummary
}

// SetSessionID sets the session identifier for continuity
func (a *Agent) SetSessionID(sessionID string) {
	a.sessionID = sessionID
}

// GetSessionID returns the session identifier
func (a *Agent) GetSessionID() string {
	return a.sessionID
}

// autoSaveState automatically saves the current conversation state
func (a *Agent) autoSaveState() {
	// Generate session ID based on timestamp if not set
	if a.sessionID == "" {
		a.sessionID = fmt.Sprintf("session_%d", time.Now().Unix())
	}

	// Save state to persistent storage
	stateDir, err := GetStateDir()
	if err != nil {
		if a.debug {
			a.debugLog("⚠️ Failed to get state directory for auto-save: %v\n", err)
		}
		return
	}

	stateFile := filepath.Join(stateDir, fmt.Sprintf("session_%s.json", a.sessionID))

	// Create conversation state for persistence
	state := ConversationState{
		Messages:          a.messages,
		TaskActions:       a.taskActions,
		TotalCost:         a.totalCost,
		TotalTokens:       a.totalTokens,
		PromptTokens:      a.promptTokens,
		CompletionTokens:  a.completionTokens,
		CachedTokens:      a.cachedTokens,
		CachedCostSavings: a.cachedCostSavings,
		LastUpdated:       time.Now(),
		SessionID:         a.sessionID,
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		if a.debug {
			a.debugLog("⚠️ Failed to marshal state for auto-save: %v\n", err)
		}
		return
	}

	if err := os.WriteFile(stateFile, data, 0600); err != nil {
		if a.debug {
			a.debugLog("⚠️ Failed to write state file for auto-save: %v\n", err)
		}
		return
	}

	if a.debug {
		a.debugLog("💾 Auto-saved conversation state to %s\n", stateFile)
	}
}
