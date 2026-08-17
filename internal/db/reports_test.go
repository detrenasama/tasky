package db

import (
	"database/sql"
	"testing"
	"time"
)

// seedReportSubtask создаёт проект/задачу/подзадачу и возвращает id подзадачи.
func seedReportSubtask(t *testing.T, conn *sql.DB, projectName, taskTitle string, taskCreated time.Time, sortOrder int, subTitle string) int64 {
	t.Helper()
	seedSeq++
	exec(t, conn, "INSERT INTO projects (name, created_at) VALUES (?, ?)",
		projectName, taskCreated.Unix())
	var pid int64
	if err := conn.QueryRow("SELECT id FROM projects ORDER BY id DESC LIMIT 1").Scan(&pid); err != nil {
		t.Fatal(err)
	}
	exec(t, conn, "INSERT INTO tasks (project_id, title, created_at) VALUES (?, ?, ?)",
		pid, taskTitle, taskCreated.Unix())
	var tid int64
	if err := conn.QueryRow("SELECT id FROM tasks ORDER BY id DESC LIMIT 1").Scan(&tid); err != nil {
		t.Fatal(err)
	}
	exec(t, conn, "INSERT INTO subtasks (task_id, title, sort_order, created_at) VALUES (?, ?, ?, ?)",
		tid, subTitle, sortOrder, taskCreated.Unix())
	var sid int64
	if err := conn.QueryRow("SELECT id FROM subtasks ORDER BY id DESC LIMIT 1").Scan(&sid); err != nil {
		t.Fatal(err)
	}
	return sid
}

func addClosedEntry(t *testing.T, conn *sql.DB, sid int64, start, end time.Time) {
	t.Helper()
	exec(t, conn, "INSERT INTO time_entries (subtask_id, started_at, ended_at) VALUES (?, ?, ?)",
		sid, start.Unix(), end.Unix())
}

func TestReportEntriesToday(t *testing.T) {
	conn := openTestDB(t)
	day := time.Date(2026, 7, 30, 0, 0, 0, 0, time.Local)

	s1 := seedReportSubtask(t, conn, "p1", "T1", day.AddDate(0, 0, -1), 0, "S1")
	s2 := seedReportSubtask(t, conn, "p1b", "T1", day.AddDate(0, 0, -1), 1, "S2")
	s3 := seedReportSubtask(t, conn, "p1c", "T1", day.AddDate(0, 0, -1), 2, "S3")
	s4 := seedReportSubtask(t, conn, "p1d", "T2", day.AddDate(0, 0, -1).Add(1*time.Hour), 0, "S4")
	s5 := seedReportSubtask(t, conn, "p2", "T3", day.AddDate(0, 0, -1).Add(2*time.Hour), 0, "S5")

	addClosedEntry(t, conn, s1, day.Add(10*time.Hour), day.Add(11*time.Hour))                                     // 3600с
	addClosedEntry(t, conn, s1, day.Add(13*time.Hour), day.Add(14*time.Hour+30*time.Minute))                      // 5400с → 9000с
	addClosedEntry(t, conn, s3, day.Add(9*time.Hour), day.Add(9*time.Hour))                                       // 0с — пропустить
	addClosedEntry(t, conn, s4, day.AddDate(0, 0, -1).Add(23*time.Hour), day.Add(30*time.Minute))                 // пересекает полночь: 1800с
	addClosedEntry(t, conn, s5, day.AddDate(0, 0, -1).Add(10*time.Hour), day.AddDate(0, 0, -1).Add(11*time.Hour)) // целиком вчера
	exec(t, conn, "INSERT INTO time_entries (subtask_id, started_at) VALUES (?, ?)",
		s2, day.Add(14*time.Hour).Unix()) // активная сессия — не учитывать

	entries, err := ReportEntries(conn, day, day.AddDate(0, 0, 1), 0)
	if err != nil {
		t.Fatalf("ReportEntries: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("записей: %d, ожидалось 2 (S1, S4; S5 целиком вчера)", len(entries))
	}
	got := []int64{entries[0].SubtaskID, entries[1].SubtaskID}
	want := []int64{s1, s4}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("порядок: entries[%d].SubtaskID = %d, ожидалось %d", i, got[i], want[i])
		}
	}
	if entries[0].Seconds != 9000 {
		t.Errorf("S1 Seconds = %d, ожидалось 9000", entries[0].Seconds)
	}
	if entries[1].Seconds != 1800 {
		t.Errorf("S4 Seconds = %d, ожидалось 1800 (обрезано по полночи)", entries[1].Seconds)
	}
}

