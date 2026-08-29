// Package browser открывает URL в браузере по умолчанию кроссплатформенно.
// Используется командой `tasky web` и системным индикатором (трей).
package browser

import (
	"os/exec"
	"runtime"
)

// Open открывает url в браузере по умолчанию. На Windows использует
// rundll32, на macOS — open, на остальных — xdg-open.
func Open(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}
