#!/bin/bash

PLUGIN_DIR=$HOME/.config/sketchybar/plugins
CONFIG_DIR=$HOME/.config/sketchybar

source "$CONFIG_DIR/icons.sh"

wifi=(
    padding_right=7
    label.width=0
    icon="$WIFI_DISCONNECTED"
    script="$PLUGIN_DIR/wifi.sh"
)

sketchybar --add item wifi right \
    --set wifi "${wifi[@]}" \
    --subscribe wifi wifi_change mouse.clicked
