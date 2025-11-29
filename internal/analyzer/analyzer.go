package analyzer

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/defilan/issueparser/internal/github"
	"github.com/defilan/issueparser/internal/llm"
)

type Analyzer struct {
	llm *llm.Client
}

type Options struct {
	FocusAreas []string
	Verbose    bool
}

type Analysis struct {
	Themes        []Theme       `json:"themes"`
	KeyInsights   []string      `json:"key_insights"`
	Quotes        []Quote       `json:"quotes"`
	ActionItems   []string      `json:"action_items"`
	RawIssueCount int           `json:"raw_issue_count"`
}

type Theme struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	IssueCount  int      `json:"issue_count"`
	Severity    string   `json:"severity"` // high, medium, low
	IssueURLs   []string `json:"issue_urls"`
	Examples    []string `json:"examples"`
}

type Quote struct {
	Text     string `json:"text"`
	Source   string `json:"source"`
	IssueURL string `json:"issue_url"`
}

func New(llmClient *llm.Client) *Analyzer {
	return &Analyzer{llm: llmClient}
}

func (a *Analyzer) AnalyzeIssues(ctx context.Context, issues []github.Issue, opts Options) (*Analysis, error) {
	// Process in batches to avoid overwhelming the LLM context
	batchSize := 20
	var batchAnalyses []string

	for i := 0; i < len(issues); i += batchSize {
		end := i + batchSize
		if end > len(issues) {
			end = len(issues)
		}

		batch := issues[i:end]
		fmt.Printf("  Analyzing batch %d-%d of %d issues...\n", i+1, end, len(issues))

		batchAnalysis, err := a.analyzeBatch(ctx, batch, opts)
		if err != nil {
			fmt.Printf("  Warning: batch analysis failed: %v\n", err)
			continue
		}
		batchAnalyses = append(batchAnalyses, batchAnalysis)
	}

	// Synthesize all batch analyses into final themes
	fmt.Println("  Synthesizing themes across all batches...")
	return a.synthesizeAnalyses(ctx, batchAnalyses, issues, opts)
}

func (a *Analyzer) analyzeBatch(ctx context.Context, issues []github.Issue, opts Options) (string, error) {
	// Build issue summaries for the prompt
	var issueSummaries strings.Builder
	for _, issue := range issues {
		body := issue.Body
		if len(body) > 500 {
			body = body[:500] + "..."
		}
		// Clean up markdown and newlines for cleaner prompt
		body = strings.ReplaceAll(body, "\r\n", " ")
		body = strings.ReplaceAll(body, "\n", " ")

		labels := make([]string, len(issue.Labels))
		for i, l := range issue.Labels {
			labels[i] = l.Name
		}

		fmt.Fprintf(&issueSummaries, "---\nIssue #%d [%s]: %s\nLabels: %s\nComments: %d\nBody: %s\nURL: %s\n",
			issue.Number, issue.State, issue.Title,
			strings.Join(labels, ", "),
			issue.Comments,
			body,
			issue.HTMLURL)
	}

	systemPrompt := `You are an expert software analyst. Analyze GitHub issues to identify recurring themes and pain points.

IMPORTANT: Respond with ONLY valid JSON, no markdown, no explanations. Keep responses concise.

Required JSON structure:
{"themes":[{"name":"string","description":"string","issue_numbers":[1,2],"severity":"high|medium|low","example_quotes":["quote"]}],"notable_quotes":[{"text":"quote","issue_number":1}]}`

	focusAreas := strings.Join(opts.FocusAreas, ", ")
	userPrompt := fmt.Sprintf(`Analyze these issues for themes about: %s

%s

Respond with JSON only. Identify 3-5 themes with severity ratings.`, focusAreas, issueSummaries.String())

	response, err := a.llm.Complete(ctx, systemPrompt, userPrompt, 1000)
	if err != nil {
		return "", err
	}

	return response, nil
}

