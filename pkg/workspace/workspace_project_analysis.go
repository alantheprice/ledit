package workspace

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/alantheprice/ledit/pkg/workspaceinfo"
)

// detectProjectInsightsHeuristics scans the repo to infer insights without LLM
func detectProjectInsightsHeuristics(rootDir string, ws workspaceinfo.WorkspaceFile) workspaceinfo.ProjectInsights {
	insights := workspaceinfo.ProjectInsights{}

	detectMonorepoStructure(&insights, rootDir)
	detectCIProviders(&insights, rootDir)
	detectPackageManagers(&insights, rootDir)
	detectRuntimeTargets(&insights, ws.Languages)
	detectFrameworks(&insights, rootDir)
	detectDeploymentTargets(&insights, rootDir)
	detectRepoLayout(&insights, rootDir)

	return insights
}

// detectMonorepoStructure determines if this is a monorepo
func detectMonorepoStructure(insights *workspaceinfo.ProjectInsights, rootDir string) {
	// Check for explicit monorepo indicators
	monorepoFiles := []string{
		"pnpm-workspace.yaml", "pnpm-workspace.yml",
		"lerna.json", "nx.json", "turbo.json", "go.work",
	}

	for _, file := range monorepoFiles {
		if exists(filepath.Join(rootDir, file)) {
			insights.Monorepo = "yes"
			return
		}
	}

	// Check for multiple package.json or go.mod files
	if isMultiModuleProject(rootDir) {
		insights.Monorepo = "yes"
	} else {
		insights.Monorepo = "no"
	}
}

// isMultiModuleProject checks if there are multiple package files
func isMultiModuleProject(rootDir string) bool {
	pkgCount, gomodCount := 0, 0
	
	filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			if d.IsDir() && shouldSkipDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}

		base := filepath.Base(path)
		if base == "package.json" {
			pkgCount++
		} else if base == "go.mod" {
			gomodCount++
		}

		return nil
	})

	return pkgCount > 1 || gomodCount > 1
}

// shouldSkipDir determines if a directory should be skipped during analysis
func shouldSkipDir(name string) bool {
	skipDirs := []string{".git", "node_modules", "vendor", "dist", "build", ".next", "target"}
	for _, skip := range skipDirs {
		if name == skip {
			return true
		}
	}
	return false
}

// detectCIProviders detects continuous integration providers
func detectCIProviders(insights *workspaceinfo.ProjectInsights, rootDir string) {
	var providers []string

	ciChecks := map[string]string{
		filepath.Join(".github", "workflows"):     "GitHub Actions",
		".gitlab-ci.yml":                          "GitLab CI",
		filepath.Join(".circleci", "config.yml"):  "CircleCI",
		".azure-pipelines.yml":                    "Azure Pipelines",
		".drone.yml":                              "Drone",
		".travis.yml":                             "TravisCI",
		"Jenkinsfile":                             "Jenkins",
		".buildkite":                              "Buildkite",
	}

	for file, provider := range ciChecks {
		if exists(filepath.Join(rootDir, file)) {
			providers = append(providers, provider)
		}
	}

	insights.CIProviders = strings.Join(providers, ", ")
}

// detectPackageManagers detects package managers in use
func detectPackageManagers(insights *workspaceinfo.ProjectInsights, rootDir string) {
	var managers []string

	pmChecks := map[string]string{
		"package-lock.json": "npm",
		"yarn.lock":         "yarn",
		"pnpm-lock.yaml":    "pnpm",
		"go.mod":            "go modules",
		"Cargo.toml":        "cargo",
		"Gemfile":           "bundler",
		"composer.json":     "composer",
	}

	for file, manager := range pmChecks {
		if exists(filepath.Join(rootDir, file)) {
			managers = append(managers, manager)
		}
	}

	// Python package managers
	pythonFiles := []string{"requirements.txt", "Pipfile", "poetry.lock", "pyproject.toml"}
	for _, file := range pythonFiles {
		if exists(filepath.Join(rootDir, file)) {
			managers = append(managers, "pip/poetry")
			break
		}
	}

	insights.PackageManagers = strings.Join(managers, ", ")
}

// detectRuntimeTargets detects runtime targets based on languages
func detectRuntimeTargets(insights *workspaceinfo.ProjectInsights, languages []string) {
	var targets []string
	langSet := make(map[string]bool)
	
	for _, lang := range languages {
		langSet[lang] = true
	}

	if langSet["javascript"] || langSet["typescript"] {
		targets = append(targets, "Node.js", "Browser")
	}
	if langSet["python"] {
		targets = append(targets, "Python")
	}
	if langSet["java"] || langSet["kotlin"] {
		targets = append(targets, "JVM")
	}
	if langSet["go"] {
		targets = append(targets, "Go")
	}
	if langSet["rust"] {
		targets = append(targets, "Rust")
	}
	if langSet["c"] || langSet["cpp"] {
		targets = append(targets, "Native")
	}

	insights.RuntimeTargets = strings.Join(uniqueStrings(targets), ", ")
}

// detectFrameworks detects frameworks based on package files
func detectFrameworks(insights *workspaceinfo.ProjectInsights, rootDir string) {
	var frameworks []string

	// Node.js frameworks
	if frameworks = append(frameworks, detectNodeFrameworks(rootDir)...); len(frameworks) == 0 {
		// Go frameworks
		frameworks = append(frameworks, detectGoFrameworks(rootDir)...)
	}

	insights.PrimaryFrameworks = strings.Join(uniqueStrings(frameworks), ", ")
}

