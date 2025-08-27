package agent

import (
	"regexp"
	"strings"
)

// refineTodosWithAnalysis updates remaining todos based on analysis results
func refineTodosWithAnalysis(ctx *SimplifiedAgentContext, completedTodo *TodoItem) {
	analysis := strings.TrimSpace(ctx.AnalysisResults[completedTodo.ID])
	if analysis == "" {
		return
	}

	foundFiles := extractFilePathsFromAnalysis(analysis)
	updateTodosWithFoundFiles(ctx, foundFiles)
	createFollowUpTodos(ctx, analysis, foundFiles)
}

// extractFilePathsFromAnalysis finds file paths mentioned in analysis output
func extractFilePathsFromAnalysis(analysis string) map[string]bool {
	pathRe := regexp.MustCompile(`(?m)(?:^|\s)([\w./-]+\.[A-Za-z0-9]+)`)
	matches := pathRe.FindAllStringSubmatch(analysis, -1)
	
	foundFiles := map[string]bool{}
	for _, m := range matches {
		if len(m) >= 2 {
			p := strings.TrimSpace(m[1])
			if p != "" && !strings.HasSuffix(p, "/") {
				foundFiles[p] = true
			}
		}
	}
	return foundFiles
}

// updateTodosWithFoundFiles updates pending todos with discovered file paths
func updateTodosWithFoundFiles(ctx *SimplifiedAgentContext, foundFiles map[string]bool) {
	for i := range ctx.Todos {
		t := &ctx.Todos[i]
		if !isPendingOrInProgress(t) || hasFilePath(t) {
			continue
		}

		for f := range foundFiles {
			if isFileRelevantToTodo(f, t, ctx.AnalysisResults) {
				t.FilePath = f
				break
			}
		}
	}
}

// createFollowUpTodos creates additional todos based on analysis suggestions
func createFollowUpTodos(ctx *SimplifiedAgentContext, analysis string, foundFiles map[string]bool) {
	if len(foundFiles) == 0 {
		return
	}

	suggestRe := regexp.MustCompile(`(?i)\b(add|implement|update|modify|refactor|create)\b`)
	if suggestRe.MatchString(analysis) {
		for f := range foundFiles {
			ctx.Todos = append(ctx.Todos, TodoItem{
				ID:          generateTodoID(),
				Content:     "Apply changes based on analysis",
				Description: "Implement the changes identified by the analysis for: " + f,
				Status:      "pending",
				FilePath:    f,
				Priority:    5,
			})
			break
		}
	}
}

// isPendingOrInProgress checks if todo is in a state that can be updated
func isPendingOrInProgress(todo *TodoItem) bool {
	return todo.Status == "pending" || todo.Status == "in_progress"
}

// hasFilePath checks if todo already has a file path
func hasFilePath(todo *TodoItem) bool {
	return strings.TrimSpace(todo.FilePath) != ""
}

// isFileRelevantToTodo checks if a file is relevant to a todo
func isFileRelevantToTodo(filePath string, todo *TodoItem, analysisResults map[string]string) bool {
	stem := extractFileStem(filePath)
	
	// Check analysis results for relevance
	for _, analysis := range analysisResults {
		if strings.Contains(strings.ToLower(analysis), strings.ToLower(stem)) {
			return true
		}
	}
	
	// Check todo content for relevance
	return strings.Contains(strings.ToLower(todo.Content), strings.ToLower(stem))
}

// extractFileStem gets the filename without path
func extractFileStem(filePath string) string {
	stem := filePath
	if idx := strings.LastIndex(stem, "/"); idx != -1 {
		stem = stem[idx+1:]
	}
	return stem
}

// scoreTodoForDynamicPriority computes a dynamic priority score for a todo
func scoreTodoForDynamicPriority(ctx *SimplifiedAgentContext, todo *TodoItem) int {
	baseScore := calculateBaseScore(todo.Priority)
	urgencyBoost := calculateUrgencyBoost(todo)
	contextBoost := calculateContextBoost(ctx, todo)
	
	return baseScore + urgencyBoost + contextBoost
}

// calculateBaseScore derives base score from static priority
func calculateBaseScore(priority int) int {
	base := 100 - (priority * 10)
	if base < 0 {
		base = 0
	}
	return base
}

// calculateUrgencyBoost adds points for urgent keywords
func calculateUrgencyBoost(todo *TodoItem) int {
	content := strings.ToLower(todo.Content + " " + todo.Description)
	urgencyKeywords := []string{
		"fix", "error", "failing", "fail", "build", "lint", "security", 
		"vuln", "panic", "crash", "broken", "blocking",
	}
	
	boost := 0
	for _, keyword := range urgencyKeywords {
		if strings.Contains(content, keyword) {
			boost += 6
		}
	}
	return boost
}

// calculateContextBoost adds points based on findings and knowledge
func calculateContextBoost(ctx *SimplifiedAgentContext, todo *TodoItem) int {
	if ctx.ContextManager == nil || ctx.PersistentCtx == nil {
		return 0
	}

	boost := 0
	pc := ctx.PersistentCtx
	filePath := strings.TrimSpace(todo.FilePath)
	content := strings.ToLower(todo.Content + " " + todo.Description)

	// Recent findings boost
	boost += calculateFindingsBoost(pc.Findings, filePath, content)
	
	// Knowledge boost
	boost += calculateKnowledgeBoost(pc.KnowledgeBase, filePath, content)

	return boost
}