func (a *Analyzer) synthesizeAnalyses(ctx context.Context, batchAnalyses []string, issues []github.Issue, opts Options) (*Analysis, error) {
	// Build issue URL lookup
	issueURLs := make(map[int]string)
	for _, issue := range issues {
		issueURLs[issue.Number] = issue.HTMLURL
	}

	if len(batchAnalyses) == 0 {
		return &Analysis{RawIssueCount: len(issues)}, nil
	}

	// If only one batch, parse it directly
	if len(batchAnalyses) == 1 {
		return a.parseAnalysis(batchAnalyses[0], issueURLs, len(issues))
	}

	// Otherwise, ask LLM to synthesize
	systemPrompt := `You synthesize multiple issue analyses into a final report. Merge similar themes, rank by importance.

IMPORTANT: Respond with ONLY valid JSON. No markdown, no explanations. Be concise.

Required JSON structure:
{"themes":[{"name":"string","description":"string","issue_count":10,"severity":"high|medium|low","examples":["quote1","quote2"]}],"key_insights":["insight1"],"action_items":["action1"]}`

	var analysesText strings.Builder
	for i, analysis := range batchAnalyses {
		fmt.Fprintf(&analysesText, "Batch %d:\n%s\n", i+1, analysis)
	}

	userPrompt := fmt.Sprintf(`Synthesize these analyses about %s into 5-7 final themes:

%s

Respond with JSON only.`, strings.Join(opts.FocusAreas, ", "), analysesText.String())

	response, err := a.llm.Complete(ctx, systemPrompt, userPrompt, 1500)
	if err != nil {
		return nil, fmt.Errorf("synthesis failed: %w", err)
	}

	return a.parseAnalysis(response, issueURLs, len(issues))
}

func (a *Analyzer) parseAnalysis(response string, issueURLs map[int]string, issueCount int) (*Analysis, error) {
	// Extract JSON from response (it might have markdown code blocks)
	jsonStr := response
	if idx := strings.Index(response, "```json"); idx != -1 {
		jsonStr = response[idx+7:]
		if endIdx := strings.Index(jsonStr, "```"); endIdx != -1 {
			jsonStr = jsonStr[:endIdx]
		}
	} else if idx := strings.Index(response, "```"); idx != -1 {
		jsonStr = response[idx+3:]
		if endIdx := strings.Index(jsonStr, "```"); endIdx != -1 {
			jsonStr = jsonStr[:endIdx]
		}
	}
	jsonStr = strings.TrimSpace(jsonStr)

	// Try to parse the JSON
	var rawAnalysis struct {
		Themes []struct {
			Name         string   `json:"name"`
			Description  string   `json:"description"`
			IssueNumbers []int    `json:"issue_numbers"`
			IssueCount   int      `json:"issue_count"`
			Severity     string   `json:"severity"`
			Examples     []string `json:"examples"`
			ExampleQuotes []string `json:"example_quotes"`
		} `json:"themes"`
		KeyInsights   []string `json:"key_insights"`
		NotableQuotes []struct {
			Text        string `json:"text"`
			IssueNumber int    `json:"issue_number"`
		} `json:"notable_quotes"`
		ActionItems []string `json:"action_items"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &rawAnalysis); err != nil {
		// If JSON parsing fails, create a basic analysis with the raw response
		return &Analysis{
			Themes: []Theme{{
				Name:        "Raw Analysis",
				Description: response,
				IssueCount:  issueCount,
				Severity:    "medium",
			}},
			RawIssueCount: issueCount,
		}, nil
	}

	// Convert to our Analysis struct
	analysis := &Analysis{
		RawIssueCount: issueCount,
		KeyInsights:   rawAnalysis.KeyInsights,
		ActionItems:   rawAnalysis.ActionItems,
	}

	for _, t := range rawAnalysis.Themes {
		theme := Theme{
			Name:        t.Name,
			Description: t.Description,
			Severity:    t.Severity,
			IssueCount:  t.IssueCount,
		}

		// Use issue_numbers if issue_count not set
		if theme.IssueCount == 0 && len(t.IssueNumbers) > 0 {
			theme.IssueCount = len(t.IssueNumbers)
		}

		// Map issue numbers to URLs
		for _, num := range t.IssueNumbers {
			if url, ok := issueURLs[num]; ok {
				theme.IssueURLs = append(theme.IssueURLs, url)
			}
		}

		// Combine examples
		theme.Examples = append(theme.Examples, t.Examples...)
		theme.Examples = append(theme.Examples, t.ExampleQuotes...)

		analysis.Themes = append(analysis.Themes, theme)
	}

	// Convert notable quotes
	for _, q := range rawAnalysis.NotableQuotes {
		quote := Quote{
			Text: q.Text,
		}
		if url, ok := issueURLs[q.IssueNumber]; ok {
			quote.IssueURL = url
			quote.Source = fmt.Sprintf("Issue #%d", q.IssueNumber)
		}
		analysis.Quotes = append(analysis.Quotes, quote)
	}

	return analysis, nil
}
