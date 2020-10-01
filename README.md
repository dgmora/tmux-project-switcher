# Tmux Project Switcher

Switch between tmux sessions that represent a project/repo.

Given you have a root folder with all your repos like:

```
$HOME
│
└── src
    ├── github.com
    │     ├── dgmora
    │     │   ├── repo1
    │     │   ├── tmux
    │     │   └── tmux-project-switcher
    │     └── tmux
    │         ├── tmux
    │         └── repo3
    ├── github.myenterprise.com
    │     └── DavidMora
    │         └── an-enterprise-repo
    │
    └── gitlab.com
          └── dgmora
               └── i-have-repos-all-over-the-place
```

`tmux-project-switcher` will open a popup with all those leaf folders, where
you'll be able to fuzzy find a project you can switch to. When selected, it will
either switch to that session or create a new session, and change into the right directory.

## Requirements

- [fzf](https://github.com/junegunn/fzf)
- Ruby
- Tmux >= 3.2 
- (optional) Works well with [`gh`](https://github.com/jdxcode/gh) for quickly cloning projects
  
## Tmux 3.2 popup window

This plugin works with the popup window, which is only supported in some of the
newer tmux versions. If you use `brew` you can install it with `--HEAD`

To check if that's working fine run inside a tmux session `tmux popup`. An empty popup should appear.

## Installation with [TPM](https://github.com/tmux-plugins/tpm)

Add plugin to the list of TPM plugins in ~/.tmux.conf:

```
set -g @plugin 'dgmora/tmux-project-switcher'
```

`tmux-prefix` + `I` to install the plugin

## Setup

The default configuration assumes that you have a file structure as defined in the beginning:
- A root folder with all your projects in `$HOME/src`.
- Your projects are at depth `3` from that folder. This is important because of what you see
in the popup will be all folders at that depth from the root. 1st level would be `github.com`,
second `dgmora` and third `tmux-project-switcher`.
- The "meaningful name" of the project is the last `2` folders. i.e. `dgmora/tmux-project-switcher`.
this will be used for the tmux session name. 2 is used because of git forks. 1 can be used but
won't work super well if you have forks

These settings can be overwritten. To overwrite the root folder, these settings are taken into account:

`GH_BASE_DIR` if set (used by [`gh`](https://github.com/jdxcode/gh) too)
`TMUX_PROJECT_SWITCHER_ROOT_FOLDER` if set
`$HOME/src`

For the depth:

`TMUX_PROJECT_SWITCHER_PROJECT_DEPTH` if set.
`3`

For the number of folders used to name the session:
`TMUX_PROJECT_SWITCHER_FOLDERS_AMOUNT` if set.
`2`
