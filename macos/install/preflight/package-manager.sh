#!/bin/bash

/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"

if [[ ! -d ~/.zprofile ]]; then
	echo >> ~/.zprofile
fi

if [[ ! -d ~/.bashrc ]]; then
	echo >> ~/.bashrc
fi

echo 'eval "$(/opt/homebrew/bin/brew shellenv)"' >> ~/.zprofile
echo 'eval "$(/opt/homebrew/bin/brew shellenv)"' >> ~/.bashrc
eval "$(/opt/homebrew/bin/brew shellenv)"
