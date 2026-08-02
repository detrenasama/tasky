package main

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbletea"

	"github.com/kalpamer/tasky/internal/db"
)

type settingsMode int

const (
	settingsBrowse settingsMode = iota
	settingsDirInput
	settingsProjList
	settingsPeriodList
	settingsPeriodInput
)

// settingsScreen — страница «Настройки»: настройки отчёта (период, фильтр
// проекта, журнал, каталог сохранения). Значения пишутся в общий
// reportConfig экрана отчётов.
type settingsScreen struct {
	db          *sql.DB
	cfg         *reportConfig
	mode        settingsMode
	sel         int
	projects    []db.Project
	dirInput    textinput.Model
	periodInput textinput.Model
	projPick    pickList
	periodPick  pickList
	lastErr     error

	midH int
}

// pickItem — элемент списка выбора в модалках настроек (value — id проекта
// или код периода).
type pickItem struct {
	value int64
	label string
}

// pickList — простой скроллируемый список: курсор ▸, при выходе за видимую
// область содержимое плавно прокручивается (без постраничных прыжков).
type pickList struct {
	items   []pickItem
	sel     int
	scroll  int
	visible int
}

func (p *pickList) setVisible(n int) {
	p.visible = max(n, 1)
	if len(p.items) > 0 && p.sel >= len(p.items) {
		p.sel = len(p.items) - 1
	}
	p.clampScroll()
}

func (p *pickList) clampScroll() {
	if p.sel < p.scroll {
		p.scroll = p.sel
	}
	if p.sel >= p.scroll+p.visible {
		p.scroll = p.sel - p.visible + 1
	}
	if p.scroll < 0 {
		p.scroll = 0
	}
}

// move сдвигает курсор по кругу и прокручивает список при необходимости.
func (p *pickList) move(d int) {
	n := len(p.items)
	if n == 0 {
		return
	}
	p.sel = (p.sel + d + n) % n
	p.clampScroll()
}

func (p *pickList) view() string {
	var lines []string
	end := min(len(p.items), p.scroll+p.visible)
	for i := p.scroll; i < end; i++ {
		label := "  " + p.items[i].label
		if i == p.sel {
			label = "▸ " + p.items[i].label
		}
		lines = append(lines, label)
	}
	return strings.Join(lines, "\n")
}

func (p *pickList) selected() (pickItem, bool) {
	if p.sel < 0 || p.sel >= len(p.items) {
		return pickItem{}, false
	}
	return p.items[p.sel], true
}

func newSettingsScreen(conn *sql.DB, cfg *reportConfig) *settingsScreen {
	ti := textinput.New()
	ti.Placeholder = "reports"
	pi := textinput.New()
	pi.Placeholder = "02.08.2026 или 01.08.2026-05.08.2026"

	s := &settingsScreen{db: conn, cfg: cfg, dirInput: ti, periodInput: pi}
	s.projPick.setVisible(12)
	s.periodPick.setVisible(12)
	items := make([]pickItem, 0, len(periodNames)+1)
	for i, name := range periodNames {
		items = append(items, pickItem{value: int64(i), label: name})
	}
	items = append(items, pickItem{value: int64(periodCustom), label: "свой…"})
	s.periodPick.items = items
	s.load()
	return s
}

func (s *settingsScreen) load() {
	s.projects, _ = db.Projects(s.db)
	if s.cfg.projectID != 0 {
		found := false
		for _, p := range s.projects {
			if p.ID == s.cfg.projectID {
				found = true
			}
		}
		if !found {
			s.cfg.projectID = 0
		}
	}
	items := make([]pickItem, 0, len(s.projects)+1)
	items = append(items, pickItem{value: 0, label: "все проекты"})
	for _, p := range s.projects {
		items = append(items, pickItem{value: p.ID, label: p.Name})
	}
	s.projPick.items = items
}

func (s *settingsScreen) resize(w, h int) {
	s.midH = h
	visible := max(4, min(h-8, 12))
	s.projPick.setVisible(visible)
	s.periodPick.setVisible(visible)
}

func (s *settingsScreen) header(w int) string {
	return padW(headerStyle.Render("Tasky")+"  "+faint("Настройки"), w)
}

func (s *settingsScreen) footer(w int) string {
	return padW(faint("↑/↓ — выбор · Enter — изменить · esc — назад"), w)
}

