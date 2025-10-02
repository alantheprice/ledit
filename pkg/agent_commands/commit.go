package commands

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/alantheprice/ledit/pkg/agent"
	api "github.com/alantheprice/ledit/pkg/agent_api"
	"github.com/alantheprice/ledit/pkg/factory"
	"github.com/alantheprice/ledit/pkg/utils"
	"golang.org/x/term"
)

// CommitCommand implements the /commit slash command
type CommitCommand struct {
	skipPrompt   bool
	dryRun       bool
	allowSecrets bool
}

// wrapText wraps text to a specific line length
func wrapText(text string, lineLength int) string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return ""
	}

	var lines []string
	currentLine := words[0]

	for i := 1; i < len(words); i++ {
		word := words[i]
		if len(currentLine)+1+len(word) <= lineLength {
			currentLine += " " + word
		} else {
			lines = append(lines, currentLine)
			currentLine = word
		}
	}

	if currentLine != "" {
		lines = append(lines, currentLine)
	}

	return strings.Join(lines, "\n")
}

// Name returns the command name
func (c *CommitCommand) Name() string {
	return "commit"
}

// Description returns the command description
func (c *CommitCommand) Description() string {
	return "Interactive commit workflow with dropdown selection - stage files and generate commit messages"
}

// console-safe output helpers
func normalizeNewlines(s string) string {
	if term.IsTerminal(int(os.Stdout.Fd())) {
		return strings.ReplaceAll(s, "\n", "\r\n")
	}
	return s
}

func (c *CommitCommand) printf(format string, args ...interface{}) {
	fmt.Fprint(os.Stdout, normalizeNewlines(fmt.Sprintf(format, args...)))
}

func (c *CommitCommand) println(text string) {
	fmt.Fprint(os.Stdout, normalizeNewlines(text)+"\r\n")
}

// --- Small helpers to reduce duplication ---

// getStagedFiles returns the list of staged file paths.
func getStagedFiles() ([]string, error) {
	out, err := exec.Command("git", "diff", "--staged", "--name-only").CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to get staged files: %v", err)
	}
	raw := strings.Split(strings.TrimSpace(string(out)), "\n")
	var files []string
	for _, f := range raw {
		if t := strings.TrimSpace(f); t != "" {
			files = append(files, t)
		}
	}
	return files, nil
}

// getPorcelainStatusLines returns non-empty lines from `git status --porcelain`.
func getPorcelainStatusLines() ([]string, error) {
	out, err := exec.Command("git", "status", "--porcelain").CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to get git status: %v", err)
	}
	if len(out) == 0 {
		return nil, nil
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var valid []string
	for _, l := range lines {
		if strings.TrimSpace(l) != "" {
			valid = append(valid, l)
		}
	}
	return valid, nil
}

// parseFilenameFromStatusLine extracts the filename from a porcelain status line.
func parseFilenameFromStatusLine(line string) (string, bool) {
	parts := strings.Fields(line)
	if len(parts) >= 2 {
		return strings.Join(parts[1:], " "), true
	}
	return "", false
}

// stageFiles stages a list of files and reports results.
func (c *CommitCommand) stageFiles(files []string) {
	c.println("\n📦 Staging files...")
	for _, file := range files {
		cmd := exec.Command("git", "add", file)
		output, err := cmd.CombinedOutput()
		if err != nil {
			c.printf("❌ Failed to stage %s: %v\n", file, err)
			if len(output) > 0 {
				c.printf("Output: %s\n", string(output))
			}
		} else {
			c.printf("✅ Staged: %s\n", file)
		}
	}
}

// selectAllModifiedFiles converts porcelain lines to filenames.
func selectAllModifiedFiles(validStatusLines []string) []string {
	var files []string
	for _, line := range validStatusLines {
		if name, ok := parseFilenameFromStatusLine(line); ok {
			files = append(files, name)
		}
	}
	return files
}

