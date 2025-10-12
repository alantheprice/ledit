package agent

import (
	"strings"
	"testing"
)

func TestGetEmbeddedSystemPrompt(t *testing.T) {
	prompt := GetEmbeddedSystemPrompt()

	if prompt == "" {
		t.Error("Expected non-empty system prompt")
	}

	if !strings.Contains(prompt, "You are") {
		t.Error("System prompt should contain agent description")
	}
}

func TestGetEmbeddedSystemPromptWithProvider(t *testing.T) {
	basePrompt := GetEmbeddedSystemPrompt()

	// Test with ZAI provider (GLM-4.6) - should have specific constraints
	zaiPrompt := GetEmbeddedSystemPromptWithProvider("zai")

	if len(zaiPrompt) <= len(basePrompt) {
		t.Error("ZAI prompt should be longer than base prompt (has GLM-4.6 specific constraints)")
	}

	// Test with other provider (should return base prompt)
	otherPrompt := GetEmbeddedSystemPromptWithProvider("openai")

	if len(otherPrompt) != len(basePrompt) {
		t.Error("Non-ZAI providers should return base prompt only")
	}

	// Verify ZAI-specific constraints
	if !strings.Contains(zaiPrompt, "GLM-4.6 Critical Constraints") {
		t.Error("ZAI prompt should contain GLM-4.6 specific constraints")
	}

	if !strings.Contains(zaiPrompt, "LIMIT concurrent cognitive tasks to maximum 3-5 todos") {
		t.Error("ZAI prompt should contain todo limits")
	}

	// Verify base prompt has consolidated efficiency guidelines
	if !strings.Contains(basePrompt, "Be concise and direct") {
		t.Error("Base prompt should contain consolidated conciseness instruction")
	}

	if !strings.Contains(basePrompt, "Limit tool usage") {
		t.Error("Base prompt should contain tool usage limits")
	}
}

func TestConsolidatedEfficiencyGuidelines(t *testing.T) {
	// Test that efficiency guidelines are integrated throughout the base prompt
	basePrompt := GetEmbeddedSystemPrompt()

	// Check that efficiency concepts are integrated into existing sections
	expectedIntegrations := []string{
		"Be concise and direct",         // Core Principles
		"Focus on results",              // Core Principles
		"Focus on results, not process", // Tool Usage Guidelines
		"Make decisive choices",         // Tool Usage Guidelines
		"Get straight to the point",     // Progress Updates
		"most straightforward solution", // Implementation Process
	}

	for _, integration := range expectedIntegrations {
		if !strings.Contains(basePrompt, integration) {
			t.Errorf("Expected to find integrated efficiency instruction: %s", integration)
		}
	}

	// Verify the redundant section was removed
	if strings.Contains(basePrompt, "Efficiency and Communication Guidelines") {
		t.Error("Redundant efficiency section should have been removed")
	}

	// Verify non-ZAI providers get the consolidated base prompt
	providers := []string{"openai", "deepinfra", "ollama"}
	for _, provider := range providers {
		providerPrompt := GetEmbeddedSystemPromptWithProvider(provider)
		if len(providerPrompt) != len(basePrompt) {
			t.Errorf("Provider %s should get same consolidated base prompt", provider)
		}
	}

	// Verify ZAI gets extra constraints
	zaiPrompt := GetEmbeddedSystemPromptWithProvider("zai")
	if len(zaiPrompt) <= len(basePrompt) {
		t.Error("ZAI should get base prompt plus extra constraints")
	}

	t.Logf("✅ Consolidated efficiency guidelines verified")
	t.Logf("Base prompt length: %d", len(basePrompt))
	t.Logf("ZAI prompt length: %d (with constraints)", len(zaiPrompt))
	t.Logf("Non-ZAI providers get consolidated base prompt")
}
