#!/bin/bash

rm -rf ~/.config/nvim

git clone https://github.com/LazyVim/LazyVim ~/.config/nvim

rm -rf ~/.config/.git

cp -r ./shared/nvim/lua/* $HOME/.config/nvim/lua
