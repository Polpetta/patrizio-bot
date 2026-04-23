# Build stage
FROM golang:1.26@sha256:1e598ea5752ae26c093b746fd73c5095af97d6f2d679c43e83e0eac484a33dc3 AS builder

WORKDIR /app

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o patrizio ./cmd/patrizio && \
    mkdir -p data/db data/media

# Download deltachat-rpc-server from GitHub releases
FROM alpine:3@sha256:5b10f432ef3da1b8d4c7eb6c487f2f5a8f096bc91145e68878dd4a5019afde11 AS rpc-server

ARG TARGETARCH
ARG DELTACHAT_RPC_VERSION=v2.49.0

RUN ARCH=$(case ${TARGETARCH} in amd64) echo "x86_64" ;; arm64) echo "aarch64" ;; *) echo "x86_64" ;; esac) && \
    wget -q -O /deltachat-rpc-server \
      "https://github.com/chatmail/core/releases/download/${DELTACHAT_RPC_VERSION}/deltachat-rpc-server-${ARCH}-linux" && \
    chmod +x /deltachat-rpc-server

# Runtime stage
FROM gcr.io/distroless/static-debian12@sha256:20bc6c0bc4d625a22a8fde3e55f6515709b32055ef8fb9cfbddaa06d1760f838

COPY --from=builder /app/patrizio /usr/local/bin/patrizio
COPY --from=builder --chown=nonroot:nonroot /app/data /data
COPY --from=rpc-server /deltachat-rpc-server /usr/local/bin/deltachat-rpc-server

USER nonroot:nonroot
VOLUME ["/data"]

ENTRYPOINT ["/usr/local/bin/patrizio"]
CMD ["serve"]