// editInEditor opens $VISUAL or $EDITOR to edit content, returns the edited text
func editInEditor(initial string) (string, error) {
	// Create temp file
	f, err := os.CreateTemp("", "ledit_commit_*.txt")
	if err != nil {
		return "", err
	}
	path := f.Name()
	_, _ = f.WriteString(initial)
	f.Close()

	// Choose editor
	editor := os.Getenv("VISUAL")
	if editor == "" {
		editor = os.Getenv("EDITOR")
	}
	if editor == "" {
		editor = "vi"
	}

	cmd := exec.Command(editor, path)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", err
	}

	// Read back
	data, err := os.ReadFile(path)
	_ = os.Remove(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

// Execute runs the commit command
func (c *CommitCommand) Execute(args []string, chatAgent *agent.Agent) error {
	// Parse flags from args
	var cleanArgs []string
	for _, arg := range args {
		switch arg {
		case "--skip-prompt":
			c.skipPrompt = true
		case "--dry-run":
			c.dryRun = true
		case "--allow-secrets":
			c.allowSecrets = true
		default:
			cleanArgs = append(cleanArgs, arg)
		}
	}

	// Handle subcommands
	if len(cleanArgs) > 0 {
		switch cleanArgs[0] {
		case "help", "--help", "-h":
			return c.showHelp()
		default:
			return fmt.Errorf("unknown subcommand: %s. Use '/commit help' for usage", cleanArgs[0])
		}
	}

	// Default behavior: use new interactive commit flow with flags
	flow := NewCommitFlowWithFlags(chatAgent, c.skipPrompt, c.dryRun, c.allowSecrets)
	return flow.Execute()
}

func (c *CommitCommand) selectAndStageFiles(chatAgent *agent.Agent, reader *bufio.Reader) ([]string, error) {
	validStatusLines, err := getPorcelainStatusLines()
	if err != nil {
		return nil, err
	}
	var filesToAdd []string
	fmt.Println("\n💡 Enter file numbers to commit (comma-separated, 'a' for all, 'q' to quit):")
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	switch strings.ToLower(input) {
	case "q", "quit":
		fmt.Println("❌ Commit cancelled")
		return nil, nil
	case "a", "all":
		filesToAdd = selectAllModifiedFiles(validStatusLines)
		fmt.Println("✅ Adding all modified files")
	default:
		selections := strings.Split(input, ",")
		for _, sel := range selections {
			sel = strings.TrimSpace(sel)
			if sel == "" {
				continue
			}
			var index int
			if _, err := fmt.Sscanf(sel, "%d", &index); err != nil || index < 1 || index > len(validStatusLines) {
				fmt.Printf("❌ Invalid selection: %s\n", sel)
				continue
			}
			if name, ok := parseFilenameFromStatusLine(validStatusLines[index-1]); ok {
				filesToAdd = append(filesToAdd, name)
				fmt.Printf("✅ Adding: %s\n", name)
			}
		}
	}
	c.stageFiles(filesToAdd)

	return filesToAdd, nil
}

func (c *CommitCommand) checkForAnyChanges(chatAgent *agent.Agent) (bool, error) {
	validStatusLines, err := getPorcelainStatusLines()
	if err != nil {
		return false, err
	}
	if len(validStatusLines) == 0 {
		chatAgent.PrintLine("✅ No changes to commit")
		return false, nil
	}
	return true, nil
}

func (c *CommitCommand) printStatus(chatAgent *agent.Agent) error {
	validStatusLines, err := getPorcelainStatusLines()
	if err != nil {
		chatAgent.PrintLine("Failed to get git status")
		return err
	}
	// Print the current git status
	chatAgent.PrintLine("📊 Current git status:")
	chatAgent.PrintLine("\n📁 Modified files:")
	for i, line := range validStatusLines {
		chatAgent.PrintLine(fmt.Sprintf("%2d. %s", i+1, line))
	}

	return nil
}

// executeMultiFileCommit handles the original multi-file commit workflow
func (c *CommitCommand) executeMultiFileCommit(chatAgent *agent.Agent) error {
	if ok, err := c.checkForAnyChanges(chatAgent); !ok {
		return err
	}
	reader := bufio.NewReader(os.Stdin)

	// Step 1: Check for staged files first
	staged, err := getStagedFiles()
	if err != nil {
		return err
	}
	if len(staged) == 0 {
		chatAgent.PrintLine("✅ No staged files found")
		staged, err = c.selectAndStageFiles(chatAgent, reader)
		if err != nil {
			return err
		}
	} else {
		chatAgent.PrintLine(fmt.Sprintf("📦 Found %d staged file(s):", len(staged)))
	}

	if err := c.printStatus(chatAgent); err != nil {
		return err
	}

	if len(staged) == 0 {
		fmt.Println("❌ No files selected")
		return nil
	}

	// Step 4: Stage the selected files
	return c.generateAndCommit(chatAgent, reader)
}

// showHelp displays commit command usage
func (c *CommitCommand) showHelp() error {
	fmt.Println("📝 Commit Command Usage:")
	fmt.Println("========================")
	fmt.Println("/commit          - Interactive commit workflow for staged files")
	fmt.Println("/commit help     - Show this help message")
	fmt.Println()
	fmt.Println("The interactive workflow helps you commit staged files")
	fmt.Println()
	return nil
}

// generateAndCommit handles commit message generation and commit creation
func (c *CommitCommand) generateAndCommit(chatAgent *agent.Agent, reader *bufio.Reader) error {
	// If reader is nil, create one
	if reader == nil {
		reader = bufio.NewReader(os.Stdin)
	}

	// Generate commit message
	c.println("")
	c.printf("Using model: %s\n From Provider: %s\n", chatAgent.GetModel(), chatAgent.GetProvider())

	// Get staged diff
	diffOutput, err := exec.Command("git", "diff", "--staged").CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to get staged diff: %v", err)
	}

	if len(strings.TrimSpace(string(diffOutput))) == 0 {
		c.println("❌ No changes staged")
		return nil
	}

	// Prepare LLM client if agent is available; otherwise fall back to manual prompt
	var client api.ClientInterface
	var clientType api.ClientType
	var model string
	if chatAgent != nil {
		configManager := chatAgent.GetConfigManager()
		if configManager != nil {
			if ct, e := configManager.GetProvider(); e == nil {
				clientType = ct
				model = configManager.GetModelForProvider(clientType)
				if cl, ce := factory.CreateProviderClient(clientType, model); ce == nil {
					client = cl
				}
			}
		}
	}

	// Get current branch name
	branchOutput, err := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to get branch name: %v", err)
	}
	branch := strings.TrimSpace(string(branchOutput))

	// Check if it's a default branch
	defaultBranches := []string{"master", "main", "develop", "dev"}
	isDefaultBranch := false
	for _, db := range defaultBranches {
		if branch == db {
			isDefaultBranch = true
			break
		}
	}

	// Get staged files with their status
	stagedFilesOutput, err := exec.Command("git", "diff", "--cached", "--name-status").CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to get staged files status: %v", err)
	}

	// Parse file actions and filenames
	fileActions := []string{}
	primaryAction := ""
	stagedFilenames := []string{}
	lines := strings.Split(strings.TrimSpace(string(stagedFilesOutput)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			status := parts[0]
			filepath := strings.Join(parts[1:], " ")
			stagedFilenames = append(stagedFilenames, filepath)

			action := ""
			switch status {
			case "A":
				action = "Adds"
			case "D":
				action = "Deletes"
			case "R":
				action = "Renames"
			default:
				action = "Updates"
			}

			if primaryAction == "" {
				primaryAction = action
			}

			fileActions = append(fileActions, fmt.Sprintf("%s %s", action, filepath))
		}
	}

	// Build the file actions summary (detailed for single file, generic for multi-file)
	var fileActionsSummary string
	if len(fileActions) == 1 {
		// Single file: include the specific action
		fileActionsSummary = fileActions[0]
	} else {
		// Multi-file: use generic summary
		fileActionsSummary = fmt.Sprintf("%s %d files", primaryAction, len(fileActions))
	}

	// Build branch prefix if not on default branch
	branchPrefix := ""
	if !isDefaultBranch {
		branchPrefix = fmt.Sprintf("[%s] ", branch)
	}

	var commitMessage string

	// Retry loop for commit message generation (LLM if available, otherwise manual input)
