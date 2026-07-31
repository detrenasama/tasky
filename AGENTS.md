# AGENTS.md

TUI-приложение «Tasky» на Go: менеджер задач с учётом времени.

## Команды

- `go run .` — запуск приложения (требуется настоящий TTY)
- `go build -o tasky .` — сборка бинарника (заметь: `go build ./...` НЕ создаёт файл в каталоге)
- `go vet ./...` и `gofmt -l .` — проверки перед завершением задачи
- `go test ./...` — тесты (пока нет)

## Структура и стек

- `main.go` — BubbleTea-модель (`Init`/`Update`/`View`), выход по `q`/`ctrl+c`
- `internal/db` — пакет `db.Open(path)` поверх `database/sql` + `modernc.org/sqlite` (чистый Go, без CGO)
- БД: файл `tasky.db` в рабочем каталоге
- UI: `bubbletea` + `bubbles` + `lipgloss`; запуск через `tea.NewProgram(..., tea.WithAltScreen())`

## Нюансы

- Тестировать TUI в неинтерактивной оболочке нельзя (`/dev/tty` недоступен). Проверка: `printf 'q' | timeout 10 script -qec "./tasky" /dev/null` — должен завершиться с кодом 0.
- `tasky.db` и бинарник `tasky` в `.gitignore`.
- Режим приложения только альтернативный экран (WithAltScreen).
