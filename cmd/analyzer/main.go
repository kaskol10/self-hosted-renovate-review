package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/google/go-github/v60/github"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
	"github.com/tmc/langchaingo/prompts"
	"golang.org/x/oauth2"
)

// LLMProvider represents the LLM provider type
type LLMProvider string

const (
	ProviderVLLM    LLMProvider = "vllm"    // Direct vLLM
	ProviderLiteLLM LLMProvider = "litellm" // LiteLLM proxy
)

// PRAnalyzer handles PR analysis
type PRAnalyzer struct {
	client      *github.Client
	repo        string
	prNumber    int
	llmBaseURL  string      // Base URL for LLM (vLLM or LiteLLM)
	llmAPIKey   string      // API key for LLM
	llmProvider LLMProvider // Provider type
	llmModel    string      // Model name
	llm         llms.Model  // LangChainGo LLM instance (required)
}

// NewPRAnalyzer creates a new PR analyzer instance
func NewPRAnalyzer(repo string, prNumber int, githubToken, llmBaseURL, llmAPIKey string, llmProvider LLMProvider) (*PRAnalyzer, error) {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: githubToken},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	// Default LLM URL if not provided
	if llmBaseURL == "" {
		switch llmProvider {
		case ProviderLiteLLM:
			llmBaseURL = "http://localhost:4000/v1" // LiteLLM default port
		case ProviderVLLM:
			llmBaseURL = "http://localhost:8000/v1" // vLLM default port
		default:
			llmBaseURL = "http://localhost:8000/v1"
		}
	}

	// Normalize URL - remove /chat/completions if present (should be base URL)
	llmBaseURL = strings.TrimSuffix(llmBaseURL, "/chat/completions")
	llmBaseURL = strings.TrimSuffix(llmBaseURL, "/v1/chat/completions")
	// Ensure it ends with /v1
	if !strings.HasSuffix(llmBaseURL, "/v1") {
		if strings.HasSuffix(llmBaseURL, "/") {
			llmBaseURL = llmBaseURL + "v1"
		} else {
			llmBaseURL = llmBaseURL + "/v1"
		}
	}

	// Get model name from env or use default
	modelName := os.Getenv("LLM_MODEL")
	if modelName == "" {
		modelName = "qwen3" // Default model name
	}

	analyzer := &PRAnalyzer{
		client:      client,
		repo:        repo,
		prNumber:    prNumber,
		llmBaseURL:  llmBaseURL,
		llmAPIKey:   llmAPIKey,
		llmProvider: llmProvider,
		llmModel:    modelName,
	}

	// Initialize LangChainGo LLM (required)

	// LangChainGo requires an API key, but vLLM doesn't need one
	// Use provided key, or a dummy key if none provided
	apiKey := llmAPIKey
	if apiKey == "" {
		// Check environment variable
		apiKey = os.Getenv("OPENAI_API_KEY")
		if apiKey == "" {
			// Use dummy key for vLLM (it doesn't validate the key)
			apiKey = "not-needed"
		}
	}

	// Set OPENAI_API_KEY env var for LangChainGo (it reads from env)
	// This is a workaround since LangChainGo checks for the env var
	if os.Getenv("OPENAI_API_KEY") == "" {
		os.Setenv("OPENAI_API_KEY", apiKey)
	}

	llm, err := openai.New(
		openai.WithBaseURL(llmBaseURL),
		openai.WithModel(modelName),
		openai.WithAPIType(openai.APITypeOpenAI),
		openai.WithToken(apiKey), // Explicitly set the token
	)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize LangChainGo LLM: %w", err)
	}
	analyzer.llm = llm

	return analyzer, nil
}

// GetPRInfo fetches PR information
func (a *PRAnalyzer) GetPRInfo() (*github.PullRequest, error) {
	parts := strings.Split(a.repo, "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid repo format: %s (expected owner/repo)", a.repo)
	}

	ctx := context.Background()
	pr, _, err := a.client.PullRequests.Get(ctx, parts[0], parts[1], a.prNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to get PR: %w", err)
	}

	return pr, nil
}

