package workspace

import (
	"fmt"
	"sort"
	"strings"

	"github.com/alantheprice/ledit/pkg/workspaceinfo"
)

// buildSyntacticOverview creates a compact, deterministic overview string for LLM context
func buildSyntacticOverview(ws workspaceinfo.WorkspaceFile) string {
	var b strings.Builder
	
	addBasicInfo(&b, ws)
	addInsightsInfo(&b, ws)
	addFilesInfo(&b, ws)
	
	return b.String()
}

// addBasicInfo adds basic workspace information
func addBasicInfo(b *strings.Builder, ws workspaceinfo.WorkspaceFile) {
	b.WriteString("Languages: ")
	b.WriteString(strings.Join(ws.Languages, ", "))
	b.WriteString("\n")
	
	if ws.BuildCommand != "" {
		b.WriteString(fmt.Sprintf("Build: %s\n", ws.BuildCommand))
	}
	if ws.TestCommand != "" {
		b.WriteString(fmt.Sprintf("Test: %s\n", ws.TestCommand))
	}
	if len(ws.BuildRunners) > 0 {
		b.WriteString(fmt.Sprintf("Build runners: %s\n", strings.Join(ws.BuildRunners, ", ")))
	}
	if len(ws.TestRunnerPaths) > 0 {
		b.WriteString(fmt.Sprintf("Test configs: %s\n", strings.Join(ws.TestRunnerPaths, ", ")))
	}
}

// addInsightsInfo adds project insights to overview
func addInsightsInfo(b *strings.Builder, ws workspaceinfo.WorkspaceFile) {
	if (ws.ProjectInsights == workspaceinfo.ProjectInsights{}) {
		return
	}

	b.WriteString("Insights: ")
	parts := buildInsightParts(ws.ProjectInsights)
	if len(parts) > 0 {
		b.WriteString(strings.Join(parts, "; "))
		b.WriteString("\n")
	}
}

// buildInsightParts builds insight parts from project insights
func buildInsightParts(insights workspaceinfo.ProjectInsights) []string {
	var parts []string
	
	appendIf := func(name, val string) {
		if strings.TrimSpace(val) != "" {
			parts = append(parts, fmt.Sprintf("%s=%s", name, val))
		}
	}
	
	appendIf("frameworks", insights.PrimaryFrameworks)
	appendIf("ci", insights.CIProviders)
	appendIf("pkg", insights.PackageManagers)
	appendIf("runtime", insights.RuntimeTargets)
	appendIf("deploy", insights.DeploymentTargets)
	appendIf("monorepo", insights.Monorepo)
	appendIf("layout", insights.RepoLayout)
	
	return parts
}

// addFilesInfo adds file information to overview
func addFilesInfo(b *strings.Builder, ws workspaceinfo.WorkspaceFile) {
	b.WriteString("\nFiles (path, overview, exports, references):\n")
	
	files := getSortedFiles(ws.Files)
	const maxFiles = 400
	if len(files) > maxFiles {
		files = files[:maxFiles]
	}
	
	for _, path := range files {
		fileInfo := ws.Files[path]
		addFileDetails(b, path, fileInfo)
	}
}

// getSortedFiles returns sorted list of file paths
func getSortedFiles(files map[string]workspaceinfo.WorkspaceFileInfo) []string {
	var paths []string
	for p := range files {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	return paths
}

// addFileDetails adds individual file details to overview  
func addFileDetails(b *strings.Builder, path string, info workspaceinfo.WorkspaceFileInfo) {
	b.WriteString(path)
	b.WriteString("\n  overview: ")
	b.WriteString(info.Summary)
	
	if strings.TrimSpace(info.Exports) != "" {
		b.WriteString("\n  exports: ")
		b.WriteString(info.Exports)
	}
	
	if strings.TrimSpace(info.References) != "" {
		b.WriteString("\n  references: ")
		b.WriteString(info.References)
	}
	
	b.WriteString("\n\n")
}