retryLoop:
	for {
		if client == nil {
			// Manual fallback when LLM client isn't available
			c.println("")
			c.println("🧾 Staged diff (truncated):")
			preview := string(diffOutput)
			if len(preview) > 2000 {
				preview = preview[:2000] + "\n... (truncated)"
			}
			c.println(preview)
			c.println("")
			c.println("✏️  Enter commit message (end with a blank line):")
			var b strings.Builder
			empty := 0
			for {
				line, _ := reader.ReadString('\n')
				if strings.TrimSpace(line) == "" {
					empty++
					if empty >= 1 {
						break
					}
				} else {
					empty = 0
				}
				b.WriteString(line)
			}
			commitMessage = strings.TrimSpace(b.String())
			if commitMessage == "" {
				c.println("❌ Empty commit message; aborting")
				return nil
			}
			break
		}
		// Multi-file mode - full format with file actions
		// Calculate available space for title
		prefixAndActions := branchPrefix + fileActionsSummary + " - "
		availableSpace := 72 - len(prefixAndActions)

		// Optimize diff for API efficiency
		optimizer := utils.NewDiffOptimizer()
		optimizedDiff := optimizer.OptimizeDiff(string(diffOutput))

		// Build context info for large files
		var contextInfo string
		if len(optimizedDiff.FileSummaries) > 0 {
			contextInfo = "\n\nFile summaries for optimized content:\n"
			for file, summary := range optimizedDiff.FileSummaries {
				contextInfo += fmt.Sprintf("- %s: %s\n", file, summary)
			}
		}

		// Create the commit message generation prompt
		commitPrompt := fmt.Sprintf(`Base responses on the following changes:

%s%s

Generate a concise git commit title starting with the word: '%s'. 
The total length MUST be under %d characters. Don't include the file name or any 
colons. The title should be a single line without any markdown formatting. Only 
return the short title and nothing else.`, optimizedDiff.OptimizedContent, contextInfo, primaryAction, availableSpace)

		// Get commit message title using fast model
		messages := []api.Message{
			{
				Role:    "system",
				Content: "You are a git commit message generator. Generate concise, clear commit messages following conventional commit standards.",
			},
			{
				Role:    "user",
				Content: commitPrompt,
			},
		}

		resp, err := client.SendChatRequest(messages, nil, "")
		if err != nil {
			return fmt.Errorf("failed to generate commit message: %v", err)
		}

		if len(resp.Choices) == 0 {
			return fmt.Errorf("no response from model")
		}

		shortTitle := strings.TrimSpace(resp.Choices[0].Message.Content)

		// Now generate the description (reuse the optimized diff)
		descPrompt := fmt.Sprintf(`Base responses on the following changes:

%s%s

Generate a Git commit message summary. The message should follow these rules:
1. The total length MUST be under 500 characters.
2. DO NOT include a title.
3. DO NOT include any code blocks or filenames.
4. DO NOT include any user messages.
5. Message will be a single paragraph without any markdown formatting.
6. The message should be clear and concise and only give reasoning for the change 
   if provided by the user.`, optimizedDiff.OptimizedContent, contextInfo)

		// Get description
		messages = []api.Message{
			{
				Role:    "system",
				Content: "You are a git commit message generator. Generate clear, concise descriptions.",
			},
			{
				Role:    "user",
				Content: descPrompt,
			},
		}

		resp, err = client.SendChatRequest(messages, nil, "")
		if err != nil {
			return fmt.Errorf("failed to generate description: %v", err)
		}

		if len(resp.Choices) == 0 {
			return fmt.Errorf("no response from model for description")
		}

		description := strings.TrimSpace(resp.Choices[0].Message.Content)

		// Wrap description at 72 characters
		wrappedDesc := wrapText(description, 72)

		// Build the full commit message
		commitTitle := branchPrefix + fileActionsSummary + " - " + shortTitle
		commitMessage = commitTitle + "\n\n" + wrappedDesc

		// Show token usage (both requests)
		c.printf("\n💰 Tokens used: ~%d (model: %s/%s)\n", resp.Usage.TotalTokens*2, clientType, model)

		// Show staged files summary and commit message (minimal, no emoji)
		c.println("")
		if len(stagedFilenames) > 0 {
			c.printf("Committing %d staged file(s):\n", len(stagedFilenames))
			const maxList = 10
			for i, name := range stagedFilenames {
				if i >= maxList {
					remaining := len(stagedFilenames) - maxList
					if remaining > 0 {
						c.printf("... (+%d more)\n", remaining)
					}
					break
				}
				c.printf("- %s\n", name)
			}
		}
		c.println("")
		c.println("With message:")
		c.println("")
		c.println(commitMessage)
		c.println("")

		// Handle confirmation (or auto-proceed if skipPrompt)
		if c.skipPrompt {
			c.println("")
			c.println("✅ Auto-proceeding with commit (--skip-prompt)")
			break // Exit retry loop
		} else {
			// If TUI is active use dropdown, otherwise stdin prompt
			if os.Getenv("LEDIT_AGENT_CONSOLE") == "1" && chatAgent != nil {
				// Include the commit title in the prompt so users see context even if overlay obscures preview
				title := ""
				if parts := strings.Split(commitMessage, "\n"); len(parts) > 0 {
					title = strings.TrimSpace(parts[0])
					if len(title) > 80 {
						title = title[:77] + "..."
					}
				}
				choices := []agent.ChoiceOption{
					{Label: "Approve", Value: "y"},
					{Label: "Retry", Value: "r"},
					{Label: "Edit", Value: "e"},
					{Label: "Cancel", Value: "n"},
				}
				c.println("-----------------------------\n")
				prompt := "Proceed with commit?"

				choice, err := chatAgent.PromptChoice(prompt, choices)
				if err != nil {
					return fmt.Errorf("confirmation failed: %w", err)
				}
				switch choice {
				case "r":
					c.println("Regenerating commit message...")
					continue
				case "e":
					edited, err := editInEditor(commitMessage)
					if err != nil {
						return fmt.Errorf("editor failed: %w", err)
					}
					if strings.TrimSpace(edited) == "" {
						c.println("Empty commit message; aborting")
						return nil
					}
					commitMessage = edited
					break retryLoop
				case "y":
					break retryLoop
				case "n":
					c.println("Commit cancelled")
					return nil
				default:
					continue
				}
			} else {
				// Confirmation with retry option via stdin
				c.println("")
				c.printf("Proceed with commit? (y/n/e to edit/r to retry): ")
				input, _ := reader.ReadString('\n')
				input = strings.TrimSpace(strings.ToLower(input))

				if input == "r" || input == "retry" {
					c.println("Regenerating commit message...")
					continue // Go back to start of loop to regenerate
				} else if input == "e" || input == "edit" {
					// Open editor for editing
					edited, err := editInEditor(commitMessage)
					if err != nil {
						return fmt.Errorf("editor failed: %w", err)
					}
					if strings.TrimSpace(edited) == "" {
						c.println("Empty commit message; aborting")
						return nil
					}
					commitMessage = edited
					break
				} else if input == "y" || input == "yes" || input == "" {
					break // Exit retry loop and proceed with commit
				} else if input == "n" || input == "no" {
					c.println("Commit cancelled")
					return nil
				} else {
					c.printf("Invalid option: %s. Please use y/n/e/r\n", input)
					continue // Show the confirmation prompt again
				}
			}
		}

	} // End of retry loop

	// Handle dry-run mode
	if c.dryRun {
		c.println("")
		c.println("🔍 Dry-run mode: Commit message generated successfully!")
		c.println("💡 The commit was not created due to --dry-run flag")
		c.println("📝 To create the commit, run the command again without --dry-run")
		return nil
	}

	// Create the commit
	c.println("")
	c.println("💾 Creating commit...")

	// Write commit message to temporary file
	tempFile := "commit_msg.txt"
	err = os.WriteFile(tempFile, []byte(commitMessage), 0644)
	if err != nil {
		return fmt.Errorf("failed to create temporary commit message file: %v", err)
	}
	defer os.Remove(tempFile)

	cmd := exec.Command("git", "commit", "-F", tempFile)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create commit: %v\nOutput: %s", err, string(output))
	}

	c.println("✅ Commit created successfully!")
	c.printf("Output: %s\n", string(output))

	return nil
}
