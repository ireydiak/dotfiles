#!/bin/bash

if [[ ! -d "/Applications/Ghostty.app" ]]; then
	brew install --cask ghostty
fi

if [[ -d "$HOME/Library/Application\ Support/com.mitchellh.ghostty" ]]; then
    echo "theme=nord-wave" >>$HOME/Library/Application\ Support/com.mitchellh.ghostty/config
fi
