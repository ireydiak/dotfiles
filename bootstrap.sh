#!/bin/bash
# Fresh-Mac entry point for ireydiak/dotfiles.
# Prerequisite: xcode-select --install (and wait for it to finish).
set -euo pipefail

REPO_URL="https://github.com/ireydiak/dotfiles"
REPO_DIR="$HOME/.dotfiles"

if ! xcode-select -p >/dev/null 2>&1; then
  echo "Xcode Command Line Tools are missing. Run: xcode-select --install, then rerun this script." >&2
  exit 1
fi

if [ ! -x /opt/homebrew/bin/brew ]; then
  /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)" </dev/tty
fi
eval "$(/opt/homebrew/bin/brew shellenv)"

command -v go >/dev/null 2>&1 || brew install go

if [ ! -d "$REPO_DIR/.git" ]; then
  git clone "$REPO_URL" "$REPO_DIR"
fi

mkdir -p "$HOME/.local/bin"
(cd "$REPO_DIR" && go build -o "$HOME/.local/bin/dot" ./cmd/dot)

"$HOME/.local/bin/dot" install

echo
echo "Done. Open a new terminal to pick up the shell configuration."
