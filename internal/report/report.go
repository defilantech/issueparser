package report

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/defilan/issueparser/internal/analyzer"
)

type Report struct {
	analysis *analyzer.Analysis
	opts     Options
}

type Options struct {
	Title      string
	Repos      []string
	Keywords   []string
	IssueCount int
}

func New(analysis *analyzer.Analysis, opts Options) *Report {
	return &Report{
		analysis: analysis,
		opts:     opts,
	}
}

func (r *Report) WriteMarkdown(filename string) error {
	var sb strings.Builder

	// Header
	sb.WriteString(fmt.Sprintf("# %s\n\n", r.opts.Title))
	sb.WriteString(fmt.Sprintf("**Generated:** %s\n", time.Now().Format("January 2, 2006")))
	sb.WriteString(fmt.Sprintf("**Repositories:** %s\n", strings.Join(r.opts.Repos, ", ")))
	sb.WriteString(fmt.Sprintf("**Keywords:** %s\n", strings.Join(r.opts.Keywords, ", ")))
	sb.WriteString(fmt.Sprintf("**Issues Analyzed:** %d\n\n", r.opts.IssueCount))
	sb.WriteString("---\n\n")

	// Executive Summary
	sb.WriteString("## Executive Summary\n\n")
	if len(r.analysis.KeyInsights) > 0 {
		for _, insight := range r.analysis.KeyInsights {
			sb.WriteString(fmt.Sprintf("- %s\n", insight))
		}
		sb.WriteString("\n")
	} else {
		sb.WriteString(fmt.Sprintf("Analyzed %d issues and identified %d common themes.\n\n",
			r.opts.IssueCount, len(r.analysis.Themes)))
	}

	// Themes
	sb.WriteString("---\n\n")
	sb.WriteString("## Identified Themes\n\n")

	for i, theme := range r.analysis.Themes {
		// Theme header with severity badge
		severityBadge := r.severityBadge(theme.Severity)
		sb.WriteString(fmt.Sprintf("### %d. %s %s\n\n", i+1, theme.Name, severityBadge))

		// Issue count
		if theme.IssueCount > 0 {
			sb.WriteString(fmt.Sprintf("**Issues:** %d\n\n", theme.IssueCount))
		}

		// Description
		sb.WriteString(fmt.Sprintf("%s\n\n", theme.Description))

		// Examples/Quotes
		if len(theme.Examples) > 0 {
			sb.WriteString("**Example quotes:**\n")
			for _, example := range theme.Examples {
				if example != "" {
					sb.WriteString(fmt.Sprintf("> %s\n\n", example))
				}
			}
		}

		// Issue links
		if len(theme.IssueURLs) > 0 {
			sb.WriteString("**Related Issues:**\n")
			for _, url := range theme.IssueURLs {
				sb.WriteString(fmt.Sprintf("- %s\n", url))
			}
			sb.WriteString("\n")
		}

		sb.WriteString("---\n\n")
	}

	// Notable Quotes Section
	if len(r.analysis.Quotes) > 0 {
		sb.WriteString("## Notable Quotes\n\n")
		for _, quote := range r.analysis.Quotes {
			sb.WriteString(fmt.Sprintf("> \"%s\"\n", quote.Text))
			if quote.Source != "" {
				sb.WriteString(fmt.Sprintf("> — %s", quote.Source))
				if quote.IssueURL != "" {
					sb.WriteString(fmt.Sprintf(" ([link](%s))", quote.IssueURL))
				}
				sb.WriteString("\n")
			}
			sb.WriteString("\n")
		}
		sb.WriteString("---\n\n")
	}

	// Action Items
	if len(r.analysis.ActionItems) > 0 {
		sb.WriteString("## Potential Action Items\n\n")
		for _, item := range r.analysis.ActionItems {
			sb.WriteString(fmt.Sprintf("- [ ] %s\n", item))
		}
		sb.WriteString("\n---\n\n")
	}

	// LLMKube Attribution
	sb.WriteString("## Methodology\n\n")
	sb.WriteString("This analysis was performed using:\n")
	sb.WriteString("- **IssueParser** - GitHub issue theme analyzer\n")
	sb.WriteString("- **LLMKube** - Kubernetes-native LLM inference platform\n")
	sb.WriteString("- **Model:** Qwen 2.5 14B (dual GPU inference)\n\n")
	sb.WriteString("Issues were fetched via GitHub REST API, batched, and analyzed for common themes using LLM-powered pattern recognition.\n")

	return os.WriteFile(filename, []byte(sb.String()), 0644)
}

func (r *Report) severityBadge(severity string) string {
	switch strings.ToLower(severity) {
	case "high":
		return "🔴"
	case "medium":
		return "🟡"
	case "low":
		return "🟢"
	default:
		return ""
	}
}
