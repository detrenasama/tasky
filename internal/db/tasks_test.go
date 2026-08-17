package db

import (
	"database/sql"
	"strings"
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
	if task.ID == 0 || task.Status != "Новая" {
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

func TestSubtasksByProject(t *testing.T) {
	conn := openTestDB(t)
	pid := seedProject(t, conn)
	t1, err := CreateTask(conn, pid, "t1")
	if err != nil {
		t.Fatal(err)
	}
	t2, err := CreateTask(conn, pid, "t2")
	if err != nil {
		t.Fatal(err)
	}
	for _, title := range []string{"a1", "a2"} {
		if _, err := CreateSubtask(conn, t1.ID, title); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := CreateSubtask(conn, t2.ID, "b1"); err != nil {
		t.Fatal(err)
	}

	subs, err := SubtasksByProject(conn, pid)
	if err != nil {
		t.Fatal(err)
	}
	if len(subs) != 3 {
		t.Fatalf("подзадач: %d, ожидалось 3", len(subs))
	}
	want := []struct {
		taskID int64
		title  string
	}{
		{t1.ID, "a1"}, {t1.ID, "a2"}, {t2.ID, "b1"},
	}
	for i, w := range want {
		if subs[i].TaskID != w.taskID || subs[i].Title != w.title {
			t.Errorf("subs[%d] = task %d %q, ожидалось task %d %q",
				i, subs[i].TaskID, subs[i].Title, w.taskID, w.title)
		}
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

func TestTaskSubtaskDescription(t *testing.T) {
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

	if err := UpdateTaskDescription(conn, task.ID, "описание задачи"); err != nil {
		t.Fatalf("UpdateTaskDescription: %v", err)
	}
	got, err := TaskDescription(conn, task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got != "описание задачи" {
		t.Errorf("TaskDescription = %q", got)
	}

	if err := UpdateSubtaskDescription(conn, sub.ID, "описание подзадачи"); err != nil {
		t.Fatalf("UpdateSubtaskDescription: %v", err)
	}
	got, err = SubtaskDescription(conn, sub.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got != "описание подзадачи" {
		t.Errorf("SubtaskDescription = %q", got)
	}

	// по умолчанию описания пустые
	t2, err := CreateTask(conn, pid, "t2")
	if err != nil {
		t.Fatal(err)
	}
	if d, _ := TaskDescription(conn, t2.ID); d != "" {
		t.Errorf("новое описание задачи = %q, ожидалось пустое", d)
	}
}

func TestTaskSubtaskTitleUpdate(t *testing.T) {
	conn := openTestDB(t)
	pid := seedProject(t, conn)
	task, err := CreateTask(conn, pid, "Задача")
	if err != nil {
		t.Fatal(err)
	}
	sub, err := CreateSubtask(conn, task.ID, "Подзадача")
	if err != nil {
		t.Fatal(err)
	}

	if err := UpdateTaskTitle(conn, task.ID, "Задача (изм.)"); err != nil {
		t.Fatalf("UpdateTaskTitle: %v", err)
	}
	if err := UpdateSubtaskTitle(conn, sub.ID, "Подзадача (изм.)"); err != nil {
		t.Fatalf("UpdateSubtaskTitle: %v", err)
	}

	tasks, err := TasksByProject(conn, pid)
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 1 || tasks[0].Title != "Задача (изм.)" {
		t.Errorf("TasksByProject: %+v", tasks)
	}
	subs, err := SubtasksByProject(conn, pid)
	if err != nil {
		t.Fatal(err)
	}
	if len(subs) != 1 || subs[0].Title != "Подзадача (изм.)" {
		t.Errorf("SubtasksByProject: %+v", subs)
	}
}

// TestProjectQueriesIncludeDescription — описания задач и подзадач приходят
// из запросов по проекту (нужно для поиска).
func TestProjectQueriesIncludeDescription(t *testing.T) {
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
	if err := UpdateTaskDescription(conn, task.ID, "desc-t"); err != nil {
		t.Fatal(err)
	}
	if err := UpdateSubtaskDescription(conn, sub.ID, "desc-s"); err != nil {
		t.Fatal(err)
	}

	tasks, err := TasksByProject(conn, pid)
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 1 || tasks[0].Description != "desc-t" {
		t.Errorf("TasksByProject: %+v", tasks)
	}
	subs, err := SubtasksByProject(conn, pid)
	if err != nil {
		t.Fatal(err)
	}
	if len(subs) != 1 || subs[0].Description != "desc-s" {
		t.Errorf("SubtasksByProject: %+v", subs)
	}
	byTask, err := SubtasksWithTime(conn, task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(byTask) != 1 || byTask[0].Description != "desc-s" {
		t.Errorf("SubtasksWithTime: %+v", byTask)
	}
}

// TestJournalTexts — карта подзадача → объединённый текст записей журнала.
func TestJournalTexts(t *testing.T) {
	conn := openTestDB(t)
	pid := seedProject(t, conn)
	t1, err := CreateTask(conn, pid, "t1")
	if err != nil {
		t.Fatal(err)
	}
	t2, err := CreateTask(conn, pid, "t2")
	if err != nil {
		t.Fatal(err)
	}
	st1, err := CreateSubtask(conn, t1.ID, "s1")
	if err != nil {
		t.Fatal(err)
	}
	st2, err := CreateSubtask(conn, t1.ID, "s2")
	if err != nil {
		t.Fatal(err)
	}
	st3, err := CreateSubtask(conn, t2.ID, "s3")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := CreateJournalEntry(conn, st1.ID, "первая"); err != nil {
		t.Fatal(err)
	}
	if _, err := CreateJournalEntry(conn, st1.ID, "вторая"); err != nil {
		t.Fatal(err)
	}
	if _, err := CreateJournalEntry(conn, st3.ID, "третья"); err != nil {
		t.Fatal(err)
	}

	texts, err := JournalTexts(conn, pid)
	if err != nil {
		t.Fatal(err)
	}
	if len(texts) != 2 {
		t.Fatalf("подзадач с журналом: %d, ожидалось 2", len(texts))
	}
	joined := texts[st1.ID]
	if !strings.Contains(joined, "первая") || !strings.Contains(joined, "вторая") {
		t.Errorf("журнал s1 = %q", joined)
	}
	if texts[st2.ID] != "" {
		t.Errorf("у s2 не должно быть журнала: %q", texts[st2.ID])
	}
	if texts[st3.ID] != "третья" {
		t.Errorf("журнал s3 = %q", texts[st3.ID])
	}
}

func TestTaskLinks(t *testing.T) {
	conn := openTestDB(t)
	pid := seedProject(t, conn)
	task, err := CreateTask(conn, pid, "t")
	if err != nil {
		t.Fatal(err)
	}
	l1, err := CreateTaskLink(conn, task.ID, "Доки", "https://example.com")
	if err != nil {
		t.Fatalf("CreateTaskLink: %v", err)
	}
	if l1.OwnerID != task.ID || l1.Name != "Доки" || l1.ID == 0 {
		t.Errorf("ссылка = %+v", l1)
	}
	if _, err := CreateTaskLink(conn, task.ID, "", "https://example.org"); err != nil {
		t.Fatal(err)
	}
	links, err := TaskLinks(conn, task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(links) != 2 || links[1].URL != "https://example.org" {
		t.Errorf("ссылки задачи = %+v", links)
	}
	if err := DeleteTaskLink(conn, l1.ID); err != nil {
		t.Fatalf("DeleteTaskLink: %v", err)
	}
	links, _ = TaskLinks(conn, task.ID)
	if len(links) != 1 || links[0].ID != l1.ID+1 {
		t.Errorf("после удаления ссылки = %+v", links)
	}
}

func TestSubtaskLinks(t *testing.T) {
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
	l, err := CreateSubtaskLink(conn, sub.ID, "Доки", "https://example.com")
	if err != nil {
		t.Fatalf("CreateSubtaskLink: %v", err)
	}
	if l.OwnerID != sub.ID {
		t.Errorf("владелец ссылки = %d, ожидался %d", l.OwnerID, sub.ID)
	}
	links, err := SubtaskLinks(conn, sub.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(links) != 1 || links[0].URL != "https://example.com" {
		t.Errorf("ссылки подзадачи = %+v", links)
	}
}

func TestJournalEntries(t *testing.T) {
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
	if _, err := CreateJournalEntry(conn, sub.ID, "первая"); err != nil {
		t.Fatal(err)
	}
	time.Sleep(2 * time.Millisecond)
	e2, err := CreateJournalEntry(conn, sub.ID, "вторая")
	if err != nil {
		t.Fatal(err)
	}
	entries, err := JournalEntries(conn, sub.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("записей: %d, ожидалось 2", len(entries))
	}
	// хронологический порядок: первая сверху
	if entries[0].Text != "первая" || entries[1].Text != "вторая" {
		t.Errorf("порядок записей = %+v", entries)
	}

	if err := UpdateJournalEntry(conn, e2.ID, "вторая (изм.)"); err != nil {
		t.Fatalf("UpdateJournalEntry: %v", err)
	}
	entries, _ = JournalEntries(conn, sub.ID)
	if entries[1].Text != "вторая (изм.)" {
		t.Errorf("после обновления = %+v", entries[1])
	}

	// удаление подзадачи каскадно убирает записи
	if err := DeleteSubtask(conn, sub.ID); err != nil {
		t.Fatal(err)
	}
	entries, _ = JournalEntries(conn, sub.ID)
	if len(entries) != 0 {
		t.Errorf("записей после удаления подзадачи: %d", len(entries))
	}
}
