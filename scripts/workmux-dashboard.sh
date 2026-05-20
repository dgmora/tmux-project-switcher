#!/usr/bin/env bash
set -euo pipefail

# All commands here assume you are already inside tmux

if ! command -v workmux >/dev/null 2>&1; then
    tmux display-message "tmux-project-switcher: workmux is not available in PATH"
    exit 1
fi

tmux display-popup -w 95% -h 95% -E "workmux dashboard"
