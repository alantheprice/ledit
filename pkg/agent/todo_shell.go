package agent

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/alantheprice/ledit/pkg/llm"
	"github.com/alantheprice/ledit/pkg/prompts"
	"github.com/alantheprice/ledit/pkg/utils"
)

// executeShellCommandTodo executes todos that require shell commands
func executeShellCommandTodo(ctx *SimplifiedAgentContext, todo *TodoItem) error {
	ctx.Logger.LogProcessStep("🖥️ Executing shell command todo")

	commandPlan, err := generateCommandPlan(ctx, todo)
	if err != nil {
		return fmt.Errorf("failed to generate command plan: %w", err)
	}

	logCommandPlan(ctx, commandPlan)
	return executeCommands(ctx, commandPlan, todo)
}

// commandPlan represents a shell command execution plan
type commandPlan struct {
	Commands    []string `json:"commands"`
	Explanation string   `json:"explanation"`
	SafetyNotes string   `json:"safety_notes"`
}

// generateCommandPlan creates a shell command execution plan using LLM
func generateCommandPlan(ctx *SimplifiedAgentContext, todo *TodoItem) (*commandPlan, error) {
	prompt := buildShellPrompt(ctx, todo)
	messages := []prompts.Message{
		{Role: "system", Content: "You are an expert at generating safe shell commands for development tasks. Always respond with valid JSON containing an array of shell commands."},
		{Role: "user", Content: prompt},
	}

	response, tokenUsage, err := llm.GetLLMResponse(
		ctx.Config.OrchestrationModel, 
		messages, 
		"", 
		ctx.Config, 
		llm.GetSmartTimeout(ctx.Config, ctx.Config.OrchestrationModel, "analysis"),
	)
	if err != nil {
		return nil, fmt.Errorf("LLM request failed: %w", err)
	}

	trackTokenUsage(ctx, tokenUsage, ctx.Config.OrchestrationModel)
	return parseCommandPlan(response)
}

// buildShellPrompt creates the prompt for shell command generation
func buildShellPrompt(ctx *SimplifiedAgentContext, todo *TodoItem) string {
	return fmt.Sprintf(`You are an expert system administrator. Generate safe shell commands to accomplish this task:

Task: %s
Description: %s
Overall Goal: %s

Generate the appropriate shell commands to complete this task. Be very careful about:
1. Only use safe commands that won't harm the system
2. Create directories and files as needed
3. Follow standard conventions for project structure
4. Use relative paths from current directory
5. For multi-line file content, use SINGLE commands with proper heredoc syntax
6. Each array item should be ONE complete command, not broken into parts
7. For Go commands (go get, go mod, go build), always run them in the directory with go.mod
8. Use "cd directory && command" format for commands that need specific working directory

IMPORTANT: 
- When creating files with content, use: "cat > filename <<'EOF'\ncontent here\nEOF"  
- For Go module operations, use: "cd backend && go get package" or "cd backend && go mod tidy"
- Never run Go commands from root if the go.mod is in a subdirectory
- Use 'find' to locate files, 'grep' to search content, 'ls' to list directories
- Example: "find . -name '*.py' -path '*/routes*'" to find Python files in routes directories

Respond with JSON:
{
  "commands": ["command1", "command2", "command3"],
  "explanation": "What these commands accomplish",  
  "safety_notes": "Any important safety considerations"
}`, todo.Content, todo.Description, ctx.UserIntent)
}

// parseCommandPlan parses the LLM response into a command plan
func parseCommandPlan(response string) (*commandPlan, error) {
	clean, err := utils.ExtractJSON(response)
	if err != nil {
		return nil, fmt.Errorf("failed to extract JSON from response: %w", err)
	}

	var plan commandPlan
	if err := json.Unmarshal([]byte(clean), &plan); err != nil {
		return nil, fmt.Errorf("failed to unmarshal command plan: %w", err)
	}

	return &plan, nil
}

// logCommandPlan logs the command execution plan
func logCommandPlan(ctx *SimplifiedAgentContext, plan *commandPlan) {
	ctx.Logger.LogProcessStep(fmt.Sprintf("📋 Execution plan: %s", plan.Explanation))
	if plan.SafetyNotes != "" {
		ctx.Logger.LogProcessStep(fmt.Sprintf("⚠️ Safety notes: %s", plan.SafetyNotes))
	}
}

// executeCommands executes the shell commands in the plan
func executeCommands(ctx *SimplifiedAgentContext, plan *commandPlan, todo *TodoItem) error {
	for i, command := range plan.Commands {
		if err := executeSingleCommand(ctx, command, i+1, len(plan.Commands)); err != nil {
			return err
		}
	}

	markFilesModified(ctx, todo, plan)
	ctx.Logger.LogProcessStep("✅ Shell command todo completed successfully")
	return nil
}

