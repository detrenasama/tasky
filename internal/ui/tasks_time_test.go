package ui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/detrenasama/tasky/internal/db"
)

func TestTimeEntriesRequiresSubtask(t *testing.T) {
	_, s, task, _ := tasksSeedProject(t)
	s.selectByKindID(kindTask, task.ID)
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	if s.mode != taskBrowse {
		t.Fatalf("mode=%v, ожидался taskBrowse (t не должен открывать на задаче)", s.mode)
	}
	if s.notice == "" {
		t.Error("ожидалась подсказка «выберите подзадачу»")
	}
}

func TestTimeEntriesOpenAndEdit(t *testing.T) {
	conn, s, task, st := tasksSeedProject(t)
	start := time.Now().Add(-2 * time.Hour).Truncate(time.Second)
	end := start.Add(time.Hour)
	db.StartSession(conn, st.ID, start)
	db.StopSession(conn, st.ID, end)

	s.expanded[task.ID] = true
	s.buildItems()
	s.selectByKindID(kindSubtask, st.ID)
	s.loadInfo()

	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	if s.mode != taskTimeList {
		t.Fatalf("mode=%v, ожидался taskTimeList", s.mode)
	}
	if len(s.timePick.items) != 1 {
		t.Fatalf("записей в списке: %d, ожидалось 1", len(s.timePick.items))
	}

	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyEnter})
	if s.mode != taskTimeEdit || !s.editHasEnd {
		t.Fatalf("mode=%v editHasEnd=%v, ожидался редактор с концом", s.mode, s.editHasEnd)
	}

	// 4 раза вправо -> поле «минуты» (индекс 4)
	for i := 0; i < 4; i++ {
		s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyRight})
	}
	if s.timeField != 4 {
		t.Fatalf("timeField=%d, ожидалось 4 (минуты)", s.timeField)
	}
	before := s.editStart.Minute()
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyShiftUp})
	if s.editStart.Minute() != (before+5)%60 {
		t.Errorf("минуты после shift+up: %d, ожидалось %d", s.editStart.Minute(), (before+5)%60)
	}

	// обычный ↓ меняет только на 1
	down := s.editStart.Minute()
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyDown})
	if s.editStart.Minute() != (down-1+60)%60 {
		t.Errorf("минуты после ↓: %d, ожидалось %d", s.editStart.Minute(), (down-1+60)%60)
	}

	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyEnter})
	if s.mode != taskTimeList {
		t.Fatalf("после сохранения mode=%v, ожидался taskTimeList", s.mode)
	}
	entries, _ := db.TimeEntriesBySubtask(conn, st.ID)
	if len(entries) != 1 {
		t.Fatalf("записей в БД: %d, ожидалось 1", len(entries))
	}
	if entries[0].StartedAt.Minute() != s.editStart.Minute() {
		t.Errorf("сохранённая минута=%d, ожидалось %d", entries[0].StartedAt.Minute(), s.editStart.Minute())
	}
	// список должен обновиться и показывать новое время
	found := false
	for _, it := range s.timePick.items {
		if it.value == s.editTimeID {
			if !strings.Contains(it.label, fmt.Sprintf("%02d:%02d", s.editStart.Hour(), s.editStart.Minute())) {
				t.Errorf("метка списка не отражает новое время: %q", it.label)
			}
			if s.timePick.sel != 0 {
				t.Errorf("курсор после сохранения sel=%d, ожидалось 0 (выбранная запись)", s.timePick.sel)
			}
			found = true
		}
	}
	if !found {
		t.Error("отредактированная запись не найдена в списке после сохранения")
	}
}

