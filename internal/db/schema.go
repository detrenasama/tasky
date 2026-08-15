package db

import (
	"database/sql"
	"strings"
)

const schema = `
CREATE TABLE IF NOT EXISTS projects (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    name        TEXT    NOT NULL UNIQUE,
    description TEXT    NOT NULL DEFAULT '',
    created_at  INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS tasks (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id   INTEGER NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    title        TEXT    NOT NULL,
    description  TEXT    NOT NULL DEFAULT '',
    status       TEXT    NOT NULL DEFAULT '',
    created_at   INTEGER NOT NULL,
    completed_at INTEGER
);

CREATE INDEX IF NOT EXISTS idx_tasks_project ON tasks(project_id);

CREATE TABLE IF NOT EXISTS subtasks (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    task_id      INTEGER NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    title        TEXT    NOT NULL,
    description  TEXT    NOT NULL DEFAULT '',
    status       TEXT    NOT NULL DEFAULT '',
    sort_order   INTEGER NOT NULL DEFAULT 0,
    created_at   INTEGER NOT NULL,
    completed_at INTEGER
);

CREATE INDEX IF NOT EXISTS idx_subtasks_task ON subtasks(task_id);

CREATE TABLE IF NOT EXISTS time_entries (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    subtask_id INTEGER NOT NULL REFERENCES subtasks(id) ON DELETE CASCADE,
    started_at INTEGER NOT NULL,
    ended_at   INTEGER,
    note       TEXT    NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_time_entries_subtask ON time_entries(subtask_id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_time_entries_open
    ON time_entries(subtask_id) WHERE ended_at IS NULL;

CREATE TABLE IF NOT EXISTS project_links (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id INTEGER NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name       TEXT    NOT NULL DEFAULT '',
    url        TEXT    NOT NULL,
    created_at INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_project_links_project ON project_links(project_id);

CREATE TABLE IF NOT EXISTS task_links (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    task_id    INTEGER NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    name       TEXT    NOT NULL DEFAULT '',
    url        TEXT    NOT NULL,
    created_at INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_task_links_task ON task_links(task_id);

CREATE TABLE IF NOT EXISTS subtask_links (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    subtask_id INTEGER NOT NULL REFERENCES subtasks(id) ON DELETE CASCADE,
    name       TEXT    NOT NULL DEFAULT '',
    url        TEXT    NOT NULL,
    created_at INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_subtask_links_subtask ON subtask_links(subtask_id);

CREATE TABLE IF NOT EXISTS journal_entries (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    subtask_id INTEGER NOT NULL REFERENCES subtasks(id) ON DELETE CASCADE,
    created_at INTEGER NOT NULL,
    text       TEXT    NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_journal_entries_subtask ON journal_entries(subtask_id);

-- Каталог статусов: настраиваемый список (см. statuses.go, сид по умолчанию —
-- в миграции). type: new | in_progress | done; is_quick — участвует в быстрой
-- цепочке смены статуса; note_prompt — подсказка для обязательной заметки.
CREATE TABLE IF NOT EXISTS statuses (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    name        TEXT    NOT NULL UNIQUE,
    type        TEXT    NOT NULL DEFAULT 'new'
                CHECK (type IN ('new', 'in_progress', 'done')),
    color       TEXT    NOT NULL DEFAULT '#8a8a8a',
    note_prompt TEXT    NOT NULL DEFAULT '',
    is_quick    INTEGER NOT NULL DEFAULT 0,
    sort_order  INTEGER NOT NULL DEFAULT 0
);

-- История смены статусов задач (переходы подзадач пишутся в journal_entries).
CREATE TABLE IF NOT EXISTS status_history (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    task_id     INTEGER REFERENCES tasks(id) ON DELETE CASCADE,
    subtask_id  INTEGER REFERENCES subtasks(id) ON DELETE CASCADE,
    from_status TEXT    NOT NULL DEFAULT '',
    to_status   TEXT    NOT NULL,
    note        TEXT    NOT NULL DEFAULT '',
    created_at  INTEGER NOT NULL,
    CHECK ((task_id IS NULL) != (subtask_id IS NULL))
);

-- Простые настройки приложения (ключ → значение).
CREATE TABLE IF NOT EXISTS settings (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

-- Каталог типов тегов: настраиваемый список (см. tags.go, сид по умолчанию —
-- в миграции). kind: text | task_id — «текст» или номер задачи внешнего сервиса.
CREATE TABLE IF NOT EXISTS tag_types (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    name       TEXT    NOT NULL UNIQUE,
    kind       TEXT    NOT NULL DEFAULT 'text'
                CHECK (kind IN ('text', 'task_id')),
    color      TEXT    NOT NULL DEFAULT '#8a8a8a',
    sort_order INTEGER NOT NULL DEFAULT 0
);

-- Теги задач: значение + ссылка на тип (цвет берётся из типа) + URL опционально.
CREATE TABLE IF NOT EXISTS task_tags (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    task_id    INTEGER NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    type_id    INTEGER NOT NULL REFERENCES tag_types(id),
    text       TEXT    NOT NULL,
    url        TEXT    NOT NULL DEFAULT '',
    created_at INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_task_tags_task ON task_tags(task_id);
`

