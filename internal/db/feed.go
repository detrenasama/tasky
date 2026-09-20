package db

import (
	"database/sql"
	"time"
)

// FeedItem — запись ленты новостей.
// Kind: overdue, due_today, due_tomorrow, due_week, scheduled_today, update, plugin:*
type FeedItem struct {
	ID           int64     `json:"id"`
	Kind         string    `json:"kind"`
	Title        string    `json:"title"`
	Body         string    `json:"body"`
	RefTaskID    *int64    `json:"ref_task_id"`
	RefSubtaskID *int64    `json:"ref_subtask_id"`
	DedupKey     string    `json:"dedup_key"`
	CreatedAt    time.Time `json:"created_at"`
	IsRead       bool      `json:"is_read"`
	IsArchived   bool      `json:"is_archived"`
}

// FeedFilter — фильтр выборки ленты.
type FeedFilter struct {
	Kind            string
	UnreadOnly      bool
	IncludeArchived bool
}

// ListFeed возвращает записи ленты по фильтру, новые сверху.
func ListFeed(conn *sql.DB, f FeedFilter) ([]FeedItem, error) {
	q := `SELECT id, kind, title, body, ref_task_id, ref_subtask_id, dedup_key, created_at, is_read, is_archived
FROM feed_items WHERE 1=1`
	var args []any
	if !f.IncludeArchived {
		q += " AND is_archived = 0"
	}
	if f.UnreadOnly {
		q += " AND is_read = 0"
	}
	if f.Kind != "" {
		if f.Kind == "plugin" {
			q += " AND kind LIKE 'plugin:%'"
		} else {
			q += " AND kind = ?"
			args = append(args, f.Kind)
		}
	}
	q += " ORDER BY created_at DESC, id DESC"
	rows, err := conn.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []FeedItem
	for rows.Next() {
		var it FeedItem
		var created int64
		var isRead, isArch int
		var refTask, refSub sql.NullInt64
		if err := rows.Scan(&it.ID, &it.Kind, &it.Title, &it.Body, &refTask, &refSub, &it.DedupKey, &created, &isRead, &isArch); err != nil {
			return nil, err
		}
		it.CreatedAt = time.Unix(created, 0)
		it.IsRead = isRead != 0
		it.IsArchived = isArch != 0
		if refTask.Valid {
			v := refTask.Int64
			it.RefTaskID = &v
		}
		if refSub.Valid {
			v := refSub.Int64
			it.RefSubtaskID = &v
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

// UpsertFeed вставляет записи игноря дубликаты по dedup_key.
func UpsertFeed(conn *sql.DB, items []FeedItem) (int, error) {
	if len(items) == 0 {
		return 0, nil
	}
	tx, err := conn.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	n := 0
	for _, it := range items {
		var refTask, refSub any
		if it.RefTaskID != nil {
			refTask = *it.RefTaskID
		}
		if it.RefSubtaskID != nil {
			refSub = *it.RefSubtaskID
		}
		res, err := tx.Exec(`INSERT OR IGNORE INTO feed_items
(kind, title, body, ref_task_id, ref_subtask_id, dedup_key, created_at, is_read, is_archived)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			it.Kind, it.Title, it.Body, refTask, refSub, it.DedupKey, it.CreatedAt.Unix(), 0, 0)
		if err != nil {
			return n, err
		}
		if c, _ := res.RowsAffected(); c > 0 {
			n++
		}
	}
	if err := tx.Commit(); err != nil {
		return n, err
	}
	return n, nil
}

// MarkFeedRead отмечает запись прочитанной.
func MarkFeedRead(conn *sql.DB, id int64) error {
	_, err := conn.Exec("UPDATE feed_items SET is_read = 1 WHERE id = ?", id)
	return err
}

// FeedUnreadCount возвращает число непрочитанных неархивированных.
func FeedUnreadCount(conn *sql.DB) (int, error) {
	var n int
	err := conn.QueryRow("SELECT COUNT(*) FROM feed_items WHERE is_read = 0 AND is_archived = 0").Scan(&n)
	return n, err
}

// ArchiveFeedBefore архивирует записи старше cutoff (исключая).
func ArchiveFeedBefore(conn *sql.DB, cutoff time.Time) (int, error) {
	res, err := conn.Exec("UPDATE feed_items SET is_archived = 1 WHERE created_at < ? AND is_archived = 0", cutoff.Unix())
	if err != nil {
		return 0, err
	}
	c, _ := res.RowsAffected()
	return int(c), nil
}

// DeleteArchivedFeed удаляет архивированные записи.
func DeleteArchivedFeed(conn *sql.DB) (int, error) {
	res, err := conn.Exec("DELETE FROM feed_items WHERE is_archived = 1")
	if err != nil {
		return 0, err
	}
	c, _ := res.RowsAffected()
	return int(c), nil
}
