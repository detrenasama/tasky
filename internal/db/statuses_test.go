package db

import (
	"database/sql"
	"errors"
	"testing"
	"time"
)

// openLegacyDB создаёт БД со старой схемой (status с CHECK), добавляет
// данные и открывает через db.Open — проверяем миграцию.
func openLegacyDB(t *testing.T) *sql.DB {
	t.Helper()
	path := t.TempDir() + "/legacy.db"
	conn, err := sql.Open("sqlite", path+"?_pragma=foreign_keys(1)")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	legacy := `
CREATE TABLE projects (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    name        TEXT    NOT NULL UNIQUE,
    description TEXT    NOT NULL DEFAULT '',
    created_at  INTEGER NOT NULL
);
CREATE TABLE tasks (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id   INTEGER NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    title        TEXT    NOT NULL,
    description  TEXT    NOT NULL DEFAULT '',
    status       TEXT    NOT NULL DEFAULT 'todo'
                 CHECK (status IN ('todo', 'in_progress', 'done')),
    created_at   INTEGER NOT NULL,
    completed_at INTEGER
);
CREATE TABLE subtasks (
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
CREATE TABLE time_entries (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    subtask_id INTEGER NOT NULL REFERENCES subtasks(id) ON DELETE CASCADE,
    started_at INTEGER NOT NULL,
    ended_at   INTEGER,
    note       TEXT    NOT NULL DEFAULT ''
);
`
	if _, err := conn.Exec(legacy); err != nil {
		t.Fatal(err)
	}
	now := time.Now().Unix()
	if _, err := conn.Exec(
		"INSERT INTO projects (id, name, created_at) VALUES (1, 'p', ?)", now); err != nil {
		t.Fatal(err)
	}
	for _, st := range []string{"todo", "in_progress", "done"} {
		if _, err := conn.Exec(
			"INSERT INTO tasks (project_id, title, status, created_at, completed_at) VALUES (1, ?, ?, ?, ?)",
			"t"+st, st, now, now); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := conn.Exec(
		"INSERT INTO subtasks (task_id, title, status, created_at) VALUES (1, 's1', 'in_progress', ?)",
		now); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Exec(
		"INSERT INTO time_entries (subtask_id, started_at, ended_at) VALUES (1, ?, ?)",
		now, now); err != nil {
		t.Fatal(err)
	}
	return open(t, path)
}

// open открывает БД через Open и регистрирует закрытие.
func open(t *testing.T, path string) *sql.DB {
	t.Helper()
	conn, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	return conn
}

func TestMigrateLegacyStatuses(t *testing.T) {
	conn := openLegacyDB(t)
	defer conn.Close()

	// CHECK снят: можно вставить произвольный статус.
	if _, err := conn.Exec(
		"INSERT INTO tasks (project_id, title, status, created_at) VALUES (1, 't', 'Произвольный', ?)",
		time.Now().Unix()); err != nil {
		t.Fatalf("CHECK не снят: %v", err)
	}
	// старые значения замапплены на статусы по умолчанию
	for _, want := range []string{"Новая", "В работе", "Выполнена"} {
		var n int
		if err := conn.QueryRow(
			"SELECT COUNT(*) FROM tasks WHERE status = ?", want).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n != 1 {
			t.Errorf("status %q: найдено %d", want, n)
		}
	}
	// каталог статусов засеян
	sts, err := Statuses(conn)
	if err != nil {
		t.Fatal(err)
	}
	if len(sts) != 8 || sts[0].Name != "Новая" || !sts[0].IsQuick {
		t.Errorf("сид статусов: %d элементов, первый %q", len(sts), sts[0].Name)
	}
}

// TestMigrateLegacyFKS — после миграции legacy-БД внешние ключи ссылающихся
// таблиц должны указывать на актуальные tasks/subtasks (регрессия: старая
// миграция переписывала их на удалённые tasks_old/subtasks_old, и запись
// переходов статусов падала на FK-ограничении).
func TestMigrateLegacyFKS(t *testing.T) {
	conn := openLegacyDB(t)
	defer conn.Close()
	now := time.Now()

	// смена статуса задачи пишет status_history
	if err := SetStatus(conn, OwnerTask, 1, "В работе", "", now); err != nil {
		t.Fatalf("SetStatus задачи: %v", err)
	}
	// смена статуса подзадачи пишет журнал
	if err := SetStatus(conn, OwnerSubtask, 1, "Выполнена", "", now); err != nil {
		t.Fatalf("SetStatus подзадачи: %v", err)
	}
	// записи журнала и времени по-прежнему создаются
	if _, err := CreateJournalEntry(conn, 1, "запись"); err != nil {
		t.Fatalf("CreateJournalEntry: %v", err)
	}
	if _, err := conn.Exec(
		"INSERT INTO time_entries (subtask_id, started_at) VALUES (1, ?)", now.Unix()); err != nil {
		t.Fatalf("time_entries: %v", err)
	}
	if _, err := CreateTaskLink(conn, 1, "l", "https://example.com"); err != nil {
		t.Fatalf("CreateTaskLink: %v", err)
	}
	// данные не потеряны
	var n int
	if err := conn.QueryRow("SELECT COUNT(*) FROM time_entries").Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Errorf("time_entries: %d записей, ждали 2", n)
	}
}

