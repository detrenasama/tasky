//go:build windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"

	"github.com/detrenasama/tasky/internal/browser"
)

// serverCmd — запущенный процесс сервера (nil, если не запущен).
var serverCmd *exec.Cmd

// taskyexe возвращает путь к бинарнику tasky рядом с индикатором (или из PATH).
func taskyexe() string {
	if self, err := os.Executable(); err == nil {
		dir := filepath.Dir(self)
		candidate := filepath.Join(dir, "tasky")
		if runtime.GOOS == "windows" {
			candidate += ".exe"
		}
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	if p, err := exec.LookPath("tasky"); err == nil {
		return p
	}
	if runtime.GOOS == "windows" {
		return "tasky.exe"
	}
	return "tasky"
}

// startServer запускает `tasky serve` отвязанным процессом.
func startServer() error {
	if serverCmd != nil && serverCmd.Process != nil {
		return fmt.Errorf("сервер уже запущен")
	}
	cmd := exec.Command(taskyexe(), "serve")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return err
	}
	serverCmd = cmd
	return nil
}

// stopServer останавливает запущенный сервер. На Unix шлёт SIGTERM; на Windows
// Go не поддерживает SIGTERM для процесса — откатываемся к Kill.
func stopServer() error {
	if serverCmd == nil || serverCmd.Process == nil {
		return fmt.Errorf("сервер не запущен")
	}
	if err := serverCmd.Process.Signal(syscall.SIGTERM); err != nil {
		_ = serverCmd.Process.Kill()
	}
	serverCmd = nil
	return nil
}

// openBrowser открывает веб-интерфейс в браузере по умолчанию.
func openBrowser() error {
	addr := os.Getenv("TASKY_HTTP_ADDR")
	if addr == "" {
		addr = "127.0.0.1:9110"
	}
	return browser.Open("http://" + addr)
}
