# Contributing to IssueParser

Thank you for your interest in contributing to IssueParser! This document provides guidelines and information for contributors.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [Making Changes](#making-changes)
- [Pull Request Process](#pull-request-process)
- [Coding Standards](#coding-standards)

---

## Code of Conduct

### Our Pledge

We are committed to making participation in this project a harassment-free experience for everyone, regardless of background or identity.

### Expected Behavior

- Be respectful and inclusive
- Provide constructive feedback
- Focus on what is best for the community
- Show empathy towards other contributors

### Unacceptable Behavior

- Harassment, trolling, or personal attacks
- Publishing others' private information
- Other conduct which could reasonably be considered inappropriate

---

## Getting Started

### Reporting Bugs

Before creating a bug report, please check existing issues to avoid duplicates.

**When filing a bug, include:**

1. **Description** - Clear description of the bug
2. **Steps to Reproduce** - Numbered steps to reproduce the issue
3. **Expected Behavior** - What you expected to happen
4. **Actual Behavior** - What actually happened
5. **Environment** - Go version, OS, Kubernetes version if applicable
6. **Logs** - Relevant error messages or logs

### Suggesting Features

Feature requests are welcome! Please include:

1. **Problem Statement** - What problem does this solve?
2. **Proposed Solution** - How would you like it to work?
3. **Alternatives Considered** - Other approaches you've thought about
4. **Additional Context** - Any other relevant information

### First-Time Contributors

Look for issues labeled:
- `good first issue` - Simple issues for newcomers
- `help wanted` - Issues where we need community help

---

## Development Setup

### Prerequisites

- Go 1.21 or later
- Docker (for container builds)
- kubectl (for Kubernetes testing)
- Access to an OpenAI-compatible LLM endpoint (optional, for integration testing)

### Clone and Build

```bash
# Clone the repository
git clone https://github.com/defilan/issueparser
cd issueparser

# Build the binary
make build

# Run tests
make test

# Run locally (requires LLM endpoint)
export GITHUB_TOKEN=your_token  # optional
./bin/issueparser --llm-endpoint="http://localhost:8080"
```

### Project Structure

```
issueparser/
├── cmd/
│   └── issueparser/
│       └── main.go          # CLI entry point
├── internal/
│   ├── analyzer/
│   │   └── analyzer.go      # Theme extraction logic
│   ├── github/
│   │   └── client.go        # GitHub API client
│   ├── llm/
│   │   └── client.go        # LLM API client
│   └── report/
│       └── report.go        # Markdown report generator
├── deploy/
│   ├── llmkube-qwen-14b.yaml
│   └── job.yaml
├── scripts/
│   └── deploy-to-remote.sh
├── Dockerfile
├── Makefile
└── go.mod
```

---

## Making Changes

### Branching Strategy

Create branches from `main` using this naming convention:

| Branch Type | Pattern | Example |
|-------------|---------|---------|
| Feature | `feat/*` | `feat/add-label-filter` |
| Bug Fix | `fix/*` | `fix/rate-limit-handling` |
| Documentation | `docs/*` | `docs/improve-readme` |
| Refactor | `refactor/*` | `refactor/simplify-analyzer` |

### Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>: <description>

[optional body]

[optional footer]
```

**Types:**
- `feat:` - New feature
- `fix:` - Bug fix
- `docs:` - Documentation only
- `refactor:` - Code change that neither fixes a bug nor adds a feature
- `test:` - Adding or updating tests
- `chore:` - Maintenance tasks

**Examples:**
```
feat: add support for filtering by issue state

fix: handle GitHub API rate limit errors gracefully

docs: add examples for custom LLM endpoints
```

---

## Pull Request Process

### Before Submitting

- [ ] Code builds without errors (`make build`)
- [ ] Tests pass (`make test`)
- [ ] Code follows Go style guidelines
- [ ] Commit messages follow conventional commits format
- [ ] Documentation updated if needed

### PR Title Format

Use the same format as commit messages:
```
feat: add support for filtering by issue state
```

### Review Process

1. Submit your PR against `main`
2. Ensure CI checks pass
3. Wait for maintainer review
4. Address any feedback
5. Once approved, a maintainer will merge

---

## Coding Standards

### Go Style

- Follow [Effective Go](https://golang.org/doc/effective_go.html)
- Use `gofmt` for formatting
- Keep functions focused and small
- Handle errors explicitly

### Code Organization

```go
// Good: Clear error handling
resp, err := client.FetchIssues(ctx, owner, repo, opts)
if err != nil {
    return nil, fmt.Errorf("failed to fetch issues: %w", err)
}

// Good: Descriptive variable names
issuesByTheme := make(map[string][]Issue)

// Good: Comments explain "why", not "what"
// Skip closed issues older than 1 year to focus on relevant themes
if issue.ClosedAt.Before(cutoffDate) {
    continue
}
```

### Error Handling

- Wrap errors with context using `fmt.Errorf("context: %w", err)`
- Return errors rather than logging and continuing
- Provide actionable error messages

### Testing

- Write table-driven tests where appropriate
- Test error cases, not just happy paths
- Keep tests focused on one behavior

---

## Questions?

- Open a [GitHub Issue](https://github.com/defilan/issueparser/issues) for bugs or feature requests
- Check existing issues before creating new ones

Thank you for contributing!
