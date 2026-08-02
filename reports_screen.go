package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"

	"github.com/kalpamer/tasky/internal/db"
)

// reportPeriod — тип периода отчёта.
type reportPeriod int

const (
	periodToday reportPeriod = iota
	periodYesterday
	periodWeek
	periodMonth
	periodCustom
)

var periodNames = []string{"сегодня", "вчера", "неделя", "месяц"}

// reportConfig — настройки отчёта, общие для экранов «Отчеты» и «Настройки».
type reportConfig struct {
	period         reportPeriod
	projectID      int64 // 0 — все проекты
	includeJournal bool
	saveDir        string
	customFrom     time.Time // границы своего периода (00:00)
	customTo       time.Time // исключая конец (последний день + 24ч)
}

var monthNames = []string{
	"январь", "февраль", "март", "апрель", "май", "июнь",
	"июль", "август", "сентябрь", "октябрь", "ноябрь", "декабрь",
}

var saveOKStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))

// reportsScreen — страница «Отчеты»: общее время за период, задачи и
// подзадачи с учтённым временем (только завершённые записи).
type reportsScreen struct {
	db       *sql.DB
	cfg      *reportConfig
	now      time.Time
	rep      []db.TaskReport
	journal  map[int64][]db.ReportJournalEntry
	total    time.Duration
	repV     viewport.Model
	lastErr  error
	lastSave string
}

func newReportsScreen(conn *sql.DB, cfg *reportConfig) *reportsScreen {
	s := &reportsScreen{db: conn, cfg: cfg, now: time.Now()}
	s.repV = viewport.New(1, 1)
	return s
}

func (s *reportsScreen) load() {
	s.now = time.Now()
	s.lastErr = nil
	s.lastSave = ""
	from, to := s.periodRange()
	entries, err := db.ReportEntries(s.db, from, to, s.cfg.projectID)
	if err != nil {
		s.rep = nil
		s.total = 0
		s.journal = nil
		s.refresh()
		return
	}
	s.rep = db.ReportByTask(entries)
	var total int64
	for _, t := range s.rep {
		total += t.Seconds
	}
	s.total = time.Duration(total) * time.Second
	if s.cfg.includeJournal {
		s.journal = map[int64][]db.ReportJournalEntry{}
		jl, err := db.JournalEntriesByRange(s.db, from, to)
		if err == nil {
			for _, e := range jl {
				s.journal[e.SubtaskID] = append(s.journal[e.SubtaskID], e)
			}
		}
	} else {
		s.journal = nil
	}
	s.refresh()
}

// periodRange возвращает границы периода отчёта [from, to).
func (s *reportsScreen) periodRange() (time.Time, time.Time) {
	now := s.now
	day := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	switch s.cfg.period {
	case periodYesterday:
		return day.AddDate(0, 0, -1), day
	case periodWeek:
		mon := day.AddDate(0, 0, -(int(now.Weekday())+6)%7)
		return mon, mon.AddDate(0, 0, 7)
	case periodMonth:
		mStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		return mStart, mStart.AddDate(0, 1, 0)
	case periodCustom:
		if s.cfg.customFrom.IsZero() {
			return day, day.AddDate(0, 0, 1)
		}
		return s.cfg.customFrom, s.cfg.customTo
	default:
		return day, day.AddDate(0, 0, 1)
	}
}

// periodLabel — заголовок периода для шапки отчёта.
func (s *reportsScreen) periodLabel() string {
	from, to := s.periodRange()
	switch s.cfg.period {
	case periodYesterday:
		return "Отчет за вчера · " + from.Format("02.01.2006")
	case periodWeek:
		return "Отчет за неделю · " + from.Format("02.01") + " – " +
			to.AddDate(0, 0, -1).Format("02.01.2006")
	case periodMonth:
		return "Отчет за " + monthNames[from.Month()-1] + " " + from.Format("2006")
	case periodCustom:
		if s.cfg.customFrom.IsZero() {
			return "Отчет · свой период"
		}
		if s.cfg.customTo.Sub(s.cfg.customFrom) > 24*time.Hour {
			return "Отчет · " + from.Format("02.01.2006") + " – " +
				to.AddDate(0, 0, -1).Format("02.01.2006")
		}
		return "Отчет за день · " + from.Format("02.01.2006")
	default:
		return "Отчет за сегодня · " + from.Format("02.01.2006")
	}
}

