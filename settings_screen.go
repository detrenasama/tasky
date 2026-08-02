package main

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/kalpamer/tasky/internal/db"
)

type settingsMode int

const (
	settingsBrowse settingsMode = iota
	settingsDirInput
	settingsProjList
	settingsPeriodList
	settingsPeriodInput
	settingsStatusList
	settingsStatusEdit
	settingsColorPick
	settingsStatusConfirm
)

// statusPalette — палитра цветов статусов (приглушённые, видны на тёмном
// фоне). Хранится индексом в pickList.
var statusPalette = []string{
	"#6a9955", "#4f7942", "#569cd6", "#c586c0", "#d7ba7d",
	"#ce9178", "#8a8a8a", "#8f3e3e", "#4ec9b0", "#d47a9e",
}

var paletteNames = []string{
	"зелёный", "тёмно-зелёный", "синий", "фиолетовый", "жёлтый",
	"оранжевый", "серый", "тёмно-красный", "голубой", "розовый",
}

var statusTypeNames = []string{"Новый", "В работе", "Завершённый"}
var statusTypeCodes = []string{"new", "in_progress", "done"}

// settingsScreen — страница «Настройки»: настройки отчёта (период, фильтр
// проекта, журнал, каталог сохранения) и каталог статусов. Значения отчёта
// пишутся в общий reportConfig экрана отчётов.
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

	statuses     []db.StatusDef
	statusPick   pickList
	colorPick    pickList
	editName     textinput.Model
	editNote     textinput.Model
	editType     int
	editColor    int
	editQuick    bool
	editFocus    int
	statusEditID int64
	statusDelID  int64

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
	s.statusPick.setVisible(12)
	s.colorPick.setVisible(12)
	items := make([]pickItem, 0, len(periodNames)+1)
	for i, name := range periodNames {
		items = append(items, pickItem{value: int64(i), label: name})
	}
	items = append(items, pickItem{value: int64(periodCustom), label: "свой…"})
	s.periodPick.items = items
	s.editName = textinput.New()
	s.editName.Placeholder = "Название статуса"
	s.editName.Prompt = "> "
	s.editName.Width = 30
	s.editNote = textinput.New()
	s.editNote.Placeholder = "пусто — без заметки"
	s.editNote.Prompt = "> "
	s.editNote.Width = 30
	for i, c := range statusPalette {
		s.colorPick.items = append(s.colorPick.items, pickItem{
			value: int64(i),
			label: colorPreview(c) + " " + paletteNames[i],
		})
	}
	s.load()
	return s
}

// colorPreview — цветной квадрат-превью для палитры.
func colorPreview(color string) string {
	return lipgloss.NewStyle().Background(lipgloss.Color(color)).Render("  ")
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
	s.statuses, _ = db.Statuses(s.db)
	sItems := make([]pickItem, 0, len(s.statuses))
	for _, st := range s.statuses {
		sItems = append(sItems, pickItem{value: st.ID, label: st.Name})
	}
	s.statusPick.items = sItems
}

