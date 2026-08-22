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

rm -rf dist/release && mkdir -p dist/release dist/release/stage
for t in $RELEASES; do
    os="${t%-*}"; arch="${t#*-}"
    just build-target "$os" "$arch" "$VERSION"
    cp "dist/release/tasky-$os-$arch" dist/release/stage/tasky
    tar czf "dist/release/tasky-$os-$arch.tar.gz" -C dist/release/stage tasky
done
rm -rf dist/release/stage
( cd dist/release && sha256sum *.tar.gz > SHA256SUMS )
echo ""
echo "Готово: dist/release/*.tar.gz + SHA256SUMS"
echo "Публикация:"
echo "  gh release create $VERSION dist/release/*.tar.gz dist/release/SHA256SUMS \\"
echo "      --title 'Tasky $VERSION' --notes '...'"