func TestTimeEntriesListNewestFirst(t *testing.T) {
	conn, s, task, st := tasksSeedProject(t)
	old := time.Now().Add(-3 * time.Hour).Truncate(time.Second)
	newT := time.Now().Add(-1 * time.Hour).Truncate(time.Second)
	db.StartSession(conn, st.ID, old)
	db.StopSession(conn, st.ID, old.Add(time.Hour))
	db.StartSession(conn, st.ID, newT)
	db.StopSession(conn, st.ID, newT.Add(time.Hour))

	s.expanded[task.ID] = true
	s.buildItems()
	s.selectByKindID(kindSubtask, st.ID)
	s.loadInfo()

	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	if len(s.timePick.items) != 2 {
		t.Fatalf("записей в списке: %d, ожидалось 2", len(s.timePick.items))
	}
	first := s.timePick.items[0].label
	last := s.timePick.items[len(s.timePick.items)-1].label
	if !strings.Contains(first, fmt.Sprintf("%02d:%02d", newT.Hour(), newT.Minute())) {
		t.Errorf("первая запись не самая новая: %q", first)
	}
	if !strings.Contains(last, fmt.Sprintf("%02d:%02d", old.Hour(), old.Minute())) {
		t.Errorf("последняя запись не самая старая: %q", last)
	}
}

func TestTimeEntriesDelete(t *testing.T) {
	conn, s, task, st := tasksSeedProject(t)
	start := time.Now().Add(-2 * time.Hour).Truncate(time.Second)
	end := start.Add(time.Hour)
	db.StartSession(conn, st.ID, start)
	db.StopSession(conn, st.ID, end)

	s.expanded[task.ID] = true
	s.buildItems()
	s.selectByKindID(kindSubtask, st.ID)
	s.loadInfo()

	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	if s.mode != taskTimeDelete {
		t.Fatalf("mode=%v, ожидался taskTimeDelete", s.mode)
	}
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if s.mode != taskTimeList {
		t.Fatalf("после удаления mode=%v, ожидался taskTimeList", s.mode)
	}
	entries, _ := db.TimeEntriesBySubtask(conn, st.ID)
	if len(entries) != 0 {
		t.Errorf("после удаления записей: %d, ожидалось 0", len(entries))
	}
}

func TestTimeEntriesEditorFormatAndNav(t *testing.T) {
	conn, s, task, st := tasksSeedProject(t)
	start := time.Now().Add(-2 * time.Hour).Truncate(time.Second)
	end := start.Add(time.Hour)
	db.StartSession(conn, st.ID, start)
	db.StopSession(conn, st.ID, end)

	s.expanded[task.ID] = true
	s.buildItems()
	s.selectByKindID(kindSubtask, st.ID)
	s.loadInfo()

	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyEnter})
	if s.mode != taskTimeEdit || !s.editHasEnd {
		t.Fatalf("mode=%v editHasEnd=%v, ожидался редактор с концом", s.mode, s.editHasEnd)
	}
	// формат как в списке: дата через дефис, время через двоеточие
	body := s.renderTimeEdit()
	if !strings.Contains(body, "-") || !strings.Contains(body, ":") {
		t.Errorf("редактор не использует формат «дата-время»: %q", body)
	}

	// 4× вправо -> минуты начала (индекс 4)
	for i := 0; i < 4; i++ {
		s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyRight})
	}
	if s.timeField != 4 {
		t.Fatalf("timeField=%d, ожидалось 4 (минуты начала)", s.timeField)
	}

	// вправо внутри группы начала -> оборачивается в год (0), не перескакивает в конец
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyRight})
	if s.timeField != 0 {
		t.Errorf("timeField=%d, ожидалось 0 (год начала, без перехода в конец)", s.timeField)
	}
	// влево из года начала -> минуты начала (4)
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyLeft})
	if s.timeField != 4 {
		t.Errorf("timeField=%d, ожидалось 4 (минуты начала)", s.timeField)
	}

	// Tab -> минуты конца (9), та же позиция
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyTab})
	if s.timeField != 9 {
		t.Errorf("timeField=%d, ожидалось 9 (минуты конца)", s.timeField)
	}
	// влево внутри группы конца -> часы конца (8)
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyLeft})
	if s.timeField != 8 {
		t.Errorf("timeField=%d, ожидалось 8 (часы конца)", s.timeField)
	}
	// Tab обратно -> часы начала (3, та же позиция «часы»)
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyTab})
	if s.timeField != 3 {
		t.Errorf("timeField=%d, ожидалось 3 (часы начала после Tab)", s.timeField)
	}
}
