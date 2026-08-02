package db

import (
	"database/sql"
	"time"
)

func CreateTask(conn *sql.DB, projectID int64, title string) (Task, error) {
	var t Task
	now := time.Now().Unix()
	status, _ := DefaultStatus(conn)
	res, err := conn.Exec(
		"INSERT INTO tasks (project_id, title, status, created_at) VALUES (?, ?, ?, ?)",
		projectID, title, status, now)
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

func SubtaskLinks(conn *sql.DB, subtaskID int64) ([]Link, error) {
	return linksFor(conn, "subtask_links", "subtask_id", subtaskID)
}

func CreateSubtaskLink(conn *sql.DB, subtaskID int64, name, url string) (Link, error) {
	return createLink(conn, "subtask_links", "subtask_id", subtaskID, name, url)
}

func DeleteSubtaskLink(conn *sql.DB, id int64) error {
	return deleteLink(conn, "subtask_links", id)
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
SELECT t.id, t.project_id, t.title, t.status, t.created_at, t.completed_at,
       (SELECT COUNT(*) FROM subtasks s WHERE s.task_id = t.id)
FROM tasks t
WHERE t.project_id = ?
ORDER BY t.created_at, t.id`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Task
	for rows.Next() {
		var t Task
		var created int64
		var completed sql.NullInt64
		if err := rows.Scan(&t.ID, &t.ProjectID, &t.Title, &t.Status, &created,
			&completed, &t.SubCount); err != nil {
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
SELECT s.id, s.task_id, s.title, s.status, s.sort_order, s.created_at, s.completed_at,
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
		if err := rows.Scan(&s.ID, &s.TaskID, &s.Title, &s.Status, &s.SortOrder,
			&created, &completed, &total, &active); err != nil {
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

func SubtasksWithTime(conn *sql.DB, taskID int64) ([]SubtaskWithTime, error) {
	rows, err := conn.Query(`
SELECT s.id, s.task_id, s.title, s.status, s.sort_order, s.created_at, s.completed_at,
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
		if err := rows.Scan(&s.ID, &s.TaskID, &s.Title, &s.Status, &s.SortOrder,
			&created, &completed, &total, &active); err != nil {
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
