# First, declare ARGs before any FROM statement
ARG GO_VERSION=1.23
ARG DEBIAN_VERSION=bookworm-slim

# Then use them in FROM statements
FROM golang:${GO_VERSION} AS builder
# Maintainer information
ARG VCS_REF
LABEL maintainer="Lucas Albers <https://github.com/lalbers-lz4>"
LABEL org.opencontainers.image.source="https://hub.docker.com/r/lalberslz4/powertop-monitoring"

USER root

WORKDIR /app
COPY go.mod .
COPY go.sum .
RUN go mod download
COPY . .

# Set up cross-compilation arguments
ARG TARGETARCH
ARG TARGETOS

RUN apt-get update -y && \
    apt-get install -y curl powertop

# Build for the target architecture with verbose output
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -v -o main ./cmd/

FROM debian:${DEBIAN_VERSION}

RUN apt-get update -y && \
    apt-get install -y powertop && \
    rm -rf /var/lib/apt/lists/*

COPY --from=builder /app/main /app/main

EXPOSE 8887
ENTRYPOINT ["/app/main"]
