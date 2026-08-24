package db

import (
	"database/sql"
	"time"
)

func CreateTask(conn *sql.DB, projectID int64, title string) (Task, error) {
	var t Task
	now := time.Now().Unix()
	status, _ := DefaultStatus(conn)
	res, err := conn.Exec(`
INSERT INTO tasks (project_id, title, status, sort_order, created_at)
SELECT ?, ?, ?, COALESCE(MAX(sort_order), 0) + 1, ? FROM tasks WHERE project_id = ?`,
		projectID, title, status, now, projectID)
	if err != nil {
		return t, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return t, err
	}
	t = Task{ID: id, ProjectID: projectID, Title: title, Status: status,
		CreatedAt: time.Unix(now, 0)}
	return t, nil
}

func DeleteTask(conn *sql.DB, id int64) error {
	_, err := conn.Exec("DELETE FROM tasks WHERE id = ?", id)
	return err
}

func CreateSubtask(conn *sql.DB, taskID int64, title string) (SubtaskWithTime, error) {
	var s SubtaskWithTime
	now := time.Now().Unix()
	status, _ := DefaultStatus(conn)
	res, err := conn.Exec(`
INSERT INTO subtasks (task_id, title, status, sort_order, created_at)
SELECT ?, ?, ?, COALESCE(MAX(sort_order), 0) + 1, ? FROM subtasks WHERE task_id = ?`,
		taskID, title, status, now, taskID)
	if err != nil {
		return s, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return s, err
	}
	s = SubtaskWithTime{ID: id, TaskID: taskID, Title: title, Status: status,
		CreatedAt: time.Unix(now, 0)}
	return s, nil
}

func DeleteSubtask(conn *sql.DB, id int64) error {
	_, err := conn.Exec("DELETE FROM subtasks WHERE id = ?", id)
	return err
}

// MoveTask перемещает задачу на одну позицию вверх (dir = -1) или вниз
// (dir = 1) в пределах её проекта. Порядок задаёт sort_order.
func MoveTask(conn *sql.DB, id int64, dir int) error {
	return moveOrderedRow(conn, "tasks", "project_id", id, dir)
}

// MoveSubtask перемещает подзадачу на одну позицию в пределах родительской
// задачи.
func MoveSubtask(conn *sql.DB, id int64, dir int) error {
	return moveOrderedRow(conn, "subtasks", "task_id", id, dir)
}

// moveOrderedRow двигает строку таблицы table (parentCol — колонка родителя)
// в списке соседей. Если движение выходит за границы — ничего не делает.
// Позиции соседей нормализуются к 1..N (устойчиво даже к legacy-нулям), затем
// сортируемые элементы обмениваются местами.
func moveOrderedRow(conn *sql.DB, table, parentCol string, id int64, dir int) error {
	var parentID int64
	if err := conn.QueryRow(
		"SELECT "+parentCol+" FROM "+table+" WHERE id = ?", id).Scan(&parentID); err != nil {
		return err
	}
	rows, err := conn.Query(
		"SELECT id FROM "+table+" WHERE "+parentCol+" = ? ORDER BY sort_order, id", parentID)
	if err != nil {
		return err
	}
	var ids []int64
	for rows.Next() {
		var x int64
		if err := rows.Scan(&x); err != nil {
			rows.Close()
			return err
		}
		ids = append(ids, x)
	}
	if err := rows.Close(); err != nil {
		return err
	}
	idx := -1
	for i, x := range ids {
		if x == id {
			idx = i
			break
		}
	}
	if idx < 0 {
		return nil
	}
	j := idx + dir
	if j < 0 || j >= len(ids) {
		return nil
	}
	tx, err := conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for i, x := range ids {
		if _, err := tx.Exec("UPDATE "+table+" SET sort_order = ? WHERE id = ?", i+1, x); err != nil {
			return err
		}
	}
	if _, err := tx.Exec("UPDATE "+table+" SET sort_order = ? WHERE id = ?", j+1, ids[idx]); err != nil {
		return err
	}
	if _, err := tx.Exec("UPDATE "+table+" SET sort_order = ? WHERE id = ?", idx+1, ids[j]); err != nil {
		return err
	}
	return tx.Commit()
}

