#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_DIR="$(dirname "$SCRIPT_DIR")"
BIN_DIR="$REPO_DIR/bin"
BINARY="$BIN_DIR/tmux-project-switcher"

if ! command -v go >/dev/null 2>&1; then
    echo "tmux-project-switcher: Go is required" >&2
    exit 1
fi

mkdir -p "$BIN_DIR"

should_build=0
if [ ! -x "$BINARY" ]; then
    should_build=1
else
    check_files=(go.mod go.sum)
    while IFS= read -r file; do
        check_files+=("$file")
    done < <(cd "$REPO_DIR" && find cmd/project-switcher -type f -name '*.go')

    for relative in "${check_files[@]}"; do
        local_path="$REPO_DIR/$relative"
        if [ -f "$local_path" ] && [ "$local_path" -nt "$BINARY" ]; then
            should_build=1
            break
        fi
    done
fi

if [ "$should_build" -eq 0 ]; then
    exit 0
fi

(cd "$REPO_DIR" && GO111MODULE=on GOFLAGS="" go build -o "$BINARY" ./scripts/project-switcher.go)