// GetPRFiles fetches changed files in the PR
func (a *PRAnalyzer) GetPRFiles() ([]*github.CommitFile, error) {
	parts := strings.Split(a.repo, "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid repo format: %s (expected owner/repo)", a.repo)
	}

	ctx := context.Background()
	files, _, err := a.client.PullRequests.ListFiles(ctx, parts[0], parts[1], a.prNumber, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get PR files: %w", err)
	}

	return files, nil
}

// isDependencyFile checks if a file is likely to contain dependency information
func isDependencyFile(fileName string) bool {
	fileName = strings.ToLower(fileName)

	// Package manager files
	dependencyFiles := []string{
		"package.json", "package-lock.json", "yarn.lock", "pnpm-lock.yaml",
		"requirements.txt", "pipfile", "poetry.lock", "pyproject.toml",
		"go.mod", "go.sum",
		"cargo.toml", "cargo.lock",
		"pom.xml", "build.gradle", "gradle.properties",
		"*.csproj", "*.sln", "packages.config",
		"gemfile", "gemfile.lock",
		"composer.json", "composer.lock",
		"pubspec.yaml",
		"mix.exs", "mix.lock",
		"podfile", "podfile.lock",
	}

	for _, depFile := range dependencyFiles {
		if strings.Contains(fileName, depFile) || strings.HasSuffix(fileName, depFile) {
			return true
		}
	}

	// Docker and Kubernetes files
	if strings.HasSuffix(fileName, ".yml") || strings.HasSuffix(fileName, ".yaml") {
		// Check if it's likely a Kubernetes/Helm/Docker Compose file
		if strings.Contains(fileName, "docker-compose") ||
			strings.Contains(fileName, "kubernetes") ||
			strings.Contains(fileName, "k8s") ||
			strings.Contains(fileName, "values.yaml") ||
			strings.Contains(fileName, "chart") ||
			strings.Contains(fileName, "helm") {
			return true
		}
	}

	// Dockerfile
	if strings.Contains(fileName, "dockerfile") {
		return true
	}

	return false
}

// GetFileDiffs collects file diffs from PR files for LLM analysis
// Only includes dependency-related files to avoid processing unrelated changes
func (a *PRAnalyzer) GetFileDiffs(files []*github.CommitFile) []struct {
	FileName string
	Diff     string
} {
	var diffs []struct {
		FileName string
		Diff     string
	}

	fmt.Printf("📄 Processing %d file(s)...\n", len(files))
	dependencyFileCount := 0

	for _, file := range files {
		if file.Patch == nil {
			continue
		}

		fileName := *file.Filename

		// Only process dependency-related files
		if !isDependencyFile(fileName) {
			continue
		}

		patch := *file.Patch
		dependencyFileCount++

		fmt.Printf("  📝 Collecting diff for %s\n", fileName)
		diffs = append(diffs, struct {
			FileName string
			Diff     string
		}{
			FileName: fileName,
			Diff:     patch,
		})
	}

	fmt.Printf("  ✅ Found %d dependency-related file(s) out of %d total file(s)\n", dependencyFileCount, len(files))
	return diffs
}

