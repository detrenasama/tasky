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
	for _, table := range []string{"projects", "tasks", "subtasks", "time_entries"} {
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
