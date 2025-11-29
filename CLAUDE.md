# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Run Commands

```bash
make build          # Build binary to bin/issueparser
make test           # Run all tests
make run            # Build and run with defaults (requires LLM endpoint at localhost:8080)
make run-k8s        # Run with port-forwarded LLMKube service
make docker-build   # Build Docker image for AMD64
make deploy         # Deploy LLMKube model and job to Kubernetes
make get-results    # Copy report from completed K8s job
```

## Architecture

IssueParser is a Go CLI tool that analyzes GitHub issues using an LLM to identify recurring themes and pain points.

**Data Flow:**
1. `cmd/issueparser/main.go` - CLI entry point, parses flags, orchestrates workflow
2. `internal/github/client.go` - Fetches issues via GitHub REST/Search API
3. `internal/analyzer/analyzer.go` - Batches issues (20 per batch), sends to LLM, synthesizes results
4. `internal/llm/client.go` - OpenAI-compatible `/v1/chat/completions` client
5. `internal/report/report.go` - Generates Markdown report with themes, quotes, severity badges

**Key Design Decisions:**
- Pure Go with no external dependencies (standard library only)
- Works with any OpenAI-compatible endpoint (LLMKube, Ollama, OpenAI)
- Batch processing to handle LLM context limits
- LLM responses are parsed as JSON; falls back to raw text if parsing fails

## Environment Variables

- `GITHUB_TOKEN` - Optional GitHub PAT for higher API rate limits (60 → 5000 req/hr)

## CI/CD

- **Tests & Lint** - Run on every PR and push to main
- **Release Please** - Automates version bumps and changelogs based on conventional commits
- **GoReleaser** - Builds cross-platform binaries (linux/darwin × amd64/arm64) and Docker images on release
- Docker images published to `ghcr.io/defilantech/issueparser`

## Conventions

- Follow [Conventional Commits](https://www.conventionalcommits.org/): `feat:`, `fix:`, `docs:`, `refactor:`, `test:`, `chore:`
- Branch naming: `feat/*`, `fix/*`, `docs/*`, `refactor/*`
- Error handling: wrap errors with context using `fmt.Errorf("context: %w", err)`
