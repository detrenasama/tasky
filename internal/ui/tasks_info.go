package ui

import (
	"fmt"
	"github.com/charmbracelet/lipgloss"
	"github.com/detrenasama/tasky/internal/db"
	"github.com/detrenasama/tasky/internal/ui/theme"
	"strings"
	"time"
)

// infoTop — верхняя часть правой колонки (выбранный элемент): статус,
// записи времени/подзадачи, теги, история статусов. Без панели — обёртку
// добавляет rightRail.
func (s *tasksScreen) infoTop(topH int) string {
	kind, id := s.selectedKindID()
	var body []string
	switch {
	case kind == kindTask && id == 0:
		body = append(body, theme.Faint("Выберите задачу или подзадачу."))
	case kind == kindSubtask:
		var st *db.SubtaskWithTime
		for i := range s.subs {
			if s.subs[i].ID == id {
				st = &s.subs[i]
			}
		}
		if st == nil {
			body = append(body, theme.Faint("Подзадача не найдена."))
			break
		}
		w := max(rightW-4, 1)
		body = append(body, s.parentTaskLines(st, w)...)
		body = append(body, st.Title)
		body = append(body, theme.Faint("Статус: ")+s.statusText(st.Status))
		if line, ok := s.dueLine(st.DueDate, st.Status); ok {
			body = append(body, line)
		}
		if st.ScheduleDate != nil {
			body = append(body, theme.Faint("Запланировано: ")+st.ScheduleDate.Format("02.01.2006"))
		}
		total := time.Duration(st.TotalSeconds) * time.Second
		if st.ActiveSince != nil {
			total += s.now.Sub(time.Unix(*st.ActiveSince, 0))
		}
		body = append(body, theme.Faint("Время всего: ")+fmtDur(total))
		body = append(body, "")
		if len(s.entries) == 0 {
			body = append(body, theme.Faint("Записей нет."))
		} else {
			if len(s.entries) > 8 {
				body = append(body, theme.Faint(fmt.Sprintf("... еще %d записей", len(s.entries)-8)))
			}
			ents := s.entries
			if len(ents) > 8 {
				ents = ents[len(ents)-8:]
			}
			for _, e := range ents {
				body = append(body, entryLine(e, s.now))
			}
		}
		if len(s.checklistItems) > 0 {
			body = append(body, "", theme.Faint("Чеклист:"))
			visible := visibleChecklist(s.checklistItems, s.now)
			if len(visible) == 0 {
				body = append(body, theme.Faint("нет активных"))
			} else {
				for _, ci := range visible {
					line := "  " + checkMark(ci.Status) + " " + ci.Text
					body = append(body, lipgloss.NewStyle().Foreground(lipgloss.Color(checkColor(ci.Status))).Render(line))
				}
			}
		}
		body = append(body, s.historyLines()...)
	case kind == kindTask:
		t := s.taskByID(id)
		if t == nil {
			body = append(body, theme.Faint("Задача не найдена."))
			break
		}
		body = append(body, t.Title)
		body = append(body, theme.Faint("Статус: ")+s.statusText(t.Status))
		if line, ok := s.dueLine(t.DueDate, t.Status); ok {
			body = append(body, line)
		}
		if t.ScheduleDate != nil {
			body = append(body, theme.Faint("Запланировано: ")+t.ScheduleDate.Format("02.01.2006"))
		}
		plural := "подзадач"
		if t.SubCount == 1 {
			plural = "подзадача"
		}
		var sum time.Duration
		for _, st := range s.subs {
			if st.TaskID != t.ID {
				continue
			}
			d := time.Duration(st.TotalSeconds) * time.Second
			if st.ActiveSince != nil {
				d += s.now.Sub(time.Unix(*st.ActiveSince, 0))
			}
			sum += d
			body = append(body, "  ├ "+st.Title+" · "+fmtDur(d))
		}
		body = append(body, theme.Faint(fmt.Sprintf("%d %s, всего: %s", t.SubCount, plural, fmtDur(sum))))
		if len(s.tagsMap[t.ID]) > 0 {
			body = append(body, "", theme.Faint("Теги:"))
			for _, tg := range s.tagsMap[t.ID] {
				line := "  " + tagChip(tg)
				if tg.URL != "" {
					line += " " + theme.Faint(tg.URL)
				}
				body = append(body, line)
			}
		}
		body = append(body, s.historyLines()...)
	default:
		body = append(body, theme.Faint("Выберите задачу или подзадачу."))
	}
	inner := strings.Join(body, "\n")
	inner = padLines(inner, max(rightW-4, 1), topH)
	return inner
}

// taskByID возвращает задачу текущего проекта по ID (задачи загружаются
// целиком, включая скрытые фильтром), nil — если не найдена.
func (s *tasksScreen) taskByID(id int64) *db.Task {
	for i := range s.tasks {
		if s.tasks[i].ID == id {
			return &s.tasks[i]
		}
	}
	return nil
}

