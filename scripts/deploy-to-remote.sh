#!/bin/bash
set -e

# Deploy IssueParser to a remote MicroK8s cluster
# Usage: SSH_HOST=my-server ./scripts/deploy-to-remote.sh
#
# Environment Variables:
#   SSH_HOST - SSH hostname for the remote server (required)

if [ -z "${SSH_HOST}" ]; then
    echo "Error: SSH_HOST environment variable is required"
    echo "Usage: SSH_HOST=my-server ./scripts/deploy-to-remote.sh"
    exit 1
fi

IMAGE_NAME="issueparser"
IMAGE_TAG="latest"
REMOTE_IMAGE="localhost:32000/${IMAGE_NAME}:${IMAGE_TAG}"

echo "=== IssueParser Deployment to ${SSH_HOST} ==="
echo ""

# Step 1: Build AMD64 image
echo "[1/5] Building AMD64 container image..."
docker buildx build --platform linux/amd64 -t ${IMAGE_NAME}:${IMAGE_TAG} --load .

# Step 2: Save image to tarball
echo "[2/5] Saving image to tarball..."
docker save ${IMAGE_NAME}:${IMAGE_TAG} | gzip > /tmp/issueparser.tar.gz
echo "    Image size: $(du -h /tmp/issueparser.tar.gz | cut -f1)"

# Step 3: Copy to remote and load into MicroK8s
echo "[3/5] Transferring to ${SSH_HOST} and loading into MicroK8s..."
scp /tmp/issueparser.tar.gz ${SSH_HOST}:/tmp/
ssh ${SSH_HOST} "gunzip -c /tmp/issueparser.tar.gz | microk8s ctr image import -"

# Tag for local registry
echo "    Tagging for local registry..."
ssh ${SSH_HOST} "microk8s ctr image tag docker.io/library/${IMAGE_NAME}:${IMAGE_TAG} ${REMOTE_IMAGE}"

# Step 4: Deploy manifests
echo "[4/5] Deploying to Kubernetes..."
scp deploy/llmkube-qwen-14b.yaml ${SSH_HOST}:/tmp/
scp deploy/job.yaml ${SSH_HOST}:/tmp/

ssh ${SSH_HOST} "microk8s kubectl apply -f /tmp/llmkube-qwen-14b.yaml"
echo "    LLMKube Model and InferenceService applied"

# Step 5: Check status
echo "[5/5] Checking deployment status..."
echo ""
echo "=== Model Status ==="
ssh ${SSH_HOST} "microk8s kubectl get model qwen-14b-issueparser" || true
echo ""
echo "=== InferenceService Status ==="
ssh ${SSH_HOST} "microk8s kubectl get inferenceservice qwen-14b-issueparser-service" || true

echo ""
echo "=== Deployment Complete ==="
echo ""
echo "Next steps:"
echo "  1. Wait for model to download:"
echo "     ssh ${SSH_HOST} 'microk8s kubectl get model -w'"
echo ""
echo "  2. Create GitHub token secret:"
echo "     ssh ${SSH_HOST} \"microk8s kubectl create secret generic github-token --from-literal=token=ghp_xxx\""
echo ""
echo "  3. Run the analysis job:"
echo "     ssh ${SSH_HOST} 'microk8s kubectl apply -f /tmp/job.yaml'"
echo ""
echo "  4. Watch logs:"
echo "     ssh ${SSH_HOST} 'microk8s kubectl logs -f job/issueparser-analysis'"

# Cleanup local tarball
rm -f /tmp/issueparser.tar.gz
