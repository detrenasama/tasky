package main

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"

	"github.com/kalpamer/tasky/internal/db"
)

// reportPeriod — тип периода отчёта (этап 2 добавит вчера/неделю/месяц).
type reportPeriod int

const (
	periodToday reportPeriod = iota
	periodYesterday
	periodWeek
	periodMonth
)

// reportConfig — настройки отчёта. На этапе 1 используется только periodToday;
// остальные поля задействуются на этапах 2–3.
type reportConfig struct {
	period         reportPeriod
	projectID      int64 // 0 — все проекты
	includeJournal bool
	saveDir        string
}

// reportsScreen — страница «Отчеты»: общее время за период, задачи и
// подзадачи с учтённым временем (только завершённые записи).
type reportsScreen struct {
	db    *sql.DB
	cfg   reportConfig
	now   time.Time
	rep   []db.TaskReport
	total time.Duration
	repV  viewport.Model

	midH int
}

func newReportsScreen(conn *sql.DB) *reportsScreen {
	s := &reportsScreen{db: conn, cfg: reportConfig{period: periodToday}, now: time.Now()}
	s.repV = viewport.New(1, 1)
	return s
}

func (s *reportsScreen) load() {
	s.now = time.Now()
	from, to := s.periodRange()
	entries, err := db.ReportEntries(s.db, from, to, s.cfg.projectID)
	if err != nil {
		s.rep = nil
		s.total = 0
		s.refresh()
		return
	}
	s.rep = db.ReportByTask(entries)
	var total int64
	for _, t := range s.rep {
		total += t.Seconds
	}
	s.total = time.Duration(total) * time.Second
	s.refresh()
}

// periodRange возвращает границы периода отчёта [from, to) — на этапе 1
// только «сегодня».
func (s *reportsScreen) periodRange() (time.Time, time.Time) {
	now := s.now
	day := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	return day, day.AddDate(0, 0, 1)
}

// periodLabel — заголовок периода для шапки отчёта.
func (s *reportsScreen) periodLabel() string {
	return "Отчет за сегодня · " + s.now.Format("02.01.2006")
}

// refresh пересобирает контент viewport отчёта.
func (s *reportsScreen) refresh() {
	var body []string
	if s.total == 0 {
		body = append(body, faint("Времени за период ещё не учтено."))
	} else {
		for _, t := range s.rep {
			body = append(body, fmt.Sprintf("%s · %s", t.TaskTitle,
				fmtDur(time.Duration(t.Seconds)*time.Second)))
			for _, st := range t.Subs {
				body = append(body, "  ├ "+st.Title+" · "+fmtDur(time.Duration(st.Seconds)*time.Second))
			}
			body = append(body, "")
		}
	}
	s.repV.SetContent(strings.Join(body, "\n"))
}

func (s *reportsScreen) resize(w, h int) {
	s.midH = max(h, 3)
	s.repV.Width = max(w-4, 1)
	s.repV.Height = max(h-6, 1)
	s.refresh()
}

func (s *reportsScreen) header(w int) string {
	return padW(headerStyle.Render("Tasky")+"  "+faint("Отчеты"), w)
}

func (s *reportsScreen) footer(w int) string {
	return padW(faint("↑/↓ скролл · esc — назад · q — выход"), w)
}

// view — фикс-шапка с периодом и общим временем + скроллируемый список
// задач с подзадачами.
func (s *reportsScreen) view(w, h int) string {
	top := s.topBox(w)
	body := top + "\n\n" + boxStyle.Render(s.repV.View())
	return padH(body, w, h)
}

func (s *reportsScreen) topBox(w int) string {
	inner := s.periodLabel() + "  " + faint("Общее время: ") + fmtDur(s.total)
	return boxStyle.Render(padLines(inner, max(w-4, 1), 1))
}

func (s *reportsScreen) dialog() (string, bool) { return "", false }
