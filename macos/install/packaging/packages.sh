#!/bin/bash

brew install awscli \
    docker \
    fzf \
    go \
    jq \
    ripgrep \
    tmux \
    zoxide

if [[ ! -d "/Applications/Google Chrome.app" ]]; then
	brew install --cask google-chrome
fi

if [[ ! -d "/Applications/JetBrains Toolbox.app" ]]; then
	brew install --cask jetbrains-toolbox
fi
