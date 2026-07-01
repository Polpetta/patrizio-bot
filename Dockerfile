# Build stage
FROM golang:1.26@sha256:f96cc555eb8db430159a3aa6797cd5bae561945b7b0fe7d0e284c63a3b291609 AS builder

WORKDIR /app

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o patrizio ./cmd/patrizio && \
    mkdir -p data/db data/media

# Download deltachat-rpc-server from GitHub releases
FROM alpine:3@sha256:28bd5fe8b56d1bd048e5babf5b10710ebe0bae67db86916198a6eec434943f8b AS rpc-server

ARG TARGETARCH
# renovate: datasource=go packageName=github.com/chatmail/rpc-client-go/v2 versioning=semver
ARG DELTACHAT_RPC_VERSION=v2.53.0

RUN ARCH=$(case ${TARGETARCH} in amd64) echo "x86_64" ;; arm64) echo "aarch64" ;; *) echo "x86_64" ;; esac) && \
    wget -q -O /deltachat-rpc-server \
      "https://github.com/chatmail/core/releases/download/${DELTACHAT_RPC_VERSION}/deltachat-rpc-server-${ARCH}-linux" && \
    chmod +x /deltachat-rpc-server

# Runtime stage
FROM gcr.io/distroless/static-debian12@sha256:9c346e4be81b5ca7ff31a0d89eaeade58b0f95cfd3baed1f36083ddb47ca3160

COPY --from=builder /app/patrizio /usr/local/bin/patrizio
COPY --from=builder --chown=nonroot:nonroot /app/data /data
COPY --from=rpc-server /deltachat-rpc-server /usr/local/bin/deltachat-rpc-server

USER nonroot:nonroot
VOLUME ["/data"]

ENTRYPOINT ["/usr/local/bin/patrizio"]
CMD ["serve"]
