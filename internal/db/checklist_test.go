package db

import (
	"database/sql"
	"testing"
)

func seedChecklistSubtask(t *testing.T, conn *sql.DB) int64 {
	t.Helper()
	pid := seedProject(t, conn)
	task, err := CreateTask(conn, pid, "t")
	if err != nil {
		t.Fatal(err)
	}
	sub, err := CreateSubtask(conn, task.ID, "s")
	if err != nil {
		t.Fatal(err)
	}
	return sub.ID
}

func TestChecklistCRUD(t *testing.T) {
	conn := openTestDB(t)
	subID := seedChecklistSubtask(t, conn)

	a, err := CreateChecklistItem(conn, subID, "Пункт A")
	if err != nil {
		t.Fatal(err)
	}
	if a.ID == 0 || a.Status != "new" {
		t.Errorf("CreateChecklistItem: id=%d status=%q", a.ID, a.Status)
	}
	b, err := CreateChecklistItem(conn, subID, "Пункт B")
	if err != nil {
		t.Fatal(err)
	}
	// порядок добавления: b после a
	items, err := ChecklistItems(conn, subID)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("ожидалось 2 элемента, получено %d", len(items))
	}
	if items[0].ID != a.ID || items[1].ID != b.ID {
		t.Error("порядок элементов чек-листа нарушен")
	}

	if err := SetChecklistItemStatus(conn, a.ID, "done"); err != nil {
		t.Fatal(err)
	}
	if err := SetChecklistItemStatus(conn, b.ID, "cancelled"); err != nil {
		t.Fatal(err)
	}
	// счётчик для значка: done+cancelled = 2, total = 2
	counts, err := ChecklistCounts(conn, 0)
	if err != nil {
		t.Fatal(err)
	}
	// projectID=0 — подзадача привязана к проекту с id=1
	_ = counts

	counts, err = ChecklistCounts(conn, 1)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := counts[subID]
	if !ok {
		t.Fatal("нет счётчика для подзадачи")
	}
	if got[0] != 2 || got[1] != 2 {
		t.Errorf("ChecklistCounts = %v, ожидалось [2,2]", got)
	}

	// перемещение: a (индекс 0) вниз — должен стать после b
	if err := MoveChecklistItem(conn, a.ID, 1); err != nil {
		t.Fatal(err)
	}
	items, _ = ChecklistItems(conn, subID)
	if items[0].ID != b.ID || items[1].ID != a.ID {
		t.Error("после перемещения порядок неверен")
	}

	if err := UpdateChecklistItemText(conn, a.ID, "Пункт A!"); err != nil {
		t.Fatal(err)
	}
	items, _ = ChecklistItems(conn, subID)
	if items[1].Text != "Пункт A!" {
		t.Errorf("текст не обновился: %q", items[1].Text)
	}

	if err := DeleteChecklistItem(conn, a.ID); err != nil {
		t.Fatal(err)
	}
	items, _ = ChecklistItems(conn, subID)
	if len(items) != 1 || items[0].ID != b.ID {
		t.Error("после удаления остался не тот элемент")
	}
}

func TestChecklistCountsEmpty(t *testing.T) {
	conn := openTestDB(t)
	subID := seedChecklistSubtask(t, conn)
	counts, err := ChecklistCounts(conn, 1)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := counts[subID]
	if !ok {
		t.Fatal("нет счётчика для подзадачи без чек-листа")
	}
	if got[0] != 0 || got[1] != 0 {
		t.Errorf("ChecklistCounts пустого чек-листа = %v, ожидалось [0,0]", got)
	}
}

func TestChecklistCascadeOnSubtaskDelete(t *testing.T) {
	conn := openTestDB(t)
	pid := seedProject(t, conn)
	task, err := CreateTask(conn, pid, "t")
	if err != nil {
		t.Fatal(err)
	}
	sub, err := CreateSubtask(conn, task.ID, "s")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := CreateChecklistItem(conn, sub.ID, "x"); err != nil {
		t.Fatal(err)
	}
	if err := DeleteTask(conn, task.ID); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := conn.QueryRow("SELECT COUNT(*) FROM checklist_items").Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Errorf("каскад: в checklist_items осталось %d строк", n)
	}
}

// TestChecklistStatusChangedAt — SetChecklistItemStatus обновляет метку
// status_changed_at, а ChecklistItems возвращает её.
func TestChecklistStatusChangedAt(t *testing.T) {
	conn := openTestDB(t)
	subID := seedChecklistSubtask(t, conn)

	it, err := CreateChecklistItem(conn, subID, "Пункт")
	if err != nil {
		t.Fatal(err)
	}
	if it.StatusChangedAt.IsZero() {
		t.Error("при создании status_changed_at не должен быть нулевым")
	}
	if err := SetChecklistItemStatus(conn, it.ID, "done"); err != nil {
		t.Fatal(err)
	}
	items, err := ChecklistItems(conn, subID)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("ожидался 1 элемент, got %d", len(items))
	}
	if items[0].Status != "done" {
		t.Errorf("статус=%q, ожидался done", items[0].Status)
	}
	if items[0].StatusChangedAt.IsZero() {
		t.Error("после смены статуса status_changed_at не должен быть нулевым")
	}
}
