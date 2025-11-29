# IssueParser

**LLM-Powered GitHub Issue Theme Analyzer**

Scan GitHub repositories for issues and use AI to identify common pain points, recurring themes, and actionable insights.

<p>
  <img src="https://img.shields.io/badge/license-Apache%202.0-blue.svg" alt="License">
  <img src="https://img.shields.io/badge/go-1.21+-00ADD8.svg" alt="Go Version">
  <img src="https://img.shields.io/badge/platform-kubernetes-326CE5.svg" alt="Platform">
</p>

---

## Why IssueParser?

Understanding user pain points across large open-source projects is time-consuming. Manually reading through hundreds of GitHub issues to identify patterns is tedious and error-prone.

IssueParser automates this by:

- **Searching** GitHub issues for relevant keywords (multi-GPU, scaling, performance, etc.)
- **Batching** issues and sending them to an LLM for analysis
- **Extracting** themes, severity ratings, and notable quotes
- **Generating** a structured Markdown report with actionable insights

Built to work with [LLMKube](https://github.com/defilantech/LLMKube) for Kubernetes-native GPU inference.

---

## Quick Start

### Prerequisites

- Kubernetes cluster with NVIDIA GPUs (or any OpenAI-compatible LLM endpoint)
- [LLMKube](https://github.com/defilantech/LLMKube) controller installed (optional, for GPU inference)
- GitHub token (optional, increases API rate limits from 60 to 5,000 requests/hour)

### Option 1: Kubernetes Deployment (Recommended)

```bash
# 1. Install LLMKube (if not already installed)
helm repo add llmkube https://defilantech.github.io/LLMKube
helm install llmkube llmkube/llmkube --namespace llmkube-system --create-namespace

# 2. Create GitHub token secret (optional, for higher rate limits)
kubectl create secret generic github-token --from-literal=token=ghp_your_token

# 3. Deploy the LLM model
kubectl apply -f deploy/llmkube-qwen-14b.yaml

# 4. Wait for model to be ready (~10GB download)
kubectl wait --for=condition=Ready model/qwen-14b-issueparser --timeout=600s

# 5. Run the analysis job
kubectl apply -f deploy/job.yaml

# 6. Watch the logs
kubectl logs -f job/issueparser-analysis

# 7. Get the report
make get-results
```

### Option 2: Local Development

```bash
# Build
make build

# Port-forward LLMKube service (if using Kubernetes)
kubectl port-forward svc/qwen-14b-issueparser-service 8080:8080 &

# Run with any OpenAI-compatible endpoint
export GITHUB_TOKEN=ghp_your_token  # optional
./bin/issueparser \
  --llm-endpoint="http://localhost:8080" \
  --repos="ollama/ollama,vllm-project/vllm" \
  --keywords="multi-gpu,scale,performance"
```

---

## Features

### Analysis Capabilities
- **Multi-repo scanning** - Analyze issues from multiple repositories in a single run
- **Keyword search** - Filter issues by keywords in title/body
- **Label filtering** - Focus on specific issue labels (bug, enhancement, etc.)
- **Severity assessment** - LLM rates each theme as high/medium/low severity
- **Quote extraction** - Captures notable user quotes with source links

### Output
- **Structured Markdown report** with:
  - Executive summary
  - Identified themes with severity badges
  - Issue counts and example quotes
  - Links back to original GitHub issues
  - Actionable recommendations

### Technical
- **Pure Go** - No external dependencies, single static binary
- **OpenAI-compatible** - Works with any `/v1/chat/completions` endpoint
- **Batch processing** - Groups issues into manageable batches for LLM context
- **Rate limit aware** - Monitors GitHub API rate limits

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                    Kubernetes Cluster                           │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌──────────────────┐      ┌──────────────────────────────────┐│
│  │  IssueParser Job │      │  LLMKube InferenceService        ││
│  │                  │      │  (Qwen 2.5 14B on dual GPUs)     ││
│  │  1. Fetch issues │─────▶│                                  ││
│  │  2. Batch & send │      │  Endpoint: /v1/chat/completions  ││
│  │  3. Extract themes│◀────│                                  ││
│  │  4. Generate report│     │                                  ││
│  └──────────────────┘      └──────────────────────────────────┘│
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### How It Works

1. **Fetch Issues** - Uses GitHub REST API to search for issues containing specified keywords
2. **Batch Processing** - Groups issues into batches of 20 for LLM analysis
3. **Theme Extraction** - LLM identifies patterns, severity, and example quotes
4. **Synthesis** - Multiple batch analyses are combined into coherent themes
5. **Report Generation** - Outputs a structured Markdown report

---

## CLI Reference

```
Usage: issueparser [options]

Options:
  -repos string
        Comma-separated repos to analyze (default "ollama/ollama,vllm-project/vllm")
  -keywords string
        Keywords to search for (default "multi-gpu,scale,concurrency,production,performance")
  -labels string
        Filter by GitHub labels (comma-separated)
  -max-issues int
        Max issues per repo (default 100)
  -llm-endpoint string
        LLMKube/OpenAI-compatible service URL (default "http://qwen-14b-issueparser-service:8080")
  -llm-model string
        Model name (default "qwen-2.5-14b")
  -output string
        Output file (default "issue-analysis-report.md")
  -verbose
        Verbose output

Environment Variables:
  GITHUB_TOKEN    Optional GitHub personal access token for higher rate limits
```

### Examples

```bash
# Analyze Kubernetes repos for scaling issues
./issueparser \
  --repos="kubernetes/kubernetes,kubernetes-sigs/karpenter" \
  --keywords="autoscaling,resource,memory,OOM" \
  --max-issues=200

# Focus on bug reports only
./issueparser \
  --repos="ollama/ollama" \
  --labels="bug" \
  --keywords="crash,error,fail"

# Use a different LLM endpoint (e.g., local Ollama)
./issueparser \
  --llm-endpoint="http://localhost:11434" \
  --llm-model="llama3.2"
```

---

## Example Output

The tool generates a Markdown report like:

```markdown
# GitHub Issue Theme Analysis

**Generated:** November 28, 2025
**Repositories:** ollama/ollama, vllm-project/vllm
**Keywords:** multi-gpu, scale, concurrency, production
**Issues Analyzed:** 150

---

## Executive Summary

Users consistently report challenges with multi-GPU configurations,
particularly around layer distribution and VRAM utilization...

## Identified Themes

### 1. Multi-GPU Layer Distribution [HIGH]

**Issues:** 23

Multi-GPU setups don't distribute layers evenly, causing one GPU to be
overloaded while others sit idle...

**Example quotes:**
> "When I add a second GPU, performance actually gets worse..."

**Related Issues:**
- https://github.com/ollama/ollama/issues/1234
- https://github.com/ollama/ollama/issues/5678
```

---

## Performance

Benchmarked on dual NVIDIA RTX 5060 Ti (32GB VRAM total):

| Metric | Value |
|--------|-------|
| **Issues analyzed** | 200 |
| **Total job time** | ~12 minutes |
| **Prompt processing** | 1,200 tokens/sec |
| **Token generation** | 30 tokens/sec |
| **Cost per run** | ~$0.01 (electricity) |

See [BENCHMARK.md](BENCHMARK.md) for detailed performance data.

---

## Deployment

### Kubernetes Job

The `deploy/` directory contains manifests for running IssueParser as a Kubernetes Job:

- `llmkube-qwen-14b.yaml` - LLMKube Model CRD for Qwen 2.5 14B
- `job.yaml` - Kubernetes Job that runs the analysis

### Makefile Targets

```bash
make build          # Build the binary
make run            # Run locally with defaults
make run-k8s        # Run with port-forwarded LLMKube
make docker-build   # Build Docker image
make deploy         # Deploy to Kubernetes
make get-results    # Copy report from completed job
make logs           # Watch job logs
make clean          # Clean build artifacts
```

---

## Requirements

- Go 1.21+
- Docker (for container builds)
- kubectl (for Kubernetes deployment)
- LLMKube or any OpenAI-compatible LLM endpoint

---

## Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## Acknowledgments

- [LLMKube](https://github.com/defilantech/LLMKube) - Kubernetes-native LLM inference
- [llama.cpp](https://github.com/ggerganov/llama.cpp) - Efficient LLM inference engine
- [Qwen](https://huggingface.co/Qwen) - Open-weight LLM models

## License

Apache 2.0 - See [LICENSE](LICENSE) for details.
