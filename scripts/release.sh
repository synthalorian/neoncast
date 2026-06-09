#!/usr/bin/env bash
set -euo pipefail

# Release build script for neoncast
# Usage: ./scripts/release.sh [version]

VERSION="${1:-$(git describe --tags --always --dirty 2>/dev/null || echo 'dev')}"
COMMIT="$(git rev-parse --short HEAD 2>/dev/null || echo 'unknown')"
BUILD_TIME="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
BINARY_NAME="neoncast"
BUILD_DIR="build"
CMD_PATH="./cmd/neoncast"

LDFLAGS="-X neoncast/internal/version.Version=${VERSION} -X neoncast/internal/version.Commit=${COMMIT} -X neoncast/internal/version.BuildTime=${BUILD_TIME} -s -w"

PLATFORMS=(
    "linux:amd64"
    "linux:arm64"
    "darwin:amd64"
    "darwin:arm64"
    "windows:amd64"
)

echo "Building neoncast ${VERSION} (${COMMIT})..."
echo ""

rm -rf "${BUILD_DIR}"
mkdir -p "${BUILD_DIR}"

for platform in "${PLATFORMS[@]}"; do
    GOOS="${platform%%:*}"
    GOARCH="${platform##*:}"
    
    output="${BUILD_DIR}/${BINARY_NAME}-${GOOS}-${GOARCH}"
    if [ "${GOOS}" = "windows" ]; then
        output="${output}.exe"
    fi
    
    echo "Building ${GOOS}/${GOARCH} -> ${output}"
    GOOS="${GOOS}" GOARCH="${GOARCH}" go build -ldflags "${LDFLAGS}" -o "${output}" "${CMD_PATH}"
done

echo ""
echo "All builds complete:"
ls -la "${BUILD_DIR}/"
echo ""
echo "Release artifacts ready in ${BUILD_DIR}/"
