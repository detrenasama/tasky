package ui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/detrenasama/tasky/internal/db"
	"time"
)

// statusDef возвращает определение статуса по имени.
func (s *tasksScreen) statusDef(name string) (db.StatusDef, bool) {
	for _, st := range s.statuses {
		if st.Name == name {
			return st, true
		}
	}
	return db.StatusDef{}, false
}

// statusColor возвращает цвет статуса (серый для неизвестных).
func (s *tasksScreen) statusColor(name string) string {
	if st, ok := s.statusDef(name); ok {
		return st.Color
	}
	return "#8a8a8a"
}

// statusText — название статуса, окрашенное его цветом.
func (s *tasksScreen) statusText(name string) string {
	return lipgloss.NewStyle().Foreground(lipgloss.Color(s.statusColor(name))).Render(name)
}

// statusBar — цветная полоса слева от элемента списка.
func statusBar(color string) string {
	return lipgloss.NewStyle().Background(lipgloss.Color(color)).Render(" ")
}

// quickStatuses — статусы быстрой цепочки в порядке сортировки.
func (s *tasksScreen) quickStatuses() []db.StatusDef {
	var out []db.StatusDef
	for _, st := range s.statuses {
		if st.IsQuick {
			out = append(out, st)
		}
	}
	return out
}

// currentStatusName — статус выбранного элемента.
func (s *tasksScreen) currentStatusName(kind paneKind, id int64) string {
	if kind == kindTask {
		for _, t := range s.tasks {
			if t.ID == id {
				return t.Status
			}
		}
		return ""
	}
	for _, st := range s.subs {
		if st.ID == id {
			return st.Status
		}
	}
	return ""
}

// shiftStatus двигает статус по быстрой цепочке (dir = ±1). Из «Выполнена»
// вперёд переходов нет (без зацикливания); статус вне цепочки прыгает на
// её первый/последний элемент.
func (s *tasksScreen) shiftStatus(dir int) {
	kind, id := s.selectedKindID()
	if id == 0 {
		return
	}
	qs := s.quickStatuses()
	if len(qs) == 0 {
		return
	}
	cur := s.currentStatusName(kind, id)
	idx := -1
	for i, st := range qs {
		if st.Name == cur {
			idx = i
			break
		}
	}
	target := -1
	if idx >= 0 {
		target = idx + dir
	} else if dir > 0 {
		target = 0
	} else {
		target = len(qs) - 1
	}
	if target < 0 || target >= len(qs) {
		return
	}
	s.applyStatus(kind, id, qs[target])
}

// applyStatus применяет статус; при обязательной заметке открывает модалку
// ввода (statusTarget запоминается до сохранения).
func (s *tasksScreen) applyStatus(kind paneKind, id int64, st db.StatusDef) {
	s.statusKind, s.statusID = kind, id
	if st.NotePrompt != "" {
		t := st
		s.statusTarget = &t
		s.statusNote.SetValue("")
		s.statusNote.Focus()
		s.mode = taskStatusNote
		return
	}
	s.statusTarget = nil
	if err := s.store.SetStatus(dbOwner(kind), id, st.Name, "", time.Now()); err == nil {
		s.now = time.Now()
		s.loadData()
	} else {
		s.lastErr = err
	}
	s.mode = taskBrowse
}

func dbOwner(kind paneKind) db.StatusOwner {
	if kind == kindSubtask {
		return db.OwnerSubtask
	}
	return db.OwnerTask
}

// openStatusPick открывает модалку выбора статуса для выбранного элемента.
func (s *tasksScreen) openStatusPick() {
	kind, id := s.selectedKindID()
	if id == 0 {
		return
	}
	cur := s.currentStatusName(kind, id)
	s.statusKind, s.statusID = kind, id
	s.statusPick.sel = 0
	for i, it := range s.statusPick.items {
		if it.label == cur {
			s.statusPick.sel = i
		}
	}
	s.statusPick.clampScroll()
	s.mode = taskStatusPick
}
