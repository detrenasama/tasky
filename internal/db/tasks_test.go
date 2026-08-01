package db

import (
	"database/sql"
	"testing"
	"time"
)

func TestCreateTask(t *testing.T) {
	conn := openTestDB(t)
	now := time.Now().Unix()
	exec(t, conn, "INSERT INTO projects (name, created_at) VALUES ('p', ?)", now)
	var pid int64
	if err := conn.QueryRow("SELECT id FROM projects").Scan(&pid); err != nil {
		t.Fatal(err)
	}

	task, err := CreateTask(conn, pid, "Задача 1")
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	if task.ID == 0 || task.Status != "todo" {
		t.Errorf("CreateTask: id=%d status=%q", task.ID, task.Status)
	}

	tasks, err := TasksByProject(conn, pid)
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 1 || tasks[0].Title != "Задача 1" {
		t.Errorf("TasksByProject: %+v", tasks)
	}
}

func TestCreateSubtaskOrder(t *testing.T) {
	conn := openTestDB(t)
	pid := seedProject(t, conn)
	task, err := CreateTask(conn, pid, "t")
	if err != nil {
		t.Fatal(err)
	}

	for _, title := range []string{"s1", "s2", "s3"} {
		if _, err := CreateSubtask(conn, task.ID, title); err != nil {
			t.Fatalf("CreateSubtask(%s): %v", title, err)
		}
	}
	subs, err := SubtasksWithTime(conn, task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(subs) != 3 {
		t.Fatalf("подзадач: %d, ожидалось 3", len(subs))
	}
	for i, s := range subs {
		if s.SortOrder != int64(i+1) {
			t.Errorf("subtask %q sort_order = %d, ожидалось %d", s.Title, s.SortOrder, i+1)
		}
	}
}

func TestDeleteSubtaskCascadesTime(t *testing.T) {
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
	now := time.Now().Unix()
	exec(t, conn, "INSERT INTO time_entries (subtask_id, started_at, ended_at) VALUES (?, ?, ?)",
		sub.ID, now, now+60)

	if err := DeleteSubtask(conn, sub.ID); err != nil {
		t.Fatalf("DeleteSubtask: %v", err)
	}
	var n int
	if err := conn.QueryRow(
		"SELECT COUNT(*) FROM time_entries WHERE subtask_id = ?", sub.ID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Errorf("после удаления подзадачи осталось %d записей времени", n)
	}
	subs, err := SubtasksWithTime(conn, task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(subs) != 0 {
		t.Errorf("подзадач после удаления: %d", len(subs))
	}
}

func seedProject(t *testing.T, conn *sql.DB) int64 {
	t.Helper()
	now := time.Now().Unix()
	exec(t, conn, "INSERT INTO projects (name, created_at) VALUES ('p', ?)", now)
	var pid int64
	if err := conn.QueryRow("SELECT id FROM projects").Scan(&pid); err != nil {
		t.Fatal(err)
	}
	return pid
}
