package db

import (
	"database/sql"
	"testing"
	"time"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	conn, err := Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { conn.Close() })
	return conn
}

func exec(t *testing.T, conn *sql.DB, query string, args ...any) {
	t.Helper()
	if _, err := conn.Exec(query, args...); err != nil {
		t.Fatalf("Exec %q: %v", query, err)
	}
}

func TestSchemaCreatesTables(t *testing.T) {
	conn := openTestDB(t)
	var n int
	for _, table := range []string{"projects", "tasks", "subtasks", "time_entries",
		"project_links", "task_links", "subtask_links", "journal_entries"} {
		err := conn.QueryRow(
			"SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?", table,
		).Scan(&n)
		if err != nil {
			t.Fatalf("query table %s: %v", table, err)
		}
		if n != 1 {
			t.Errorf("таблица %s не создана", table)
		}
	}
}

func TestCascadeDelete(t *testing.T) {
	conn := openTestDB(t)
	now := time.Now().Unix()
	exec(t, conn, "INSERT INTO projects (name, created_at) VALUES ('p', ?)", now)
	var projectID int64
	if err := conn.QueryRow("SELECT id FROM projects").Scan(&projectID); err != nil {
		t.Fatal(err)
	}
	exec(t, conn, "INSERT INTO tasks (project_id, title, created_at) VALUES (?, 't', ?)", projectID, now)
	var taskID int64
	if err := conn.QueryRow("SELECT id FROM tasks").Scan(&taskID); err != nil {
		t.Fatal(err)
	}
	exec(t, conn, "INSERT INTO subtasks (task_id, title, created_at) VALUES (?, 's', ?)", taskID, now)
	var subtaskID int64
	if err := conn.QueryRow("SELECT id FROM subtasks").Scan(&subtaskID); err != nil {
		t.Fatal(err)
	}
	exec(t, conn, "INSERT INTO time_entries (subtask_id, started_at, ended_at) VALUES (?, ?, ?)",
		subtaskID, now, now+60)

	exec(t, conn, "DELETE FROM projects WHERE id = ?", projectID)

	for _, table := range []string{"tasks", "subtasks", "time_entries"} {
		var n int
		if err := conn.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&n); err != nil {
			t.Fatalf("count %s: %v", table, err)
		}
		if n != 0 {
			t.Errorf("каскадное удаление: в %s осталось %d строк", table, n)
		}
	}
}

func TestOnlyOneOpenSession(t *testing.T) {
	conn := openTestDB(t)
	now := time.Now().Unix()
	exec(t, conn, "INSERT INTO projects (name, created_at) VALUES ('p', ?)", now)
	var projectID int64
	if err := conn.QueryRow("SELECT id FROM projects").Scan(&projectID); err != nil {
		t.Fatal(err)
	}
	exec(t, conn, "INSERT INTO tasks (project_id, title, created_at) VALUES (?, 't', ?)", projectID, now)
	var taskID int64
	if err := conn.QueryRow("SELECT id FROM tasks").Scan(&taskID); err != nil {
		t.Fatal(err)
	}
	exec(t, conn, "INSERT INTO subtasks (task_id, title, created_at) VALUES (?, 's', ?)", taskID, now)
	var subtaskID int64
	if err := conn.QueryRow("SELECT id FROM subtasks").Scan(&subtaskID); err != nil {
		t.Fatal(err)
	}

	exec(t, conn, "INSERT INTO time_entries (subtask_id, started_at) VALUES (?, ?)", subtaskID, now)

	if _, err := conn.Exec(
		"INSERT INTO time_entries (subtask_id, started_at) VALUES (?, ?)", subtaskID, now+10,
	); err == nil {
		t.Fatal("вторая открытая сессия на той же подзадаче не запрещена")
	}
}

// TestMigrateAddsSubtaskDescription — старая БД без колонки description у
// подзадач: CreateSchema добавляет колонку через ALTER TABLE.
func TestMigrateAddsSubtaskDescription(t *testing.T) {
	conn := openTestDB(t)
	// имитация старой схемы: subtasks без description
	exec(t, conn, "DROP TABLE subtasks")
	exec(t, conn, `
CREATE TABLE subtasks (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    task_id      INTEGER NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    title        TEXT    NOT NULL,
    status       TEXT    NOT NULL DEFAULT 'todo',
    sort_order   INTEGER NOT NULL DEFAULT 0,
    created_at   INTEGER NOT NULL,
    completed_at INTEGER
)`)

	if err := CreateSchema(conn); err != nil {
		t.Fatalf("CreateSchema: %v", err)
	}
	var n int
	err := conn.QueryRow(
		"SELECT COUNT(*) FROM pragma_table_info('subtasks') WHERE name = 'description'",
	).Scan(&n)
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Error("колонка description у subtasks не добавлена миграцией")
	}

	// существующие строки получают пустое описание
	now := time.Now().Unix()
	exec(t, conn, "INSERT INTO projects (name, created_at) VALUES ('p', ?)", now)
	var pid int64
	if err := conn.QueryRow("SELECT id FROM projects").Scan(&pid); err != nil {
		t.Fatal(err)
	}
	exec(t, conn, "INSERT INTO tasks (project_id, title, created_at) VALUES (?, 't', ?)", pid, now)
	var tid int64
	if err := conn.QueryRow("SELECT id FROM tasks").Scan(&tid); err != nil {
		t.Fatal(err)
	}
	exec(t, conn, "INSERT INTO subtasks (task_id, title, created_at) VALUES (?, 's', ?)", tid, now)
	var desc string
	if err := conn.QueryRow("SELECT description FROM subtasks").Scan(&desc); err != nil {
		t.Fatalf("чтение description: %v", err)
	}
	if desc != "" {
		t.Errorf("description старой строки = %q, ожидалось пустое", desc)
	}
}

// TestNewLinkTablesCascade — каскадное удаление для task_links, subtask_links
// и journal_entries.
func TestNewLinkTablesCascade(t *testing.T) {
	conn := openTestDB(t)
	pid := seedProject(t, conn)
	task, err := CreateTask(conn, pid, "t")
	if err != nil {
		t.Fatal(err)
	}
	sub, err := CreateSubtask(conn, task.ID, "s")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := CreateTaskLink(conn, task.ID, "t1", "https://a"); err != nil {
		t.Fatal(err)
	}
	if _, err := CreateSubtaskLink(conn, sub.ID, "s1", "https://b"); err != nil {
		t.Fatal(err)
	}
	if _, err := CreateJournalEntry(conn, sub.ID, "запись"); err != nil {
		t.Fatal(err)
	}

	if err := DeleteTask(conn, task.ID); err != nil {
		t.Fatalf("DeleteTask: %v", err)
	}
	for _, table := range []string{"task_links", "subtask_links", "journal_entries"} {
		var n int
		if err := conn.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&n); err != nil {
			t.Fatalf("count %s: %v", table, err)
		}
		if n != 0 {
			t.Errorf("каскад: в %s осталось %d строк", table, n)
		}
	}
}
