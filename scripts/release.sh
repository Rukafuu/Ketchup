#!/bin/bash
# Ketchup Release Script
# Builds multi-platform binaries and generates release manifest
#
# Usage: ./scripts/release.sh <version> [channel]
# Example: ./scripts/release.sh 0.8.1 stable
#
# Environment variables required for upload:
#   R2_ENDPOINT - Cloudflare R2 endpoint URL
#   R2_ACCESS_KEY_ID - R2 access key
#   R2_SECRET_ACCESS_KEY - R2 secret key
#   R2_BUCKET - R2 bucket name (e.g., "ketchup-releases")

set -euo pipefail

VERSION="${1:-}"
CHANNEL="${2:-stable}"

if [[ -z "$VERSION" ]]; then
    echo "Usage: $0 <version> [channel]"
    echo "Example: $0 0.8.1 stable"
    exit 1
fi

if [[ ! "$VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.]+)?$ ]]; then
    echo "Error: Invalid version format. Use SemVer (e.g., 0.8.1 or 0.9.0-beta.1)"
    exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
RELEASE_DIR="$PROJECT_ROOT/releases/$VERSION"

echo "=== Ketchup Release Script ==="
echo "Version: $VERSION"
echo "Channel: $CHANNEL"
echo ""

# Create release directory
mkdir -p "$RELEASE_DIR"

# Platforms to build
PLATFORMS=(
    "linux/amd64"
    "linux/arm64"
    "darwin/amd64"
    "darwin/arm64"
    "windows/amd64"
)

# Build all platforms
echo "Building binaries..."
for platform in "${PLATFORMS[@]}"; do
    OS="${platform%/*}"
    ARCH="${platform#*/}"
    
    OUTPUT_NAME="ketchup-$OS-$ARCH"
    if [[ "$OS" == "windows" ]]; then
        OUTPUT_NAME="$OUTPUT_NAME.exe"
    fi
    
    echo "  Building $OS/$ARCH -> $OUTPUT_NAME"
    
    CGO_ENABLED=0 GOOS="$OS" GOARCH="$ARCH" go build \
        -ldflags "-X main.Version=$VERSION" \
        -o "$RELEASE_DIR/$OUTPUT_NAME" \
        "$PROJECT_ROOT/cmd/ketchup"

    FF_NAME="ff-$OS-$ARCH"
    if [[ "$OS" == "windows" ]]; then
        FF_NAME="$FF_NAME.exe"
    fi
    cp "$RELEASE_DIR/$OUTPUT_NAME" "$RELEASE_DIR/$FF_NAME"
    echo "  Alias $OS/$ARCH -> $FF_NAME"
done

# Generate SHA-256 checksums
echo ""
echo "Generating checksums..."
declare -A CHECKSUMS
for file in "$RELEASE_DIR"/ketchup-* "$RELEASE_DIR"/ff-*; do
    filename="$(basename "$file")"
    checksum="$(sha256sum "$file" | awk '{print $1}')"
    CHECKSUMS["$filename"]="$checksum"
    echo "  $filename: $checksum"
done

# Generate manifest JSON
echo ""
echo "Generating manifest..."
MANIFEST_FILE="$RELEASE_DIR/manifest.json"

cat > "$MANIFEST_FILE" << EOF
{
  "version": "$VERSION",
  "channel": "$CHANNEL",
  "released_at": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "downloads": {
EOF

first=true
for platform in "${PLATFORMS[@]}"; do
    OS="${platform%/*}"
    ARCH="${platform#*/}"
    platform_key="$OS-$ARCH"
    
    OUTPUT_NAME="ketchup-$OS-$ARCH"
    if [[ "$OS" == "windows" ]]; then
        OUTPUT_NAME="$OUTPUT_NAME.exe"
    fi
    
    checksum="${CHECKSUMS[$OUTPUT_NAME]}"
    
    if [[ "$first" != true ]]; then
        echo "," >> "$MANIFEST_FILE"
    fi
    first=false
    
    cat >> "$MANIFEST_FILE" << EOF
    "$platform_key": {
      "url": "https://releases.ketchup.dev/$VERSION/$OUTPUT_NAME",
      "sha256": "$checksum"
    }
EOF
done

cat >> "$MANIFEST_FILE" << EOF

  }
}
EOF

echo "Manifest written to: $MANIFEST_FILE"

# Update latest.json for the channel
LATEST_MANIFEST="$PROJECT_ROOT/releases/latest-$CHANNEL.json"
cp "$MANIFEST_FILE" "$LATEST_MANIFEST"
echo "Latest manifest updated: $LATEST_MANIFEST"

# Show summary
echo ""
echo "=== Release Summary ==="
echo "Version: $VERSION"
echo "Channel: $CHANNEL"
echo "Output directory: $RELEASE_DIR"
echo ""
echo "Files created:"
ls -lh "$RELEASE_DIR"

echo ""
echo "=== Upload Instructions ==="
echo "To upload to Cloudflare R2, configure these environment variables:"
echo "  export R2_ENDPOINT=https://<account-id>.r2.cloudflarestorage.com"
echo "  export R2_ACCESS_KEY_ID=<your-access-key>"
echo "  export R2_SECRET_ACCESS_KEY=<your-secret-key>"
echo "  export R2_BUCKET=ketchup-releases"
echo ""
echo "Then run the upload script or use aws CLI:"
echo "  aws s3 cp --recursive --endpoint-url \$R2_ENDPOINT \$RELEASE_DIR s3://\$R2_BUCKET/$VERSION/"
echo "  aws s3 cp --endpoint-url \$R2_ENDPOINT \$LATEST_MANIFEST s3://\$R2_BUCKET/latest-$CHANNEL.json"
echo ""
echo "Release preparation complete!"
