#!/usr/bin/env sh

ru() {
  CURRENT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
  ruby "$CURRENT_DIR/tmux-project-switcher.rb" "$@"
}

# Use GH_BASE_DIR in case is set, or $HOME/src
GH_BASE_DIR=${GH_BASE_DIR:-$HOME/src}
# Use PROJECTS_ROOT if it's set or GH_BASE_DIR
PROJECTS_ROOT=${PROJECTS_ROOT:-$GH_BASE_DIR}

# sessions only with name
SESSIONS=$(tmux list-sessions -F '#S')
PROJECTS=$(fd -t d -a --exact-depth 3 --base-directory "$HOME/src")

GOOD=$(ru list "$SESSIONS" "$PROJECTS") 
# DEPTH=3
# --preview 'bat --color=always --style=header,grid {}/README.md'
RES=$(echo "$GOOD" | fzf-tmux -w80% -h100% --preview '')

if [ -n "$RES" ]; then
  THEPATH=$(ru path "$PROJECTS" "$RES")
  tmux new-session -d -s "$RES" -c "$THEPATH" 2>/dev/null
  tmux switch-client -t "$RES"
fi
