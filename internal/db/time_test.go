package db

import (
	"database/sql"
	"testing"
	"time"
)

func seedSubtask(t *testing.T, conn *sql.DB) int64 {
	t.Helper()
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
	var sid int64
	if err := conn.QueryRow("SELECT id FROM subtasks").Scan(&sid); err != nil {
		t.Fatal(err)
	}
	return sid
}

func TestStartStopSession(t *testing.T) {
	conn := openTestDB(t)
	sid := seedSubtask(t, conn)

	now := time.Unix(1_700_000_000, 0)
	if err := StartSession(conn, sid, now); err != nil {
		t.Fatalf("StartSession: %v", err)
	}
	var open int
	if err := conn.QueryRow(
		"SELECT COUNT(*) FROM time_entries WHERE subtask_id = ? AND ended_at IS NULL", sid,
	).Scan(&open); err != nil {
		t.Fatal(err)
	}
	if open != 1 {
		t.Errorf("открытых сессий: %d, ожидалось 1", open)
	}

	if err := StartSession(conn, sid, now.Add(10*time.Second)); err == nil {
		t.Fatal("вторая открытая сессия не запрещена")
	}

	if err := StopSession(conn, sid, now.Add(5*time.Minute)); err != nil {
		t.Fatalf("StopSession: %v", err)
	}

	subs, err := SubtasksWithTime(conn, taskIDOfSubtask(t, conn, sid))
	if err != nil {
		t.Fatal(err)
	}
	if len(subs) != 1 || subs[0].TotalSeconds != 300 {
		t.Errorf("TotalSeconds = %d, ожидалось 300 (время сессии 5 минут)", subs[0].TotalSeconds)
	}
	if subs[0].ActiveSince != nil {
		t.Error("ActiveSince должен быть nil после остановки")
	}
}

func taskIDOfSubtask(t *testing.T, conn *sql.DB, sid int64) int64 {
	t.Helper()
	var tid int64
	if err := conn.QueryRow("SELECT task_id FROM subtasks WHERE id = ?", sid).Scan(&tid); err != nil {
		t.Fatal(err)
	}
	return tid
}

func TestWeeklyTotal(t *testing.T) {
	conn := openTestDB(t)
	sid := seedSubtask(t, conn)

	mon := time.Date(2026, 7, 27, 0, 0, 0, 0, time.Local)  // понедельник
	now := time.Date(2026, 7, 29, 15, 0, 0, 0, time.Local) // среда

	addClosed := func(start, end time.Time) {
		exec(t, conn, "INSERT INTO time_entries (subtask_id, started_at, ended_at) VALUES (?, ?, ?)",
			sid, start.Unix(), end.Unix())
	}
	addClosed(mon.Add(10*time.Hour), mon.Add(11*time.Hour))                     // 3600с в середине недели
	addClosed(mon.AddDate(0, 0, -1).Add(23*time.Hour), mon.Add(30*time.Minute)) // пересекает начало недели: 1800с
	addClosed(mon.AddDate(0, 0, -2), mon.AddDate(0, 0, -1))                     // целиком до недели

	exec(t, conn, "INSERT INTO time_entries (subtask_id, started_at) VALUES (?, ?)",
		sid, mon.AddDate(0, 0, -1).Add(23*time.Hour).Unix()) // активная с воскресенья 23:00

	want := 3600 + 1800 + int64(now.Sub(mon).Seconds())
	got, err := WeeklyTotal(conn, now)
	if err != nil {
		t.Fatalf("WeeklyTotal: %v", err)
	}
	if int64(got.Seconds()) != want {
		t.Errorf("WeeklyTotal = %ds, ожидалось %ds", int64(got.Seconds()), want)
	}
}
