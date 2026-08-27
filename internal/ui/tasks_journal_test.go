package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/detrenasama/tasky/internal/db"
)

func TestTaskJournalAddAndEdit(t *testing.T) {
	conn, s, _, st := tasksSeedProject(t)
	m := &model{tasks: s}
	selectFirstSubtask(m)
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyTab})

	// Ctrl+J в фокусе описания на подзадаче открывает модалку записи
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyCtrlJ})
	if s.mode != taskJournal {
		t.Fatalf("ctrl+j не открыл журнал (mode=%d)", s.mode)
	}
	if _, open := s.dialog(); !open {
		t.Error("запись журнала не рендерится как модалка")
	}

	// Esc с несохранёнными изменениями — подтверждение, а не выход
	s.journalText.SetValue("не сохранять")
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyEsc})
	if s.mode != taskJournalDiscard {
		t.Fatalf("Esc с изменениями должен открыть подтверждение (mode=%d)", s.mode)
	}
	if entries, _ := db.JournalEntries(conn, st.ID); len(entries) != 0 {
		t.Errorf("подтверждение создало запись: %d", len(entries))
	}
	// Esc в подтверждении — возврат к редактированию
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyEsc})
	if s.mode != taskJournal {
		t.Fatalf("Esc в подтверждении должен вернуть к редактированию (mode=%d)", s.mode)
	}
	// пустое (без изменений) — Esc сразу выходит
	s.journalText.SetValue("")
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyEsc})
	if s.mode != taskBrowse {
		t.Fatalf("Esc без изменений должен выйти (mode=%d)", s.mode)
	}
	if entries, _ := db.JournalEntries(conn, st.ID); len(entries) != 0 {
		t.Errorf("отменённая запись создалась: %d", len(entries))
	}

	// Ctrl+J → Ctrl+S сохраняет
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyCtrlJ})
	s.journalText.SetValue("работал над задачей")
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyCtrlS})
	if s.mode != taskBrowse {
		t.Fatalf("Ctrl+S не сохранил запись (mode=%d)", s.mode)
	}
	entries, _ := db.JournalEntries(conn, st.ID)
	if len(entries) != 1 || entries[0].Text != "работал над задачей" {
		t.Fatalf("записи = %+v", entries)
	}
	plain := stripANSI(s.descBox())
	if !strings.Contains(plain, "работал над задачей") {
		t.Error("запись не отобразилась в колонке описания")
	}
	if !strings.Contains(plain, entries[0].CreatedAt.Format("02.01.2006 15:04")) {
		t.Error("в колонке нет даты/времени записи")
	}

	// j — редактирование свежей записи текущего дня
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if s.mode != taskJournal || s.journalEditID != entries[0].ID {
		t.Fatalf("j не открыл редактирование записи (mode=%d, id=%d)", s.mode, s.journalEditID)
	}
	if s.journalText.Value() != "работал над задачей" {
		t.Errorf("textarea не заполнен текстом записи: %q", s.journalText.Value())
	}
	s.journalText.SetValue("доработал")
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyCtrlS})
	entries, _ = db.JournalEntries(conn, st.ID)
	if len(entries) != 1 || entries[0].Text != "доработал" {
		t.Errorf("запись не обновилась: %+v", entries)
	}
}

// TestTaskJournalEditOnlyToday — редактировать можно только записи текущего
// дня: для вчерашней записи j показывает ошибку.

func TestTaskJournalEditOnlyToday(t *testing.T) {
	conn, s, _, st := tasksSeedProject(t)
	m := &model{tasks: s}
	if _, err := conn.Exec(
		"INSERT INTO journal_entries (subtask_id, created_at, text) VALUES (?, ?, 'вчера')",
		st.ID, time.Now().Add(-24*time.Hour).Unix()); err != nil {
		t.Fatal(err)
	}
	s.loadDesc()
	selectFirstSubtask(m)
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyTab})

	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if s.mode != taskBrowse {
		t.Fatalf("j открыл редактирование вчерашней записи (mode=%d)", s.mode)
	}
	if s.lastErr == nil {
		t.Error("для вчерашней записи не выставлена ошибка")
	}
}

// TestTasksTitleEdit — e в фокусе списка открывает модалку изменения названия:
// input префиллен текущим названием, Enter сохраняет (задачи и подзадачи),
// Esc отменяет, пустое название показывает ошибку и не закрывает модалку.
func TestTasksJournalEscDoesNotQuit(t *testing.T) {
	_, s, _, _ := tasksSeedProject(t)
	m := &model{tasks: s, screen: screenTasks}
	selectFirstSubtask(m)
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyTab})
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyCtrlJ})
	if s.mode != taskJournal {
		t.Fatalf("ctrl+j не открыл журнал (mode=%d)", s.mode)
	}
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd != nil {
		t.Error("Esc из модалки журнала вернул команду (выход из приложения)")
	}
	if s.mode != taskBrowse {
		t.Error("Esc не закрыл модалку журнала")
	}
}

// TestQuitConfirmRunningSession — при выходе с запущенным учётом времени
// показывается предупреждение: Enter останавливает подзадачу и выходит,
// Esc отменяет выход.
