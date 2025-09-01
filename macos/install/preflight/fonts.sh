#!/bin/bash

if ! ls ~/Library/Fonts/JetBrainsMono*.ttf >/dev/null 2>&1; then
    curl -L -o $HOME/Downloads/JetBrainsMono.zip https://github.com/ryanoasis/nerd-fonts/releases/download/v3.4.0/JetBrainsMono.zip
    mkdir -p ~/Library/Fonts
    unzip -o JetBrainsMono.zip -d ~/Library/Fonts
    rm $HOME/Downloads/JetBrainsMono.zip
fi
