# If you come from bash you might have to change your $PATH.

# Path to your Oh My Zsh installation.
export ZSH="$HOME/.oh-my-zsh"

# Set name of the theme to load --- if set to "random", it will
# load a random theme each time Oh My Zsh is loaded, in which case,
# to know which specific one was loaded, run: echo $RANDOM_THEME
# See https://github.com/ohmyzsh/ohmyzsh/wiki/Themes
ZSH_THEME="robbyrussell"

# Set list of themes to pick from when loading at random
# Setting this variable when ZSH_THEME=random will cause zsh to load
# a theme from this variable instead of looking in $ZSH/themes/
# If set to an empty array, this variable will have no effect.
# ZSH_THEME_RANDOM_CANDIDATES=( "robbyrussell" "agnoster" )
export EDITOR=nvim
export GPG_TTY=$(tty)

# Uncomment the following line to use case-sensitive completion.
# CASE_SENSITIVE="true"

# Uncomment the following line to use hyphen-insensitive completion.
# Case-sensitive completion must be off. _ and - will be interchangeable.
# HYPHEN_INSENSITIVE="true"

# Uncomment one of the following lines to change the auto-update behavior
# zstyle ':omz:update' mode disabled  # disable automatic updates
# zstyle ':omz:update' mode auto      # update automatically without asking
# zstyle ':omz:update' mode reminder  # just remind me to update when it's time

# Uncomment the following line to change how often to auto-update (in days).
# zstyle ':omz:update' frequency 13

# Uncomment the following line if pasting URLs and other text is messed up.
# DISABLE_MAGIC_FUNCTIONS="true"

# Uncomment the following line to disable colors in ls.
# DISABLE_LS_COLORS="true"

# Uncomment the following line to disable auto-setting terminal title.
# DISABLE_AUTO_TITLE="true"

# Uncomment the following line to enable command auto-correction.
# ENABLE_CORRECTION="true"

# Uncomment the following line to display red dots whilst waiting for completion.
# You can also set it to another string to have that shown instead of the default red dots.
# e.g. COMPLETION_WAITING_DOTS="%F{yellow}waiting...%f"
# Caution: this setting can cause issues with multiline prompts in zsh < 5.7.1 (see #5765)
# COMPLETION_WAITING_DOTS="true"

# Uncomment the following line if you want to disable marking untracked files
# under VCS as dirty. This makes repository status check for large repositories
# much, much faster.
# DISABLE_UNTRACKED_FILES_DIRTY="true"

# Uncomment the following line if you want to change the command execution time
# stamp shown in the history command output.
# You can set one of the optional three formats:
# "mm/dd/yyyy"|"dd.mm.yyyy"|"yyyy-mm-dd"
# or set a custom format using the strftime function format specifications,
# see 'man strftime' for details.
# HIST_STAMPS="mm/dd/yyyy"

# Would you like to use another custom folder than $ZSH/custom?
# ZSH_CUSTOM=/path/to/new-custom-folder

# Which plugins would you like to load?
# Standard plugins can be found in $ZSH/plugins/
# Custom plugins may be added to $ZSH_CUSTOM/plugins/
# Example format: plugins=(rails git textmate ruby lighthouse)
# Add wisely, as too many plugins slow down shell startup.
plugins=(git tmux)
ZSH_TMUX_AUTOSTART=true

source $ZSH/oh-my-zsh.sh

# User configuration

# export MANPATH="/usr/local/man:$MANPATH"

# You may need to manually set your language environment
# export LANG=en_US.UTF-8

# Preferred editor for local and remote sessions
# if [[ -n $SSH_CONNECTION ]]; then
#   export EDITOR='vim'
# else
#   export EDITOR='nvim'
# fi

# Compilation flags
# export ARCHFLAGS="-arch $(uname -m)"
# zoxide (smart cd replacement)
eval "$(zoxide init zsh)"
# Set personal aliases, overriding those provided by Oh My Zsh libs,
# plugins, and themes. Aliases can be placed here, though Oh My Zsh
# users are encouraged to define aliases within a top-level file in
# the $ZSH_CUSTOM folder, with .zsh extension. Examples:
# - $ZSH_CUSTOM/aliases.zsh
# - $ZSH_CUSTOM/macos.zsh
# For a full list of active aliases, run `alias`.
#
# Example aliases
# alias zshconfig="mate ~/.zshrc"
# alias ohmyzsh="mate ~/.oh-my-zsh"

alias venv="make venv && source venv/bin/activate"
alias py='python3'

# pnpm
export PNPM_HOME="$HOME/Library/pnpm"
case ":$PATH:" in
  *":$PNPM_HOME:"*) ;;
  *) export PATH="$PNPM_HOME:$PATH" ;;
esac

# PATH
export PATH="/opt/homebrew/opt/openjdk@11/bin:$HOME/go/bin:$PATH"
export PATH=$HOME/bin:$HOME/.local/bin:/usr/local/bin:$HOME/go/bin:$PATH
export PATH=$PATH:~/.cache/rebar3/bin

# Function to activate Python venv if present.
# Must stay AFTER every static PATH export above: activating a venv prepends
# its bin/ to PATH, and any later prepend would shadow the venv's tools.
# auto_activate_venv() {
#     local dir="$PWD" candidate found=""
#
#     while [[ -n "$dir" ]]; do
#         for candidate in \
#             "$dir/dist/uv/venvs/python-default/bin/activate" \
#             "$dir/venv/bin/activate" \
#             "$dir/.venv/bin/activate"; do
#             if [[ -f "$candidate" ]]; then
#                 found="$candidate"
#                 break 2
#             fi
#         done
#         [[ "$dir" == "/" ]] && break
#         dir="${dir:h}"
#     done
#
#     # Skip only if the venv is active AND still wins on PATH; an inherited
#     # VIRTUAL_ENV can outlive a PATH that no longer points at it.
#     if [[ -n "$found" ]]; then
#         local venv_bin="${found:h}"
#         if [[ "$VIRTUAL_ENV" == "${venv_bin:h}" && "${PATH%%:*}" == "$venv_bin" ]]; then
#             return
#         fi
#     fi
#
#     if [[ -n "$VIRTUAL_ENV" ]]; then
#         deactivate 2>/dev/null || unset VIRTUAL_ENV
#     fi
#
#     [[ -n "$found" ]] && source "$found"
# }

# Hook into directory changes
autoload -U add-zsh-hook
# add-zsh-hook chpwd auto_activate_venv

# Run once on shell startup
# auto_activate_venv

# The following lines have been added by Docker Desktop to enable Docker CLI completions.
fpath=($HOME/.docker/completions $fpath)
autoload -Uz compinit
compinit
# End of Docker CLI completions

# Misc
export CLAUDE_CODE_DISABLE_MOUSE_CLICKS=1

alias python312="/opt/homebrew/bin/python3.12"

# Utilities
alias cleanmypy="find . -type d -name \".mypy_cache\" -exec rm -rf {} +"

# Machine-local settings and secrets. Not tracked.
[[ -f ~/.zshrc.local ]] && source ~/.zshrc.local