// calculateFindingsBoost calculates boost from recent findings
func calculateFindingsBoost(findings []AnalysisFinding, filePath, content string) int {
	boost := 0
	startIdx := len(findings) - 8
	if startIdx < 0 {
		startIdx = 0
	}

	for i := startIdx; i < len(findings); i++ {
		finding := findings[i]
		
		// File path match boost
		if filePath != "" && strings.TrimSpace(finding.FilePath) != "" && 
		   strings.EqualFold(finding.FilePath, filePath) {
			boost += getSeverityBoost(finding.Severity)
		}
		
		// Content relevance boost
		if containsRelevantTerms(finding.Title, content) {
			boost += 4
		}
	}
	
	return boost
}

// getSeverityBoost returns boost points based on finding severity
func getSeverityBoost(severity string) int {
	switch strings.ToLower(severity) {
	case "critical":
		return 20
	case "high":
		return 12
	case "medium":
		return 6
	case "low":
		return 2
	default:
		return 1
	}
}

// calculateKnowledgeBoost calculates boost from accumulated knowledge
func calculateKnowledgeBoost(knowledge []KnowledgeItem, filePath, content string) int {
	boost := 0
	startIdx := len(knowledge) - 5
	if startIdx < 0 {
		startIdx = 0
	}

	for i := startIdx; i < len(knowledge); i++ {
		item := knowledge[i]
		
		// File path relevance
		if filePath != "" && strings.Contains(strings.ToLower(item.Content), strings.ToLower(filePath)) {
			boost += 3
		}
		
		// Content relevance
		if containsRelevantTerms(item.Title+" "+item.Content, content) {
			boost += 2
		}
	}
	
	return boost
}

// containsRelevantTerms checks if any significant terms overlap between texts
func containsRelevantTerms(text1, text2 string) bool {
	// Extract meaningful terms (length >= 4) from text1
	words1 := extractMeaningfulWords(strings.ToLower(text1))
	text2Lower := strings.ToLower(text2)
	
	for _, word := range words1 {
		if strings.Contains(text2Lower, word) {
			return true
		}
	}
	return false
}

// extractMeaningfulWords extracts words of significant length
func extractMeaningfulWords(text string) []string {
	words := strings.Fields(text)
	var meaningful []string
	
	for _, word := range words {
		cleaned := strings.Trim(word, ".,!?;:")
		if len(cleaned) >= 4 {
			meaningful = append(meaningful, cleaned)
		}
	}
	
	return meaningful
}

// extractFindingsFromAnalysis parses analysis text to extract structured findings
func extractFindingsFromAnalysis(analysisText string, todo *TodoItem) []AnalysisFinding {
	var findings []AnalysisFinding
	
	lines := strings.Split(analysisText, "\n")
	for _, line := range lines {
		if finding := tryParseFindingFromLine(line, todo); finding != nil {
			findings = append(findings, *finding)
		}
	}
	
	// If no structured findings found, create a general finding
	if len(findings) == 0 {
		findings = append(findings, AnalysisFinding{
			Type:        "analysis",
			Title:       "Analysis completed: " + todo.Content,
			Description: analysisText,
			FilePath:    todo.FilePath,
			Severity:    "medium",
		})
	}
	
	return findings
}

// tryParseFindingFromLine attempts to extract a finding from a text line
func tryParseFindingFromLine(line string, todo *TodoItem) *AnalysisFinding {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil
	}

	// Look for structured finding patterns
	patterns := []struct {
		regex *regexp.Regexp
		typ   string
	}{
		{regexp.MustCompile(`(?i)^[-•*]\s*(.+?):\s*(.+)$`), "observation"},
		{regexp.MustCompile(`(?i)^(error|warning|issue):\s*(.+)$`), "issue"},
		{regexp.MustCompile(`(?i)^(found|discovered|identified):\s*(.+)$`), "discovery"},
		{regexp.MustCompile(`(?i)^(recommendation|suggest|should):\s*(.+)$`), "recommendation"},
	}

	for _, pattern := range patterns {
		if matches := pattern.regex.FindStringSubmatch(line); matches != nil {
			title := matches[1]
			details := ""
			if len(matches) > 2 {
				details = matches[2]
			}
			
			return &AnalysisFinding{
				Type:        pattern.typ,
				Title:       title,
				Description: details,
				FilePath:    todo.FilePath,
				Severity:    determineSeverity(line),
			}
		}
	}

	return nil
}

// determineSeverity determines finding severity from text content
func determineSeverity(text string) string {
	textLower := strings.ToLower(text)
	
	criticalKeywords := []string{"critical", "severe", "urgent", "blocking", "broken"}
	highKeywords := []string{"error", "fail", "warning", "important", "security"}
	mediumKeywords := []string{"issue", "problem", "concern", "should", "recommend"}
	
	for _, keyword := range criticalKeywords {
		if strings.Contains(textLower, keyword) {
			return "critical"
		}
	}
	for _, keyword := range highKeywords {
		if strings.Contains(textLower, keyword) {
			return "high"
		}
	}
	for _, keyword := range mediumKeywords {
		if strings.Contains(textLower, keyword) {
			return "medium"
		}
	}
	
	return "low"
}

// selectNextTodoIndex selects the next todo to execute based on dynamic priority
func selectNextTodoIndex(ctx *SimplifiedAgentContext) int {
	bestIndex := -1
	bestScore := -1
	
	for i, todo := range ctx.Todos {
		if todo.Status == "pending" {
			score := scoreTodoForDynamicPriority(ctx, &todo)
			if score > bestScore {
				bestScore = score
				bestIndex = i
			}
		}
	}
	
	return bestIndex
}