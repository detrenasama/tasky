package db

import (
	"database/sql"
	"errors"
	"time"
)

// ErrStatusInUse — попытка удалить статус, который используется задачами
// или подзадачами.
var ErrStatusInUse = errors.New("статус используется задачами или подзадачами")

// StatusOwner — тип владельца статуса (задача или подзадача).
type StatusOwner int

const (
	OwnerTask StatusOwner = iota
	OwnerSubtask
)

// defaultStatuses — статусы по умолчанию: имя, тип, цвет, подсказка заметки,
// участие в быстрой цепочке.
var defaultStatuses = []struct {
	name, typ, color, note string
	quick                  bool
}{
	{"Новая", "new", "#6a9955", "", true},
	{"Ожидает подтверждения", "new", "#4f7942", "", false},
	{"В работе", "in_progress", "#569cd6", "", true},
	{"На проверке", "in_progress", "#c586c0", "", true},
	{"Ожидает ответа", "in_progress", "#d7ba7d", "", false},
	{"Делегирована", "in_progress", "#ce9178", "Имя коллеги:", false},
	{"Выполнена", "done", "#8a8a8a", "", true},
	{"Отменена", "done", "#8f3e3e", "Причина отмены:", false},
}

// seedStatuses заполняет каталог статусов значениями по умолчанию, если он
// пуст.
func seedStatuses(conn *sql.DB) error {
	var n int
	if err := conn.QueryRow("SELECT COUNT(*) FROM statuses").Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	tx, err := conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for i, st := range defaultStatuses {
		quick := 0
		if st.quick {
			quick = 1
		}
		if _, err := tx.Exec(`
INSERT INTO statuses (name, type, color, note_prompt, is_quick, sort_order)
VALUES (?, ?, ?, ?, ?, ?)`,
			st.name, st.typ, st.color, st.note, quick, i); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// Statuses возвращает каталог статусов в порядке сортировки.
func Statuses(conn *sql.DB) ([]StatusDef, error) {
	rows, err := conn.Query(`
SELECT id, name, type, color, note_prompt, is_quick, sort_order FROM statuses
ORDER BY sort_order, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []StatusDef
	for rows.Next() {
		var st StatusDef
		var quick int
		if err := rows.Scan(&st.ID, &st.Name, &st.Type, &st.Color,
			&st.NotePrompt, &quick, &st.SortOrder); err != nil {
			return nil, err
		}
		st.IsQuick = quick != 0
		out = append(out, st)
	}
	return out, rows.Err()
}

// CreateStatus добавляет статус в конец каталога.
func CreateStatus(conn *sql.DB, name, typ, color, note string, quick bool) (StatusDef, error) {
	var st StatusDef
	var maxOrder sql.NullInt64
	if err := conn.QueryRow(
		"SELECT MAX(sort_order) FROM statuses").Scan(&maxOrder); err != nil {
		return st, err
	}
	order := 0
	if maxOrder.Valid {
		order = int(maxOrder.Int64) + 1
	}
	q := 0
	if quick {
		q = 1
	}
	res, err := conn.Exec(`
INSERT INTO statuses (name, type, color, note_prompt, is_quick, sort_order)
VALUES (?, ?, ?, ?, ?, ?)`, name, typ, color, note, q, order)
	if err != nil {
		return st, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return st, err
	}
	st = StatusDef{ID: id, Name: name, Type: typ, Color: color,
		NotePrompt: note, IsQuick: quick, SortOrder: order}
	return st, nil
}

// UpdateStatus изменяет статус каталога.
func UpdateStatus(conn *sql.DB, id int64, name, typ, color, note string, quick bool) error {
	q := 0
	if quick {
		q = 1
	}
	_, err := conn.Exec(`
UPDATE statuses SET name = ?, type = ?, color = ?, note_prompt = ?, is_quick = ?
WHERE id = ?`, name, typ, color, note, q, id)
	return err
}

// statusInUse проверяет, используется ли статус задачами или подзадачами.
func statusInUse(conn *sql.DB, name string) (bool, error) {
	var used int
	if err := conn.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM tasks WHERE status = ?) + EXISTS(SELECT 1 FROM subtasks WHERE status = ?)",
		name, name).Scan(&used); err != nil {
		return false, err
	}
	return used > 0, nil
}

// DeleteStatus удаляет статус из каталога; ErrStatusInUse, если он
// используется.
func DeleteStatus(conn *sql.DB, id int64) error {
	var name string
	if err := conn.QueryRow(
		"SELECT name FROM statuses WHERE id = ?", id).Scan(&name); err != nil {
		return err
	}
	used, err := statusInUse(conn, name)
	if err != nil {
		return err
	}
	if used {
		return ErrStatusInUse
	}
	_, err = conn.Exec("DELETE FROM statuses WHERE id = ?", id)
	return err
}

// DefaultStatus возвращает имя статуса по умолчанию для новых элементов —
// первый статус типа new в порядке сортировки.
func DefaultStatus(conn *sql.DB) (string, error) {
	var name string
	err := conn.QueryRow(`
SELECT name FROM statuses WHERE type = 'new' ORDER BY sort_order, id LIMIT 1`).
		Scan(&name)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return name, err
}

// SetStatus меняет статус задачи или подзадачи: обновляет status (и
// completed_at при входе/выходе из завершённого типа) и записывает переход
// в status_history (и для задач, и для подзадач).
// statusMergeWindow — окно в секундах: переход в течение этого времени от
// последней записи истории заменяет её вместо добавления новой.
const statusMergeWindow = 60

// SetStatus меняет статус задачи/подзадачи и пишет переход в status_history.
// Если с последней записи владельца прошло не больше statusMergeWindow
// секунд — запись заменяется (обновляются to_status, note, created_at,
// from_status остаётся первым статусом группы); возврат к from_status этой
// записи (в окне слияния) удаляет её (чтобы не было «X → X»); иначе
// добавляется новая запись. Смена статуса на тот же с другим комментарием
// (статусы с note_prompt) тоже пишется: в окне слияния заменяет запись,
// после окна добавляет «X → X (комментарий)»; повторный ввод того же
// комментария запись не создаёт.
func SetStatus(conn *sql.DB, owner StatusOwner, id int64, to, note string, now time.Time) error {
	table := "tasks"
	ownerCol := "task_id"
	if owner == OwnerSubtask {
		table = "subtasks"
		ownerCol = "subtask_id"
	}
	var target StatusDef
	found := false
	sts, err := Statuses(conn)
	if err != nil {
		return err
	}
	for _, st := range sts {
		if st.Name == to {
			target = st
			found = true
		}
	}
	if !found {
		return errors.New("неизвестный статус: " + to)
	}

	tx, err := conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var from string
	if err := tx.QueryRow("SELECT status FROM "+table+" WHERE id = ?", id).
		Scan(&from); err != nil {
		return err
	}

	nowUnix := now.Unix()
	var lastID int64
	var lastFrom, lastNote string
	lastTS := int64(-1)
	err = tx.QueryRow(`
SELECT id, from_status, note, created_at FROM status_history
WHERE `+ownerCol+` = ?
ORDER BY created_at DESC, id DESC LIMIT 1`, id).Scan(&lastID, &lastFrom, &lastNote, &lastTS)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	// тот же статус и тот же комментарий — запись не нужна
	if from == to && lastNote == note {
		return nil
	}

	var completed any
	if target.Type == "done" {
		completed = now.Unix()
	}
	if _, err := tx.Exec(
		"UPDATE "+table+" SET status = ?, completed_at = ? WHERE id = ?",
		to, completed, id); err != nil {
		return err
	}

	if lastTS >= 0 && nowUnix-lastTS <= statusMergeWindow {
		if to == lastFrom && to != from {
			_, err = tx.Exec("DELETE FROM status_history WHERE id = ?", lastID)
		} else {
			_, err = tx.Exec(`
UPDATE status_history SET to_status = ?, note = ?, created_at = ?
WHERE id = ?`, to, note, nowUnix, lastID)
		}
	} else {
		_, err = tx.Exec(`
INSERT INTO status_history (`+ownerCol+`, from_status, to_status, note, created_at)
VALUES (?, ?, ?, ?, ?)`, id, from, to, note, nowUnix)
	}
	if err != nil {
		return err
	}
	return tx.Commit()
}

// StatusHistory возвращает историю смены статусов (задачи или подзадачи)
// хронологически (старые сверху).
func StatusHistory(conn *sql.DB, owner StatusOwner, id int64) ([]StatusHistoryEntry, error) {
	ownerCol := "task_id"
	if owner == OwnerSubtask {
		ownerCol = "subtask_id"
	}
	rows, err := conn.Query(`
SELECT from_status, to_status, note, created_at FROM status_history
WHERE `+ownerCol+` = ?
ORDER BY created_at, id`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []StatusHistoryEntry
	for rows.Next() {
		var e StatusHistoryEntry
		var ts int64
		if err := rows.Scan(&e.From, &e.To, &e.Note, &ts); err != nil {
			return nil, err
		}
		e.CreatedAt = time.Unix(ts, 0)
		out = append(out, e)
	}
	return out, rows.Err()
}
