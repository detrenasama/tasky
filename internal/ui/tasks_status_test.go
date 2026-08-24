package ui

import (
	"database/sql"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/detrenasama/tasky/internal/db"
)

func TestTaskStatusQuickCycle(t *testing.T) {
	conn, s, task, _ := tasksSeedProject(t)
	if len(s.statuses) != 8 {
		t.Fatalf("статусов %d, ожидалось 8", len(s.statuses))
	}
	down := func() {
		s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	}
	statusOf := func() string {
		var st string
		if err := conn.QueryRow("SELECT status FROM tasks WHERE id = ?", task.ID).Scan(&st); err != nil {
			t.Fatal(err)
		}
		return st
	}

	down() // В работе
	if statusOf() != "В работе" {
		t.Fatalf("после x статус %q", statusOf())
	}
	down() // На проверке
	down() // Выполнена
	if statusOf() != "Выполнена" {
		t.Fatalf("после x×3 статус %q", statusOf())
	}
	var completed sql.NullInt64
	if err := conn.QueryRow("SELECT completed_at FROM tasks WHERE id = ?", task.ID).Scan(&completed); err != nil {
		t.Fatal(err)
	}
	if !completed.Valid {
		t.Error("completed_at не выставлен для Выполнена")
	}

	down() // без зацикливания: остаёмся на Выполнена
	if statusOf() != "Выполнена" {
		t.Fatalf("x с Выполнена зациклился: %q", statusOf())
	}

	// z назад: На проверке, completed_at очищен
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'z'}})
	if statusOf() != "На проверке" {
		t.Fatalf("после z статус %q", statusOf())
	}
	if err := conn.QueryRow("SELECT completed_at FROM tasks WHERE id = ?", task.ID).Scan(&completed); err != nil {
		t.Fatal(err)
	}
	if completed.Valid {
		t.Error("completed_at не очищен при выходе из Выполнена")
	}

	// статус вне цепочки: x прыгает на первый элемент
	db.SetStatus(conn, db.OwnerTask, task.ID, "Отменена", "", time.Now())
	s.loadData()
	down()
	if statusOf() != "Новая" {
		t.Fatalf("x из внецепочного статуса: %q", statusOf())
	}

	// полоса и цветной статус в списке
	plain := stripANSI(s.list.View())
	if !strings.Contains(plain, "Новая · ") {
		t.Errorf("в списке нет статуса: %q", plain)
	}

	// быстрый возврат к исходному статусу очищает историю («Новая → Новая»
	// не пишется)
	hist, _ := db.StatusHistory(conn, db.OwnerTask, task.ID)
	if len(hist) != 0 {
		t.Errorf("возврат к «Новой» оставил записи истории: %+v", hist)
	}
}

// TestTaskStatusPickAndNote — c открывает модалку выбора, «Делегирована»
// требует заметку (имя коллеги), переход пишется в журнал подзадачи.

func TestTaskStatusPickAndNote(t *testing.T) {
	conn, s, _, st := tasksSeedProject(t)
	m := &model{tasks: s}
	selectFirstSubtask(m)

	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	if s.mode != taskStatusPick {
		t.Fatalf("c не открыл выбор статуса (mode=%d)", s.mode)
	}
	if _, open := s.dialog(); !open {
		t.Error("выбор статуса не рендерится как модалка")
	}
	// преселект на текущем статусе («Новая»)
	if s.statusPick.sel != 0 {
		t.Errorf("преселект курсора: %d", s.statusPick.sel)
	}

	// ↓ до «Делегирована» (индекс 5) и Enter — модалка заметки
	for i := 0; i < 5; i++ {
		s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyDown})
	}
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyEnter})
	if s.mode != taskStatusNote {
		t.Fatalf("Делегирована не открыла заметку (mode=%d)", s.mode)
	}
	if _, open := s.dialog(); !open {
		t.Error("заметка не рендерится как модалка")
	}

	// Esc отменяет — статус не меняется
	s.statusNote.SetValue("Иван")
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyEsc})
	if s.mode != taskBrowse {
		t.Fatalf("Esc не отменил заметку (mode=%d)", s.mode)
	}
	var status string
	if err := conn.QueryRow("SELECT status FROM subtasks WHERE id = ?", st.ID).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "Новая" {
		t.Fatalf("отменённый переход применился: %q", status)
	}

	// снова: Делегирована + Ctrl+S с именем коллеги
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	for i := 0; i < 5; i++ {
		s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyDown})
	}
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyEnter})
	s.statusNote.SetValue("Иван Петров")
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyCtrlS})
	if s.mode != taskBrowse {
		t.Fatalf("Ctrl+S не применил статус (mode=%d)", s.mode)
	}
	if err := conn.QueryRow("SELECT status FROM subtasks WHERE id = ?", st.ID).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "Делегирована" {
		t.Fatalf("статус после заметки: %q", status)
	}
	hist, _ := db.StatusHistory(conn, db.OwnerSubtask, st.ID)
	if len(hist) != 1 || hist[0].From != "Новая" || hist[0].To != "Делегирована" ||
		hist[0].Note != "Иван Петров" {
		t.Errorf("запись истории: %+v", hist)
	}
	entries, _ := db.JournalEntries(conn, st.ID)
	if len(entries) != 0 {
		t.Errorf("журнал не должен содержать переход статуса: %+v", entries)
	}

	// повторный выбор того же статуса с другим именем — замена записи
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyEnter}) // преселект на «Делегирована»
	if s.mode != taskStatusNote {
		t.Fatalf("повторная Делегирована не открыла заметку (mode=%d)", s.mode)
	}
	s.statusNote.SetValue("Мария Сидорова")
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyCtrlS})
	hist, _ = db.StatusHistory(conn, db.OwnerSubtask, st.ID)
	if len(hist) != 1 || hist[0].Note != "Мария Сидорова" ||
		hist[0].From != "Новая" || hist[0].To != "Делегирована" {
		t.Errorf("запись после замены имени: %+v", hist)
	}

	// повторный ввод того же имени — запись не создаётся
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyEnter})
	s.statusNote.SetValue("Мария Сидорова")
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyCtrlS})
	hist, _ = db.StatusHistory(conn, db.OwnerSubtask, st.ID)
	if len(hist) != 1 {
		t.Errorf("повторный ввод имени создал запись: %+v", hist)
	}
}

