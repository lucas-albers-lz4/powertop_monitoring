#!/bin/bash
set -e  # Exit on any error

# Set variables
IMAGE_NAME=lalberslz4/powertop-monitoring
PLATFORMS=linux/amd64,linux/arm64
BUILDKIT_COMPRESSION=zstd:chunked
BUILDKIT_COMPRESSION_LEVEL=9

# Define version matrices
GO_VERSIONS=("1.23")
#not building bookworm until I decide to determine why it does not work
DEBIAN_VERSIONS=("bullseye")

# Enable BuildKit
export DOCKER_BUILDKIT=1

# Build for each combination
for GO_VERSION in "${GO_VERSIONS[@]}"; do
    for DEBIAN_VERSION in "${DEBIAN_VERSIONS[@]}"; do
        echo "Building for Go ${GO_VERSION} on ${DEBIAN_VERSION}"
        
        # Create tag based on versions
        TAG="${GO_VERSION}-${DEBIAN_VERSION}"
        
        # Build the image
        docker buildx build \
            --platform $PLATFORMS \
            --build-arg GO_VERSION=${GO_VERSION} \
            --build-arg DEBIAN_VERSION=${DEBIAN_VERSION} \
            --tag $IMAGE_NAME:${TAG} \
            --provenance=mode=max \
            --progress=plain \
            --push .
        
        # If this is the latest Go version and Debian version, tag as latest
        if [[ "$GO_VERSION" == "1.23" && "$DEBIAN_VERSION" == "bookworm" ]]; then
            docker buildx build \
                --platform $PLATFORMS \
                --build-arg GO_VERSION=${GO_VERSION} \
                --build-arg DEBIAN_VERSION=${DEBIAN_VERSION} \
                --tag $IMAGE_NAME:latest \
                --provenance=mode=max \
                --progress=plain \
                --push .
        fi
        # If this is the latest Go version and Debian version, tag as latest
        if [[ "$GO_VERSION" == "1.23" && "$DEBIAN_VERSION" == "bullseye" ]]; then
            docker buildx build \
                --platform $PLATFORMS \
                --build-arg GO_VERSION=${GO_VERSION} \
                --build-arg DEBIAN_VERSION=${DEBIAN_VERSION} \
                --tag $IMAGE_NAME:stable \
                --provenance=mode=max \
                --progress=plain \
                --push .
        fi
done
done
