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

DATA_DIR="${XDG_DATA_HOME:-$HOME/.local/share}/tasky"
mkdir -p "$DATA_DIR"

REPO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
IND_SRC="$REPO_DIR/indicators/gnome"

echo ""
echo "Установка завершена."
echo "  Бинарник:    $PREFIX/bin/tasky ($tag)"
echo "  Данные:      $DATA_DIR"
echo "  Обновление:  tasky upgrade"
echo ""

# Проверка окружения и предложение системного индикатора.
desktop="${XDG_CURRENT_DESKTOP:-}"
case "$desktop" in
	*gnome*) : ;;
	*) command -v gnome-shell >/dev/null 2>&1 && desktop="gnome" ;;
esac

if [ "${desktop#*gnome}" != "$desktop" ] || command -v gnome-shell >/dev/null 2>&1; then
	if [ -d "$IND_SRC" ]; then
		echo "Обнаружен GNOME Shell. Доступен системный индикатор (иконка в трее):"
		echo "  показывает время за сегодня, запускает/останавливает сервер"
		echo "  и открывает веб-интерфейс. Установить индикатор? [y/N]"
		printf "> "
		read -r ans || ans=""
		case "$ans" in
			y|Y|yes|YES|д|Д|да|Да)
				EXT_DIR="${XDG_DATA_HOME:-$HOME/.local/share}/gnome-shell/extensions/tasky-indicator@detrenasama"
				mkdir -p "$EXT_DIR"
				cp "$IND_SRC/extension.js" "$IND_SRC/metadata.json" "$EXT_DIR/"
				echo "Индикатор установлен: $EXT_DIR"
				if command -v gnome-extensions >/dev/null 2>&1; then
					gnome-extensions enable tasky-indicator@detrenasama 2>/dev/null || \
						echo "Включите вручную (или перезайдите в сессию): gnome-extensions enable tasky-indicator@detrenasama"
				fi
				;;
			*)
				echo "Пропущено. Позже: cd $IND_SRC && ./install.sh"
				;;
		esac
	fi
else
	echo "Системный индикатор доступен только для GNOME Shell (установите позже из $IND_SRC)."
	echo "На Windows индикатор (tasky-indicator.exe) уже входит в zip релиза рядом с tasky.exe."
fi