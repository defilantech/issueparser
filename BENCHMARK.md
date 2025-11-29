# IssueParser Benchmark Results

**Date:** November 29, 2025
**Platform:** LLMKube on MicroK8s
**Server:** ShadowStack (home lab server designed for LLMKube workloads)
**Hardware:** Dual NVIDIA RTX 5060 Ti (16GB VRAM each, 32GB total)

---

## Hardware Configuration

| Component | Specification |
|-----------|---------------|
| **GPUs** | 2x NVIDIA GeForce RTX 5060 Ti |
| **VRAM per GPU** | 16 GB GDDR7 |
| **Total VRAM** | 32 GB |
| **GPU Sharding** | Layer-based (tensor-split 1,1) |
| **Kubernetes** | MicroK8s 1.28 |
| **Inference Engine** | llama.cpp (CUDA) |

---

## Model Configuration

| Setting | Value |
|---------|-------|
| **Model** | Qwen 2.5 14B Instruct |
| **Quantization** | Q5_K_M |
| **Model Size** | 9.8 GB |
| **Context Window** | 4,096 tokens (truncated from 131K) |
| **VRAM Usage** | ~6 GB per GPU (~12 GB total) |
| **VRAM Headroom** | ~20 GB available for larger batches |

---

## Inference Performance

### Prompt Processing (Prefill)
| Metric | Value |
|--------|-------|
| **Speed** | 1,080 - 1,296 tokens/second |
| **Latency** | 0.77 - 0.92 ms/token |
| **Typical batch** | ~2,000-4,000 tokens |
| **Prefill time** | 1.6 - 3.7 seconds |

### Token Generation (Decode)
| Metric | Value |
|--------|-------|
| **Speed** | 29.7 - 30.0 tokens/second |
| **Latency** | 33.3 - 33.7 ms/token |
| **Max tokens/request** | 2,000-3,000 |
| **Generation time** | 35 - 100 seconds |

### End-to-End Request Times
| Request Type | Tokens | Time |
|--------------|--------|------|
| Small batch (20 issues) | ~3,500 total | 37 seconds |
| Medium batch (20 issues) | ~5,000 total | 70 seconds |
| Large synthesis | ~7,000 total | 104 seconds |

---

## Workload Summary

### IssueParser Job Metrics
| Metric | Value |
|--------|-------|
| **Repositories scanned** | 2 (ollama/ollama, vllm-project/vllm) |
| **Issues fetched** | 200 |
| **Batch size** | 20 issues |
| **Total batches** | 10 + 1 synthesis |
| **LLM requests** | 11 |
| **Total tokens processed** | ~50,000+ |
| **Total job time** | ~12 minutes |

### Throughput
| Metric | Value |
|--------|-------|
| **Issues/minute** | ~17 |
| **Tokens/minute** | ~4,200 |
| **GPU utilization** | Variable (0-100% during inference) |

---

## Cost Analysis (If Cloud-Hosted)

For comparison, if this workload ran on cloud infrastructure:

| Provider | GPU Type | $/hr | Job Cost |
|----------|----------|------|----------|
| **Self-hosted (RTX 5060 Ti x2)** | Consumer GPUs | ~$0.05* | ~$0.01 |
| AWS | g5.xlarge (A10G) | $1.00 | ~$0.20 |
| GCP | L4 | $0.70 | ~$0.14 |
| Azure | NC A10 v4 | $0.90 | ~$0.18 |

*Electricity cost estimate only

### Key Insight
**Self-hosted inference is 10-20x cheaper** than cloud GPU instances for workloads like this.

---

## Performance Comparison

### vs Ollama (Single GPU)
Based on community benchmarks for Qwen 14B:

| Metric | LLMKube (Dual GPU) | Ollama (Single GPU) | Improvement |
|--------|-------------------|---------------------|-------------|
| Prompt processing | 1,200 tok/s | 400-600 tok/s | **2-3x faster** |
| Token generation | 30 tok/s | 25-30 tok/s | Similar |
| Max model size | 32GB VRAM | 16GB VRAM | **2x capacity** |
| Context handling | Smooth at 4K | Struggles at 4K | Better stability |

### vs vLLM
| Feature | LLMKube | vLLM |
|---------|---------|------|
| Multi-GPU support | Layer sharding | Tensor parallelism |
| Setup complexity | Kubernetes CRD | Python config |
| Model format | GGUF (quantized) | Native weights |
| VRAM efficiency | High (Q5 quant) | Lower (FP16/BF16) |
| Production readiness | K8s-native | Requires wrapper |

---

## Observations

### What Worked Well
1. **Dual GPU layer sharding** - Even split across both 5060 Ti cards
2. **VRAM efficiency** - Only ~12GB used for 14B model, leaving headroom
3. **Prompt processing** - 1,200+ tok/s is excellent for prefill
4. **Kubernetes integration** - Declarative deployment via CRDs

### Areas for Improvement
1. **Token generation** - 30 tok/s is limited by memory bandwidth
2. **Context truncation** - 4K context limit truncated some large prompts
3. **DNS issues** - MicroK8s networking required hostNetwork workaround

### Recommendations
- For longer context: Use Q4 quantization to reduce memory footprint
- For production: Increase context window to 8K+ with memory optimization
- For larger models: LLMKube's automatic layer sharding makes it easy to scale to 70B+ models

---

## Reproducibility

### Deploy LLMKube Model
```yaml
apiVersion: inference.llmkube.dev/v1alpha1
kind: Model
metadata:
  name: qwen-14b-issueparser
spec:
  source: https://huggingface.co/bartowski/Qwen2.5-14B-Instruct-GGUF/resolve/main/Qwen2.5-14B-Instruct-Q5_K_M.gguf
  format: gguf
  quantization: Q5_K_M
  hardware:
    accelerator: cuda
    gpu:
      enabled: true
      count: 2
      vendor: nvidia
      layers: -1
      sharding:
        strategy: layer
```

### Run Benchmark
```bash
# Clone and build
git clone https://github.com/defilan/issueparser
cd issueparser
make build

# Deploy to Kubernetes (see README for full instructions)
kubectl apply -f deploy/llmkube-qwen-14b.yaml
kubectl wait --for=condition=Ready model/qwen-14b-issueparser --timeout=600s
kubectl apply -f deploy/job.yaml

# Watch logs
kubectl logs -f job/issueparser-analysis
```

---

## Summary

| Highlight | Value |
|-----------|-------|
| **Model** | Qwen 2.5 14B (Q5_K_M) |
| **Hardware** | Dual RTX 5060 Ti (32GB VRAM) |
| **Prompt throughput** | 1,200 tokens/second |
| **Generation speed** | 30 tokens/second |
| **Issues analyzed** | 200 in 12 minutes |
| **Cost per analysis** | ~$0.01 (electricity) |

**Bottom line:** A $800 GPU pair can run production-quality 14B parameter LLM inference at 30 tok/s, processing enterprise workloads for pennies instead of dollars.
