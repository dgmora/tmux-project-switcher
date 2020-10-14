CURRENT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

# ctrl + option + P as default shortcut, without prefix
ASSIGNED_KEY=$(tmux show-option -gqv "@switcher-key")
ASSIGNED_KEY=${ASSIGNED_KEY:-'-n C-M-p'}
KEY_ARRAY=("$ASSIGNED_KEY")

tmux bind-key ${KEY_ARRAY[@]} run-shell "$CURRENT_DIR/scripts/tmux-project-switcher.sh"
