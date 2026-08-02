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
`

func CreateSchema(conn *sql.DB) error {
	if _, err := conn.Exec(schema); err != nil {
		return err
	}
	return migrate(conn)
}

// migrate приводит существующие БД к актуальной схеме.
func migrate(conn *sql.DB) error {
	rows, err := conn.Query("PRAGMA table_info(project_links)")
	if err != nil {
		return err
	}
	defer rows.Close()
	hasName := false
	for rows.Next() {
		var cid, notnull, pk int
		var name, ctype string
		var dflt any
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return err
		}
		if name == "name" {
			hasName = true
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if !hasName {
		_, err = conn.Exec("ALTER TABLE project_links ADD COLUMN name TEXT NOT NULL DEFAULT ''")
	}
	return err
}