// executeSingleCommand executes a single shell command safely
func executeSingleCommand(ctx *SimplifiedAgentContext, command string, index, total int) error {
	if containsUnsafeCommand(command) {
		return fmt.Errorf("unsafe command detected and blocked: %s", command)
	}

	if err := validateShellCommand(command); err != nil {
		ctx.Logger.LogProcessStep(fmt.Sprintf("⚠️ Skipping invalid command: %s", err.Error()))
		return nil
	}

	safeCommand := prepareCommand(command)
	ctx.Logger.LogProcessStep(fmt.Sprintf("🔧 Executing command %d/%d: %s", index, total, safeCommand))

	output, err := runCommand(safeCommand)
	if err != nil {
		ctx.Logger.LogProcessStep(fmt.Sprintf("❌ Command failed: %s", string(output)))
		return fmt.Errorf("command failed: %s - %w", command, err)
	}

	if len(output) > 0 {
		ctx.Logger.LogProcessStep(fmt.Sprintf("📤 Output: %s", string(output)))
	}

	return nil
}

// prepareCommand makes command safer and ensures directories exist
func prepareCommand(command string) string {
	safeCommand := makeCommandIdempotent(command)
	return prepareDirectoriesForCommand(safeCommand)
}

// runCommand executes a shell command
func runCommand(command string) ([]byte, error) {
	cmd := exec.Command("bash", "-c", command)
	cmd.Dir = "."
	return cmd.CombinedOutput()
}

// markFilesModified marks files as potentially modified and stores results
func markFilesModified(ctx *SimplifiedAgentContext, todo *TodoItem, plan *commandPlan) {
	ctx.FilesModified = true
	resultKey := todo.ID + "_shell_result"
	result := fmt.Sprintf("Successfully executed %d commands: %s", len(plan.Commands), plan.Explanation)
	ctx.AnalysisResults[resultKey] = result
}

// containsUnsafeCommand checks if command contains unsafe operations
func containsUnsafeCommand(command string) bool {
	unsafeCommands := []string{
		"rm -rf /", "rm -rf /*", ":(){ :|:& };:", "mv /* /dev/null",
		"dd if=/dev/zero", "mkfs", "fdisk", "format", "deltree",
		"shutdown", "reboot", "halt", "poweroff", "init 0", "init 6",
		"chmod 777 /", "chown root /", "su -", "sudo su", "sudo -s",
		"> /dev/", "curl.*|.*sh", "wget.*|.*sh",
		"eval.*", "exec.*", "${.*}", "$(.*)",
	}

	cmdLower := strings.ToLower(command)
	for _, unsafe := range unsafeCommands {
		if strings.Contains(cmdLower, unsafe) {
			return true
		}
	}

	return false
}

// validateShellCommand performs basic validation on shell command
func validateShellCommand(command string) error {
	if strings.TrimSpace(command) == "" {
		return fmt.Errorf("empty command")
	}

	if len(command) > 2000 {
		return fmt.Errorf("command too long (max 2000 characters)")
	}

	// Check for suspicious patterns
	suspiciousPatterns := []string{
		"../../../", "../../../../", "/etc/passwd", "/etc/shadow",
		"/proc/", "/sys/", "/dev/sd", "/dev/hd",
	}

	cmdLower := strings.ToLower(command)
	for _, pattern := range suspiciousPatterns {
		if strings.Contains(cmdLower, pattern) {
			return fmt.Errorf("potentially dangerous path pattern: %s", pattern)
		}
	}

	return nil
}

// makeCommandIdempotent makes commands safer and more idempotent
func makeCommandIdempotent(command string) string {
	// Convert destructive operations to safer versions
	if strings.HasPrefix(command, "mkdir ") && !strings.Contains(command, "-p") {
		command = strings.Replace(command, "mkdir ", "mkdir -p ", 1)
	}

	// Add existence checks for file operations
	if strings.Contains(command, "mv ") && !strings.Contains(command, "if [") {
		parts := strings.Fields(command)
		if len(parts) >= 3 {
			src := parts[1]
			return fmt.Sprintf("if [ -e %s ]; then %s; fi", src, command)
		}
	}

	return command
}

// prepareDirectoriesForCommand ensures directories exist before file operations
func prepareDirectoriesForCommand(command string) string {
	if strings.Contains(command, "cat >") && strings.Contains(command, "/") {
		parts := strings.Split(command, "cat >")
		if len(parts) > 1 {
			filePart := strings.TrimSpace(parts[1])
			if spaceIdx := strings.Index(filePart, " "); spaceIdx > 0 {
				filePart = filePart[:spaceIdx]
			}
			if strings.Contains(filePart, "/") {
				dirPath := filePart[:strings.LastIndex(filePart, "/")]
				return fmt.Sprintf("mkdir -p %s && %s", dirPath, command)
			}
		}
	}

	return command
}