// TestRepairDanglingRefs — БД, уже сломанная старой миграцией (FK указывают
// на tasks_old/subtasks_old), чинится при открытии.
func TestRepairDanglingRefs(t *testing.T) {
	path := t.TempDir() + "/broken.db"
	// сид без foreign_keys: данные пишутся в таблицы со ссылками на
	// несуществующие *_old (как это было в реально сломанных БД)
	conn, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	broken := `
CREATE TABLE projects (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    name        TEXT    NOT NULL UNIQUE,
    description TEXT    NOT NULL DEFAULT '',
    created_at  INTEGER NOT NULL
);
CREATE TABLE tasks (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    project_id   INTEGER NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    title        TEXT    NOT NULL,
    description  TEXT    NOT NULL DEFAULT '',
    status       TEXT    NOT NULL DEFAULT '',
    created_at   INTEGER NOT NULL,
    completed_at INTEGER
);
CREATE TABLE subtasks (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    task_id      INTEGER NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    title        TEXT    NOT NULL,
    description  TEXT    NOT NULL DEFAULT '',
    status       TEXT    NOT NULL DEFAULT '',
    sort_order   INTEGER NOT NULL DEFAULT 0,
    created_at   INTEGER NOT NULL,
    completed_at INTEGER
);
CREATE TABLE status_history (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    task_id     INTEGER REFERENCES "tasks_old"(id) ON DELETE CASCADE,
    subtask_id  INTEGER REFERENCES "subtasks_old"(id) ON DELETE CASCADE,
    from_status TEXT    NOT NULL DEFAULT '',
    to_status   TEXT    NOT NULL,
    note        TEXT    NOT NULL DEFAULT '',
    created_at  INTEGER NOT NULL,
    CHECK ((task_id IS NULL) != (subtask_id IS NULL))
);
CREATE TABLE journal_entries (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    subtask_id INTEGER NOT NULL REFERENCES "subtasks_old"(id) ON DELETE CASCADE,
    created_at INTEGER NOT NULL,
    text       TEXT    NOT NULL
);
`
	if _, err := conn.Exec(broken); err != nil {
		t.Fatal(err)
	}
	now := time.Now().Unix()
	if _, err := conn.Exec(
		"INSERT INTO projects (id, name, created_at) VALUES (1, 'p', ?)", now); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Exec(
		"INSERT INTO tasks (id, project_id, title, status, created_at) VALUES (1, 1, 't', 'Новая', ?)",
		now); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Exec(
		"INSERT INTO subtasks (id, task_id, title, status, created_at) VALUES (1, 1, 's', 'Новая', ?)",
		now); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Exec(
		"INSERT INTO status_history (task_id, from_status, to_status, created_at) VALUES (1, 'x', 'y', ?)",
		now); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Exec(
		"INSERT INTO journal_entries (subtask_id, created_at, text) VALUES (1, ?, 'запись')", now); err != nil {
		t.Fatal(err)
	}
	conn.Close()

	// открытие через Open чинит FK-ссылки
	conn = open(t, path)
	defer conn.Close()
	var brokenLeft int
	if err := conn.QueryRow(`
SELECT COUNT(*) FROM sqlite_master
WHERE type = 'table' AND (sql LIKE '%tasks_old%' OR sql LIKE '%subtasks_old%')`).
		Scan(&brokenLeft); err != nil {
		t.Fatal(err)
	}
	if brokenLeft != 0 {
		t.Fatalf("остались таблицы со ссылками на *_old: %d", brokenLeft)
	}
	// история и журнал по-прежнему пишутся
	if err := SetStatus(conn, OwnerTask, 1, "В работе", "", time.Now()); err != nil {
		t.Fatalf("SetStatus после починки: %v", err)
	}
	if _, err := CreateJournalEntry(conn, 1, "новая запись"); err != nil {
		t.Fatalf("CreateJournalEntry после починки: %v", err)
	}
	// старые данные не потеряны
	var n int
	if err := conn.QueryRow(
		"SELECT COUNT(*) FROM status_history").Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Errorf("status_history: %d записей, ждали 2", n)
	}
}

