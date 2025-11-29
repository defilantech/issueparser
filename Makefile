.PHONY: build run test docker-build docker-push deploy clean

# Variables
IMAGE_NAME ?= issueparser
IMAGE_TAG ?= latest
# MicroK8s registry runs on localhost:32000
REGISTRY ?= localhost:32000

# Local development
build:
	go build -o bin/issueparser ./cmd/issueparser

run: build
	./bin/issueparser \
		--repos="ollama/ollama,vllm-project/vllm" \
		--keywords="multi-gpu,scale,concurrency,production" \
		--max-issues=50 \
		--llm-endpoint="http://localhost:8080" \
		--output="issue-analysis-report.md"

# Run with port-forward to LLMKube
run-k8s: build
	@echo "Make sure you have port-forwarded: kubectl port-forward svc/qwen-14b-issueparser-service 8080:8080"
	./bin/issueparser \
		--repos="ollama/ollama,vllm-project/vllm" \
		--keywords="multi-gpu,scale,concurrency,production,performance,memory,VRAM" \
		--max-issues=100 \
		--llm-endpoint="http://localhost:8080" \
		--output="issue-analysis-report.md" \
		--verbose

test:
	go test ./...

# Docker - build for AMD64 (Linux servers)
docker-build:
	docker buildx build --platform linux/amd64 -t $(IMAGE_NAME):$(IMAGE_TAG) --load .

# Build and push to MicroK8s registry
docker-push: docker-build
	docker tag $(IMAGE_NAME):$(IMAGE_TAG) $(REGISTRY)/$(IMAGE_NAME):$(IMAGE_TAG)
	docker push $(REGISTRY)/$(IMAGE_NAME):$(IMAGE_TAG)

# Build multi-arch if needed
docker-build-multiarch:
	docker buildx build --platform linux/amd64,linux/arm64 -t $(REGISTRY)/$(IMAGE_NAME):$(IMAGE_TAG) --push .

# Kubernetes deployment
deploy-llmkube:
	kubectl apply -f deploy/llmkube-qwen-14b.yaml
	@echo "Waiting for model to be ready..."
	kubectl wait --for=condition=Ready model/qwen-14b-issueparser --timeout=600s || true
	@echo "Waiting for inference service..."
	kubectl wait --for=condition=Available deployment -l app=issueparser --timeout=300s || true

deploy-job:
	kubectl apply -f deploy/job.yaml

deploy: deploy-llmkube deploy-job

# Get results from job
get-results:
	@POD=$$(kubectl get pods -l job-name=issueparser-analysis -o jsonpath='{.items[0].metadata.name}' 2>/dev/null) && \
	if [ -n "$$POD" ]; then \
		kubectl cp default/$$POD:/output/issue-analysis-report.md ./issue-analysis-report.md && \
		echo "Report saved to issue-analysis-report.md"; \
	else \
		echo "No job pod found"; \
	fi

# Watch job logs
logs:
	kubectl logs -f job/issueparser-analysis

# Cleanup
clean:
	rm -rf bin/
	rm -f issue-analysis-report.md

clean-k8s:
	kubectl delete -f deploy/job.yaml --ignore-not-found
	kubectl delete -f deploy/llmkube-qwen-14b.yaml --ignore-not-found
	kubectl delete pvc issueparser-output --ignore-not-found

# Port forward for local testing
port-forward:
	kubectl port-forward svc/qwen-14b-issueparser-service 8080:8080

# Status check
status:
	@echo "=== LLMKube Model ==="
	kubectl get model qwen-14b-issueparser -o wide 2>/dev/null || echo "Not deployed"
	@echo ""
	@echo "=== LLMKube InferenceService ==="
	kubectl get inferenceservice qwen-14b-issueparser-service -o wide 2>/dev/null || echo "Not deployed"
	@echo ""
	@echo "=== IssueParser Job ==="
	kubectl get job issueparser-analysis 2>/dev/null || echo "Not running"
	@echo ""
	@echo "=== Pods ==="
	kubectl get pods -l app=issueparser
