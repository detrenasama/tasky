// Package xdg определяет каталог данных приложения по XDG Base Directory.
package xdg

import (
	"os"
	"path/filepath"
)

// DataDir возвращает корень данных Tasky: $TASKY_HOME, иначе
// $XDG_DATA_HOME/tasky, иначе ~/.local/share/tasky.
func DataDir() string {
	if v := os.Getenv("TASKY_HOME"); v != "" {
		return v
	}
	base := os.Getenv("XDG_DATA_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return filepath.Join(".", "tasky")
		}
		base = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(base, "tasky")
}
