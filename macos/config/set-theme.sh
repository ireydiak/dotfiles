#!/bin/bash
set_wallpaper() {
    osascript -e "tell application \"Finder\" to set desktop picture to POSIX file \"$1\""
}

# Usage
set_wallpaper "$HOME/.local/share/ireydiak/themes/nord/backgrounds/1-nord.png"
