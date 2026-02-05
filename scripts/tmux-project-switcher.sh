#!/usr/bin/env bash
set -euo pipefail

# All commands here assume you are already inside tmux

CURRENT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_DIR="$(dirname "$CURRENT_DIR")"
BIN_DIR="$REPO_DIR/bin"
BINARY="$BIN_DIR/tmux-project-switcher"
BUILD_SCRIPT="$CURRENT_DIR/build-project-switcher.sh"

if [ ! -x "$BINARY" ] || \
   [ "$CURRENT_DIR/project-switcher.go" -nt "$BINARY" ] || \
   [ "$REPO_DIR/go.mod" -nt "$BINARY" ] || \
   [ "$REPO_DIR/go.sum" -nt "$BINARY" ]; then
    "$BUILD_SCRIPT"
fi

if [ ! -x "$BINARY" ]; then
    tmux display-message "tmux-project-switcher: helper binary is missing (expected at $BINARY)"
    exit 1
fi

GH_BASE_DIR=${GH_BASE_DIR:-$HOME/src}
export TMUX_PROJECT_SWITCHER_ROOT_FOLDER=${TMUX_PROJECT_SWITCHER_ROOT_FOLDER:-$GH_BASE_DIR}
export TMUX_PROJECT_SWITCHER_PROJECT_DEPTH=${TMUX_PROJECT_SWITCHER_PROJECT_DEPTH:-3}
export TMUX_PROJECT_SWITCHER_FOLDERS_AMOUNT=${TMUX_PROJECT_SWITCHER_FOLDERS_AMOUNT:-2}
export TMUX_PROJECT_SWITCHER_FZF_COMMAND=${TMUX_PROJECT_SWITCHER_FZF_COMMAND:-"fzf-tmux -w80% -h100% --preview ''"}

ENTRIES=$("$BINARY" \
    --root "$TMUX_PROJECT_SWITCHER_ROOT_FOLDER" \
    --project-depth "$TMUX_PROJECT_SWITCHER_PROJECT_DEPTH" \
    --name-depth "$TMUX_PROJECT_SWITCHER_FOLDERS_AMOUNT")

SESSION_NAME=$(printf '%s\n' "$ENTRIES" | cut -f1 | eval "$TMUX_PROJECT_SWITCHER_FZF_COMMAND" || true)

if [ -z "$SESSION_NAME" ]; then
    exit 0
fi

PATH_FOR_SESSION=$(printf '%s\n' "$ENTRIES" | awk -F'\t' -v target="$SESSION_NAME" '($1 == target) { print $2; exit }')

if [ -n "$PATH_FOR_SESSION" ]; then
    if ! tmux has-session -t "$SESSION_NAME" 2>/dev/null; then
        tmux new-session -d -s "$SESSION_NAME" -c "$PATH_FOR_SESSION"
    fi
else
    if ! tmux has-session -t "$SESSION_NAME" 2>/dev/null; then
        echo "tmux-project-switcher: session '$SESSION_NAME' has no project path and does not exist" >&2
        exit 1
    fi
fi

tmux switch-client -t "$SESSION_NAME"
