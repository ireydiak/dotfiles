#!/bin/bash

ansi_art='                 ▄▄▄
 ▄█████   ▄███████   ▄██████  ▄█   █▄   ▄███████   ▄█████   ▄███████   ▄█   ▄█▄
 ███      ███   ███  ███      ███   ███  ███   ███  ███      ███   ███  ███ ▄███▀
 ███      ███   ███  ███      ███   ███  ███   ███  ███      ███   ███  ███▐██▀
 ███     ▄███▄▄▄██▀ ▄███▄▄▄  ▄███▄▄▄███▄ ███   ███  ███     ▄███▄▄▄███ ▄█████▀
 ███     ▀███▀▀▀▀   ▀███▀▀▀  ▀▀███▀▀▀███  ███   ███  ███    ▀▀███▀▀▀███ ▀▀█████▄
 ███     ██████████  ███      ███   ███  ███   ███  ███      ███   ███  ███▐██▄
 ███      ███   ███  ███      ███   ███  ███   ███  ███      ███   ███  ███ ▀███▄
 ▄█████   ███   ███  ██████   ███   █▀   ███████▀   ▄█████   ███   █▀   ███   ▀█▀
         ███   █▀                                                       ▀        '
clear
echo -e "\n$ansi_art\n"

rm -rf $HOME/.local/share/ireydiak

source ./setup.sh

echo "Cloning installer into $HOME/.local/share/ireydiak"
git clone -b feature/macos https://github.com/ireydiak/dotfiles $HOME/.local/share/ireydiak

echo "Starting installation..."
source $HOME/.local/share/ireydiak/macos/install.sh
