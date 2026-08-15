#!/bin/sh
# Установка Tasky: скачивает последний релиз с GitHub, проверяет SHA256
# и ставит бинарник в ${PREFIX:-$HOME/.local}/bin.
set -eu

REPO="detrenasama/tasky"
API="https://api.github.com/repos/${REPO}/releases/latest"
PREFIX="${PREFIX:-$HOME/.local}"

require() {
	command -v "$1" >/dev/null 2>&1 || {
		echo "Ошибка: не найден $1" >&2
		exit 1
	}
}

os="$(uname -s)"
case "$os" in
Linux) os=linux ;;
*)
	echo "Ошибка: ОС $os не поддерживается (нужен Linux)." >&2
	exit 1
	;;
esac

arch="$(uname -m)"
case "$arch" in
x86_64 | amd64) arch=amd64 ;;
aarch64 | arm64) arch=arm64 ;;
*)
	echo "Ошибка: архитектура $arch не поддерживается (amd64 или arm64)." >&2
	exit 1
	;;
esac

require curl
require sha256sum
require tar

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

tag="$(curl -fsSL "$API" | sed -n 's/.*"tag_name":"\([^"]*\)".*/\1/p')"
[ -n "$tag" ] || {
	echo "Ошибка: не удалось получить последнюю версию." >&2
	exit 1
}
echo "Установка Tasky $tag ($os/$arch)..."

asset="tasky-${os}-${arch}.tar.gz"
curl -fsSL "https://github.com/${REPO}/releases/download/${tag}/${asset}" -o "$tmp/tasky.tar.gz"
curl -fsSL "https://github.com/${REPO}/releases/download/${tag}/SHA256SUMS" -o "$tmp/SHA256SUMS"

want="$(awk -v a="$asset" '$2 == a {print $1}' "$tmp/SHA256SUMS")"
[ -n "$want" ] || {
	echo "Ошибка: в SHA256SUMS нет записи для $asset." >&2
	exit 1
}
got="$(sha256sum "$tmp/tasky.tar.gz" | awk '{print $1}')"
[ "$want" = "$got" ] || {
	echo "Ошибка: контрольная сумма не совпадает." >&2
	exit 1
}

mkdir -p "$PREFIX/bin"
tar xzf "$tmp/tasky.tar.gz" -C "$tmp"
install -m755 "$tmp/tasky" "$PREFIX/bin/tasky"

mkdir -p "$HOME/.local/share/tasky"
echo "Установлено: $PREFIX/bin/tasky ($tag)"
echo "Обновление: tasky upgrade"