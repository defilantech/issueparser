package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/defilan/issueparser/internal/analyzer"
	"github.com/defilan/issueparser/internal/github"
	"github.com/defilan/issueparser/internal/llm"
	"github.com/defilan/issueparser/internal/report"
)

func main() {
	// CLI flags
	var (
		repos       string
		labels      string
		keywords    string
		maxIssues   int
		llmEndpoint string
		llmModel    string
		outputFile  string
		verbose     bool
	)

	flag.StringVar(&repos, "repos", "ollama/ollama,vllm-project/vllm", "Comma-separated list of repos (owner/repo). Examples: ollama/ollama, vllm-project/vllm, ggerganov/llama.cpp")
	flag.StringVar(&labels, "labels", "", "Filter by labels (comma-separated)")
	flag.StringVar(&keywords, "keywords", "multi-gpu,scale,concurrency,production,performance", "Keywords to search for in issues")
	flag.IntVar(&maxIssues, "max-issues", 100, "Maximum issues to fetch per repo")
	flag.StringVar(&llmEndpoint, "llm-endpoint", "http://qwen-14b-issueparser-service:8080", "LLMKube service endpoint")
	flag.StringVar(&llmModel, "llm-model", "qwen-2.5-14b", "Model name for API calls")
	flag.StringVar(&outputFile, "output", "issue-analysis-report.md", "Output file for the report")
	flag.BoolVar(&verbose, "verbose", false, "Enable verbose output")
	flag.Parse()

	ctx := context.Background()

	// Get GitHub token from environment
	ghToken := os.Getenv("GITHUB_TOKEN")
	if ghToken == "" {
		fmt.Fprintln(os.Stderr, "Warning: GITHUB_TOKEN not set, API rate limits will be restrictive")
	}

	// Initialize components
	ghClient := github.NewClient(ghToken)
	llmClient := llm.NewClient(llmEndpoint, llmModel)
	themeAnalyzer := analyzer.New(llmClient)

	fmt.Println("=== IssueParser: GitHub Issue Theme Analyzer ===")
	fmt.Printf("Repos: %s\n", repos)
	fmt.Printf("Keywords: %s\n", keywords)
	fmt.Printf("LLM Endpoint: %s\n", llmEndpoint)
	fmt.Println()

	// Parse repos
	repoList := strings.Split(repos, ",")
	keywordList := strings.Split(keywords, ",")

	var allIssues []github.Issue

	// Fetch issues from each repo
	for _, repo := range repoList {
		repo = strings.TrimSpace(repo)
		parts := strings.Split(repo, "/")
		if len(parts) != 2 {
			fmt.Fprintf(os.Stderr, "Invalid repo format: %s (expected owner/repo)\n", repo)
			continue
		}

		fmt.Printf("Fetching issues from %s...\n", repo)
		issues, err := ghClient.FetchIssues(ctx, parts[0], parts[1], github.FetchOptions{
			Labels:   strings.Split(labels, ","),
			Keywords: keywordList,
			MaxItems: maxIssues,
			State:    "all", // both open and closed
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error fetching issues from %s: %v\n", repo, err)
			continue
		}

		fmt.Printf("  Found %d relevant issues\n", len(issues))
		allIssues = append(allIssues, issues...)
	}

	if len(allIssues) == 0 {
		fmt.Println("No issues found matching criteria")
		os.Exit(0)
	}

	fmt.Printf("\nTotal issues to analyze: %d\n", len(allIssues))
	fmt.Println("\nAnalyzing issues with LLM (this may take a while)...")

	// Analyze issues for themes
	analysis, err := themeAnalyzer.AnalyzeIssues(ctx, allIssues, analyzer.Options{
		FocusAreas: keywordList,
		Verbose:    verbose,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error analyzing issues: %v\n", err)
		os.Exit(1)
	}

	// Generate report
	fmt.Printf("\nGenerating report to %s...\n", outputFile)
	rpt := report.New(analysis, report.Options{
		Title:      "GitHub Issue Theme Analysis",
		Repos:      repoList,
		Keywords:   keywordList,
		IssueCount: len(allIssues),
	})

	if err := rpt.WriteMarkdown(outputFile); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing report: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\n=== Analysis Complete ===")
	fmt.Printf("Report saved to: %s\n", outputFile)
	fmt.Printf("Themes identified: %d\n", len(analysis.Themes))

	// Print summary to stdout
	fmt.Println("\n--- Quick Summary ---")
	for i, theme := range analysis.Themes {
		if i >= 5 {
			fmt.Printf("  ... and %d more themes\n", len(analysis.Themes)-5)
			break
		}
		fmt.Printf("  %d. %s (%d issues)\n", i+1, theme.Name, theme.IssueCount)
	}
}
