package feed

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/detrenasama/tasky/internal/db"
)

// dayStart возвращает 00:00 локального дня.
func dayStart(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// parseHour читает feed_gen_hour из settings (default 6).
func genHour(conn *sql.DB) int {
	s, ok, _ := db.GetSetting(conn, "feed_gen_hour")
	if !ok {
		return 6
	}
	var h int
	if _, err := fmt.Sscanf(s, "%d", &h); err != nil || h < 0 || h > 23 {
		return 6
	}
	return h
}

// NeedsGenerate проверяет нужно ли генерировать сегодня.
// Логика: lastGen != todayStr && now >= today 00:00+genHour.
func NeedsGenerate(conn *sql.DB, now time.Time) bool {
	todayStr := dayStart(now).Format("2006-01-02")
	last, ok, _ := db.GetSetting(conn, "feed_last_gen")
	if ok && last == todayStr {
		return false
	}
	h := genHour(conn)
	genTime := time.Date(now.Year(), now.Month(), now.Day(), h, 0, 0, 0, now.Location())
	return !now.Before(genTime)
}

// MarkGenerated ставит feed_last_gen = today.
func MarkGenerated(conn *sql.DB, now time.Time) error {
	return db.SetSetting(conn, "feed_last_gen", dayStart(now).Format("2006-01-02"))
}

// GenerateDaily создаёт записи ленты на сегодня. Идемпотентно (dedup_key).
func GenerateDaily(conn *sql.DB, now time.Time) (int, error) {
	today := dayStart(now)
	tomorrow := today.AddDate(0, 0, 1)
	weekEnd := monday(today).AddDate(0, 0, 7) // [today, monday+7)
	var items []db.FeedItem

	// helper to append
	add := func(kind, title, body string, taskID, subID *int64, dedup string) {
		items = append(items, db.FeedItem{
			Kind:         kind,
			Title:        title,
			Body:         body,
			RefTaskID:    taskID,
			RefSubtaskID: subID,
			DedupKey:     dedup,
			CreatedAt:    now,
		})
	}

	// Задачи — overdue
	if err := genOverdue(conn, today, add); err != nil {
		return 0, err
	}
	// due_today / tomorrow / week, scheduled_today для задач и подзадач
	if err := genDue(conn, today, tomorrow, weekEnd, add); err != nil {
		return 0, err
	}
	if err := genScheduled(conn, today, add); err != nil {
		return 0, err
	}

	n, err := db.UpsertFeed(conn, items)
	if err != nil {
		return n, err
	}
	if n > 0 || len(items) == 0 {
		// даже если нет новых — фиксируем что генерация была сегодня, чтобы не крутить каждый тик
		_ = MarkGenerated(conn, now)
	}
	// архивация старше 7 дней
	cutoff := now.AddDate(0, 0, -7)
	_, _ = db.ArchiveFeedBefore(conn, cutoff)
	return n, nil
}

func monday(today time.Time) time.Time {
	wd := int(today.Weekday())
	// Sunday=0 -> 6, Monday=1 ->0 ...
	off := (wd + 6) % 7
	return today.AddDate(0, 0, -off)
}

func genOverdue(conn *sql.DB, today time.Time, add func(string, string, string, *int64, *int64, string)) error {
	todayUnix := today.Unix()
	// задачи overdue: due_date < today, not done (completed_at IS NULL и type != done)
	rows, err := conn.Query(`
SELECT t.id, t.title, p.name FROM tasks t JOIN projects p ON p.id=t.project_id
WHERE t.due_date IS NOT NULL AND t.due_date < ?
  AND (t.completed_at IS NULL)
  AND COALESCE((SELECT type FROM statuses s WHERE s.name=t.status), 'new') != 'done'`, todayUnix)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var title, proj string
		if err := rows.Scan(&id, &title, &proj); err != nil {
			return err
		}
		dedup := fmt.Sprintf("overdue:task:%d:%s", id, today.Format("2006-01-02"))
		add("overdue", fmt.Sprintf("Просрочено: %s", title), fmt.Sprintf("Проект %s · дедлайн просрочен", proj), &id, nil, dedup)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	// подзадачи overdue
	rows2, err := conn.Query(`
SELECT s.id, s.title, t.title, p.name FROM subtasks s
JOIN tasks t ON t.id=s.task_id JOIN projects p ON p.id=t.project_id
WHERE s.due_date IS NOT NULL AND s.due_date < ?
  AND (s.completed_at IS NULL)
  AND COALESCE((SELECT type FROM statuses s2 WHERE s2.name=s.status), 'new') != 'done'`, todayUnix)
	if err != nil {
		return err
	}
	defer rows2.Close()
	for rows2.Next() {
		var sid int64
		var stitle, ttitle, proj string
		if err := rows2.Scan(&sid, &stitle, &ttitle, &proj); err != nil {
			return err
		}
		dedup := fmt.Sprintf("overdue:subtask:%d:%s", sid, today.Format("2006-01-02"))
		add("overdue", fmt.Sprintf("Просрочено: %s", stitle), fmt.Sprintf("Задача %s · проект %s", ttitle, proj), nil, &sid, dedup)
	}
	return rows2.Err()
}

func genDue(conn *sql.DB, today, tomorrow, weekEnd time.Time, add func(string, string, string, *int64, *int64, string)) error {
	cases := []struct {
		kind string
		from int64
		to   int64 // exclusive, 0 = exact one day
	}{
		{"due_today", today.Unix(), 0},
		{"due_tomorrow", tomorrow.Unix(), 0},
		{"due_week", tomorrow.AddDate(0, 0, 1).Unix(), weekEnd.Unix()},
	}
	for _, c := range cases {
		// задачи
		var rows *sql.Rows
		var err error
		if c.to == 0 {
			rows, err = conn.Query(`
SELECT t.id, t.title, p.name FROM tasks t JOIN projects p ON p.id=t.project_id
WHERE t.due_date = ? AND COALESCE((SELECT type FROM statuses s WHERE s.name=t.status),'new')!='done'`, c.from)
		} else {
			rows, err = conn.Query(`
SELECT t.id, t.title, p.name FROM tasks t JOIN projects p ON p.id=t.project_id
WHERE t.due_date >= ? AND t.due_date < ? AND COALESCE((SELECT type FROM statuses s WHERE s.name=t.status),'new')!='done'`, c.from, c.to)
		}
		if err != nil {
			return err
		}
		for rows.Next() {
			var id int64
			var title, proj string
			if err := rows.Scan(&id, &title, &proj); err != nil {
				rows.Close()
				return err
			}
			dateStr := time.Unix(c.from, 0).Format("2006-01-02")
			if c.kind == "due_week" {
				dateStr = today.Format("2006-01-02")
			}
			dedup := fmt.Sprintf("%s:task:%d:%s", c.kind, id, dateStr)
			// пропускаем если уже есть overdue сегодня (дедуп другой kind, но логически overdue важнее — оставляем оба? оставляем оба, фильтр покажет)
			add(c.kind, labelDue(c.kind, title), fmt.Sprintf("Проект %s", proj), &id, nil, dedup)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}
		// подзадачи
		var rows2 *sql.Rows
		if c.to == 0 {
			rows2, err = conn.Query(`
SELECT s.id, s.title, t.title, p.name FROM subtasks s
JOIN tasks t ON t.id=s.task_id JOIN projects p ON p.id=t.project_id
WHERE s.due_date = ? AND COALESCE((SELECT type FROM statuses s2 WHERE s2.name=s.status),'new')!='done'`, c.from)
		} else {
			rows2, err = conn.Query(`
SELECT s.id, s.title, t.title, p.name FROM subtasks s
JOIN tasks t ON t.id=s.task_id JOIN projects p ON p.id=t.project_id
WHERE s.due_date >= ? AND s.due_date < ? AND COALESCE((SELECT type FROM statuses s2 WHERE s2.name=s.status),'new')!='done'`, c.from, c.to)
		}
		if err != nil {
			return err
		}
		for rows2.Next() {
			var sid int64
			var stitle, ttitle, proj string
			if err := rows2.Scan(&sid, &stitle, &ttitle, &proj); err != nil {
				rows2.Close()
				return err
			}
			dateStr := time.Unix(c.from, 0).Format("2006-01-02")
			if c.kind == "due_week" {
				dateStr = today.Format("2006-01-02")
			}
			dedup := fmt.Sprintf("%s:subtask:%d:%s", c.kind, sid, dateStr)
			add(c.kind, labelDue(c.kind, stitle), fmt.Sprintf("Задача %s · проект %s", ttitle, proj), nil, &sid, dedup)
		}
		rows2.Close()
		if err := rows2.Err(); err != nil {
			return err
		}
	}
	return nil
}

func genScheduled(conn *sql.DB, today time.Time, add func(string, string, string, *int64, *int64, string)) error {
	todayUnix := today.Unix()
	rows, err := conn.Query(`
SELECT t.id, t.title, p.name FROM tasks t JOIN projects p ON p.id=t.project_id
WHERE t.schedule_date = ? AND COALESCE((SELECT type FROM statuses s WHERE s.name=t.status),'new')!='done'`, todayUnix)
	if err != nil {
		return err
	}
	for rows.Next() {
		var id int64
		var title, proj string
		if err := rows.Scan(&id, &title, &proj); err != nil {
			rows.Close()
			return err
		}
		dedup := fmt.Sprintf("scheduled_today:task:%d:%s", id, today.Format("2006-01-02"))
		add("scheduled_today", fmt.Sprintf("Запланировано: %s", title), fmt.Sprintf("Проект %s · начало сегодня", proj), &id, nil, dedup)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	rows2, err := conn.Query(`
SELECT s.id, s.title, t.title, p.name FROM subtasks s
JOIN tasks t ON t.id=s.task_id JOIN projects p ON p.id=t.project_id
WHERE s.schedule_date = ? AND COALESCE((SELECT type FROM statuses s2 WHERE s2.name=s.status),'new')!='done'`, todayUnix)
	if err != nil {
		return err
	}
	defer rows2.Close()
	for rows2.Next() {
		var sid int64
		var stitle, ttitle, proj string
		if err := rows2.Scan(&sid, &stitle, &ttitle, &proj); err != nil {
			return err
		}
		dedup := fmt.Sprintf("scheduled_today:subtask:%d:%s", sid, today.Format("2006-01-02"))
		add("scheduled_today", fmt.Sprintf("Запланировано: %s", stitle), fmt.Sprintf("Задача %s · проект %s", ttitle, proj), nil, &sid, dedup)
	}
	return rows2.Err()
}

func labelDue(kind, title string) string {
	switch kind {
	case "due_today":
		return fmt.Sprintf("Срок сегодня: %s", title)
	case "due_tomorrow":
		return fmt.Sprintf("Срок завтра: %s", title)
	case "due_week":
		return fmt.Sprintf("Срок на неделе: %s", title)
	default:
		return title
	}
}

// GenerateUpdate создаёт запись об обновлении.
func GenerateUpdate(conn *sql.DB, version, body string, now time.Time) (int, error) {
	if version == "" {
		return 0, nil
	}
	title := fmt.Sprintf("Доступно обновление %s — tasky upgrade", version)
	dedup := fmt.Sprintf("update:%s", version)
	item := db.FeedItem{
		Kind:      "update",
		Title:     title,
		Body:      body,
		DedupKey:  dedup,
		CreatedAt: now,
	}
	return db.UpsertFeed(conn, []db.FeedItem{item})
}