// TestTaskStatusSubtaskHistory — быстрый переход подзадачи пишется в
// status_history и виден в info-панели.

func TestTaskStatusSubtaskHistory(t *testing.T) {
	conn, s, _, st := tasksSeedProject(t)
	m := &model{tasks: s}
	selectFirstSubtask(m)

	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	hist, _ := db.StatusHistory(conn, db.OwnerSubtask, st.ID)
	if len(hist) != 1 || hist[0].From != "Новая" || hist[0].To != "В работе" {
		t.Fatalf("история: %+v", hist)
	}
	entries, _ := db.JournalEntries(conn, st.ID)
	if len(entries) != 0 {
		t.Fatalf("журнал не должен содержать переход статуса: %+v", entries)
	}
	plain := stripANSI(s.infoTop(20))
	for _, want := range []string{"История статусов:", "Новая → В работе"} {
		if !strings.Contains(plain, want) {
			t.Errorf("в info нет %q", want)
		}
	}
	// в колонке описания переход не дублируется в журнале
	desc := stripANSI(s.descBox())
	if strings.Contains(desc, "Статус: ") {
		t.Error("переход статуса виден в журнале колонки описания")
	}
}

// TestTaskStatusHistoryInfo — переходы задачи видны в info-панели; быстрые
// переходы (в пределах минуты) сливаются в одну запись.

func TestTaskStatusHistoryInfo(t *testing.T) {
	conn, s, task, _ := tasksSeedProject(t)
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	hist, _ := db.StatusHistory(conn, db.OwnerTask, task.ID)
	if len(hist) != 1 {
		t.Fatalf("история: %d записей, ожидалась 1 (слияние быстрых переходов)", len(hist))
	}
	if hist[0].From != "Новая" || hist[0].To != "На проверке" {
		t.Errorf("слитая запись: %+v", hist[0])
	}
	plain := stripANSI(s.infoTop(20))
	for _, want := range []string{"История статусов:", "Новая → На проверке"} {
		if !strings.Contains(plain, want) {
			t.Errorf("в info нет %q", want)
		}
	}
	if strings.Contains(plain, "В работе") {
		t.Error("в info остался промежуточный переход")
	}
}

// TestTaskStatusPickCloses — выбор статуса без обязательной заметки в
// модалке применяет статус и закрывает модалку.

func TestTaskStatusPickCloses(t *testing.T) {
	conn, s, task, _ := tasksSeedProject(t)
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	if s.mode != taskStatusPick {
		t.Fatalf("c не открыл выбор статуса (mode=%d)", s.mode)
	}
	// ↓ до «В работе» (без note_prompt) и Enter
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyDown})
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyDown})
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyEnter})
	if s.mode != taskBrowse {
		t.Fatalf("Enter не закрыл модалку (mode=%d)", s.mode)
	}
	var status string
	if err := conn.QueryRow("SELECT status FROM tasks WHERE id = ?", task.ID).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "В работе" {
		t.Errorf("статус после выбора: %q", status)
	}
}

// TestTaskCreateRefreshesInfo — после создания задачи/подзадачи info-панель
// показывает историю нового элемента, а не старого.
