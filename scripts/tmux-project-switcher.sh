#!/usr/bin/env bash
set -euo pipefail

# All commands here assume you are already inside tmux

# Mode picks which homogeneous list to show:
#   sessions - running tmux sessions only (resume)
#   folders  - projects without a session only (start a new one)
# Keeping each list homogeneous is what lets fzf rank by fuzzy score again; there
# are no project/session sections left to preserve.
MODE=${1:-sessions}
case "$MODE" in
    sessions|folders) ;;
    *)
        echo "tmux-project-switcher: unknown mode '$MODE' (want sessions or folders)" >&2
        exit 2
        ;;
esac

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
export TMUX_PROJECT_SWITCHER_FZF_COMMAND=${TMUX_PROJECT_SWITCHER_FZF_COMMAND:-"fzf-tmux -w50% -h60% --preview ''"}

ENTRIES=$("$BINARY" \
    --mode "$MODE" \
    --root "$TMUX_PROJECT_SWITCHER_ROOT_FOLDER" \
    --project-depth "$TMUX_PROJECT_SWITCHER_PROJECT_DEPTH" \
    --name-depth "$TMUX_PROJECT_SWITCHER_FOLDERS_AMOUNT")

# Column 2 is the display text shown in fzf; column 4 is the tmux session name to
# act on (they differ for worktrees, whose display is the path but whose session
# name is workmux's "<prefix><handle>").
SELECTED=$(printf '%s\n' "$ENTRIES" | cut -f2 | eval "$TMUX_PROJECT_SWITCHER_FZF_COMMAND" || true)

if [ -z "$SELECTED" ]; then
    exit 0
fi

ENTRY=$(printf '%s\n' "$ENTRIES" | awk -F'\t' -v target="$SELECTED" '($2 == target) { print $3 "\t" $4; exit }')

if [ -z "$ENTRY" ]; then
    echo "tmux-project-switcher: selected entry '$SELECTED' was not found" >&2
    exit 1
fi

IFS=$'\t' read -r PATH_FOR_SESSION TMUX_TARGET <<<"$ENTRY"

if [ -n "$PATH_FOR_SESSION" ]; then
    if ! tmux has-session -t "$TMUX_TARGET" 2>/dev/null; then
        tmux new-session -d -s "$TMUX_TARGET" -c "$PATH_FOR_SESSION"
    fi
else
    if ! tmux has-session -t "$TMUX_TARGET" 2>/dev/null; then
        echo "tmux-project-switcher: session '$TMUX_TARGET' has no project path and does not exist" >&2
        exit 1
    fi
fi

tmux switch-client -t "$TMUX_TARGET"
