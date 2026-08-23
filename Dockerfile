# Build stage
# GO_VERSION is derived from go.mod by the caller (Makefile / CI workflow),
# so go.mod stays the single source of truth for the Go toolchain version.
# To suppress warnings a default version can by generated with
# `make -B Dockerfile`
ARG GO_VERSION=1.26.6@sha256:0d1d3a794be25f809dd2cb3160d8c73276c4056a9f8242a138e908ddeee7b6b6
FROM golang:${GO_VERSION} AS builder

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
ARG DELTACHAT_RPC_VERSION=v2.56.0

RUN ARCH=$(case ${TARGETARCH} in amd64) echo "x86_64" ;; arm64) echo "aarch64" ;; *) echo "x86_64" ;; esac) && \
    wget -q -O /deltachat-rpc-server \
      "https://github.com/chatmail/core/releases/download/${DELTACHAT_RPC_VERSION}/deltachat-rpc-server-${ARCH}-linux" && \
    chmod +x /deltachat-rpc-server

# Runtime stage
FROM gcr.io/distroless/static-debian12@sha256:9c346e4be81b5ca7ff31a0d89eaeade58b0f95cfd3baed1f36083ddb47ca3160

COPY --from=builder /app/patrizio /usr/local/bin/patrizio
COPY --from=builder --chown=nonroot:nonroot /app/data /data
COPY --from=rpc-server /deltachat-rpc-server /usr/local/bin/deltachat-rpc-server

USER 65532:65532 # nonroot user:group
VOLUME ["/data"]

ENTRYPOINT ["/usr/local/bin/patrizio"]
CMD ["serve"]
