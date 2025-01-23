# First, declare ARGs before any FROM statement
ARG GO_VERSION=1.23
ARG DEBIAN_VERSION=bullseye

# Then use them in FROM statements
FROM golang:${GO_VERSION} AS builder
# Maintainer information
ARG VCS_REF
LABEL maintainer="Lucas Albers <https://github.com/lalbers-lz4>"
LABEL org.opencontainers.image.source="https://hub.docker.com/r/lalberslz4/powertop-monitoring"

USER root

WORKDIR /app

# Copy go mod files first for better caching
COPY go.mod go.sum ./

# Clean and setup modules
RUN go mod download
RUN go mod tidy

# Copy the rest of the source code
COPY . .

# Remove any existing vendor directory
RUN rm -rf vendor/

# Set up cross-compilation arguments
ARG TARGETARCH
ARG TARGETOS

# Build without using vendor directory  e
#we are stripping the binary to reduce the size

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -mod=mod -v -o main -ldflags="-s -w" ./cmd/

FROM debian:${DEBIAN_VERSION}-slim

# Set TARGETARCH and DEBIAN_VERSION for the final stage
ARG TARGETARCH
ARG DEBIAN_VERSION

# Install required packages with conditional libraspberrypi-bin for ARM64
RUN set -ex && \
    apt-get update && \
    apt-get install -y --no-install-recommends \
        powertop \
        curl \
        gnupg \
        ca-certificates && \
    if [ "$TARGETARCH" = "arm64" ] ; then \
        echo "deb http://archive.raspberrypi.org/debian/ ${DEBIAN_VERSION} main" > /etc/apt/sources.list.d/raspi.list && \
        curl -sSL https://archive.raspberrypi.org/debian/raspberrypi.gpg.key | gpg --dearmor > /etc/apt/trusted.gpg.d/raspberrypi.gpg && \
        apt-get update && \
        apt-get install -y --no-install-recommends libraspberrypi-bin && \
        apt-get remove -y curl gnupg && \
        apt-get autoremove -y ; \
    else \
        apt-get remove -y curl gnupg && \
        apt-get autoremove -y ; \
    fi && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

# Copy the binary from builder
COPY --from=builder /app/main /main

EXPOSE 8887
ENTRYPOINT ["/main"]
