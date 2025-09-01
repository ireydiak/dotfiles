#!/bin/bash

if command -v git &>/dev/null; then
    echo "Git is installed"
    git --version
else
    echo "Installing Git..."
    curl -O https://sourceforge.net/projects/git-osx-installer/files/git-2.33.0-intel-universal-mavericks.dmg/download
    hdiutil mount git-2.33.0-intel-universal-mavericks.dmg
    sudo installer -pkg /Volumes/Git*/*.pkg -target /
    hdiutil unmount /Volumes/Git*
fi
