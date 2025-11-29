package github

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	httpClient *http.Client
	token      string
	baseURL    string
}

type Issue struct {
	Number    int       `json:"number"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	State     string    `json:"state"`
	Labels    []Label   `json:"labels"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	HTMLURL   string    `json:"html_url"`
	Comments  int       `json:"comments"`
	Repo      string    `json:"-"` // Added by us
}

type Label struct {
	Name string `json:"name"`
}

type FetchOptions struct {
	Labels   []string
	Keywords []string
	MaxItems int
	State    string // "open", "closed", "all"
}

func NewClient(token string) *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		token:      token,
		baseURL:    "https://api.github.com",
	}
}

func (c *Client) FetchIssues(ctx context.Context, owner, repo string, opts FetchOptions) ([]Issue, error) {
	var allIssues []Issue

	// Use search API for keyword filtering
	if len(opts.Keywords) > 0 {
		return c.searchIssues(ctx, owner, repo, opts)
	}

	// Otherwise use the standard issues API
	page := 1
	perPage := 100
	if opts.MaxItems < perPage {
		perPage = opts.MaxItems
	}

	for len(allIssues) < opts.MaxItems {
		endpoint := fmt.Sprintf("%s/repos/%s/%s/issues?page=%d&per_page=%d&state=%s",
			c.baseURL, owner, repo, page, perPage, opts.State)

		if len(opts.Labels) > 0 && opts.Labels[0] != "" {
			endpoint += "&labels=" + url.QueryEscape(strings.Join(opts.Labels, ","))
		}

		issues, err := c.fetchPage(ctx, endpoint)
		if err != nil {
			return nil, err
		}

		if len(issues) == 0 {
			break
		}

		for i := range issues {
			issues[i].Repo = fmt.Sprintf("%s/%s", owner, repo)
		}

		allIssues = append(allIssues, issues...)
		page++

		if len(issues) < perPage {
			break
		}
	}

	if len(allIssues) > opts.MaxItems {
		allIssues = allIssues[:opts.MaxItems]
	}

	return allIssues, nil
}

func (c *Client) searchIssues(ctx context.Context, owner, repo string, opts FetchOptions) ([]Issue, error) {
	var allIssues []Issue

	// Build search query
	// Search for issues containing any of the keywords
	for _, keyword := range opts.Keywords {
		keyword = strings.TrimSpace(keyword)
		if keyword == "" {
			continue
		}

		query := fmt.Sprintf("%s repo:%s/%s is:issue", keyword, owner, repo)
		if opts.State != "" && opts.State != "all" {
			query += " state:" + opts.State
		}

		page := 1
		for len(allIssues) < opts.MaxItems {
			endpoint := fmt.Sprintf("%s/search/issues?q=%s&page=%d&per_page=100",
				c.baseURL, url.QueryEscape(query), page)

			result, err := c.fetchSearchPage(ctx, endpoint)
			if err != nil {
				// Rate limit or other error, continue with what we have
				fmt.Printf("    Warning: search error for '%s': %v\n", keyword, err)
				break
			}

			if len(result.Items) == 0 {
				break
			}

			for i := range result.Items {
				result.Items[i].Repo = fmt.Sprintf("%s/%s", owner, repo)
			}

			// Deduplicate by issue number
			for _, issue := range result.Items {
				isDupe := false
				for _, existing := range allIssues {
					if existing.Number == issue.Number && existing.Repo == issue.Repo {
						isDupe = true
						break
					}
				}
				if !isDupe {
					allIssues = append(allIssues, issue)
				}
			}

			page++
			if len(result.Items) < 100 {
				break
			}
		}
	}

	if len(allIssues) > opts.MaxItems {
		allIssues = allIssues[:opts.MaxItems]
	}

	return allIssues, nil
}

type searchResult struct {
	Items []Issue `json:"items"`
	Total int     `json:"total_count"`
}

func (c *Client) fetchSearchPage(ctx context.Context, endpoint string) (*searchResult, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "IssueParser/1.0") // GitHub requires User-Agent
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	// Check rate limit headers
	if remaining := resp.Header.Get("X-RateLimit-Remaining"); remaining != "" {
		if remaining == "0" {
			resetTime := resp.Header.Get("X-RateLimit-Reset")
			return nil, fmt.Errorf("rate limited, resets at %s", resetTime)
		}
	}

	if resp.StatusCode == 403 {
		return nil, fmt.Errorf("rate limited or forbidden")
	}

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var result searchResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (c *Client) fetchPage(ctx context.Context, endpoint string) ([]Issue, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "IssueParser/1.0")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	// Check rate limit headers
	if remaining := resp.Header.Get("X-RateLimit-Remaining"); remaining != "" {
		if remaining == "0" {
			resetTime := resp.Header.Get("X-RateLimit-Reset")
			return nil, fmt.Errorf("rate limited, resets at %s", resetTime)
		}
	}

	if resp.StatusCode == 403 {
		return nil, fmt.Errorf("rate limited or forbidden")
	}

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var issues []Issue
	if err := json.NewDecoder(resp.Body).Decode(&issues); err != nil {
		return nil, err
	}

	return issues, nil
}
