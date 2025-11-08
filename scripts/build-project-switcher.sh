#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_DIR="$(dirname "$SCRIPT_DIR")"
BIN_DIR="$REPO_DIR/bin"
BINARY="$BIN_DIR/tmux-project-switcher"

if ! command -v go >/dev/null 2>&1; then
    echo "tmux-project-switcher: Go is required to build the helper binary" >&2
    exit 1
fi

mkdir -p "$BIN_DIR"

(cd "$REPO_DIR" && GO111MODULE=on GOFLAGS="" go build -o "$BINARY" ./scripts/project-switcher.go)
