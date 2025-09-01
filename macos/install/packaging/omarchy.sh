#!/bin/bash

rm -rf ~/.local/share/omarchy

git clone https://github.com/basecamp/omarchy.git ~/.local/share/omarchy

if [[ ! -d "~/.bashrc" ]]; then
	touch ~/.bashrc
fi

echo 'source ~/.local/share/omarchy/default/bash/rc' | cat - ~/.bashrc > ~/.bashrc.tmp && mv ~/.bashrc.tmp ~/.bashrc
