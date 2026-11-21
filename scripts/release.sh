#!/usr/bin/env bash
# Скрипт релиза: выбирает версию (тег или интерактивный ввод), собирает
# таргеты, упаковывает tar.gz, считает SHA256SUMS и печатает подсказку
# публикации. Запускается из корня репозитория (через `just release`).
set -euo pipefail

RELEASES="${RELEASES:-linux-amd64}"

DESCRIBE="$(git describe --tags --always --dirty)"
if git describe --tags --exact-match >/dev/null 2>&1; then
    VERSION="$(git describe --tags --exact-match)"
    echo "Текущий коммит помечен тегом: $VERSION"
else
    echo "Текущий коммит НЕ помечен тегом. Версия по умолчанию: $DESCRIBE"
    printf 'Введите версию (например v1.2.3) или Enter для использования "%s": ' "$DESCRIBE"
    read -r INPUT || INPUT=""
    if [ -z "$INPUT" ]; then
        VERSION="$DESCRIBE"
    else
        VERSION="$INPUT"
        case "$VERSION" in
            v*) ;;
            *) VERSION="v$VERSION" ;;
        esac
    fi
    echo "Используется версия: $VERSION"
fi

rm -rf dist/release && mkdir -p dist/release

# Заметки GitHub-релиза берём из встроенного changelog: тот же текст увидит
# пользователь в окне «Что нового» после обновления.
NOTES_FILE="dist/release/RELEASE_NOTES.md"
CHANGELOG_FILE="changelog/CHANGELOG.md"
write_release_notes() {
    local ver="${VERSION#v}"
    : > "$NOTES_FILE"
    if [ -f "$CHANGELOG_FILE" ]; then
        awk -v ver="$ver" '
            $0 ~ "^##[[:space:]]+v?" ver "([[:space:]]|$)" { found=1; next }
            found && /^##[[:space:]]/ { exit }
            found { print }
        ' "$CHANGELOG_FILE" | sed -e '/./,$!d' > "$NOTES_FILE"
    fi
    if [ ! -s "$NOTES_FILE" ] || ! grep -q '[^[:space:]]' "$NOTES_FILE"; then
        echo "ВНИМАНИЕ: в $CHANGELOG_FILE нет секции $VERSION — заметки релиза будут краткими."
        printf 'Tasky %s\n' "$VERSION" > "$NOTES_FILE"
    fi
}
write_release_notes

for t in $RELEASES; do
    os="${t%-*}"; arch="${t#*-}"
    just build-target "$os" "$arch" "$VERSION"
    rm -rf dist/release/stage && mkdir -p dist/release/stage
    if [ "$os" = "windows" ]; then
        bin="tasky.exe"
        just build-indicator "$os" "$arch" "$VERSION"
        cp "dist/release/tasky-indicator-$os-$arch" dist/release/stage/tasky-indicator.exe
    else
        bin="tasky"
    fi
    cp "dist/release/tasky-$os-$arch" "dist/release/stage/$bin"
    files="$bin"
    [ "$os" = "windows" ] && files="$files tasky-indicator.exe"
    tar czf "dist/release/tasky-$os-$arch.tar.gz" -C dist/release/stage $files
    # Временные бинарники (исходники архивов) больше не нужны.
    rm -f "dist/release/tasky-$os-$arch" "dist/release/tasky-indicator-$os-$arch"
done
rm -rf dist/release/stage
( cd dist/release && sha256sum *.tar.gz > SHA256SUMS )
echo ""
echo "Готово: dist/release/*.tar.gz + SHA256SUMS + RELEASE_NOTES.md"
echo "Публикация:"
echo "  gh release create $VERSION dist/release/*.tar.gz dist/release/SHA256SUMS \\"
echo "      --title 'Tasky $VERSION' --notes-file dist/release/RELEASE_NOTES.md"