// detectNodeFrameworks detects Node.js/JavaScript frameworks
func detectNodeFrameworks(rootDir string) []string {
	packageJSONPath := filepath.Join(rootDir, "package.json")
	if !exists(packageJSONPath) {
		return nil
	}

	data, err := os.ReadFile(packageJSONPath)
	if err != nil {
		return nil
	}

	var pkg map[string]interface{}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil
	}

	return detectJSFramework(extractDependencies(pkg))
}

// extractDependencies extracts dependencies from package.json
func extractDependencies(pkg map[string]interface{}) map[string]interface{} {
	deps := make(map[string]interface{})
	
	if d, ok := pkg["dependencies"].(map[string]interface{}); ok {
		for k, v := range d {
			deps[k] = v
		}
	}
	if d, ok := pkg["devDependencies"].(map[string]interface{}); ok {
		for k, v := range d {
			deps[k] = v
		}
	}
	
	return deps
}

// detectJSFramework detects JavaScript frameworks from dependencies
func detectJSFramework(deps map[string]interface{}) []string {
	var frameworks []string

	frameworkChecks := map[string]string{
		"react":     "React",
		"vue":       "Vue.js",
		"angular":   "Angular",
		"svelte":    "Svelte",
		"next":      "Next.js",
		"nuxt":      "Nuxt.js",
		"gatsby":    "Gatsby",
		"express":   "Express",
		"fastify":   "Fastify",
		"nestjs":    "NestJS",
		"socket.io": "Socket.IO",
	}

	for dep, framework := range frameworkChecks {
		if _, found := deps[dep]; found {
			frameworks = append(frameworks, framework)
		}
	}

	return frameworks
}

// detectGoFrameworks detects Go frameworks
func detectGoFrameworks(rootDir string) []string {
	goModPath := filepath.Join(rootDir, "go.mod")
	if !exists(goModPath) {
		return nil
	}

	data, err := os.ReadFile(goModPath)
	if err != nil {
		return nil
	}

	content := string(data)
	var frameworks []string

	goFrameworkChecks := map[string]string{
		"gin-gonic/gin":      "Gin",
		"gorilla/mux":        "Gorilla",
		"labstack/echo":      "Echo",
		"gofiber/fiber":      "Fiber",
		"beego/beego":        "Beego",
		"revel/revel":        "Revel",
	}

	for module, framework := range goFrameworkChecks {
		if strings.Contains(content, module) {
			frameworks = append(frameworks, framework)
		}
	}

	return frameworks
}

// detectDeploymentTargets detects deployment targets
func detectDeploymentTargets(insights *workspaceinfo.ProjectInsights, rootDir string) {
	var targets []string

	deploymentChecks := map[string]string{
		"Dockerfile":          "Docker",
		"docker-compose.yml":  "Docker Compose",
		"vercel.json":         "Vercel",
		"netlify.toml":        "Netlify",
		".platform.app.yaml": "Platform.sh",
		"app.yaml":            "Google App Engine",
		"Procfile":            "Heroku",
		"serverless.yml":      "Serverless",
	}

	for file, target := range deploymentChecks {
		if exists(filepath.Join(rootDir, file)) {
			targets = append(targets, target)
		}
	}

	// Kubernetes detection
	if hasKubernetesFiles(rootDir) {
		targets = append(targets, "Kubernetes")
	}

	insights.DeploymentTargets = strings.Join(targets, ", ")
}

// hasKubernetesFiles checks for Kubernetes configuration files
func hasKubernetesFiles(rootDir string) bool {
	kubernetesPatterns := []string{"k8s", "kubernetes", "*.yaml", "*.yml"}
	
	for _, pattern := range kubernetesPatterns {
		matches, _ := filepath.Glob(filepath.Join(rootDir, pattern))
		for _, match := range matches {
			if strings.Contains(match, "apiVersion") || 
			   strings.Contains(match, "kind:") ||
			   strings.Contains(match, "deployment") {
				return true
			}
		}
	}
	
	return false
}

// detectRepoLayout detects repository layout patterns
func detectRepoLayout(insights *workspaceinfo.ProjectInsights, rootDir string) {
	layout := "standard"

	// Check for common monorepo patterns
	if exists(filepath.Join(rootDir, "packages")) {
		layout = "packages-based"
	} else if exists(filepath.Join(rootDir, "apps")) && exists(filepath.Join(rootDir, "libs")) {
		layout = "apps-libs"
	} else if exists(filepath.Join(rootDir, "services")) {
		layout = "microservices"
	} else if hasMultipleMainDirectories(rootDir) {
		layout = "multi-project"
	}

	insights.RepoLayout = layout
}

// hasMultipleMainDirectories checks for multiple main project directories
func hasMultipleMainDirectories(rootDir string) bool {
	mainDirs := []string{"frontend", "backend", "api", "web", "mobile", "desktop", "server", "client"}
	count := 0
	
	for _, dir := range mainDirs {
		if exists(filepath.Join(rootDir, dir)) {
			count++
		}
	}
	
	return count > 1
}

// uniqueStrings removes duplicates from string slice
func uniqueStrings(in []string) []string {
	seen := make(map[string]bool)
	var result []string
	
	for _, str := range in {
		if !seen[str] {
			seen[str] = true
			result = append(result, str)
		}
	}
	
	return result
}

// exists checks if a file or directory exists
func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}