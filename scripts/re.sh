#!/usr/bin/env sh

ru() {
  ruby $HOME/repos.rb "$@"
}

SESSIONS=$(tmux list-sessions -F '#S')
REPOS=$(fd -t d -a --exact-depth 3 --base-directory "$HOME/src")
REPO_NAME=$(ru names "$REPOS")
SESSIONS_WITHOUT_REPO=$(ru diff "$SESSIONS" "$REPO_NAME")

#GOOD="$REPO_NAME\n$SESSIONS_WITHOUT_REPO"
GOOD=$(ru list "$SESSIONS" "$REPOS") 
# DEPTH=3
# --preview 'bat --color=always --style=header,grid {}/README.md'
RES=$(echo "$GOOD" | fzf-tmux -w80% -h100% --preview '')

if [ -n "$RES" ]; then
  THEPATH=$(ru path "$REPOS" "$RES")
  tmux new-session -d -s "$RES" -c "$THEPATH" 2>/dev/null
  tmux switch-client -t $RES 
fi
