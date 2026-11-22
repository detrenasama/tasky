package ui

import (
	"testing"

	"github.com/charmbracelet/bubbletea"
	"github.com/detrenasama/tasky/internal/db"
)

// TestChecklistEscCancelInput — Esc в добавлении пункта чек-листа требует
// второго нажатия и не создаёт пункт.
func TestChecklistEscCancelInput(t *testing.T) {
	conn, s, _, st := tasksSeedProject(t)
	m := &model{tasks: s}
	selectFirstSubtask(m)

	s.updateTasksMsg(key('i'))
	if s.mode != taskChecklist {
		t.Fatalf("i не открыл чек-лист (mode=%d)", s.mode)
	}
	s.updateTasksMsg(key('n'))
	if !s.checklistNew {
		t.Fatal("n не открыл ввод пункта")
	}
	s.checklistInput.SetValue("черновик")

	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyEsc})
	if !s.checklistNew || !s.esc.Armed() {
		t.Fatalf("первый Esc должен предупредить (new=%v armed=%v)", s.checklistNew, s.esc.Armed())
	}
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyEsc})
	if s.checklistNew || s.esc.Armed() {
		t.Fatalf("второй Esc должен отменить ввод (new=%v armed=%v)", s.checklistNew, s.esc.Armed())
	}
	if s.mode != taskChecklist {
		t.Fatalf("отмена ввода не должна закрывать чек-лист (mode=%d)", s.mode)
	}
	if items, _ := db.ChecklistItems(conn, st.ID); len(items) != 0 {
		t.Errorf("отменённый ввод создал пункт: %d", len(items))
	}
}

// TestChecklistEscAfterSaveCloses — после сохранения пункта Esc закрывает
// чек-лист сразу, без предупреждения.
func TestChecklistEscAfterSaveCloses(t *testing.T) {
	conn, s, _, st := tasksSeedProject(t)
	m := &model{tasks: s}
	selectFirstSubtask(m)

	s.updateTasksMsg(key('i'))
	s.updateTasksMsg(key('n'))
	s.checklistInput.SetValue("Пункт")
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyEnter})
	if s.checklistNew {
		t.Fatal("Enter не сохранил пункт")
	}
	items, _ := db.ChecklistItems(conn, st.ID)
	if len(items) != 1 || items[0].Text != "Пункт" {
		t.Fatalf("пункты = %+v", items)
	}
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyEsc})
	if s.mode != taskBrowse {
		t.Fatalf("Esc без изменений должен закрыть чек-лист (mode=%d)", s.mode)
	}
}

// TestChecklistEditUnchangedCloses — правка без изменений отменяется одним Esc.
func TestChecklistEditUnchangedCloses(t *testing.T) {
	conn, s, _, st := tasksSeedProject(t)
	if _, err := db.CreateChecklistItem(conn, st.ID, "Пункт"); err != nil {
		t.Fatal(err)
	}
	m := &model{tasks: s}
	selectFirstSubtask(m)

	s.updateTasksMsg(key('i'))
	s.updateTasksMsg(key('e'))
	if !s.checklistNew || s.checklistInput.Value() != "Пункт" {
		t.Fatalf("e не открыл правку пункта (new=%v value=%q)", s.checklistNew, s.checklistInput.Value())
	}
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyEsc})
	if s.checklistNew {
		t.Fatal("Esc без изменений должен сразу отменить правку")
	}
	if s.mode != taskChecklist {
		t.Fatalf("чек-лист не должен закрываться (mode=%d)", s.mode)
	}
}