// projectName — название проекта фильтра или «все проекты».
func (s *settingsScreen) projectName() string {
	if s.cfg.projectID == 0 {
		return "все проекты"
	}
	for _, p := range s.projects {
		if p.ID == s.cfg.projectID {
			return p.Name
		}
	}
	return "все проекты"
}

// periodName — текущий период для строки настроек.
func (s *settingsScreen) periodName() string {
	if s.cfg.period != periodCustom {
		return periodNames[s.cfg.period]
	}
	if s.cfg.customFrom.IsZero() {
		return "свой…"
	}
	if s.cfg.customTo.Sub(s.cfg.customFrom) > 24*time.Hour {
		return "свой · " + s.cfg.customFrom.Format("02.01.2006") + " – " +
			s.cfg.customTo.AddDate(0, 0, -1).Format("02.01.2006")
	}
	return "свой · " + s.cfg.customFrom.Format("02.01.2006")
}

func (s *settingsScreen) dirName() string {
	if s.cfg.saveDir == "" {
		return "reports"
	}
	return s.cfg.saveDir
}

func (s *settingsScreen) view(w, h int) string {
	rows := []string{
		"Период:  " + s.periodName(),
		"Проект:  " + s.projectName(),
		"Журнал:  " + boolWord(s.cfg.includeJournal),
		"Каталог: " + s.dirName(),
	}
	var lines []string
	for i, r := range rows {
		if i == s.sel {
			lines = append(lines, headerStyle.Render("▸ "+r))
		} else {
			lines = append(lines, "  "+r)
		}
	}
	inner := headerStyle.Render("Настройки отчёта") + "\n\n" +
		strings.Join(lines, "\n") + "\n\n" +
		faint("Enter — выбор из списка (журнал — вкл/выкл,")
	inner += "\n" + faint("каталог — ввод пути)")
	return boxStyle.Render(padLines(inner, max(w-4, 1), max(h-4, 1)))
}

func (s *settingsScreen) dialog() (string, bool) {
	switch s.mode {
	case settingsDirInput:
		d := dialog{title: "Каталог сохранения отчётов",
			body:    s.dirInput.View(),
			primary: "Enter — сохранить", esc: "Esc — отмена"}
		return d.render(), true
	case settingsProjList:
		d := dialog{title: "Фильтр по проекту",
			body:    s.projPick.view(),
			primary: "Enter — выбрать", esc: "Esc — отмена"}
		return d.render(), true
	case settingsPeriodList:
		d := dialog{title: "Период отчёта",
			body:    s.periodPick.view(),
			primary: "Enter — выбрать", esc: "Esc — отмена"}
		return d.render(), true
	case settingsPeriodInput:
		body := s.periodInput.View()
		if s.lastErr != nil {
			body += "\n\n" + errorStyle.Render("Ошибка: "+s.lastErr.Error())
		}
		d := dialog{title: "Свой период",
			body:    body,
			primary: "Enter — применить", esc: "Esc — отмена"}
		return d.render(), true
	}
	return "", false
}

// parseCustomPeriod разбирает «ДД.ММ.ГГГГ» или «ДД.ММ.ГГГГ–ДД.ММ.ГГГГ» в
// границы [from, to). Разделитель — «-», «–» или «—».
func parseCustomPeriod(v string) (time.Time, time.Time, error) {
	v = strings.TrimSpace(v)
	v = strings.ReplaceAll(v, "–", "-")
	v = strings.ReplaceAll(v, "—", "-")
	parseDay := func(s string) (time.Time, error) {
		return time.ParseInLocation("2.1.2006", strings.TrimSpace(s), time.Local)
	}
	dayStart := func(t time.Time) time.Time {
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	}
	if i := strings.IndexByte(v, '-'); i >= 0 {
		f, err := parseDay(v[:i])
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("неверная дата «%s», нужен формат ДД.ММ.ГГГГ", strings.TrimSpace(v[:i]))
		}
		t, err := parseDay(v[i+1:])
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("неверная дата «%s», нужен формат ДД.ММ.ГГГГ", strings.TrimSpace(v[i+1:]))
		}
		if t.Before(f) {
			return time.Time{}, time.Time{}, fmt.Errorf("начало периода позже конца")
		}
		return dayStart(f), dayStart(t).AddDate(0, 0, 1), nil
	}
	d, err := parseDay(v)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("неверная дата «%s», нужен формат ДД.ММ.ГГГГ", v)
	}
	return dayStart(d), dayStart(d).AddDate(0, 0, 1), nil
}