func (s *settingsScreen) resize(w, h int) {
	s.midH = h
	visible := max(4, min(h-8, 12))
	s.projPick.setVisible(visible)
	s.periodPick.setVisible(visible)
	s.statusPick.setVisible(visible)
	s.colorPick.setVisible(visible)
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
		"Статусы: " + fmt.Sprintf("%d", len(s.statuses)),
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
	inner += "\n" + faint("каталог — ввод пути, статусы — каталог статусов)")
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
	case settingsStatusList:
		body := s.statusPick.view()
		if s.lastErr != nil {
			body += "\n\n" + errorStyle.Render("Ошибка: "+s.lastErr.Error())
		}
		d := dialog{title: "Статусы",
			body:    body,
			primary: "Enter — изменить · n — новый · d — удалить", esc: "Esc — назад"}
		return d.render(), true
	case settingsStatusEdit:
		title := "Новый статус"
		if s.statusEditID != 0 {
			title = "Статус"
		}
		lines := []string{
			"Имя:       " + s.editName.View(),
			"Тип:       " + statusTypeNames[s.editType],
			"Цвет:      " + colorPreview(statusPalette[s.editColor]) + " " + paletteNames[s.editColor],
			"Быстрая цепочка: " + boolWord(s.editQuick),
			"Подсказка: " + s.editNote.View(),
		}
		var body []string
		for i, l := range lines {
			if i == s.editFocus {
				body = append(body, headerStyle.Render("▸ "+l))
			} else {
				body = append(body, "  "+l)
			}
		}
		body = append(body, "", faint("Тип — Enter, цвет — Enter, цепочка — Enter, Ctrl+S — сохранить"))
		inner := strings.Join(body, "\n")
		if s.lastErr != nil {
			inner += "\n\n" + errorStyle.Render("Ошибка: "+s.lastErr.Error())
		}
		d := dialog{title: title, body: inner,
			primary: "Ctrl+S — сохранить", esc: "Esc — отмена"}
		return d.render(), true
	case settingsColorPick:
		d := dialog{title: "Цвет статуса",
			body:    s.colorPick.view(),
			primary: "Enter — выбрать", esc: "Esc — отмена"}
		return d.render(), true
	case settingsStatusConfirm:
		name := ""
		for _, st := range s.statuses {
			if st.ID == s.statusDelID {
				name = st.Name
			}
		}
		d := dialog{title: "Удаление статуса",
			body:    fmt.Sprintf("Удалить статус «%s»?", name),
			primary: "y — удалить", esc: "n — нет"}
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
	case settingsStatusList:
		switch msg.String() {
		case "up":
			s.statusPick.move(-1)
		case "down":
			s.statusPick.move(1)
		case "pgup":
			s.statusPick.move(-s.statusPick.visible)
		case "pgdown":
			s.statusPick.move(s.statusPick.visible)
		case "enter":
			if it, ok := s.statusPick.selected(); ok {
				s.openStatusEdit(it.value)
			}
		case "n":
			s.openStatusEdit(0)
		case "d":
			if it, ok := s.statusPick.selected(); ok {
				s.statusDelID = it.value
				s.mode = settingsStatusConfirm
			}
		case "esc":
			s.lastErr = nil
			s.mode = settingsBrowse
		}
		return m, nil
	case settingsStatusEdit:
		switch msg.String() {
		case "up":
			s.editFocus = (s.editFocus + 4) % 5
			s.focusEditField()
		case "down", "tab":
			s.editFocus = (s.editFocus + 1) % 5
			s.focusEditField()
		case "enter":
			switch s.editFocus {
			case 0:
				s.editFocus = 1
				s.focusEditField()
			case 1:
				s.editType = (s.editType + 1) % 3
			case 2:
				s.colorPick.sel = s.editColor
				s.colorPick.clampScroll()
				s.mode = settingsColorPick
			case 3:
				s.editQuick = !s.editQuick
			case 4:
				s.saveStatusEdit()
			}
		case "ctrl+s":
			s.saveStatusEdit()
		case "esc":
			s.lastErr = nil
			s.mode = settingsStatusList
		default:
			if s.editFocus == 0 {
				s.editName, _ = s.editName.Update(msg)
			} else if s.editFocus == 4 {
				s.editNote, _ = s.editNote.Update(msg)
			}
		}
		return m, nil
	case settingsColorPick:
		switch msg.String() {
		case "up":
			s.colorPick.move(-1)
		case "down":
			s.colorPick.move(1)
		case "pgup":
			s.colorPick.move(-s.colorPick.visible)
		case "pgdown":
			s.colorPick.move(s.colorPick.visible)
		case "enter":
			if it, ok := s.colorPick.selected(); ok {
				s.editColor = int(it.value)
			}
			s.mode = settingsStatusEdit
		case "esc":
			s.mode = settingsStatusEdit
		}
		return m, nil
	case settingsStatusConfirm:
		switch msg.String() {
		case "y", "enter":
			if err := db.DeleteStatus(s.db, s.statusDelID); err != nil {
				s.lastErr = err
			} else {
				s.lastErr = nil
				s.load()
			}
			s.mode = settingsStatusList
		case "n", "esc":
			s.mode = settingsStatusList
		}
		return m, nil
	}

	switch msg.String() {
	case "up":
		s.sel = (s.sel + 4) % 5
	case "down":
		s.sel = (s.sel + 1) % 5
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
		case 4:
			s.lastErr = nil
			s.mode = settingsStatusList
		}
	}
	return m, nil
}

// focusEditField переводит фокус textinput на текущее поле редактора
// статуса (имя или подсказка).
func (s *settingsScreen) focusEditField() {
	switch s.editFocus {
	case 0:
		s.editName.Focus()
		s.editNote.Blur()
	case 4:
		s.editName.Blur()
		s.editNote.Focus()
	default:
		s.editName.Blur()
		s.editNote.Blur()
	}
}

// openStatusEdit открывает редактор статуса: id=0 — новый, иначе — правка.
func (s *settingsScreen) openStatusEdit(id int64) {
	s.statusEditID = id
	s.lastErr = nil
	s.editName.SetValue("")
	s.editNote.SetValue("")
	s.editType, s.editColor, s.editQuick = 0, 0, false
	for _, st := range s.statuses {
		if st.ID != id {
			continue
		}
		s.editName.SetValue(st.Name)
		s.editNote.SetValue(st.NotePrompt)
		s.editQuick = st.IsQuick
		for i, t := range statusTypeCodes {
			if t == st.Type {
				s.editType = i
			}
		}
		for i, c := range statusPalette {
			if c == st.Color {
				s.editColor = i
			}
		}
	}
	s.editFocus = 0
	s.editName.Focus()
	s.editNote.Blur()
	s.mode = settingsStatusEdit
}

// saveStatusEdit сохраняет статус из редактора.
func (s *settingsScreen) saveStatusEdit() {
	name := strings.TrimSpace(s.editName.Value())
	if name == "" {
		s.lastErr = fmt.Errorf("имя не может быть пустым")
		return
	}
	var err error
	if s.statusEditID == 0 {
		_, err = db.CreateStatus(s.db, name, statusTypeCodes[s.editType],
			statusPalette[s.editColor], strings.TrimSpace(s.editNote.Value()), s.editQuick)
	} else {
		err = db.UpdateStatus(s.db, s.statusEditID, name, statusTypeCodes[s.editType],
			statusPalette[s.editColor], strings.TrimSpace(s.editNote.Value()), s.editQuick)
	}
	if err != nil {
		s.lastErr = err
		return
	}
	s.lastErr = nil
	s.load()
	s.mode = settingsStatusList
}

func boolWord(b bool) string {
	if b {
		return "вкл"
	}
	return "выкл"
}