func UpdateTaskTitle(conn *sql.DB, id int64, title string) error {
	_, err := conn.Exec("UPDATE tasks SET title = ? WHERE id = ?", title, id)
	return err
}

// TaskDescription возвращает описание задачи.
func TaskDescription(conn *sql.DB, id int64) (string, error) {
	var desc string
	err := conn.QueryRow("SELECT description FROM tasks WHERE id = ?", id).Scan(&desc)
	return desc, err
}

func UpdateTaskDescription(conn *sql.DB, id int64, text string) error {
	_, err := conn.Exec("UPDATE tasks SET description = ? WHERE id = ?", text, id)
	return err
}

func UpdateSubtaskTitle(conn *sql.DB, id int64, title string) error {
	_, err := conn.Exec("UPDATE subtasks SET title = ? WHERE id = ?", title, id)
	return err
}

// SubtaskDescription возвращает описание подзадачи.
func SubtaskDescription(conn *sql.DB, id int64) (string, error) {
	var desc string
	err := conn.QueryRow("SELECT description FROM subtasks WHERE id = ?", id).Scan(&desc)
	return desc, err
}

func UpdateSubtaskDescription(conn *sql.DB, id int64, text string) error {
	_, err := conn.Exec("UPDATE subtasks SET description = ? WHERE id = ?", text, id)
	return err
}

func TaskLinks(conn *sql.DB, taskID int64) ([]Link, error) {
	return linksFor(conn, "task_links", "task_id", taskID)
}

func CreateTaskLink(conn *sql.DB, taskID int64, name, url string) (Link, error) {
	return createLink(conn, "task_links", "task_id", taskID, name, url)
}

func DeleteTaskLink(conn *sql.DB, id int64) error {
	return deleteLink(conn, "task_links", id)
}

func UpdateTaskLink(conn *sql.DB, id int64, name, url string) error {
	return updateLink(conn, "task_links", id, name, url)
}

func SubtaskLinks(conn *sql.DB, subtaskID int64) ([]Link, error) {
	return linksFor(conn, "subtask_links", "subtask_id", subtaskID)
}

func CreateSubtaskLink(conn *sql.DB, subtaskID int64, name, url string) (Link, error) {
	return createLink(conn, "subtask_links", "subtask_id", subtaskID, name, url)
}

func DeleteSubtaskLink(conn *sql.DB, id int64) error {
	return deleteLink(conn, "subtask_links", id)
}

func UpdateSubtaskLink(conn *sql.DB, id int64, name, url string) error {
	return updateLink(conn, "subtask_links", id, name, url)
}

