#!/usr/bin/env sh

# All commands here assume you are already inside tmux

CURRENT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
SCRIPT="$CURRENT_DIR/tmux-project-switcher.rb"

GH_BASE_DIR=${GH_BASE_DIR:-$HOME/src}
export TMUX_PROJECT_SWITCHER_ROOT_FOLDER=${PTMUX_PROJECT_SWITCHER_ROOT_FOLDER:-$GH_BASE_DIR}
export TMUX_PROJECT_SWITCHER_PROJECT_DEPTH=${TMUX_PROJECT_SWITCHER_PROJECT_DEPTH:-3}
export TMUX_PROJECT_SWITCHER_FOLDERS_AMOUNT=${TMUX_PROJECT_SWITCHER_FOLDERS_AMOUNT:-2}
export TMUX_PROJECT_SWITCHER_FZF_COMMAND=${TMUX_PROJECT_SWITCHER_FZF_COMMAND:-"fzf-tmux -w80% -h100% --preview ''"}

# Full path of the folders seating at depth 3 from the root
# PROJECTS=$(fd -t d -a --exact-depth "$TMUX_PROJECT_SWITCHER_PROJECT_DEPTH" --base-directory "$TMUX_PROJECT_SWITCHER_ROOT_FOLDER")

SESSIONS_AND_PROJECTS=$(ruby "$SCRIPT" list) 

# --preview 'bat --color=always --style=header,grid {}/README.md'
SESSION_NAME=$(echo "$SESSIONS_AND_PROJECTS" | eval $TMUX_PROJECT_SWITCHER_FZF_COMMAND)

if [ -n "$SESSION_NAME" ]; then
  # Given a session, i.e. dgmora/tmux-project-switcher find what is the path for it
  PATH_FOR_SESSION=$(ruby "$SCRIPT" path "$SESSION_NAME")
  # Create a new session that starts in the folder $PATH_FOR_SESSION if it does not exist yet

  tmux new-session -d -s "$SESSION_NAME" -c "$PATH_FOR_SESSION"
  # Switch to that session
  tmux switch-client -t "$SESSION_NAME"
fi
