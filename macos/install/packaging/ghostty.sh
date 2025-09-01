#!/bin/bash

if [[ ! -d "/Applications/Ghostty.app" ]]; then
	brew install --cask ghostty
fi

echo "theme=nord-wave" >>$HOME/Library/Application\ Support/com.mitchellh.ghostty/config
