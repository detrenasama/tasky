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

// ConfigDir возвращает корень конфигурации Tasky (темы): $TASKY_CONFIG_HOME,
// иначе $XDG_CONFIG_HOME/tasky, иначе ~/.config/tasky.
func ConfigDir() string {
	if v := os.Getenv("TASKY_CONFIG_HOME"); v != "" {
		return v
	}
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return filepath.Join(".", "tasky")
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "tasky")
}