func TestStatusesCRUD(t *testing.T) {
	conn := openTestDB(t)
	defer conn.Close()

	st, err := CreateStatus(conn, "Тест", "new", "#ff0000", "Заметка", true)
	if err != nil {
		t.Fatal(err)
	}
	sts, _ := Statuses(conn)
	if len(sts) != 9 || sts[8].Name != "Тест" || !sts[8].IsQuick ||
		sts[8].Type != "new" || sts[8].Color != "#ff0000" || sts[8].NotePrompt != "Заметка" {
		t.Errorf("CreateStatus: %+v", sts[8])
	}

	if err := UpdateStatus(conn, st.ID, "Тест 2", "done", "#00ff00", "", false); err != nil {
		t.Fatal(err)
	}
	sts, _ = Statuses(conn)
	if sts[8].Name != "Тест 2" || sts[8].Type != "done" || sts[8].Color != "#00ff00" ||
		sts[8].IsQuick || sts[8].NotePrompt != "" {
		t.Errorf("UpdateStatus: %+v", sts[8])
	}

	// удаление неиспользуемого статуса — ок
	if err := DeleteStatus(conn, st.ID); err != nil {
		t.Fatalf("DeleteStatus: %v", err)
	}
	// удаление используемого — ошибка
	pid := mustProject(t, conn, "p")
	if _, err := CreateTask(conn, pid, "t"); err != nil {
		t.Fatal(err)
	}
	if err := DeleteStatus(conn, 1); err == nil {
		t.Fatal("удаление используемого статуса не дало ошибку")
	} else if !errors.Is(err, ErrStatusInUse) {
		t.Fatalf("ожидался ErrStatusInUse, получено: %v", err)
	}
}

