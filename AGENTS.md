# AGENTS.md

TUI-приложение «Tasky» на Go: менеджер задач с учётом времени.

## Команды

- `go run .` — запуск приложения (требуется настоящий TTY)
- `go build -o tasky .` — сборка бинарника (заметь: `go build ./...` НЕ создаёт файл в каталоге)
- `go vet ./...` и `gofmt -l .` — проверки перед завершением задачи
- `go test ./...` — тесты (internal/db/schema_test.go)

## Структура и стек

- `main.go` — BubbleTea-модель (`Init`/`Update`/`View`), выход по `q`/`ctrl+c`
- `internal/db` — пакет `db.Open(path)` поверх `database/sql` + `modernc.org/sqlite` (чистый Go, без CGO)
- БД: файл `tasky.db` в рабочем каталоге; схема (projects → tasks → subtasks → time_entries) создаётся автоматически при открытии (см. `schema.go`)
- UI: `bubbletea` + `bubbles` + `lipgloss`; запуск через `tea.NewProgram(..., tea.WithAltScreen())`
- UI: `bubbletea` + `bubbles` + `lipgloss`; запуск через `tea.NewProgram(..., tea.WithAltScreen())`

## Нюансы

- SQLite открывается через DSN с прагмами `foreign_keys(1)` и `busy_timeout(5000)` (см. `db.go`) — не отключать: без FK каскады не работают.
- Активная сессия учёта времени — это запись `time_entries` с `ended_at IS NULL`; не более одной на подзадачу (unique partial index).
- Тестировать TUI в неинтерактивной оболочке нельзя (`/dev/tty` недоступен). Проверка: `printf 'q' | timeout 10 script -qec "./tasky" /dev/null` — должен завершиться с кодом 0.
- `tasky.db` и бинарник `tasky` в `.gitignore`.
- Режим приложения только альтернативный экран (WithAltScreen).
