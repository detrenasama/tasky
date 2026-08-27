package ui

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// copyToClipboard копирует text в системный буфер обмена. Выбирает инструмент
// по окружению: wl-copy (Wayland), xclip (X11), pbcopy (macOS). Если подходящего
// инструмента нет, возвращает ошибку.
func copyToClipboard(text string) error {
	if text == "" {
		return fmt.Errorf("нет текста для копирования")
	}
	var cmd *exec.Cmd
	switch {
	case os.Getenv("WAYLAND_DISPLAY") != "":
		cmd = exec.Command("wl-copy")
	case commandExists("xclip"):
		cmd = exec.Command("xclip", "-selection", "clipboard")
	case commandExists("xsel"):
		cmd = exec.Command("xsel", "--clipboard", "--input")
	case commandExists("pbcopy"):
		cmd = exec.Command("pbcopy")
	default:
		return fmt.Errorf("не найден инструмент буфера обмена (wl-copy/xclip/pbcopy)")
	}
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}

func commandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}
