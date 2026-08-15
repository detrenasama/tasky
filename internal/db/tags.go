package db

import (
	"database/sql"
	"errors"
	"strings"
	"time"
)

// ErrTagTypeInUse — попытка удалить тип тега, который используется тегами
// задач.
var ErrTagTypeInUse = errors.New("тип тега используется задачами")

// TagTypes возвращает каталог типов тегов в порядке сортировки.
func TagTypes(conn *sql.DB) ([]TagType, error) {
	rows, err := conn.Query(`
SELECT id, name, kind, color, sort_order FROM tag_types
ORDER BY sort_order, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []TagType
	for rows.Next() {
		var t TagType
		if err := rows.Scan(&t.ID, &t.Name, &t.Kind, &t.Color, &t.SortOrder); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// CreateTagType добавляет тип тега в конец каталога.
func CreateTagType(conn *sql.DB, name, kind, color string) (TagType, error) {
	var t TagType
	var maxOrder sql.NullInt64
	if err := conn.QueryRow(
		"SELECT MAX(sort_order) FROM tag_types").Scan(&maxOrder); err != nil {
		return t, err
	}
	order := 0
	if maxOrder.Valid {
		order = int(maxOrder.Int64) + 1
	}
	res, err := conn.Exec(`
INSERT INTO tag_types (name, kind, color, sort_order)
VALUES (?, ?, ?, ?)`, name, kind, color, order)
	if err != nil {
		return t, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return t, err
	}
	t = TagType{ID: id, Name: name, Kind: kind, Color: color, SortOrder: order}
	return t, nil
}

// UpdateTagType изменяет тип тега каталога.
func UpdateTagType(conn *sql.DB, id int64, name, kind, color string) error {
	_, err := conn.Exec(`
UPDATE tag_types SET name = ?, kind = ?, color = ? WHERE id = ?`,
		name, kind, color, id)
	return err
}

// DeleteTagType удаляет тип тега; ErrTagTypeInUse, если он используется.
func DeleteTagType(conn *sql.DB, id int64) error {
	var used int
	if err := conn.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM task_tags WHERE type_id = ?)", id).Scan(&used); err != nil {
		return err
	}
	if used > 0 {
		return ErrTagTypeInUse
	}
	_, err := conn.Exec("DELETE FROM tag_types WHERE id = ?", id)
	return err
}

// TaskTags возвращает теги задачи по порядку добавления.
func TaskTags(conn *sql.DB, taskID int64) ([]Tag, error) {
	rows, err := conn.Query(`
SELECT tg.id, tg.task_id, tg.type_id, tg.text, tg.url, tg.created_at,
       tt.name, tt.kind, tt.color
FROM task_tags tg
JOIN tag_types tt ON tt.id = tg.type_id
WHERE tg.task_id = ?
ORDER BY tg.created_at, tg.id`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanTags(rows)
}

// TagsByProject возвращает карту id задачи → теги для задач проекта.
func TagsByProject(conn *sql.DB, projectID int64) (map[int64][]Tag, error) {
	rows, err := conn.Query(`
SELECT tg.id, tg.task_id, tg.type_id, tg.text, tg.url, tg.created_at,
       tt.name, tt.kind, tt.color
FROM task_tags tg
JOIN tag_types tt ON tt.id = tg.type_id
JOIN tasks t ON t.id = tg.task_id
WHERE t.project_id = ?
ORDER BY tg.task_id, tg.created_at, tg.id`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanTagsByTask(rows)
}

// TagsByTasks возвращает карту id задачи → теги для перечисленных задач
// (для отчёта). Пустой список — пустая карта.
func TagsByTasks(conn *sql.DB, taskIDs []int64) (map[int64][]Tag, error) {
	out := map[int64][]Tag{}
	if len(taskIDs) == 0 {
		return out, nil
	}
	marks := make([]string, len(taskIDs))
	args := make([]any, len(taskIDs))
	for i, id := range taskIDs {
		marks[i] = "?"
		args[i] = id
	}
	rows, err := conn.Query(`
SELECT tg.id, tg.task_id, tg.type_id, tg.text, tg.url, tg.created_at,
       tt.name, tt.kind, tt.color
FROM task_tags tg
JOIN tag_types tt ON tt.id = tg.type_id
WHERE tg.task_id IN (`+strings.Join(marks, ", ")+`)
ORDER BY tg.task_id, tg.created_at, tg.id`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanTagsByTask(rows)
}

// CreateTag добавляет тег задаче.
func CreateTag(conn *sql.DB, taskID, typeID int64, text, url string) (Tag, error) {
	var t Tag
	now := time.Now().Unix()
	res, err := conn.Exec(`
INSERT INTO task_tags (task_id, type_id, text, url, created_at)
VALUES (?, ?, ?, ?, ?)`, taskID, typeID, text, url, now)
	if err != nil {
		return t, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return t, err
	}
	t, err = tagByID(conn, id)
	return t, err
}

// UpdateTag изменяет тип, значение и URL тега.
func UpdateTag(conn *sql.DB, id, typeID int64, text, url string) error {
	_, err := conn.Exec(`
UPDATE task_tags SET type_id = ?, text = ?, url = ? WHERE id = ?`,
		typeID, text, url, id)
	return err
}

// DeleteTag удаляет тег.
func DeleteTag(conn *sql.DB, id int64) error {
	_, err := conn.Exec("DELETE FROM task_tags WHERE id = ?", id)
	return err
}

// tagByID возвращает тег по id (с денормализованными полями типа).
func tagByID(conn *sql.DB, id int64) (Tag, error) {
	var t Tag
	var created int64
	err := conn.QueryRow(`
SELECT tg.id, tg.task_id, tg.type_id, tg.text, tg.url, tg.created_at,
       tt.name, tt.kind, tt.color
FROM task_tags tg
JOIN tag_types tt ON tt.id = tg.type_id
WHERE tg.id = ?`, id).Scan(&t.ID, &t.TaskID, &t.TypeID, &t.Text, &t.URL,
		&created, &t.TypeName, &t.Kind, &t.Color)
	if err != nil {
		return t, err
	}
	t.CreatedAt = time.Unix(created, 0)
	return t, nil
}

// scanTags сканирует строки запроса тегов в плоский список (порядок запроса).
func scanTags(rows *sql.Rows) ([]Tag, error) {
	var out []Tag
	for rows.Next() {
		t, err := scanTag(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// scanTagsByTask сканирует строки запроса тегов в карту id задачи → теги.
func scanTagsByTask(rows *sql.Rows) (map[int64][]Tag, error) {
	out := map[int64][]Tag{}
	for rows.Next() {
		t, err := scanTag(rows)
		if err != nil {
			return nil, err
		}
		out[t.TaskID] = append(out[t.TaskID], t)
	}
	return out, rows.Err()
}

// scanTag сканирует одну строку тега (вместе с полями типа).
func scanTag(rows *sql.Rows) (Tag, error) {
	var t Tag
	var created int64
	if err := rows.Scan(&t.ID, &t.TaskID, &t.TypeID, &t.Text, &t.URL,
		&created, &t.TypeName, &t.Kind, &t.Color); err != nil {
		return t, err
	}
	t.CreatedAt = time.Unix(created, 0)
	return t, nil
}