// AnalyzeWithLangChainGo uses LangChainGo for enhanced analysis with diff-based approach
func (a *PRAnalyzer) AnalyzeWithLangChainGo(pr *github.PullRequest, diffs []struct {
	FileName string
	Diff     string
}) (string, error) {
	if a.llm == nil {
		return "", fmt.Errorf("LangChainGo LLM not initialized")
	}

	prTitle := ""
	if pr.Title != nil {
		prTitle = *pr.Title
	}

	prBody := ""
	if pr.Body != nil {
		body := *pr.Body
		if len(body) > 2000 {
			body = body[:2000] + "..."
		}
		prBody = body
	}

	// Build diff summary
	var diffSummary strings.Builder
	for _, fileDiff := range diffs {
		// Limit diff size to avoid token limits (keep first 2000 chars per file)
		diffContent := fileDiff.Diff
		if len(diffContent) > 2000 {
			diffContent = diffContent[:2000] + "\n... (diff truncated)"
		}
		diffSummary.WriteString(fmt.Sprintf("\n**File: %s**\n```diff\n%s\n```\n", fileDiff.FileName, diffContent))
	}

	// Create prompt template using LangChainGo
	promptTemplate := prompts.NewPromptTemplate(
		`You are an expert software engineer specializing in dependency management and breaking change analysis. Your task is to provide clear, actionable insights that help developers make informed decisions about dependency updates.

## Context

**PR Title:** {{.pr_title}}

**PR Description:** {{.pr_description}}

**Code Changes (Diffs):**
{{.diff_summary}}

## Analysis Requirements

Analyze the provided diffs and provide a comprehensive, structured analysis. Follow this exact format:

### 📦 1. Dependency Changes Summary

List ALL dependency changes found in the diffs. For each change, specify:
- **Package/Image Name**: Exact name from the diff
- **Version Change**: Old version → New version (e.g., "1.2.3 → 2.0.0")
- **Update Type**: Major / Minor / Patch / Docker image tag
- **File Location**: Which file(s) contain this change

Supported formats:
- Node.js: package.json, package-lock.json, yarn.lock, pnpm-lock.yaml
- Python: requirements.txt, Pipfile, poetry.lock, pyproject.toml
- Go: go.mod, go.sum
- Rust: Cargo.toml, Cargo.lock
- Java: pom.xml, build.gradle
- .NET: *.csproj, *.sln, packages.config
- Ruby: Gemfile, Gemfile.lock
- PHP: composer.json, composer.lock
- Docker/Kubernetes: Look for "image:" lines or "repository:" + "tag:" pairs in YAML files

### ⚠️ 2. Breaking Changes Risk Assessment

For EACH dependency change, assess breaking change risk:

**Risk Level**: 🔴 HIGH / 🟡 MEDIUM / 🟢 LOW

**Reasoning**:
- Semantic versioning analysis (major bumps = HIGH risk)
- Known breaking changes in changelogs/release notes
- Deprecation warnings or removed features
- API/interface changes detected

**Specific Breaking Changes** (if any):
- List concrete breaking changes (e.g., "API method X removed", "Configuration format changed")
- Reference specific versions or changelog entries if known

### 📊 3. Impact Analysis

Assess the potential impact on the codebase:

**Affected Areas**:
- List specific files, modules, or components that might be affected
- Identify services or features that depend on these changes
- Note any transitive dependencies that might be impacted

**Potential Issues**:
- Runtime errors or exceptions that might occur
- Build/compilation issues
- Performance implications
- Security considerations

**Severity**: 🔴 Critical / 🟡 Moderate / 🟢 Low

### 🔄 4. Migration Requirements

Provide actionable migration steps if needed:

**Required Actions** (if breaking changes detected):
1. [Specific step 1 with code examples if applicable]
2. [Specific step 2]
3. [Continue as needed]

**Code Changes Needed**:
- List specific code locations that need updates
- Provide code examples or patterns if helpful
- Note any configuration file changes

**Estimated Effort**: [X hours/days] or "No changes required"

### 🧪 5. Testing Recommendations

Provide specific, actionable testing guidance:

**Critical Test Areas**:
- [Specific feature/component to test]
- [Specific functionality to verify]
- [Specific integration to check]

**Test Types**:
- **Unit Tests**: [Specific test files or functions to update/run]
- **Integration Tests**: [Specific integration scenarios to verify]
- **Manual Testing**: [Specific user flows or features to manually test]

**Regression Risks**:
- List specific areas where regressions are most likely
- Suggest test cases to add if missing

### 🎯 6. Confidence Level & Recommendation

**Confidence Level**: 
- 🔴 **LOW**: Significant uncertainty, requires thorough review
- 🟡 **MEDIUM**: Some uncertainty, review recommended
- 🟢 **HIGH**: High confidence, likely safe

**Reasoning**: [Explain why you assigned this confidence level]

**Recommendation**: 
- ✅ **MERGE**: Safe to merge, no action needed
- ⚠️ **REVIEW REQUIRED**: Requires human review before merging
- ❌ **DO NOT MERGE**: Contains breaking changes that need migration first

**Next Steps** (if not MERGE):
1. [Specific action item 1]
2. [Specific action item 2]
3. [Continue as needed]

## Output Format Guidelines

- Use clear markdown formatting with headers, lists, and code blocks
- Be specific and concrete - avoid vague statements
- Provide actionable guidance - tell developers exactly what to do
- Use emojis for visual clarity (as shown in the format above)
- If no issues found, clearly state "No breaking changes detected" and recommend merge
- If issues found, prioritize them by severity and provide clear remediation steps

## Important Notes

- Base your analysis ONLY on the diffs provided - do not make assumptions
- For Docker images, check both formats:
  - Direct: image: registry/image:tag
  - Structured: repository: "image" with tag: "version"
- When in doubt about breaking changes, err on the side of caution
- Provide specific file paths, function names, or code locations when possible
- If you cannot determine something from the diffs, state "Cannot determine from provided diffs" rather than guessing`,
		[]string{"pr_title", "pr_description", "diff_summary"},
	)

	// Format the prompt
	prompt, err := promptTemplate.Format(map[string]interface{}{
		"pr_title":       prTitle,
		"pr_description": prBody,
		"diff_summary":   diffSummary.String(),
	})
	if err != nil {
		return "", fmt.Errorf("failed to format prompt: %w", err)
	}

	// Build the full prompt with system message
	fullPrompt := fmt.Sprintf(`You are an expert software engineer specializing in dependency management and breaking change analysis. Your responses must be clear, actionable, and structured according to the format provided.

%s`, prompt)

	// Call LLM using LangChainGo
	ctx := context.Background()
	completion, err := a.llm.Call(ctx, fullPrompt, llms.WithTemperature(0.3), llms.WithMaxTokens(3000))
	if err != nil {
		return "", fmt.Errorf("LangChainGo LLM call failed: %w", err)
	}

	// Format the response
	analysis := fmt.Sprintf(`## 🤖 Renovate AI Analysis (Self-Hosted Models)

%s

---
*This analysis was automatically generated by Renovate AI using self-hosted models (%s/%s) via LangChainGo.*`, completion, a.llmProvider, a.llmModel)

	return analysis, nil
}

