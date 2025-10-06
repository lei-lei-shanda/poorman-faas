# syntax=docker/dockerfile:1

# ---------- Build stage ----------
FROM golang:1.24-alpine AS builder

ARG BINARY=faas
ARG CGO_ENABLED=0
ARG TARGETOS=linux
ARG TARGETARCH=amd64

WORKDIR /app

# Pre-copy go.mod and go.sum to leverage Docker layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source
COPY . .

# Optional: run code generation if needed
# Uncomment if you want CI-like generation inside the image
# RUN go generate ./...

# Build the selected binary from cmd/<BINARY>
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    GOOS=${TARGETOS} GOARCH=${TARGETARCH} CGO_ENABLED=${CGO_ENABLED} \
    go build -trimpath -ldflags "-s -w" -o /bin/${BINARY} ./cmd/${BINARY}

# ---------- Runtime stage ----------
FROM alpine:3.20 AS runner

# Create non-root user
RUN addgroup -S app && adduser -S app -G app

# Minimal certs for HTTPS calls
RUN apk add --no-cache ca-certificates tzdata

# App directories
WORKDIR /app

# Copy binary from builder
ARG BINARY=faas
COPY --from=builder /bin/${BINARY} /app/${BINARY}

# Drop privileges
USER app

# Default command can be overridden at runtime
ENTRYPOINT ["/app/faas"]
# To run other binaries, override at runtime with: docker run ... /app/<binary>