// JournalEntries возвращает записи журнала подзадачи в хронологическом порядке.
func JournalEntries(conn *sql.DB, subtaskID int64) ([]JournalEntry, error) {
	rows, err := conn.Query(`
SELECT id, subtask_id, created_at, text FROM journal_entries
WHERE subtask_id = ?
ORDER BY created_at, id`, subtaskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []JournalEntry
	for rows.Next() {
		var e JournalEntry
		var created int64
		if err := rows.Scan(&e.ID, &e.SubtaskID, &created, &e.Text); err != nil {
			return nil, err
		}
		e.CreatedAt = time.Unix(created, 0)
		out = append(out, e)
	}
	return out, rows.Err()
}

func CreateJournalEntry(conn *sql.DB, subtaskID int64, text string) (JournalEntry, error) {
	var e JournalEntry
	now := time.Now().Unix()
	res, err := conn.Exec(
		"INSERT INTO journal_entries (subtask_id, created_at, text) VALUES (?, ?, ?)",
		subtaskID, now, text)
	if err != nil {
		return e, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return e, err
	}
	e = JournalEntry{ID: id, SubtaskID: subtaskID, CreatedAt: time.Unix(now, 0), Text: text}
	return e, nil
}

func UpdateJournalEntry(conn *sql.DB, id int64, text string) error {
	_, err := conn.Exec("UPDATE journal_entries SET text = ? WHERE id = ?", text, id)
	return err
}

func TasksByProject(conn *sql.DB, projectID int64) ([]Task, error) {
	rows, err := conn.Query(`
SELECT t.id, t.project_id, t.title, t.description, t.status, t.created_at, t.completed_at,
       (SELECT COUNT(*) FROM subtasks s WHERE s.task_id = t.id)
FROM tasks t
WHERE t.project_id = ?
ORDER BY t.sort_order, t.id`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Task
	for rows.Next() {
		var t Task
		var created int64
		var completed sql.NullInt64
		if err := rows.Scan(&t.ID, &t.ProjectID, &t.Title, &t.Description, &t.Status,
			&created, &completed, &t.SubCount); err != nil {
			return nil, err
		}
		t.CreatedAt = time.Unix(created, 0)
		if completed.Valid {
			c := time.Unix(completed.Int64, 0)
			t.CompletedAt = &c
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func SubtasksByProject(conn *sql.DB, projectID int64) ([]SubtaskWithTime, error) {
	rows, err := conn.Query(`
SELECT s.id, s.task_id, s.title, s.description, s.status, s.sort_order, s.created_at, s.completed_at,
       COALESCE((SELECT SUM(te.ended_at - te.started_at) FROM time_entries te
                 WHERE te.subtask_id = s.id AND te.ended_at IS NOT NULL), 0),
       (SELECT te.started_at FROM time_entries te
        WHERE te.subtask_id = s.id AND te.ended_at IS NULL LIMIT 1)
FROM subtasks s
JOIN tasks t ON t.id = s.task_id
WHERE t.project_id = ?
ORDER BY s.task_id, s.sort_order, s.id`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []SubtaskWithTime
	for rows.Next() {
		var s SubtaskWithTime
		var created, total int64
		var completed, active sql.NullInt64
		if err := rows.Scan(&s.ID, &s.TaskID, &s.Title, &s.Description, &s.Status,
			&s.SortOrder, &created, &completed, &total, &active); err != nil {
			return nil, err
		}
		s.CreatedAt = time.Unix(created, 0)
		s.TotalSeconds = total
		if completed.Valid {
			c := time.Unix(completed.Int64, 0)
			s.CompletedAt = &c
		}
		if active.Valid {
			s.ActiveSince = &active.Int64
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// JournalTexts возвращает карту id подзадачи → объединённый текст записей её
// журнала (для полнотекстового поиска по проекту).
func JournalTexts(conn *sql.DB, projectID int64) (map[int64]string, error) {
	rows, err := conn.Query(`
SELECT s.id, GROUP_CONCAT(je.text, '\n')
FROM journal_entries je
JOIN subtasks s ON s.id = je.subtask_id
JOIN tasks t ON t.id = s.task_id
WHERE t.project_id = ?
GROUP BY s.id`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := map[int64]string{}
	for rows.Next() {
		var id int64
		var text string
		if err := rows.Scan(&id, &text); err != nil {
			return nil, err
		}
		out[id] = text
	}
	return out, rows.Err()
}

func SubtasksWithTime(conn *sql.DB, taskID int64) ([]SubtaskWithTime, error) {
	rows, err := conn.Query(`
SELECT s.id, s.task_id, s.title, s.description, s.status, s.sort_order, s.created_at, s.completed_at,
       COALESCE((SELECT SUM(te.ended_at - te.started_at) FROM time_entries te
                 WHERE te.subtask_id = s.id AND te.ended_at IS NOT NULL), 0),
       (SELECT te.started_at FROM time_entries te
        WHERE te.subtask_id = s.id AND te.ended_at IS NULL LIMIT 1)
FROM subtasks s
WHERE s.task_id = ?
ORDER BY s.sort_order, s.id`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []SubtaskWithTime
	for rows.Next() {
		var s SubtaskWithTime
		var created, total int64
		var completed, active sql.NullInt64
		if err := rows.Scan(&s.ID, &s.TaskID, &s.Title, &s.Description, &s.Status,
			&s.SortOrder, &created, &completed, &total, &active); err != nil {
			return nil, err
		}
		s.CreatedAt = time.Unix(created, 0)
		s.TotalSeconds = total
		if completed.Valid {
			c := time.Unix(completed.Int64, 0)
			s.CompletedAt = &c
		}
		if active.Valid {
			s.ActiveSince = &active.Int64
		}
		out = append(out, s)
	}
	return out, rows.Err()
}
