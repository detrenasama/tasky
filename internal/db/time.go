package db

import (
	"database/sql"
	"time"
)

// StartSession запускает сессию для подзадачи, предварительно останавливая
// любую другую активную сессию (одновременно может идти только одна).
func StartSession(conn *sql.DB, subtaskID int64, now time.Time) error {
	tx, err := conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(
		"UPDATE time_entries SET ended_at = ? WHERE ended_at IS NULL AND subtask_id != ?",
		now.Unix(), subtaskID); err != nil {
		return err
	}
	if _, err := tx.Exec(
		"INSERT INTO time_entries (subtask_id, started_at) VALUES (?, ?)",
		subtaskID, now.Unix()); err != nil {
		return err
	}
	return tx.Commit()
}

func StopSession(conn *sql.DB, subtaskID int64, now time.Time) error {
	_, err := conn.Exec(
		"UPDATE time_entries SET ended_at = ? WHERE subtask_id = ? AND ended_at IS NULL",
		now.Unix(), subtaskID)
	return err
}

// TimeEntriesBySubtask возвращает все записи учёта времени подзадачи
// (по возрастанию начала).
func TimeEntriesBySubtask(conn *sql.DB, subtaskID int64) ([]TimeEntry, error) {
	rows, err := conn.Query(`
SELECT id, subtask_id, started_at, ended_at, note FROM time_entries
WHERE subtask_id = ?
ORDER BY started_at, id`, subtaskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []TimeEntry
	for rows.Next() {
		var e TimeEntry
		var started, ended sql.NullInt64
		if err := rows.Scan(&e.ID, &e.SubtaskID, &started, &ended, &e.Note); err != nil {
			return nil, err
		}
		e.StartedAt = time.Unix(started.Int64, 0)
		if ended.Valid {
			t := time.Unix(ended.Int64, 0)
			e.EndedAt = &t
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// RunningSession возвращает заголовок подзадачи и время старта единственной
// активной сессии (или nil, если ничего не запущено).
func RunningSession(conn *sql.DB) (*SubtaskWithTime, error) {
	row := conn.QueryRow(`
SELECT s.id, s.task_id, s.title, te.started_at
FROM time_entries te
JOIN subtasks s ON s.id = te.subtask_id
WHERE te.ended_at IS NULL
LIMIT 1`)
	var st SubtaskWithTime
	var started int64
	if err := row.Scan(&st.ID, &st.TaskID, &st.Title, &started); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	st.ActiveSince = &started
	return &st, nil
}

// TodayTotal возвращает суммарное время за сегодня (00:00 — сейчас) по всем
// проектам. Активная сессия учитывается частично.
func TodayTotal(conn *sql.DB, now time.Time) (time.Duration, error) {
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	return rangeTotal(conn, dayStart, dayStart.AddDate(0, 0, 1), now)
}

// WeeklyTotal возвращает суммарное время за текущую неделю (пн 00:00 локального
// времени — сейчас) по всем проектам. Активная сессия учитывается частично.
func WeeklyTotal(conn *sql.DB, now time.Time) (time.Duration, error) {
	weekStart := mondayOf(now)
	return rangeTotal(conn, weekStart, weekStart.AddDate(0, 0, 7), now)
}

// rangeTotal суммирует время записей, пересекающих интервал [from, to),
// обрезая их по границам; активная сессия считается до now.
func rangeTotal(conn *sql.DB, from, to, now time.Time) (time.Duration, error) {
	rows, err := conn.Query(`
SELECT started_at, ended_at FROM time_entries
WHERE started_at < ? AND (ended_at IS NULL OR ended_at > ?)`,
		to.Unix(), from.Unix())
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var total int64
	for rows.Next() {
		var start int64
		var ended sql.NullInt64
		if err := rows.Scan(&start, &ended); err != nil {
			return 0, err
		}
		end := now.Unix()
		if ended.Valid {
			end = ended.Int64
		}
		start = max(start, from.Unix())
		end = min(end, to.Unix())
		if end > start {
			total += end - start
		}
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	return time.Duration(total) * time.Second, nil
}

func mondayOf(t time.Time) time.Time {
	t = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	offset := (int(t.Weekday()) + 6) % 7
	return t.AddDate(0, 0, -offset)
}
