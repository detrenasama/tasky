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
		body = append(body, st.Title)
		body = append(body, theme.Faint("Статус: ")+s.statusText(st.Status))
		total := time.Duration(st.TotalSeconds) * time.Second
		if st.ActiveSince != nil {
			total += s.now.Sub(time.Unix(*st.ActiveSince, 0))
		}
		body = append(body, theme.Faint("Время всего: ")+fmtDur(total))
		body = append(body, "")
		for _, e := range s.entries {
			body = append(body, entryLine(e, s.now))
		}
		if len(s.entries) == 0 {
			body = append(body, theme.Faint("Записей нет."))
		}
		if len(s.checklistItems) > 0 {
			body = append(body, "", theme.Faint("Чеклист:"))
			for _, ci := range s.checklistItems {
				line := "  " + checkMark(ci.Status) + " " + ci.Text
				body = append(body, lipgloss.NewStyle().Foreground(lipgloss.Color(checkColor(ci.Status))).Render(line))
			}
		}
		body = append(body, s.historyLines()...)
	case kind == kindTask:
		var t *db.Task
		for i := range s.tasks {
			if s.tasks[i].ID == id {
				t = &s.tasks[i]
			}
		}
		if t == nil {
			body = append(body, theme.Faint("Задача не найдена."))
			break
		}
		body = append(body, t.Title)
		body = append(body, theme.Faint("Статус: ")+s.statusText(t.Status))
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

// historyLines — последние 6 переходов статусов выбранного элемента: штамп
// дня отдельной строкой, сам переход и заметка wrapText.
func (s *tasksScreen) historyLines() []string {
	var body []string
	if len(s.history) == 0 {
		return body
	}
	body = append(body, "", theme.Faint("История статусов:"))
	start := 0
	if len(s.history) > 6 {
		start = len(s.history) - 6
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
func entryLine(e db.TimeEntry, now time.Time) string {
	start := e.StartedAt.Format("15:04")
	if e.EndedAt == nil {
		return start + "–… · " + theme.Faint("идет "+fmtElapsed(now.Sub(e.StartedAt)))
	}
	d := time.Duration(e.EndedAt.Sub(e.StartedAt))
	return start + "–" + e.EndedAt.Format("15:04") + " · " + fmtDur(d)
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
