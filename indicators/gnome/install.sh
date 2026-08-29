#!/usr/bin/env bash
# Установщик индикатора Tasky для GNOME Shell.
# Копирует расширение в каталог расширений пользователя и включает его.
set -euo pipefail

UUID="tasky-indicator@detrenasama"
SRC_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
DATA_HOME="${XDG_DATA_HOME:-$HOME/.local/share}"
EXT_DIR="$DATA_HOME/gnome-shell/extensions/$UUID"

echo "Tasky GNOME indicator — установка"
echo

# 1. GNOME Shell
if ! command -v gnome-shell >/dev/null 2>&1; then
    echo "Ошибка: GNOME Shell не найден." >&2
    exit 1
fi
SHELL_VERSION="$(gnome-shell --version 2>/dev/null | awk '{print $NF}')"
echo "GNOME Shell: $SHELL_VERSION"

# 2. Сервер tasky (для HTTP /status)
if ! command -v tasky >/dev/null 2>&1 && [ ! -x "$HOME/.local/bin/tasky" ]; then
    echo "Предупреждение: tasky не найден — установите его и запустите «tasky serve»:"
    echo "                 https://github.com/detrenasama/tasky"
fi

# 3. Копирование расширения
mkdir -p "$EXT_DIR"
cp "$SRC_DIR/extension.js" "$SRC_DIR/metadata.json" "$EXT_DIR/"
echo "Расширение установлено: $EXT_DIR"

# 4. Включение (GNOME Shell может подхватить файлы не сразу — пробуем несколько раз)
if command -v gnome-extensions >/dev/null 2>&1; then
    ENABLED=""
    for _ in 1 2 3 4 5; do
        if gnome-extensions enable "$UUID" >/dev/null 2>&1; then
            ENABLED=1
            break
        fi
        sleep 1
    done
    if [ -n "$ENABLED" ]; then
        echo "Расширение включено."
    else
        echo "Расширение не включилось на лету. Перезапустите сессию GNOME"
        echo "(выход и вход) или нажмите Alt+F2 → r — затем:"
        echo "  gnome-extensions enable $UUID"
    fi
else
    echo "gnome-extensions не найден. Включите расширение вручную:"
    echo "  gnome-extensions enable $UUID"
fi

echo
echo "Готово. Индикатор появится в панели, когда работает «tasky serve»."