// customInputValue — подсказка своего периода из текущих границ.
func (s *settingsScreen) customInputValue() string {
	if s.cfg.customFrom.IsZero() {
		return ""
	}
	if s.cfg.customTo.Sub(s.cfg.customFrom) > 24*time.Hour {
		return s.cfg.customFrom.Format("02.01.2006") + "-" +
			s.cfg.customTo.AddDate(0, 0, -1).Format("02.01.2006")
	}
	return s.cfg.customFrom.Format("02.01.2006")
}

// openPeriodPick открывает модалку периода, курсор — на текущем значении.
func (s *settingsScreen) openPeriodPick() {
	s.periodPick.sel = int(s.cfg.period)
	if s.periodPick.sel >= len(s.periodPick.items) {
		s.periodPick.sel = len(s.periodPick.items) - 1
	}
	s.periodPick.clampScroll()
	s.mode = settingsPeriodList
}

// openProjPick открывает модалку проекта, курсор — на текущем фильтре.
func (s *settingsScreen) openProjPick() {
	s.projPick.sel = 0
	for i, it := range s.projPick.items {
		if it.value == s.cfg.projectID {
			s.projPick.sel = i
		}
	}
	s.projPick.clampScroll()
	s.mode = settingsProjList
}

// updateSettings обрабатывает клавиши страницы настроек (включая модалки
// выбора и ввода).
func (m *model) updateSettings(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	s := m.settings
	switch s.mode {
	case settingsDirInput:
		s.dirInput, _ = s.dirInput.Update(msg)
		switch msg.String() {
		case "enter":
			v := strings.TrimSpace(s.dirInput.Value())
			if v == "" {
				v = "reports"
			}
			s.cfg.saveDir = v
			s.mode = settingsBrowse
		case "esc":
			s.mode = settingsBrowse
		}
		return m, nil
	case settingsProjList:
		switch msg.String() {
		case "up":
			s.projPick.move(-1)
		case "down":
			s.projPick.move(1)
		case "pgup":
			s.projPick.move(-s.projPick.visible)
		case "pgdown":
			s.projPick.move(s.projPick.visible)
		case "enter":
			if it, ok := s.projPick.selected(); ok {
				s.cfg.projectID = it.value
			}
			s.mode = settingsBrowse
		case "esc":
			s.mode = settingsBrowse
		}
		return m, nil
	case settingsPeriodList:
		switch msg.String() {
		case "up":
			s.periodPick.move(-1)
		case "down":
			s.periodPick.move(1)
		case "pgup":
			s.periodPick.move(-s.periodPick.visible)
		case "pgdown":
			s.periodPick.move(s.periodPick.visible)
		case "enter":
			if it, ok := s.periodPick.selected(); ok {
				if reportPeriod(it.value) == periodCustom {
					s.periodInput.SetValue(s.customInputValue())
					s.periodInput.Focus()
					s.lastErr = nil
					s.mode = settingsPeriodInput
				} else {
					s.cfg.period = reportPeriod(it.value)
					s.mode = settingsBrowse
				}
			}
		case "esc":
			s.mode = settingsBrowse
		}
		return m, nil
	case settingsPeriodInput:
		s.periodInput, _ = s.periodInput.Update(msg)
		switch msg.String() {
		case "enter":
			from, to, err := parseCustomPeriod(s.periodInput.Value())
			if err != nil {
				s.lastErr = err
				return m, nil
			}
			s.cfg.customFrom, s.cfg.customTo = from, to
			s.cfg.period = periodCustom
			s.lastErr = nil
			s.mode = settingsBrowse
		case "esc":
			s.lastErr = nil
			s.mode = settingsBrowse
		}
		return m, nil
	}

	switch msg.String() {
	case "up":
		s.sel = (s.sel + 3) % 4
	case "down":
		s.sel = (s.sel + 1) % 4
	case "enter":
		switch s.sel {
		case 0:
			s.openPeriodPick()
		case 1:
			s.openProjPick()
		case 2:
			s.cfg.includeJournal = !s.cfg.includeJournal
		case 3:
			s.dirInput.SetValue(s.cfg.saveDir)
			s.dirInput.Focus()
			s.mode = settingsDirInput
		}
	}
	return m, nil
}

func boolWord(b bool) string {
	if b {
		return "вкл"
	}
	return "выкл"
}
