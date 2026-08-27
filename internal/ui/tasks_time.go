package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/detrenasama/tasky/internal/db"
	"github.com/detrenasama/tasky/internal/ui/theme"
)

// timeEntryLabel формирует строку списка записей времени:
// «[начало] -- [конец] => ДЛИТЕЛЬНОСТЬ» (для активной — «… (идёт)»).
func timeEntryLabel(e db.TimeEntry) string {
	label := fmtTimeEntry(e.StartedAt)
	if e.EndedAt != nil {
		label += " -- " + fmtTimeEntry(*e.EndedAt) +
			" => " + fmtDurationHM(e.EndedAt.Sub(e.StartedAt))
	} else {
		label += " -- … (идёт)"
	}
	return label
}

// openTimeEntries открывает модалку списка записей времени выбранной
// подзадачи. Если выбрана задача (не подзадача) — показывает подсказку.
func (s *tasksScreen) openTimeEntries() {
	if s.selectedKind() != kindSubtask {
		s.notice = "Выберите подзадачу для редактирования времени"
		return
	}
	_, id := s.selectedKindID()
	entries, err := s.store.TimeEntriesBySubtask(id)
	if err != nil {
		s.lastErr = err
		return
	}
	s.entries = entries
	s.rebuildTimePick()
	s.lastErr = nil
	s.notice = ""
	s.mode = taskTimeList
}

// rebuildTimePick пересобирает элементы списка записей времени из s.entries.
func (s *tasksScreen) rebuildTimePick() {
	items := make([]pickItem, 0, len(s.entries))
	for i := len(s.entries) - 1; i >= 0; i-- {
		e := s.entries[i]
		items = append(items, pickItem{value: e.ID, label: timeEntryLabel(e)})
	}
	s.timePick.items = items
	s.timePick.setVisible(max(4, min(s.midH-8, 12)))
	if s.timePick.sel >= len(items) {
		s.timePick.sel = max(len(items)-1, 0)
	}
	s.timePick.clampScroll()
}

// startTimeEdit открывает модалку редактирования выбранной записи времени.
func (s *tasksScreen) startTimeEdit() {
	it, ok := s.timePick.selected()
	if !ok {
		return
	}
	var entry *db.TimeEntry
	for i := range s.entries {
		if s.entries[i].ID == it.value {
			entry = &s.entries[i]
			break
		}
	}
	if entry == nil {
		return
	}
	s.editTimeID = entry.ID
	s.editStart = entry.StartedAt
	if entry.EndedAt != nil {
		e := *entry.EndedAt
		s.editEnd = &e
		s.editHasEnd = true
	} else {
		s.editEnd = nil
		s.editHasEnd = false
	}
	s.timeField = 0
	s.lastErr = nil
	s.mode = taskTimeEdit
}

// startTimeDelete запрашивает подтверждение удаления выбранной записи.
func (s *tasksScreen) startTimeDelete() {
	it, ok := s.timePick.selected()
	if !ok {
		return
	}
	s.confirmTimeID = it.value
	s.mode = taskTimeDelete
}

// adjustTimeField меняет выбранный компонент (год/месяц/день/часы/минуты)
// начала (поля 0..4) или конца (поля 5..9) на delta. time.Date сам
// нормализует переполнение (напр. день 32 → следующий месяц). Секунды
// сохраняются как были.
func (s *tasksScreen) adjustTimeField(field, delta int) {
	if field < 5 {
		s.editStart = adjustComponent(s.editStart, field, delta)
		return
	}
	if s.editHasEnd && s.editEnd != nil {
		e := adjustComponent(*s.editEnd, field-5, delta)
		s.editEnd = &e
	}
}

// adjustComponent возвращает время t с изменённым компонентом field (0 год,
// 1 месяц, 2 день, 3 часы, 4 минуты) на delta; секунды сохраняются.
func adjustComponent(t time.Time, field, delta int) time.Time {
	y, mo, d := t.Date()
	h, mi, sec := t.Clock()
	switch field {
	case 0:
		y += delta
	case 1:
		mo += time.Month(delta)
	case 2:
		d += delta
	case 3:
		h += delta
	case 4:
		mi += delta
	}
	return time.Date(y, mo, d, h, mi, sec, 0, t.Location())
}

// shiftDelta — шаг для shift+↑/↓: ±5 для минут, иначе ±1.
func (s *tasksScreen) shiftDelta() int {
	if s.timeField == 4 || s.timeField == 9 {
		return 5
	}
	return 1
}

// renderTimeField рендерит один момент времени с подсветкой активного поля
// (baseField в 0..4 или -1, если это не текущая редактируемая метка).
func (s *tasksScreen) renderTimeField(t time.Time, baseField int) string {
	y, mo, d := t.Date()
	h, mi, _ := t.Clock()
	parts := []struct {
		text  string
		field int
	}{
		{fmt.Sprintf("%04d", y), 0},
		{"-", -2},
		{fmt.Sprintf("%02d", int(mo)), 1},
		{"-", -2},
		{fmt.Sprintf("%02d", d), 2},
		{" " + ruWeekdayShort(t) + " ", -2},
		{fmt.Sprintf("%02d", h), 3},
		{":", -2},
		{fmt.Sprintf("%02d", mi), 4},
	}
	var b strings.Builder
	b.WriteString("[")
	for _, p := range parts {
		if baseField >= 0 && p.field == baseField {
			b.WriteString(theme.HeaderStyle.Render(p.text))
		} else {
			b.WriteString(p.text)
		}
	}
	b.WriteString("]")
	return b.String()
}

// renderTimeEdit строит тело модалки редактирования: начало, конец и
// пересчитываемая длительность.
func (s *tasksScreen) renderTimeEdit() string {
	startField, endField := -1, -1
	if s.timeField < 5 {
		startField = s.timeField
	} else {
		endField = s.timeField - 5
	}
	var b strings.Builder
	b.WriteString("Начало:  " + s.renderTimeField(s.editStart, startField) + "\n")
	if s.editHasEnd && s.editEnd != nil {
		b.WriteString("Конец:   " + s.renderTimeField(*s.editEnd, endField) + "\n")
		b.WriteString("Длительность: " + fmtDurationHM(s.editEnd.Sub(s.editStart)))
	} else {
		b.WriteString("Конец:   — (запись идёт)\n")
		b.WriteString("Длительность: —")
	}
	if s.lastErr != nil {
		b.WriteString("\n\n" + theme.ErrorStyle.Render("Ошибка: "+s.lastErr.Error()))
	}
	return b.String()
}
