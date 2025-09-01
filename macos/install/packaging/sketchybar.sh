#!/bin/bash

rm -rf $HOME/.config/sketchybar

brew tap FelixKratz/formulae && brew install sketchybar
if [[ ! -f "/Users/jcverdier/Library/Fonts/HackNerdFont-Bold.ttf" ]]; then
	brew install --cask font-hack-nerd-font
	brew install --cask sf-symbols
fi

mkdir -p ~/.config/sketchybar/plugins

cp $(brew --prefix)/share/sketchybar/examples/sketchybarrc ~/.config/sketchybar/sketchybarrc
cp -r $(brew --prefix)/share/sketchybar/examples/plugins/ ~/.config/sketchybar/plugins/

cp -rf $CONFIG_DIR/sketchybar/* ~/.config/sketchybar/
cp $CONFIG_DIR/sketchybar/sketchybarrc ~/.sketchybarrc

brew services start sketchybar
