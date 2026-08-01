package db

import (
	"database/sql"
	"time"
)

func StartSession(conn *sql.DB, subtaskID int64, now time.Time) error {
	_, err := conn.Exec(
		"INSERT INTO time_entries (subtask_id, started_at) VALUES (?, ?)",
		subtaskID, now.Unix())
	return err
}

func StopSession(conn *sql.DB, subtaskID int64, now time.Time) error {
	_, err := conn.Exec(
		"UPDATE time_entries SET ended_at = ? WHERE subtask_id = ? AND ended_at IS NULL",
		now.Unix(), subtaskID)
	return err
}

// WeeklyTotal возвращает суммарное время за текущую неделю (пн 00:00 локального
// времени — сейчас) по всем проектам. Активная сессия учитывается частично.
func WeeklyTotal(conn *sql.DB, now time.Time) (time.Duration, error) {
	weekStart := mondayOf(now)
	weekEnd := weekStart.AddDate(0, 0, 7)

	rows, err := conn.Query(`
SELECT started_at, ended_at FROM time_entries
WHERE started_at < ? AND (ended_at IS NULL OR ended_at > ?)`,
		weekEnd.Unix(), weekStart.Unix())
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
		start = max(start, weekStart.Unix())
		end = min(end, weekEnd.Unix())
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
