package db

import "database/sql"

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
    status       TEXT    NOT NULL DEFAULT 'todo'
                 CHECK (status IN ('todo', 'in_progress', 'done')),
    created_at   INTEGER NOT NULL,
    completed_at INTEGER
);

CREATE INDEX IF NOT EXISTS idx_tasks_project ON tasks(project_id);

CREATE TABLE IF NOT EXISTS subtasks (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    task_id      INTEGER NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    title        TEXT    NOT NULL,
    description  TEXT    NOT NULL DEFAULT '',
    status       TEXT    NOT NULL DEFAULT 'todo'
                 CHECK (status IN ('todo', 'in_progress', 'done')),
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
	return ensureColumn(conn, "subtasks", "description", "TEXT NOT NULL DEFAULT ''")
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
