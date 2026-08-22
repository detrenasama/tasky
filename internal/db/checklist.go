package db

import (
	"database/sql"
	"time"
)

// ChecklistItems возвращает элементы чек-листа подзадачи в порядке sort_order.
func ChecklistItems(conn *sql.DB, subtaskID int64) ([]ChecklistItem, error) {
	rows, err := conn.Query(`
SELECT id, subtask_id, text, status, sort_order, created_at, status_changed_at FROM checklist_items
WHERE subtask_id = ?
ORDER BY sort_order, id`, subtaskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []ChecklistItem
	for rows.Next() {
		var ci ChecklistItem
		var created, changed int64
		if err := rows.Scan(&ci.ID, &ci.SubtaskID, &ci.Text, &ci.Status, &ci.SortOrder, &created, &changed); err != nil {
			return nil, err
		}
		ci.CreatedAt = time.Unix(created, 0)
		ci.StatusChangedAt = time.Unix(changed, 0)
		out = append(out, ci)
	}
	return out, rows.Err()
}

// ChecklistCounts возвращает для каждой подзадачи проекта [выполнено+отменено, всего].
func ChecklistCounts(conn *sql.DB, projectID int64) (map[int64][2]int, error) {
	rows, err := conn.Query(`
SELECT s.id,
       COUNT(ci.id) AS total,
       SUM(CASE WHEN ci.status IN ('done', 'cancelled') THEN 1 ELSE 0 END) AS dc
FROM subtasks s
JOIN tasks t ON t.id = s.task_id
LEFT JOIN checklist_items ci ON ci.subtask_id = s.id
WHERE t.project_id = ?
GROUP BY s.id`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[int64][2]int)
	for rows.Next() {
		var id int64
		var total, dc sql.NullInt64
		if err := rows.Scan(&id, &total, &dc); err != nil {
			return nil, err
		}
		out[id] = [2]int{int(dc.Int64), int(total.Int64)}
	}
	return out, rows.Err()
}

// CreateChecklistItem добавляет элемент в конец чек-листа подзадачи.
func CreateChecklistItem(conn *sql.DB, subtaskID int64, text string) (ChecklistItem, error) {
	var ci ChecklistItem
	now := time.Now().Unix()
	res, err := conn.Exec(`
INSERT INTO checklist_items (subtask_id, text, status, sort_order, created_at, status_changed_at)
SELECT ?, ?, 'new', COALESCE(MAX(sort_order), 0) + 1, ?, ?
FROM checklist_items WHERE subtask_id = ?`,
		subtaskID, text, now, now, subtaskID)
	if err != nil {
		return ci, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return ci, err
	}
	ci = ChecklistItem{ID: id, SubtaskID: subtaskID, Text: text, Status: "new",
		SortOrder: 0, CreatedAt: time.Unix(now, 0), StatusChangedAt: time.Unix(now, 0)}
	return ci, nil
}

// UpdateChecklistItemText меняет текст элемента чек-листа.
func UpdateChecklistItemText(conn *sql.DB, id int64, text string) error {
	_, err := conn.Exec("UPDATE checklist_items SET text = ? WHERE id = ?", text, id)
	return err
}

// SetChecklistItemStatus меняет статус элемента чек-листа и обновляет метку
// времени смены статуса (нужна для вывода выполненных пунктов в отчёте).
func SetChecklistItemStatus(conn *sql.DB, id int64, status string) error {
	now := time.Now().Unix()
	_, err := conn.Exec("UPDATE checklist_items SET status = ?, status_changed_at = ? WHERE id = ?", status, now, id)
	return err
}

// MoveChecklistItem перемещает элемент чек-листа вверх (dir=-1) или вниз (dir=+1).
func MoveChecklistItem(conn *sql.DB, id int64, dir int) error {
	return moveOrderedRow(conn, "checklist_items", "subtask_id", id, dir)
}

// DeleteChecklistItem удаляет элемент чек-листа.
func DeleteChecklistItem(conn *sql.DB, id int64) error {
	_, err := conn.Exec("DELETE FROM checklist_items WHERE id = ?", id)
	return err
}
