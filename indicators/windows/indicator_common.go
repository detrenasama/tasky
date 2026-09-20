//go:build windows

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/detrenasama/tasky/internal/browser"
)

const defaultHTTPAddr = "127.0.0.1:9110"

// serverCmd — запущенный процесс сервера (nil, если не запущен).
var serverCmd *exec.Cmd

// httpAddr возвращает адрес HTTP-сервера: TASKY_HTTP_ADDR или дефолт.
// Читается при каждом опросе/запуске, чтобы подхватить изменение env без
// перезапуска индикатора.
func httpAddr() string {
	if a := os.Getenv("TASKY_HTTP_ADDR"); a != "" {
		return a
	}
	return defaultHTTPAddr
}

// socketPath возвращает путь сокета сервера: TASKY_SOCKET или TASKY_HOME/tasky.sock.
// Пустая строка — дефолт (<XDG_DATA_HOME>/tasky/tasky.sock), передавать --socket не нужно.
func socketPath() string {
	if s := os.Getenv("TASKY_SOCKET"); s != "" {
		return s
	}
	if h := os.Getenv("TASKY_HOME"); h != "" {
		return filepath.Join(h, "tasky.sock")
	}
	return ""
}

// taskyexe возвращает путь к бинарнику tasky рядом с индикатором (или из PATH).
// Поддерживает штатные места установки: рядом с индикатором, PATH, ~/go/bin и т.п.
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
	// Фолбэки как в GNOME taskyBin(): ~/go/bin, GOPATH/bin, ~/.local/bin.
	if home, err := os.UserHomeDir(); err == nil {
		candidates := []string{
			filepath.Join(home, "go", "bin", "tasky.exe"),
			filepath.Join(home, ".local", "bin", "tasky.exe"),
		}
		if gopath := os.Getenv("GOPATH"); gopath != "" {
			candidates = append(candidates, filepath.Join(gopath, "bin", "tasky.exe"))
		}
		for _, c := range candidates {
			if _, err := os.Stat(c); err == nil {
				return c
			}
		}
	}
	if runtime.GOOS == "windows" {
		return "tasky.exe"
	}
	return "tasky"
}

// serverReachable проверяет доступность сервера через HTTP GET /status.
func serverReachable() bool {
	addr := httpAddr()
	url := "http://" + addr + "/status"
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false
	}
	var data struct {
		TodaySeconds int64 `json:"today_seconds"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return false
	}
	return true
}

// startServer запускает `tasky serve` отвязанным процессом.
func startServer() error {
	if serverCmd != nil && serverCmd.Process != nil {
		return fmt.Errorf("сервер уже запущен")
	}
	if serverReachable() {
		return fmt.Errorf("сервер уже запущен (внешний)")
	}
	bin := taskyexe()
	// exec.LookPath уже проверен, но если бинарь не найден — вернём ошибку явно.
	if _, err := exec.LookPath(bin); err != nil {
		if _, err2 := os.Stat(bin); err2 != nil {
			return fmt.Errorf("tasky не найден (checked PATH, рядом с индикатором, ~/go/bin/tasky.exe)")
		}
	}
	args := []string{"serve"}
	addr := httpAddr()
	if raw := os.Getenv("TASKY_HTTP_ADDR"); raw != "" || addr != defaultHTTPAddr {
		args = append(args, "--http-addr", addr)
	}
	if sp := socketPath(); sp != "" {
		args = append(args, "--socket", sp)
	}
	cmd := exec.Command(bin, args...)
	// Не привязываем stdout/stderr к консоли индикатора.
	if err := cmd.Start(); err != nil {
		return err
	}
	serverCmd = cmd
	// Следим за выходом своего процесса — если упал сразу (порт занят и т.п.),
	// сбросим serverCmd, меню обновится на следующем опросе.
	go func(c *exec.Cmd) {
		_ = c.Wait()
		if serverCmd == c {
			serverCmd = nil
		}
	}(cmd)
	return nil
}

// stopServer останавливает только свой сервер. На Unix шлёт SIGTERM; на Windows
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
	return browser.Open("http://" + httpAddr())
}
