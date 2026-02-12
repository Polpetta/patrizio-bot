# Build stage
FROM golang:1.25 AS builder

WORKDIR /app

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o patrizio ./cmd/patrizio

# Download deltachat-rpc-server from GitHub releases
FROM alpine:3 AS rpc-server

ARG TARGETARCH
ARG DELTACHAT_RPC_VERSION=v2.42.0

RUN ARCH=$(case ${TARGETARCH} in amd64) echo "x86_64" ;; arm64) echo "aarch64" ;; *) echo "x86_64" ;; esac) && \
    wget -q -O /deltachat-rpc-server \
      "https://github.com/chatmail/core/releases/download/${DELTACHAT_RPC_VERSION}/deltachat-rpc-server-${ARCH}-linux" && \
    chmod +x /deltachat-rpc-server

# Runtime stage
FROM gcr.io/distroless/static-debian12

COPY --from=builder /app/patrizio /patrizio
COPY --from=rpc-server /deltachat-rpc-server /usr/local/bin/deltachat-rpc-server

USER nonroot:nonroot

ENTRYPOINT ["/patrizio"]
CMD ["serve"]