// saveFileName — имя файла отчёта: одиночный день — ДД.ММ.ГГГГ, период —
// даты через подчёркивание.
func (s *reportsScreen) saveFileName() string {
	from, to := s.periodRange()
	if s.cfg.period == periodToday || s.cfg.period == periodYesterday {
		return from.Format("2006-01-02") + ".txt"
	}
	if s.cfg.period == periodMonth {
		return from.Format("2006-01") + ".txt"
	}
	if s.cfg.period == periodCustom && s.cfg.customTo.Sub(s.cfg.customFrom) <= 24*time.Hour {
		return from.Format("2006-01-02") + ".txt"
	}
	return from.Format("2006-01-02") + "_" + to.AddDate(0, 0, -1).Format("2006-01-02") + ".txt"
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
				for _, e := range s.journal[st.ID] {
					body = append(body, faint("      ["+e.CreatedAt.Format("02.01 15:04")+"] "+e.Text))
				}
			}
			body = append(body, "")
		}
	}
	s.repV.SetContent(strings.Join(body, "\n"))
}

// save записывает отчёт в файл cfg.saveDir/saveFileName().
func (s *reportsScreen) save() {
	dir := s.cfg.saveDir
	if dir == "" {
		dir = "reports"
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		s.lastErr = err
		s.lastSave = ""
		s.refresh()
		return
	}
	var sb strings.Builder
	sb.WriteString(s.periodLabel() + "\n\n")
	for _, t := range s.rep {
		sb.WriteString(fmt.Sprintf("%s · %s\n", t.TaskTitle,
			fmtDur(time.Duration(t.Seconds)*time.Second)))
		for _, st := range t.Subs {
			sb.WriteString(fmt.Sprintf("  ├ %s · %s\n", st.Title,
				fmtDur(time.Duration(st.Seconds)*time.Second)))
			for _, e := range s.journal[st.ID] {
				sb.WriteString(fmt.Sprintf("      [%s] %s\n", e.CreatedAt.Format("02.01 15:04"), e.Text))
			}
		}
		sb.WriteString("\n")
	}
	sb.WriteString("Общее время: " + fmtDur(s.total) + "\n")

	path := filepath.Join(dir, s.saveFileName())
	if err := os.WriteFile(path, []byte(sb.String()), 0o644); err != nil {
		s.lastErr = err
		s.lastSave = ""
		s.refresh()
		return
	}
	s.lastErr = nil
	s.lastSave = path
	s.refresh()
}

func (s *reportsScreen) resize(w, h int) {
	s.repV.Width = max(w-4, 1)
	s.repV.Height = max(h-6, 1)
}

func (s *reportsScreen) header(w int) string {
	return padW(headerStyle.Render("Tasky")+"  "+faint("Отчеты"), w)
}

func (s *reportsScreen) footer(w int) string {
	return padW(faint("↑/↓ скролл · ctrl+s — сохранить · esc — назад · q — выход"), w)
}

// view — фикс-шапка с периодом и общим временем + статус + скроллируемый
// список задач с подзадачами.
func (s *reportsScreen) view(w, h int) string {
	status := s.statusLine(w)
	statusH := 0
	if status != "" {
		statusH = 1
	}
	s.repV.Height = max(h-6-statusH, 1)
	parts := []string{s.topBox(w)}
	if status != "" {
		parts = append(parts, "", status)
	}
	parts = append(parts, "", boxStyle.Render(s.repV.View()))
	return padH(strings.Join(parts, "\n"), w, h)
}

// statusLine — строка результата последнего сохранения (или пустая).
func (s *reportsScreen) statusLine(w int) string {
	if s.lastSave != "" {
		return padW(saveOKStyle.Render("Отчёт сохранён: "+s.lastSave), w)
	}
	if s.lastErr != nil {
		return padW(errorStyle.Render("Не удалось сохранить: "+s.lastErr.Error()), w)
	}
	return ""
}

func (s *reportsScreen) topBox(w int) string {
	inner := s.periodLabel() + "  " + faint("Общее время: ") + fmtDur(s.total)
	return boxStyle.Render(padLines(inner, max(w-4, 1), 1))
}

func (s *reportsScreen) dialog() (string, bool) { return "", false }
