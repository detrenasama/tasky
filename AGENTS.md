# AGENTS.md

TUI-приложение «Tasky» на Go: менеджер задач с учётом времени.

## Команды

- `go run .` — запуск приложения (требуется настоящий TTY)
- `go build -o tasky .` — сборка бинарника (заметь: `go build ./...` НЕ создаёт файл в каталоге)
- `go vet ./...` и `gofmt -l .` — проверки перед завершением задачи
- `go test ./...` — тесты (internal/db/schema_test.go)

## Структура и стек

- `main.go` — корневая BubbleTea-модель: переключение экранов (`tasks`/`projects`), тик 1с (обновление таймеров и недельной суммы), `tea.WindowSizeMsg`, выход по `q`/`ctrl+c`
- `tasks_screen.go` — главный экран: задачи выбранного проекта (левая панель), подзадачи с временем (правая), внизу справа — недельная панель; `tab` — фокус панели, `n`/`d` — создать/удалить (создание идёт в сфокусированной панели, удаление с подтверждением y/n), `ctrl+l` — старт/пауза подзадачи, `[`/`]` — смена проекта, `p` — проекты
- `projects_screen.go` — экран проектов: `n` создать (textinput), `d` удалить (подтверждение y/n), `esc` — назад
- `internal/db` — пакет `db.Open(path)` поверх `database/sql` + `modernc.org/sqlite` (чистый Go, без CGO); типы в `types.go`, запросы: `projects.go`, `tasks.go`, `time.go`
- БД: файл `tasky.db` в рабочем каталоге; схема (projects → tasks → subtasks → time_entries) создаётся автоматически при открытии (см. `schema.go`)
- UI: `bubbletea` + `bubbles` (list, textinput) + `lipgloss`; запуск через `tea.NewProgram(..., tea.WithAltScreen())`
- Списки (`bubbles/list` v1.0.0): делегат — `list.NewDefaultDelegate()`, описание элемента — методы `Title()`/`Description()` интерфейса `DefaultItem` (старого `SetDescriptionFunc` нет)
- UI: `bubbletea` + `bubbles` + `lipgloss`; запуск через `tea.NewProgram(..., tea.WithAltScreen())`

## Нюансы

- SQLite открывается через DSN с прагмами `foreign_keys(1)` и `busy_timeout(5000)` (см. `db.go`) — не отключать: без FK каскады не работают.
- Активная сессия учёта времени — это запись `time_entries` с `ended_at IS NULL`; не более одной на подзадачу (unique partial index).
- Недельная сумма (`WeeklyTotal` в `time.go`): неделя с понедельника 00:00 локального времени; активная сессия учитывается частично (до текущего момента).
- Экраны — указатели (`*tasksScreen`): описания элементов подзадач читают `now` через замыкание на экран, поэтому без указателя обновления таймера не будут видны.
- `list.Select(0)` на пустом списке ставит `Index()=0` при `len(items)==0` — перед `Select` и индексацией по выбранному элементу всегда проверять длину (`canDelete()` в `tasks_screen.go`).
- Тестировать TUI в неинтерактивной оболочке нельзя (`/dev/tty` недоступен). Проверка: `printf 'q' | timeout 10 script -qec "./tasky" /dev/null` — должен завершиться с кодом 0. При прогоне клавиатурных сценариев между клавишами нужны паузы ≥0.3с (ранние нажатия теряются до инициализации TUI).
- `tasky.db` и бинарник `tasky` в `.gitignore`.
- Режим приложения только альтернативный экран (WithAltScreen).