func CreateSchema(conn *sql.DB) error {
	if _, err := conn.Exec(schema); err != nil {
		return err
	}
	return migrate(conn)
}

// migrate приводит существующие БД к актуальной схеме.
func migrate(conn *sql.DB) error {
	if err := ensureColumn(conn, "project_links", "name", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	if err := ensureColumn(conn, "subtasks", "description", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	return migrateStatuses(conn)
}

// migrateStatuses пересоздаёт tasks/subtasks без CHECK на status (старые
// значения маппятся на статусы по умолчанию) и сидит каталог статусов.
func migrateStatuses(conn *sql.DB) error {
	mapping := map[string]string{
		"todo":        "Новая",
		"in_progress": "В работе",
		"done":        "Выполнена",
	}
	for _, t := range []struct {
		table string
		ddl   string
	}{
		{
			table: "tasks",
			ddl: `CREATE TABLE tasks (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id   INTEGER NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    title        TEXT    NOT NULL,
    description  TEXT    NOT NULL DEFAULT '',
    status       TEXT    NOT NULL DEFAULT '',
    created_at   INTEGER NOT NULL,
    completed_at INTEGER
)`,
		},
		{
			table: "subtasks",
			ddl: `CREATE TABLE subtasks (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    task_id      INTEGER NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    title        TEXT    NOT NULL,
    description  TEXT    NOT NULL DEFAULT '',
    status       TEXT    NOT NULL DEFAULT '',
    sort_order   INTEGER NOT NULL DEFAULT 0,
    created_at   INTEGER NOT NULL,
    completed_at INTEGER
)`,
		},
	} {
		if err := rebuildTable(conn, t.table, t.ddl, mapping); err != nil {
			return err
		}
	}
	if err := repairDanglingRefs(conn); err != nil {
		return err
	}
	if err := migrateStatusJournal(conn); err != nil {
		return err
	}
	if err := seedStatuses(conn); err != nil {
		return err
	}
	return seedTagTypes(conn)
}

// seedTagTypes заполняет каталог типов тегов значениями по умолчанию, если
// он пуст. По умолчанию — два типа для номеров задач внешних сервисов
// (Jira и трекер компании); пользователь может изменить каталог в настройках.
func seedTagTypes(conn *sql.DB) error {
	var n int
	if err := conn.QueryRow("SELECT COUNT(*) FROM tag_types").Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	defaults := []struct {
		name, kind, color string
	}{
		{"Jira", "task_id", "#569cd6"},
		{"Трекер", "task_id", "#6a9955"},
	}
	tx, err := conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for i, t := range defaults {
		if _, err := tx.Exec(`
INSERT INTO tag_types (name, kind, color, sort_order)
VALUES (?, ?, ?, ?)`, t.name, t.kind, t.color, i); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// migrateStatusJournal переносит старые записи журнала смены статусов
// подзадач («Статус: X → Y (заметка)») из journal_entries в status_history.
// Записи, которые не удалось разобрать, остаются в журнале.
func migrateStatusJournal(conn *sql.DB) error {
	rows, err := conn.Query(`
SELECT id, subtask_id, created_at, text FROM journal_entries
WHERE text LIKE 'Статус: %'`)
	if err != nil {
		return err
	}
	var toMove []struct {
		id, subtaskID, ts int64
		from, to, note    string
	}
	for rows.Next() {
		var id, subtaskID, ts int64
		var text string
		if err := rows.Scan(&id, &subtaskID, &ts, &text); err != nil {
			rows.Close()
			return err
		}
		from, to, note, ok := parseStatusEntry(text)
		if !ok {
			continue
		}
		toMove = append(toMove, struct {
			id, subtaskID, ts int64
			from, to, note    string
		}{id, subtaskID, ts, from, to, note})
	}
	if err := rows.Close(); err != nil {
		return err
	}
	if len(toMove) == 0 {
		return nil
	}
	tx, err := conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, e := range toMove {
		if _, err := tx.Exec(`
INSERT INTO status_history (subtask_id, from_status, to_status, note, created_at)
VALUES (?, ?, ?, ?, ?)`, e.subtaskID, e.from, e.to, e.note, e.ts); err != nil {
			return err
		}
		if _, err := tx.Exec(
			"DELETE FROM journal_entries WHERE id = ?", e.id); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// parseStatusEntry разбирает строку журнала «Статус: From → To (note)».
func parseStatusEntry(text string) (from, to, note string, ok bool) {
	const prefix = "Статус: "
	if !strings.HasPrefix(text, prefix) {
		return "", "", "", false
	}
	rest := strings.TrimPrefix(text, prefix)
	arrow := strings.Index(rest, " → ")
	if arrow < 0 {
		return "", "", "", false
	}
	from = rest[:arrow]
	tail := rest[arrow+len(" → "):]
	if i := strings.LastIndex(tail, " ("); i >= 0 && strings.HasSuffix(tail, ")") {
		to = tail[:i]
		note = tail[i+2 : len(tail)-1]
	} else {
		to = tail
	}
	if from == "" || to == "" {
		return "", "", "", false
	}
	return from, to, note, true
}

// rebuildTable заменяет таблицу на версию без CHECK-ограничения на status,
// копируя данные с маппингом старых значений. Ничего не делает, если
// CHECK в таблице уже нет. Новая таблица создаётся под временным именем,
// старые данные копируются, затем старая таблица удаляется и новая
// переименовывается на её место — внешние ключи остальных таблиц
// ссылаются на имя, а не на физическую таблицу, и продолжают работать.
func rebuildTable(conn *sql.DB, table, ddl string, mapping map[string]string) error {
	var oldSQL string
	if err := conn.QueryRow(
		"SELECT sql FROM sqlite_master WHERE type = 'table' AND name = ?", table,
	).Scan(&oldSQL); err != nil {
		return err
	}
	if !strings.Contains(strings.ToUpper(oldSQL), "CHECK") {
		return nil
	}
	cols := map[string][]string{
		"tasks":    {"id", "project_id", "title", "description", "status", "created_at", "completed_at"},
		"subtasks": {"id", "task_id", "title", "description", "status", "sort_order", "created_at", "completed_at"},
	}
	// SELECT-список: все колонки кроме status, вместо status — CASE-маппинг.
	sel := make([]string, 0, len(cols[table]))
	for _, c := range cols[table] {
		if c == "status" {
			sel = append(sel, "CASE status "+statusMappingSQL(mapping)+" END")
		} else {
			sel = append(sel, c)
		}
	}
	selSQL := "INSERT INTO " + table + " (" + strings.Join(cols[table], ", ") +
		") SELECT " + strings.Join(sel, ", ") + " FROM " + table
	if err := replaceTable(conn, table, ddl, selSQL); err != nil {
		return err
	}
	indexes := map[string]string{
		"tasks":    "CREATE INDEX IF NOT EXISTS idx_tasks_project ON tasks(project_id)",
		"subtasks": "CREATE INDEX IF NOT EXISTS idx_subtasks_task ON subtasks(task_id)",
	}
	if idx, ok := indexes[table]; ok {
		_, err := conn.Exec(idx)
		return err
	}
	return nil
}

// replaceTable создаёт таблицу под временным именем `<table>_new`, копирует
// в неё данные запросом selSQL (INSERT INTO <table> ... SELECT ... FROM <table>
// — целевое имя переписывается на временное), удаляет старую таблицу и
// переименовывает новую на её место. Нужна foreign_keys = OFF на время
// операции.
func replaceTable(conn *sql.DB, table, ddl, selSQL string) error {
	tmp := table + "_new"
	ddlTmp := strings.Replace(ddl, "CREATE TABLE "+table, "CREATE TABLE "+tmp, 1)
	selSQL = strings.Replace(selSQL, "INSERT INTO "+table, "INSERT INTO "+tmp, 1)
	if _, err := conn.Exec("PRAGMA foreign_keys = OFF"); err != nil {
		return err
	}
	defer conn.Exec("PRAGMA foreign_keys = ON")
	if _, err := conn.Exec(ddlTmp); err != nil {
		return err
	}
	if _, err := conn.Exec(selSQL); err != nil {
		return err
	}
	if _, err := conn.Exec("DROP TABLE " + table); err != nil {
		return err
	}
	_, err := conn.Exec("ALTER TABLE " + tmp + " RENAME TO " + table)
	return err
}

// repairDanglingRefs пересоздаёт таблицы, чьи внешние ключи из-за старой
// миграции (RENAME со старым именем) указывают на удалённые tasks_old и
// subtasks_old: их DDL переписывается на актуальные имена, данные
// переносятся, индексы восстанавливаются.
func repairDanglingRefs(conn *sql.DB) error {
	rows, err := conn.Query(`
SELECT name, sql FROM sqlite_master
WHERE type = 'table' AND (sql LIKE '%tasks_old%' OR sql LIKE '%subtasks_old%')`)
	if err != nil {
		return err
	}
	var fix []struct{ name, ddl string }
	for rows.Next() {
		var name, ddl string
		if err := rows.Scan(&name, &ddl); err != nil {
			rows.Close()
			return err
		}
		fix = append(fix, struct{ name, ddl string }{name, ddl})
	}
	if err := rows.Close(); err != nil {
		return err
	}
	if len(fix) == 0 {
		return nil
	}
	for _, t := range fix {
		ddl := strings.ReplaceAll(t.ddl, "tasks_old", "tasks")
		ddl = strings.ReplaceAll(ddl, "subtasks_old", "subtasks")
		if err := replaceTable(conn, t.name, ddl,
			"INSERT INTO "+t.name+" SELECT * FROM "+t.name); err != nil {
			return err
		}
	}
	for _, idx := range statusIndexDDLs {
		if _, err := conn.Exec(idx); err != nil {
			return err
		}
	}
	return nil
}

// statusIndexDDLs — индексы таблиц, которые могут быть пересозданы при
// починке внешних ключей (речь только про индексы этих таблиц, не статусов).
var statusIndexDDLs = []string{
	"CREATE INDEX IF NOT EXISTS idx_time_entries_subtask ON time_entries(subtask_id)",
	"CREATE UNIQUE INDEX IF NOT EXISTS idx_time_entries_open ON time_entries(subtask_id) WHERE ended_at IS NULL",
	"CREATE INDEX IF NOT EXISTS idx_task_links_task ON task_links(task_id)",
	"CREATE INDEX IF NOT EXISTS idx_subtask_links_subtask ON subtask_links(subtask_id)",
	"CREATE INDEX IF NOT EXISTS idx_journal_entries_subtask ON journal_entries(subtask_id)",
}

// statusMappingSQL строит фрагмент CASE WHEN status = 'x' THEN 'y' ...
func statusMappingSQL(m map[string]string) string {
	var b strings.Builder
	for from, to := range m {
		b.WriteString("WHEN '")
		b.WriteString(from)
		b.WriteString("' THEN '")
		b.WriteString(to)
		b.WriteString("' ")
	}
	return b.String()
}

// ensureColumn проверяет наличие колонки через PRAGMA table_info и, если её
// нет, добавляет через ALTER TABLE.
func ensureColumn(conn *sql.DB, table, column, ddl string) error {
	rows, err := conn.Query("PRAGMA table_info(" + table + ")")
	if err != nil {
		return err
	}
	defer rows.Close()
	has := false
	for rows.Next() {
		var cid, notnull, pk int
		var name, ctype string
		var dflt any
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return err
		}
		if name == column {
			has = true
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if !has {
		_, err = conn.Exec("ALTER TABLE " + table + " ADD COLUMN " + column + " " + ddl)
	}
	return err
}
