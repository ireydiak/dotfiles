#!/bin/bash

brew tap FelixKratz/formulae
brew install sketchybar

brew install --cask font-hack-nerd-font
brew install --cask sf-symbols

mkdir -p ~/.config/sketchybar
cp -r ./* ~/.config/sketchybar

git clone https://github.com/kvndrsslr/sketchybar-app-font.git
cd sketchybar-app-font && pnpm install && pnpm build:install

brew services start sketchybar