// dueLine — строка срока: обычная или красная с пометкой «просрочено»;
// ok=false — срок не задан.
func (s *tasksScreen) dueLine(due *time.Time, status string) (string, bool) {
	if due == nil {
		return "", false
	}
	dStr := due.Format("02.01.2006")
	if isOverdue(*due, status, s.now) {
		return theme.ErrorStyle.Render("Срок: " + dStr + " (просрочено)"), true
	}
	return theme.Faint("Срок: ") + dStr, true
}

// parentTaskLines — контекст родительской задачи над выбранной подзадачей:
// название, теги, статус, срок и разделитель. Родитель берётся из s.tasks,
// поэтому блок виден, даже когда строка задачи скрыта фильтром или поиском.
func (s *tasksScreen) parentTaskLines(st *db.SubtaskWithTime, w int) []string {
	t := s.taskByID(st.TaskID)
	if t == nil {
		return nil
	}
	body := titleLines("Задача:", t.Title, w)
	if tags := s.tagsMap[t.ID]; len(tags) > 0 {
		body = append(body, tagsLines("Теги:", tags, w)...)
	}
	body = append(body, theme.Faint("Статус: ")+s.statusText(t.Status))
	if line, ok := s.dueLine(t.DueDate, t.Status); ok {
		body = append(body, line)
	}
	body = append(body, theme.Faint(strings.Repeat("─", w)))
	return body
}

// titleLines — строка «метка: текст» с переносом текста по словам; продолжения
// выравниваются под текстом (отступ на ширину метки с двоеточием).
func titleLines(label, title string, w int) []string {
	indentW := lipgloss.Width(label) + 1
	indent := strings.Repeat(" ", indentW)
	lines := strings.Split(wrapText(title, max(w-indentW, 8)), "\n")
	out := make([]string, 0, len(lines))
	for i, ln := range lines {
		prefix := indent
		if i == 0 {
			prefix = theme.Faint(label) + " "
		}
		out = append(out, prefix+ln)
	}
	return out
}

// historyLines — последние 8 переходов статусов выбранного элемента с индикатором
// скрытых записей вверху (как у записей времени): штамп отдельной строкой, сам
// переход и заметка wrapText.
func (s *tasksScreen) historyLines() []string {
	var body []string
	if len(s.history) == 0 {
		return body
	}
	body = append(body, "", theme.Faint("История статусов:"))
	if len(s.history) > 8 {
		body = append(body, theme.Faint(fmt.Sprintf("... еще %d записей", len(s.history)-8)))
	}
	start := 0
	if len(s.history) > 8 {
		start = len(s.history) - 8
	}
	w := max(rightW-4, 1)
	for _, h := range s.history[start:] {
		body = append(body, theme.Faint(h.CreatedAt.Format("2006-01-02 15:04")))
		body = append(body, wrapText(h.From+" → "+h.To, w))
		if h.Note != "" {
			body = append(body, wrapText("      "+h.Note, w))
		}
	}
	return body
}

// visibleChecklist фильтрует чек-лист для правой панели: new/in_progress всегда,
// done/cancelled — только если статус изменён сегодня или вчера (локальное время).
func visibleChecklist(items []db.ChecklistItem, now time.Time) []db.ChecklistItem {
	var out []db.ChecklistItem
	for _, ci := range items {
		if ci.Status != "done" && ci.Status != "cancelled" {
			out = append(out, ci)
			continue
		}
		if sameDay(ci.StatusChangedAt, now) || sameDay(ci.StatusChangedAt, now.AddDate(0, 0, -1)) {
			out = append(out, ci)
			continue
		}
		// будущее время (часы ушли вперёд) — считаем видимым
		if ci.StatusChangedAt.After(time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location())) {
			out = append(out, ci)
		}
	}
	return out
}

func entryLine(e db.TimeEntry, now time.Time) string {
	start := e.StartedAt.Format("15:04")
	if e.EndedAt == nil {
		return start + "–… · " + theme.Faint("идет "+fmtElapsed(now.Sub(e.StartedAt)))
	}
	d := time.Duration(e.EndedAt.Sub(e.StartedAt))
	return start + "–" + e.EndedAt.Format("15:04") + " · " + fmtDur(d)
}

func isOverdue(due time.Time, status string, now time.Time) bool {
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	dueDay := time.Date(due.Year(), due.Month(), due.Day(), 0, 0, 0, 0, due.Location())
	return dueDay.Before(today)
}

func (s *tasksScreen) infoBottom() string {
	body := []string{
		theme.Faint("За сегодня: ") + fmtDur(s.today),
		theme.Faint("Неделя (Пн–Вс): ") + fmtDur(s.weekly),
		"",
	}
	if s.run != nil {
		elapsed := s.now.Sub(time.Unix(*s.run.ActiveSince, 0))
		body = append(body, "Сейчас: "+s.run.Title)
		body = append(body, theme.Faint("идет "+fmtElapsed(elapsed)))
	} else {
		body = append(body, theme.Faint("Ничего не запущено."))
	}
	for i := range body {
		body[i] = padW(body[i], max(rightW-4, 1))
	}
	return strings.Join(body, "\n")
}
