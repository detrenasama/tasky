package db

import (
	"database/sql"
	"time"
)

// ReportEntry — суммарное время подзадачи за период (только завершённые
// записи учёта времени).
type ReportEntry struct {
	ProjectID    int64
	ProjectName  string
	TaskID       int64
	TaskTitle    string
	SubtaskID    int64
	SubtaskTitle string
	Seconds      int64
}

// SubtaskReport — подзадача в отчёте.
type SubtaskReport struct {
	ID      int64
	Title   string
	Seconds int64
}

// TaskReport — задача в отчёте с подзадачами и суммарным временем.
type TaskReport struct {
	TaskID      int64
	ProjectID   int64
	ProjectName string
	TaskTitle   string
	Seconds     int64
	Subs        []SubtaskReport
}

// ReportEntries возвращает суммарное время по подзадачам за период [from, to):
// только завершённые записи, обрезанные по границам (активная сессия не
// учитывается); подзадачи без времени за период пропускаются. projectID = 0 —
// все проекты. Порядок — по созданию: задачи по created_at, подзадачи по
// sort_order.
func ReportEntries(conn *sql.DB, from, to time.Time, projectID int64) ([]ReportEntry, error) {
	rows, err := conn.Query(`
SELECT p.id, p.name, t.id, t.title, s.id, s.title,
       SUM(MIN(te.ended_at, ?) - MAX(te.started_at, ?)) AS secs
FROM time_entries te
JOIN subtasks s ON s.id = te.subtask_id
JOIN tasks t ON t.id = s.task_id
JOIN projects p ON p.id = t.project_id
WHERE te.started_at < ? AND te.ended_at > ?
  AND (? = 0 OR t.project_id = ?)
GROUP BY te.subtask_id
HAVING secs > 0
ORDER BY t.created_at, t.id, s.sort_order, s.id`,
		to.Unix(), from.Unix(), to.Unix(), from.Unix(), projectID, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ReportEntry
	for rows.Next() {
		var e ReportEntry
		if err := rows.Scan(&e.ProjectID, &e.ProjectName, &e.TaskID, &e.TaskTitle,
			&e.SubtaskID, &e.SubtaskTitle, &e.Seconds); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// ReportByTask группирует плоские записи в задачи с подзадачами, сохраняя
// порядок записей (по созданию).
func ReportByTask(entries []ReportEntry) []TaskReport {
	var out []TaskReport
	var cur *TaskReport
	for _, e := range entries {
		if cur == nil || cur.TaskID != e.TaskID {
			out = append(out, TaskReport{
				TaskID:      e.TaskID,
				ProjectID:   e.ProjectID,
				ProjectName: e.ProjectName,
				TaskTitle:   e.TaskTitle,
			})
			cur = &out[len(out)-1]
		}
		cur.Subs = append(cur.Subs, SubtaskReport{
			ID: e.SubtaskID, Title: e.SubtaskTitle, Seconds: e.Seconds,
		})
		cur.Seconds += e.Seconds
	}
	return out
}