func TestReportEntriesProjectFilter(t *testing.T) {
	conn := openTestDB(t)
	day := time.Date(2026, 7, 30, 0, 0, 0, 0, time.Local)

	seedReportSubtask(t, conn, "p1", "T1", day.AddDate(0, 0, -1), 0, "S1")
	seedReportSubtask(t, conn, "p1b", "T2", day.AddDate(0, 0, -1), 0, "S2")
	s3 := seedReportSubtask(t, conn, "p2", "T3", day.AddDate(0, 0, -1), 0, "S3")
	addClosedEntry(t, conn, s3, day.Add(10*time.Hour), day.Add(11*time.Hour))
	var pid int64
	if err := conn.QueryRow("SELECT id FROM projects WHERE name = 'p2'").Scan(&pid); err != nil {
		t.Fatal(err)
	}

	entries, err := ReportEntries(conn, day, day.AddDate(0, 0, 1), pid)
	if err != nil {
		t.Fatalf("ReportEntries: %v", err)
	}
	if len(entries) != 1 || entries[0].SubtaskID != s3 {
		t.Errorf("по фильтру проекта ожидалась 1 запись S3, получено %+v", entries)
	}

	entries, err = ReportEntries(conn, day, day.AddDate(0, 0, 1), pid+1)
	if err != nil {
		t.Fatalf("ReportEntries: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("по несуществующему проекту записей: %d, ожидалось 0", len(entries))
	}
}

func TestReportEntriesEmpty(t *testing.T) {
	conn := openTestDB(t)
	day := time.Date(2026, 7, 30, 0, 0, 0, 0, time.Local)
	entries, err := ReportEntries(conn, day, day.AddDate(0, 0, 1), 0)
	if err != nil {
		t.Fatalf("ReportEntries: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("записей: %d, ожидалось 0", len(entries))
	}
}

func TestReportByTask(t *testing.T) {
	entries := []ReportEntry{
		{TaskID: 1, TaskTitle: "T1", ProjectID: 7, ProjectName: "p", SubtaskID: 10, SubtaskTitle: "S1", Seconds: 60},
		{TaskID: 1, TaskTitle: "T1", ProjectID: 7, ProjectName: "p", SubtaskID: 11, SubtaskTitle: "S2", Seconds: 120},
		{TaskID: 2, TaskTitle: "T2", ProjectID: 7, ProjectName: "p", SubtaskID: 12, SubtaskTitle: "S3", Seconds: 30},
	}
	rep := ReportByTask(entries)
	if len(rep) != 2 {
		t.Fatalf("задач: %d, ожидалось 2", len(rep))
	}
	if rep[0].TaskID != 1 || rep[0].Seconds != 180 || len(rep[0].Subs) != 2 {
		t.Errorf("T1: %+v, ожидалось 180с и 2 подзадачи", rep[0])
	}
	if rep[0].Subs[0].Seconds != 60 || rep[0].Subs[1].Seconds != 120 {
		t.Errorf("порядок подзадач T1 нарушен: %+v", rep[0].Subs)
	}
	if rep[1].TaskID != 2 || rep[1].Seconds != 30 || len(rep[1].Subs) != 1 {
		t.Errorf("T2: %+v", rep[1])
	}
	if rep[0].ProjectID != 7 || rep[0].ProjectName != "p" {
		t.Errorf("поля проекта: %+v", rep[0])
	}
	if n := len(ReportByTask(nil)); n != 0 {
		t.Errorf("ReportByTask(nil): задач %d, ожидалось 0", n)
	}
}

func TestJournalEntriesByRange(t *testing.T) {
	conn := openTestDB(t)
	day := time.Date(2026, 7, 30, 0, 0, 0, 0, time.Local)
	sid := seedSubtask(t, conn)

	addJournal := func(ts time.Time, text string) {
		exec(t, conn, "INSERT INTO journal_entries (subtask_id, created_at, text) VALUES (?, ?, ?)",
			sid, ts.Unix(), text)
	}
	addJournal(day.Add(9*time.Hour), "первая")                   // внутри
	addJournal(day.Add(10*time.Hour), "вторая")                  // внутри
	addJournal(day.AddDate(0, 0, -1).Add(23*time.Hour), "вчера") // до
	addJournal(day.AddDate(0, 0, 1), "завтра")                   // на границе to — вне

	entries, err := JournalEntriesByRange(conn, day, day.AddDate(0, 0, 1))
	if err != nil {
		t.Fatalf("JournalEntriesByRange: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("записей: %d, ожидалось 2", len(entries))
	}
	if entries[0].Text != "первая" || entries[1].Text != "вторая" {
		t.Errorf("порядок нарушен: %+v", entries)
	}
	if entries[0].SubtaskID != sid || !entries[0].CreatedAt.Equal(day.Add(9*time.Hour)) {
		t.Errorf("поля записи: %+v", entries[0])
	}

	empty, err := JournalEntriesByRange(conn, day.AddDate(0, 0, 10), day.AddDate(0, 0, 11))
	if err != nil {
		t.Fatalf("JournalEntriesByRange: %v", err)
	}
	if len(empty) != 0 {
		t.Errorf("записей в пустом периоде: %d", len(empty))
	}
}

// TestReportEntriesSortOrder — задачи в отчёте следуют ручной сортировке
// (sort_order), а не порядку создания.
func TestReportEntriesSortOrder(t *testing.T) {
	conn := openTestDB(t)
	day := time.Date(2026, 7, 30, 0, 0, 0, 0, time.Local)

	exec(t, conn, "INSERT INTO projects (name, created_at) VALUES ('p', ?)", day.AddDate(0, 0, -1).Unix())
	var pid int64
	if err := conn.QueryRow("SELECT id FROM projects").Scan(&pid); err != nil {
		t.Fatal(err)
	}
	// задача A создана раньше, но sort_order 2 (перемещена вниз)
	exec(t, conn, "INSERT INTO tasks (project_id, title, sort_order, created_at) VALUES (?, 'A', 2, ?)",
		pid, day.AddDate(0, 0, -1).Unix())
	var tidA int64
	if err := conn.QueryRow("SELECT id FROM tasks WHERE title = 'A'").Scan(&tidA); err != nil {
		t.Fatal(err)
	}
	// задача B создана позже, sort_order 1 — в отчёте первой
	exec(t, conn, "INSERT INTO tasks (project_id, title, sort_order, created_at) VALUES (?, 'B', 1, ?)",
		pid, day.AddDate(0, 0, -1).Add(2*time.Hour).Unix())
	var tidB int64
	if err := conn.QueryRow("SELECT id FROM tasks WHERE title = 'B'").Scan(&tidB); err != nil {
		t.Fatal(err)
	}
	// подзадачи B: созданная второй — первая по sort_order
	exec(t, conn, "INSERT INTO subtasks (task_id, title, sort_order, created_at) VALUES (?, 'B2', 2, ?)",
		tidB, day.AddDate(0, 0, -1).Unix())
	var sidB2 int64
	if err := conn.QueryRow("SELECT id FROM subtasks WHERE title = 'B2'").Scan(&sidB2); err != nil {
		t.Fatal(err)
	}
	exec(t, conn, "INSERT INTO subtasks (task_id, title, sort_order, created_at) VALUES (?, 'B1', 1, ?)",
		tidB, day.AddDate(0, 0, -1).Unix())
	var sidB1 int64
	if err := conn.QueryRow("SELECT id FROM subtasks WHERE title = 'B1'").Scan(&sidB1); err != nil {
		t.Fatal(err)
	}
	addClosedEntry(t, conn, sidB1, day.Add(10*time.Hour), day.Add(11*time.Hour))
	addClosedEntry(t, conn, sidB2, day.Add(12*time.Hour), day.Add(13*time.Hour))

	entries, err := ReportEntries(conn, day, day.AddDate(0, 0, 1), 0)
	if err != nil {
		t.Fatalf("ReportEntries: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("записей: %d, ожидалось 2", len(entries))
	}
	if entries[0].TaskID != tidB || entries[0].SubtaskID != sidB1 {
		t.Errorf("первая запись = task %d sub %d, ожидалась B/B1", entries[0].TaskID, entries[0].SubtaskID)
	}
	if entries[1].TaskID != tidB || entries[1].SubtaskID != sidB2 {
		t.Errorf("вторая запись = task %d sub %d, ожидалась B/B2", entries[1].TaskID, entries[1].SubtaskID)
	}
}
