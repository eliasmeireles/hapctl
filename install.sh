#!/bin/bash
set -e

REPO="eliasmeireles/hapctl"
INSTALL_DIR="/usr/local/bin"
BINARY_NAME="hapctl"

get_latest_release() {
    curl --silent "https://api.github.com/repos/$REPO/releases/latest" |
        grep '"tag_name":' |
        sed -E 's/.*"([^"]+)".*/\1/'
}

detect_platform() {
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)
    
    case $ARCH in
        x86_64)
            ARCH="amd64"
            ;;
        aarch64|arm64)
            ARCH="arm64"
            ;;
        *)
            echo "Unsupported architecture: $ARCH"
            exit 1
            ;;
    esac
    
    echo "${OS}-${ARCH}"
}

main() {
    echo "Installing hapctl..."
    
    VERSION=${1:-$(get_latest_release)}
    PLATFORM=$(detect_platform)
    BINARY="hapctl-${PLATFORM}"
    
    echo "Version: $VERSION"
    echo "Platform: $PLATFORM"
    
    DOWNLOAD_URL="https://github.com/$REPO/releases/download/$VERSION/$BINARY"
    
    echo "Downloading from: $DOWNLOAD_URL"
    
    TMP_DIR=$(mktemp -d)
    cd "$TMP_DIR"
    
    if ! curl -L -o "$BINARY" "$DOWNLOAD_URL"; then
        echo "Failed to download binary"
        exit 1
    fi
    
    chmod +x "$BINARY"
    
    echo "Installing to $INSTALL_DIR/$BINARY_NAME"
    sudo mv "$BINARY" "$INSTALL_DIR/$BINARY_NAME"
    
    cd -
    rm -rf "$TMP_DIR"
    
    echo ""
    echo "hapctl installed successfully!"
    echo "Version: $($BINARY_NAME version 2>/dev/null || echo $VERSION)"
    echo ""
    echo "Next steps:"
    echo "  1. Install HAProxy: sudo hapctl install"
    echo "  2. Configure: sudo vim /etc/hapctl/config.yaml"
    echo "  3. Start agent: sudo systemctl start hapctl-agent"
}

main "$@"
