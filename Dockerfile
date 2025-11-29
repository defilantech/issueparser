FROM golang:1.23-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum* ./
RUN go mod download || true

# Copy source
COPY . .

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -o /issueparser ./cmd/issueparser

# Runtime image
FROM alpine:3.19

RUN apk add --no-cache ca-certificates

COPY --from=builder /issueparser /usr/local/bin/issueparser

ENTRYPOINT ["/usr/local/bin/issueparser"]