func TestSetStatusTask(t *testing.T) {
	conn := openTestDB(t)
	defer conn.Close()
	pid := mustProject(t, conn, "p")
	task, _ := CreateTask(conn, pid, "t")
	now := time.Date(2026, 8, 2, 12, 0, 0, 0, time.Local)

	if err := SetStatus(conn, OwnerTask, task.ID, "В работе", "", now); err != nil {
		t.Fatal(err)
	}
	if err := SetStatus(conn, OwnerTask, task.ID, "Выполнена", "", now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	// completed_at выставлен при входе в done
	var completed sql.NullInt64
	var status string
	if err := conn.QueryRow(
		"SELECT status, completed_at FROM tasks WHERE id = ?", task.ID).
		Scan(&status, &completed); err != nil {
		t.Fatal(err)
	}
	if status != "Выполнена" || !completed.Valid ||
		completed.Int64 != now.Add(time.Hour).Unix() {
		t.Errorf("задача: status=%q completed=%v", status, completed)
	}

	// выход из done очищает completed_at
	if err := SetStatus(conn, OwnerTask, task.ID, "В работе", "", now.Add(2*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := conn.QueryRow(
		"SELECT completed_at FROM tasks WHERE id = ?", task.ID).Scan(&completed); err != nil {
		t.Fatal(err)
	}
	if completed.Valid {
		t.Error("completed_at не очищен после выхода из done")
	}

	// история: 3 перехода
	hist, err := StatusHistory(conn, OwnerTask, task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(hist) != 3 || hist[0].From != "Новая" || hist[0].To != "В работе" ||
		hist[1].To != "Выполнена" || hist[2].To != "В работе" {
		t.Errorf("история: %+v", hist)
	}

	// переход в тот же статус — без записи
	if err := SetStatus(conn, OwnerTask, task.ID, "В работе", "", now); err != nil {
		t.Fatal(err)
	}
	hist, _ = StatusHistory(conn, OwnerTask, task.ID)
	if len(hist) != 3 {
		t.Errorf("повторный переход создал запись: %d", len(hist))
	}
}

// TestSetStatusSubtaskHistory — переход подзадачи пишется в status_history
// (как для задач), а не в журнал.
func TestSetStatusSubtaskHistory(t *testing.T) {
	conn := openTestDB(t)
	defer conn.Close()
	pid := mustProject(t, conn, "p")
	task, _ := CreateTask(conn, pid, "t")
	st, _ := CreateSubtask(conn, task.ID, "s")
	now := time.Date(2026, 8, 2, 12, 0, 0, 0, time.Local)

	if err := SetStatus(conn, OwnerSubtask, st.ID, "Делегирована", "Иван", now); err != nil {
		t.Fatal(err)
	}
	hist, err := StatusHistory(conn, OwnerSubtask, st.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(hist) != 1 {
		t.Fatalf("в истории %d записей", len(hist))
	}
	if hist[0].From != "Новая" || hist[0].To != "Делегирована" || hist[0].Note != "Иван" {
		t.Errorf("запись истории: %+v", hist[0])
	}
	if !hist[0].CreatedAt.Equal(now) {
		t.Errorf("время записи: %v", hist[0].CreatedAt)
	}
	// журнал не затрагивается
	entries, err := JournalEntries(conn, st.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("в журнале %d записей", len(entries))
	}
	// история задач не затрагивается
	if err := SetStatus(conn, OwnerTask, task.ID, "Выполнена", "", now); err != nil {
		t.Fatal(err)
	}
	if err := SetStatus(conn, OwnerTask, task.ID, "В работе", "", now); err != nil {
		t.Fatal(err)
	}
	if hist, _ = StatusHistory(conn, OwnerTask, task.ID); len(hist) != 2 {
		t.Errorf("история задачи: %d записей", len(hist))
	}
	if hist, _ = StatusHistory(conn, OwnerSubtask, st.ID); len(hist) != 1 {
		t.Errorf("история подзадачи затронута: %d", len(hist))
	}
}

// TestMigrateStatusJournal — старые записи «Статус: X → Y» из журнала
// подзадач переносятся в status_history, ручные записи остаются.
func TestMigrateStatusJournal(t *testing.T) {
	conn := openTestDB(t)
	defer conn.Close()
	pid := mustProject(t, conn, "p")
	task, _ := CreateTask(conn, pid, "t")
	st, _ := CreateSubtask(conn, task.ID, "s")
	now := time.Now().Unix()
	for i, text := range []string{
		"Статус: Новая → В работе",
		"Статус: В работе → Делегирована (Иван)",
		"ручная запись",
	} {
		if _, err := conn.Exec(
			"INSERT INTO journal_entries (subtask_id, created_at, text) VALUES (?, ?, ?)",
			st.ID, now+int64(i), text); err != nil {
			t.Fatal(err)
		}
	}
	if err := migrateStatusJournal(conn); err != nil {
		t.Fatal(err)
	}
	hist, _ := StatusHistory(conn, OwnerSubtask, st.ID)
	if len(hist) != 2 {
		t.Fatalf("status_history: %d записей", len(hist))
	}
	if hist[0].From != "Новая" || hist[0].To != "В работе" || hist[0].Note != "" {
		t.Errorf("первая запись: %+v", hist[0])
	}
	if hist[1].From != "В работе" || hist[1].To != "Делегирована" || hist[1].Note != "Иван" {
		t.Errorf("вторая запись: %+v", hist[1])
	}
	entries, _ := JournalEntries(conn, st.ID)
	if len(entries) != 1 || entries[0].Text != "ручная запись" {
		t.Errorf("журнал: %+v", entries)
	}
	// повторный прогон ничего не меняет
	if err := migrateStatusJournal(conn); err != nil {
		t.Fatal(err)
	}
	if hist, _ = StatusHistory(conn, OwnerSubtask, st.ID); len(hist) != 2 {
		t.Errorf("повторный прогон задвоил: %d", len(hist))
	}
}

func TestSetStatusUnknown(t *testing.T) {
	conn := openTestDB(t)
	defer conn.Close()
	pid := mustProject(t, conn, "p")
	task, _ := CreateTask(conn, pid, "t")
	if err := SetStatus(conn, OwnerTask, task.ID, "Несуществующий", "", time.Now()); err == nil {
		t.Fatal("неизвестный статус не дал ошибку")
	}
}

func mustProject(t *testing.T, conn *sql.DB, name string) int64 {
	t.Helper()
	p, err := CreateProject(conn, name)
	if err != nil {
		t.Fatal(err)
	}
	return p.ID
}