// AnalyzeWithAI sends analysis request using LangChainGo (required)
func (a *PRAnalyzer) AnalyzeWithAI(pr *github.PullRequest, diffs []struct {
	FileName string
	Diff     string
}) (string, error) {
	return a.AnalyzeWithLangChainGo(pr, diffs)
}

// PostComment posts analysis as a comment on the PR
func (a *PRAnalyzer) PostComment(analysis string) error {
	parts := strings.Split(a.repo, "/")
	if len(parts) != 2 {
		return fmt.Errorf("invalid repo format: %s (expected owner/repo)", a.repo)
	}

	// If analysis already includes the header (from LangChain), use it as-is
	// Otherwise, add our header
	var commentBody string
	providerName := string(a.llmProvider)
	if strings.Contains(analysis, "## 🤖 Renovate AI Analysis") {
		commentBody = analysis
	} else {
		commentBody = fmt.Sprintf(`## 🤖 Renovate AI Analysis (Self-Hosted Models)

%s

---
*This analysis was automatically generated by Renovate AI using self-hosted models (%s) via LangChainGo.*`, analysis, providerName)
	}

	comment := &github.IssueComment{
		Body: &commentBody,
	}

	ctx := context.Background()
	_, _, err := a.client.Issues.CreateComment(ctx, parts[0], parts[1], a.prNumber, comment)
	if err != nil {
		return fmt.Errorf("failed to post comment: %w", err)
	}

	return nil
}

