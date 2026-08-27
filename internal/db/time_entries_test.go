package db

import (
	"testing"
	"time"
)

func ptrTime(t time.Time) *time.Time { return &t }

func TestUpdateTimeEntry(t *testing.T) {
	conn := openTestDB(t)
	sid := seedSubtask(t, conn)
	start := time.Unix(1_700_000_000, 0)
	end := start.Add(time.Hour)
	exec(t, conn, "INSERT INTO time_entries (subtask_id, started_at, ended_at) VALUES (?, ?, ?)",
		sid, start.Unix(), end.Unix())
	var eid int64
	if err := conn.QueryRow("SELECT id FROM time_entries").Scan(&eid); err != nil {
		t.Fatal(err)
	}

	newStart := time.Unix(1_700_000_100, 0)
	newEnd := newStart.Add(2 * time.Hour)
	if err := UpdateTimeEntry(conn, eid, newStart, &newEnd); err != nil {
		t.Fatalf("UpdateTimeEntry: %v", err)
	}
	entries, _ := TimeEntriesBySubtask(conn, sid)
	if len(entries) != 1 || !entries[0].StartedAt.Equal(newStart) || !entries[0].EndedAt.Equal(newEnd) {
		t.Fatal("границы записи не обновились")
	}

	// nil endedAt оставляет запись активной
	if err := UpdateTimeEntry(conn, eid, newStart, nil); err != nil {
		t.Fatalf("UpdateTimeEntry(nil end): %v", err)
	}
	entries, _ = TimeEntriesBySubtask(conn, sid)
	if entries[0].EndedAt != nil {
		t.Error("при nil endedAt запись должна стать активной")
	}
}

func TestDeleteTimeEntry(t *testing.T) {
	conn := openTestDB(t)
	sid := seedSubtask(t, conn)
	exec(t, conn, "INSERT INTO time_entries (subtask_id, started_at, ended_at) VALUES (?, ?, ?)",
		sid, 100, 200)
	var eid int64
	if err := conn.QueryRow("SELECT id FROM time_entries").Scan(&eid); err != nil {
		t.Fatal(err)
	}
	if err := DeleteTimeEntry(conn, eid); err != nil {
		t.Fatalf("DeleteTimeEntry: %v", err)
	}
	entries, _ := TimeEntriesBySubtask(conn, sid)
	if len(entries) != 0 {
		t.Errorf("после удаления записей: %d, ожидалось 0", len(entries))
	}
}

func TestTimeEntriesInRange(t *testing.T) {
	conn := openTestDB(t)
	sidA := seedSubtask(t, conn)
	sidB := seedSubtask(t, conn)

	inStart := time.Date(2026, 7, 29, 10, 0, 0, 0, time.Local)
	exec(t, conn, "INSERT INTO time_entries (subtask_id, started_at, ended_at) VALUES (?, ?, ?)",
		sidA, inStart.Unix(), inStart.Add(time.Hour).Unix())
	outStart := time.Date(2020, 1, 1, 0, 0, 0, 0, time.Local)
	exec(t, conn, "INSERT INTO time_entries (subtask_id, started_at, ended_at) VALUES (?, ?, ?)",
		sidB, outStart.Unix(), outStart.Add(time.Hour).Unix())

	from := time.Date(2026, 7, 29, 0, 0, 0, 0, time.Local)
	to := from.Add(24 * time.Hour)
	all, err := TimeEntriesInRange(conn, from, to, 0)
	if err != nil {
		t.Fatalf("TimeEntriesInRange: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("записей в диапазоне: %d, ожидалось 1", len(all))
	}
	if all[0].SubtaskID != sidA {
		t.Error("ожидалась запись подзадачи A")
	}
	if all[0].SubtaskTitle == "" || all[0].TaskTitle == "" || all[0].ProjectName == "" {
		t.Error("названия подзадачи/задачи/проекта должны быть заполнены")
	}
}

func TestDetectOverlaps(t *testing.T) {
	a := time.Unix(1_000_000, 0)
	b := a.Add(30 * time.Minute)
	c := a.Add(10 * time.Minute) // пересекает a
	d := a.Add(40 * time.Minute) // не пересекает a и c
	entries := []TimeEntryInfo{
		{ID: 1, StartedAt: a, EndedAt: &b},
		{ID: 2, StartedAt: c, EndedAt: ptrTime(c.Add(20 * time.Minute))},
		{ID: 3, StartedAt: d, EndedAt: ptrTime(d.Add(20 * time.Minute))},
	}
	pairs := DetectOverlaps(entries)
	if len(pairs) != 1 {
		t.Fatalf("пар пересечений: %d, ожидалось 1", len(pairs))
	}
	if pairs[0] != [2]int{0, 1} {
		t.Errorf("пара = %v, ожидалось [0,1]", pairs[0])
	}

	// активная (EndedAt nil) не даёт пересечения
	entries2 := []TimeEntryInfo{
		{ID: 1, StartedAt: a, EndedAt: &b},
		{ID: 2, StartedAt: a, EndedAt: nil},
	}
	if len(DetectOverlaps(entries2)) != 0 {
		t.Error("активная запись не должна давать пересечение")
	}

	// граничащие (end_i == start_j) не пересекаются
	e1 := a
	e2 := a // end == start -> не пересекается
	entries3 := []TimeEntryInfo{
		{ID: 1, StartedAt: a, EndedAt: &e1},
		{ID: 2, StartedAt: e2, EndedAt: ptrTime(e2.Add(time.Hour))},
	}
	if len(DetectOverlaps(entries3)) != 0 {
		t.Error("граничащие записи не должны пересекаться")
	}
}
