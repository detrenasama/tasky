#!/usr/bin/env bash
# Удаляет индикатор Tasky для GNOME Shell.
set -euo pipefail

UUID="tasky-indicator@detrenasama"
DATA_HOME="${XDG_DATA_HOME:-$HOME/.local/share}"
EXT_DIR="$DATA_HOME/gnome-shell/extensions/$UUID"

if command -v gnome-extensions >/dev/null 2>&1; then
    gnome-extensions disable "$UUID" >/dev/null 2>&1 || true
fi
rm -rf "$EXT_DIR"
echo "Расширение удалено: $EXT_DIR"