// Run executes the full analysis workflow
func (a *PRAnalyzer) Run() error {
	fmt.Printf("🔍 Analyzing PR #%d in %s...\n", a.prNumber, a.repo)

	// Fetch PR information
	pr, err := a.GetPRInfo()
	if err != nil {
		return err
	}

	// Fetch PR files
	files, err := a.GetPRFiles()
	if err != nil {
		return err
	}

	// Get file diffs for LLM analysis
	diffs := a.GetFileDiffs(files)

	if len(diffs) == 0 {
		fmt.Println("ℹ️  No file changes detected. Skipping analysis.")
		return nil
	}

	fmt.Printf("📦 Found %d file(s) with changes\n", len(diffs))

	// Analyze with AI using LangChainGo (pass diffs directly)
	providerName := string(a.llmProvider)
	fmt.Printf("🤖 Running AI analysis with LangChainGo (%s)...\n", providerName)
	analysis, err := a.AnalyzeWithAI(pr, diffs)
	if err != nil {
		return fmt.Errorf("AI analysis failed: %w", err)
	}

	// Post comment
	if err := a.PostComment(analysis); err != nil {
		return err
	}

	fmt.Printf("✅ Posted analysis comment to PR #%d\n", a.prNumber)
	fmt.Println("✅ Analysis complete!")

	return nil
}

func main() {
	var (
		repo        = flag.String("repo", "", "Repository name (owner/repo)")
		prNumber    = flag.Int("pr-number", 0, "PR number")
		githubToken = flag.String("github-token", "", "GitHub token")
		llmURL      = flag.String("llm-url", "", "LLM API base URL (default: http://localhost:8000/v1 for vLLM, http://localhost:4000/v1 for LiteLLM)")
		llmKey      = flag.String("llm-key", "", "LLM API key (optional)")
		llmProvider = flag.String("llm-provider", "vllm", "LLM provider: 'vllm' (direct) or 'litellm' (proxy)")
	)
	flag.Parse()

	if *repo == "" || *prNumber == 0 || *githubToken == "" {
		fmt.Fprintln(os.Stderr, "Error: --repo, --pr-number, and --github-token are required")
		flag.Usage()
		os.Exit(1)
	}

	// Get LLM provider
	providerStr := *llmProvider
	if providerStr == "" {
		providerStr = os.Getenv("LLM_PROVIDER")
		if providerStr == "" {
			providerStr = "vllm" // Default to vLLM
		}
	}
	provider := LLMProvider(strings.ToLower(providerStr))
	if provider != ProviderVLLM && provider != ProviderLiteLLM {
		fmt.Fprintf(os.Stderr, "Error: Invalid LLM provider '%s'. Must be 'vllm' or 'litellm'\n", providerStr)
		os.Exit(1)
	}

	// Get LLM URL from env if not provided
	llmBaseURL := *llmURL
	if llmBaseURL == "" {
		// Try new env var first, then fallback to old vLLM-specific var for backward compatibility
		llmBaseURL = os.Getenv("LLM_API_URL")
		if llmBaseURL == "" {
			llmBaseURL = os.Getenv("VLLM_API_URL") // Backward compatibility
		}
		if llmBaseURL == "" {
			// Use provider-specific defaults
			if provider == ProviderLiteLLM {
				llmBaseURL = "http://localhost:4000/v1"
			} else {
				llmBaseURL = "http://localhost:8000/v1"
			}
		}
	}

	// Get LLM key from env if not provided
	llmAPIKey := *llmKey
	if llmAPIKey == "" {
		// Try new env var first, then fallback to old vLLM-specific var for backward compatibility
		llmAPIKey = os.Getenv("LLM_API_KEY")
		if llmAPIKey == "" {
			llmAPIKey = os.Getenv("VLLM_API_KEY") // Backward compatibility
		}
	}

	// LangChainGo is now mandatory - always initialized
	analyzer, err := NewPRAnalyzer(*repo, *prNumber, *githubToken, llmBaseURL, llmAPIKey, provider)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error initializing analyzer: %v\n", err)
		os.Exit(1)
	}

	if err := analyzer.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Error: %v\n", err)
		os.Exit(1)
	}
}
