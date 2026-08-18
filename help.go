package main

import (
	"fmt"

	"github.com/detrenasama/tasky/internal/xdg"
)

// buildHelpText собирает текст справки по CLI-командам.
func buildHelpText() string {
	return fmt.Sprintf(`Tasky — TUI-менеджер задач с учётом времени

Использование:
  tasky               запустить интерфейс (поднимает локальный сервер до выхода;
                      к уже запущенному серверу подключается автоматически)
  tasky serve         запустить только сервер (продолжает работать без TUI)
  tasky attach        подключиться к работающему серверу
  tasky status        вывести время за сегодня и запущенную подзадачу (JSON)
  tasky help          показать эту справку (то же: --help, -h)
  tasky --version     показать версию (то же: -v)
  tasky upgrade       обновить до последней версии

Сокет сервера (по умолчанию <каталог данных>/tasky.sock):
  --socket PATH       задать путь сокета (то же: TASKY_SOCKET)

HTTP для внешних интеграций (по умолчанию http://127.0.0.1:9110):
  --http-addr ADDR    задать адрес HTTP-эндпоинтов (то же: TASKY_HTTP_ADDR)
                      (GET /status — время за сегодня и запущенная подзадача)

Данные: %s (переопределение — TASKY_HOME)
Версия: %s
`, xdg.DataDir(), version)
}

// runHelp печатает справку и возвращает код выхода.
func runHelp() int {
	fmt.Print(buildHelpText())
	return 0
}
