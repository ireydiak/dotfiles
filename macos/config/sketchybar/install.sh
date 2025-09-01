#!/bin/bash

brew tap FelixKratz/formulae
brew install sketchybar

brew install --cask font-hack-nerd-font
brew install --cask sf-symbols

mkdir -p ~/.config/sketchybar
cp -r ./* ~/.config/sketchybar

brew services start sketchybar
