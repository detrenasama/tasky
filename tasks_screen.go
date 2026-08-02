package main

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/kalpamer/tasky/internal/db"
)

type taskMode int

const (
	taskBrowse taskMode = iota
	taskInput
	taskConfirm
	taskDescEdit
	taskLinkInput
	taskLinks
	taskLinkConfirm
	taskJournal
	taskStatusPick
	taskStatusNote
	taskSearch
)

type taskFocus int

const (
	taskFocusList taskFocus = iota
	taskFocusDesc
)

type paneKind int

const (
	kindTask paneKind = iota
	kindSubtask
)

type taskItem struct {
	t        db.Task
	expanded bool
	scr      *tasksScreen
}

func (i taskItem) FilterValue() string { return i.t.Title }

func (i taskItem) Title() string {
	marker := "▸"
	if i.expanded {
		marker = "▾"
	}
	return statusBar(i.scr.statusColor(i.t.Status)) + " " + marker + " " + i.t.Title
}

func (i taskItem) Description() string {
	plural := "подзадач"
	if i.t.SubCount == 1 {
		plural = "подзадача"
	}
	return i.scr.statusText(i.t.Status) + " · " + fmt.Sprintf("%d %s", i.t.SubCount, plural)
}

type subItem struct {
	st  db.SubtaskWithTime
	scr *tasksScreen
}

func (i subItem) FilterValue() string { return i.st.Title }

func (i subItem) Title() string {
	return "  " + statusBar(i.scr.statusColor(i.st.Status)) + " ├ " + i.st.Title
}

func (i subItem) Description() string {
	text := i.scr.statusText(i.st.Status)
	total := time.Duration(i.st.TotalSeconds) * time.Second
	if i.st.ActiveSince != nil {
		elapsed := i.scr.now.Sub(time.Unix(*i.st.ActiveSince, 0))
		return "  " + text + " · идет " + fmtElapsed(elapsed)
	}
	return "  " + text + " · " + fmtDur(total)
}

type tasksScreen struct {
	db       *sql.DB
	projects []db.Project
	projIdx  int
	tasks    []db.Task
	subs     []db.SubtaskWithTime
	list     list.Model
	items    []list.Item
	expanded map[int64]bool
	today    time.Duration
	weekly   time.Duration
	now      time.Time

	entries []db.TimeEntry
	run     *db.SubtaskWithTime

	midH  int
	listW int
	descW int
	infoW int

	mode        taskMode
	input       textinput.Model
	inputKind   paneKind
	confirmKind paneKind
	confirmID   int64
	lastErr     error

	focus taskFocus

	desc          string
	links         []db.Link
	journal       []db.JournalEntry
	history       []db.StatusHistoryEntry
	descText      textarea.Model
	descV         viewport.Model
	linkName      textinput.Model
	linkInput     textinput.Model
	linkList      list.Model
	journalText   textarea.Model
	confirmLinkID int64
	journalEditID int64

	statuses     []db.StatusDef
	statusPick   pickList
	statusNote   textarea.Model
	statusKind   paneKind
	statusID     int64
	statusTarget *db.StatusDef

	searchInput  textinput.Model
	searchQuery  string
	journalTexts map[int64]string
}

func newTasksScreen(conn *sql.DB) *tasksScreen {
	s := &tasksScreen{db: conn, expanded: map[int64]bool{}, now: time.Now()}

	d := list.NewDefaultDelegate()
	d.ShowDescription = true
	s.list = list.New(nil, d, 80, 20)
	s.list.Title = "Задачи"
	s.list.SetShowHelp(false)
	s.list.SetShowPagination(false)
	s.list.SetShowStatusBar(false)
	s.list.DisableQuitKeybindings()
	// встроенный фильтр списка (/ + перехват букв экраном) не работает —
	// полнотекстовый поиск реализован своим режимом taskSearch
	s.list.SetFilteringEnabled(false)

	s.input = textinput.New()
	s.input.Placeholder = "Название"
	s.input.Prompt = "> "
	s.input.CharLimit = 64
	s.input.Width = 40

	s.linkName = textinput.New()
	s.linkName.Placeholder = "Название (необязательно)"
	s.linkName.Prompt = "> "
	s.linkName.CharLimit = 64
	s.linkName.Width = 60

	s.linkInput = textinput.New()
	s.linkInput.Placeholder = "https://…"
	s.linkInput.Prompt = "> "
	s.linkInput.CharLimit = 512
	s.linkInput.Width = 60

	s.descText = textarea.New()
	s.descText.Placeholder = "Описание задачи или подзадачи"
	s.descText.ShowLineNumbers = false
	s.descText.SetWidth(60)
	s.descText.SetHeight(10)

	s.journalText = textarea.New()
	s.journalText.Placeholder = "Что делаю или сделал…"
	s.journalText.ShowLineNumbers = false
	s.journalText.SetWidth(60)
	s.journalText.SetHeight(10)

	s.statusNote = textarea.New()
	s.statusNote.Placeholder = "Заметка к переходу…"
	s.statusNote.ShowLineNumbers = false
	s.statusNote.SetWidth(60)
	s.statusNote.SetHeight(3)
	s.statusPick.setVisible(12)

	s.searchInput = textinput.New()
	s.searchInput.Placeholder = "название, описание, журнал…"
	s.searchInput.Prompt = "> "
	s.searchInput.CharLimit = 64
	s.searchInput.Width = 40

	ld := list.NewDefaultDelegate()
	ld.ShowDescription = true
	s.linkList = list.New(nil, ld, 50, 8)
	s.linkList.Title = "Ссылки"
	s.linkList.SetShowHelp(false)
	s.linkList.SetShowPagination(false)
	s.linkList.SetShowStatusBar(false)
	s.linkList.DisableQuitKeybindings()

	s.descV = viewport.New(1, 1)
	return s
}

func (s *tasksScreen) load() {
	s.projects, _ = db.Projects(s.db)
	if s.projIdx >= len(s.projects) {
		s.projIdx = 0
	}
	s.statuses, _ = db.Statuses(s.db)
	items := make([]pickItem, 0, len(s.statuses))
	for _, st := range s.statuses {
		items = append(items, pickItem{value: st.ID, label: st.Name})
	}
	s.statusPick.items = items
	s.loadData()
	s.today, _ = db.TodayTotal(s.db, s.now)
	s.weekly, _ = db.WeeklyTotal(s.db, s.now)
}

func (s *tasksScreen) loadData() {
	if s.projIdx < 0 || s.projIdx >= len(s.projects) {
		s.tasks = nil
		s.subs = nil
		s.buildItems()
		s.loadInfo()
		s.loadDesc()
		return
	}
	s.tasks, _ = db.TasksByProject(s.db, s.currentProjectID())
	s.subs, _ = db.SubtasksByProject(s.db, s.currentProjectID())
	s.journalTexts, _ = db.JournalTexts(s.db, s.currentProjectID())
	s.buildItems()
	s.loadInfo()
	s.loadDesc()
}

func (s *tasksScreen) loadInfo() {
	s.run, _ = db.RunningSession(s.db)
	s.entries = nil
	s.history = nil
	kind, id := s.selectedKindID()
	if id == 0 {
		return
	}
	if kind == kindSubtask {
		s.entries, _ = db.TimeEntriesBySubtask(s.db, id)
	}
	s.history, _ = db.StatusHistory(s.db, dbOwner(kind), id)
}

// loadDesc подгружает описание, ссылки и (для подзадачи) записи журнала
// выбранного элемента и пересобирает колонку описания.
func (s *tasksScreen) loadDesc() {
	kind, id := s.selectedKindID()
	s.desc = ""
	s.links = nil
	s.journal = nil
	switch kind {
	case kindTask:
		if id != 0 {
			s.desc, _ = db.TaskDescription(s.db, id)
			s.links, _ = db.TaskLinks(s.db, id)
		}
	case kindSubtask:
		if id != 0 {
			s.desc, _ = db.SubtaskDescription(s.db, id)
			s.links, _ = db.SubtaskLinks(s.db, id)
			s.journal, _ = db.JournalEntries(s.db, id)
		}
	}
	items := make([]list.Item, len(s.links))
	for i, l := range s.links {
		items[i] = linkItem{l}
	}
	s.linkList.SetItems(items)
	s.refreshDesc()
}

// refreshDesc собирает контент viewport колонки описания: для задачи —
// описание и ссылки; для подзадачи — блок «Описание» (1/3) и блок «Журнал»
// (2/3), разделённые линией.
func (s *tasksScreen) refreshDesc() {
	w := max(s.descV.Width, 1)
	kind, id := s.selectedKindID()
	var body string
	switch {
	case id == 0:
		body = strings.Join([]string{faint("Выберите задачу или подзадачу.")}, "\n")
	case kind == kindTask:
		body = strings.Join(append([]string{faint("Описание")}, s.descBody(w)...), "\n")
	case kind == kindSubtask:
		h1, h2 := s.descSplit()
		descBlock := padLines(strings.Join(append([]string{faint("Описание")}, s.descBody(w)...), "\n"), w, h1)
		journalBlock := padLines(strings.Join(append([]string{faint("Журнал")}, s.journalBody(w)...), "\n"), w, h2)
		divider := faint(strings.Repeat("─", w))
		body = descBlock + "\n" + divider + "\n" + journalBlock
	}
	s.descV.SetContent(body)
}

// descSplit делит высоту колонки описания на блок описания и блок журнала
// в пропорции 1:2 (минус строка-разделитель).
func (s *tasksScreen) descSplit() (int, int) {
	H := max(s.descV.Height, 3)
	h1 := max((H-1)/3, 2)
	if H-h1-1 < 3 {
		h1 = H - 4
		if h1 < 2 {
			h1 = 2
		}
	}
	return h1, max(H-h1-1, 0)
}

// descBody — строки описания и ссылок выбранного элемента.
func (s *tasksScreen) descBody(w int) []string {
	var body []string
	desc := strings.TrimSpace(s.desc)
	if desc == "" {
		body = append(body, faint("Описание пустое. Нажмите e (в колонке описания), чтобы добавить."))
	} else {
		body = append(body, wrapText(desc, w))
	}
	if len(s.links) > 0 {
		body = append(body, "", faint("Ссылки:"))
		for _, l := range s.links {
			label := l.Name
			if label == "" {
				label = l.URL
			}
			body = append(body, truncateWEnd(linkStyle.Render("• "+label), w))
		}
	}
	return body
}

// journalBody — строки записей журнала в хронологическом порядке: штамп
// даты/времени и текст записи.
func (s *tasksScreen) journalBody(w int) []string {
	if len(s.journal) == 0 {
		return []string{faint("Записей нет. Ctrl+J — добавить.")}
	}
	var body []string
	for _, e := range s.journal {
		body = append(body, faint(e.CreatedAt.Format("02.01.2006 15:04")))
		body = append(body, wrapText(e.Text, w))
		body = append(body, "")
	}
	return body
}

func (s *tasksScreen) currentProjectID() int64 {
	if s.projIdx < 0 || s.projIdx >= len(s.projects) {
		return 0
	}
	return s.projects[s.projIdx].ID
}

func (s *tasksScreen) buildItems() {
	if q := strings.ToLower(strings.TrimSpace(s.searchQuery)); q != "" {
		s.buildSearchItems(q)
		return
	}
	s.items = []list.Item{}
	for _, t := range s.tasks {
		s.items = append(s.items, taskItem{t: t, expanded: s.expanded[t.ID], scr: s})
		if s.expanded[t.ID] {
			for _, st := range s.subs {
				if st.TaskID == t.ID {
					s.items = append(s.items, subItem{st: st, scr: s})
				}
			}
		}
	}
	s.finishBuildItems()
}

// buildSearchItems собирает дерево по запросу q (регистронезависимо):
// задача попадает, если совпадает её название или описание (тогда видны все
// её подзадачи); подзадача — если совпадает название, описание или текст
// записей журнала (тогда видна только она).
func (s *tasksScreen) buildSearchItems(q string) {
	s.items = []list.Item{}
	for _, t := range s.tasks {
		taskMatch := strings.Contains(strings.ToLower(t.Title), q) ||
			strings.Contains(strings.ToLower(t.Description), q)
		var subs []db.SubtaskWithTime
		if taskMatch {
			for _, st := range s.subs {
				if st.TaskID == t.ID {
					subs = append(subs, st)
				}
			}
		} else {
			for _, st := range s.subs {
				if st.TaskID == t.ID && s.subMatches(st, q) {
					subs = append(subs, st)
				}
			}
		}
		if taskMatch || len(subs) > 0 {
			s.items = append(s.items, taskItem{t: t, expanded: true, scr: s})
			for _, st := range subs {
				s.items = append(s.items, subItem{st: st, scr: s})
			}
		}
	}
	s.finishBuildItems()
}

func (s *tasksScreen) subMatches(st db.SubtaskWithTime, q string) bool {
	return strings.Contains(strings.ToLower(st.Title), q) ||
		strings.Contains(strings.ToLower(st.Description), q) ||
		strings.Contains(strings.ToLower(s.journalTexts[st.ID]), q)
}

func (s *tasksScreen) finishBuildItems() {
	selKind, selID := s.selectedKindID()
	s.list.SetItems(s.items)
	if len(s.items) > 0 {
		idx := s.indexByKindID(selKind, selID)
		if idx < 0 {
			idx = 0
		}
		s.list.Select(idx)
	}
}

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
	if err := db.SetStatus(s.db, dbOwner(kind), id, st.Name, "", time.Now()); err == nil {
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

func (s *tasksScreen) selectedKindID() (paneKind, int64) {
	switch item := s.list.SelectedItem().(type) {
	case taskItem:
		return kindTask, item.t.ID
	case subItem:
		return kindSubtask, item.st.ID
	}
	return kindTask, 0
}

func (s *tasksScreen) indexByKindID(kind paneKind, id int64) int {
	if id == 0 {
		return -1
	}
	for i, item := range s.items {
		switch it := item.(type) {
		case taskItem:
			if kind == kindTask && it.t.ID == id {
				return i
			}
		case subItem:
			if kind == kindSubtask && it.st.ID == id {
				return i
			}
		}
	}
	return -1
}

func (s *tasksScreen) selectByKindID(kind paneKind, id int64) {
	idx := s.indexByKindID(kind, id)
	if idx < 0 {
		return
	}
	s.list.Select(idx)
}

func (s *tasksScreen) selectedKind() paneKind {
	switch s.list.SelectedItem().(type) {
	case taskItem:
		return kindTask
	case subItem:
		return kindSubtask
	}
	return kindTask
}

// selectedTaskID возвращает ID задачи: выбранной или родителя выбранной подзадачи.
func (s *tasksScreen) selectedTaskID() int64 {
	switch item := s.list.SelectedItem().(type) {
	case taskItem:
		return item.t.ID
	case subItem:
		return item.st.TaskID
	}
	return 0
}

func (s *tasksScreen) toggleExpand() {
	item, ok := s.list.SelectedItem().(taskItem)
	if !ok {
		return
	}
	s.expanded[item.t.ID] = !s.expanded[item.t.ID]
	s.buildItems()
}

func (s *tasksScreen) canCreate(kind paneKind) bool {
	if kind == kindTask {
		return s.projIdx >= 0 && s.projIdx < len(s.projects)
	}
	return s.selectedTaskID() != 0
}

func (s *tasksScreen) canDelete() bool {
	return s.list.Index() >= 0 && len(s.items) > 0
}

func (s *tasksScreen) switchProject(dir int) {
	if len(s.projects) < 2 {
		return
	}
	s.projIdx = (s.projIdx + dir + len(s.projects)) % len(s.projects)
	s.loadData()
}

func (s *tasksScreen) toggleTimer() {
	item, ok := s.list.SelectedItem().(subItem)
	if !ok {
		return
	}
	now := time.Now()
	if item.st.ActiveSince != nil {
		db.StopSession(s.db, item.st.ID, now)
	} else {
		db.StartSession(s.db, item.st.ID, now)
	}
	s.now = now
	s.loadData()
	s.today, _ = db.TodayTotal(s.db, now)
	s.weekly, _ = db.WeeklyTotal(s.db, now)
}

func (s *tasksScreen) createItem(kind paneKind, title string) (int64, error) {
	if kind == kindTask {
		t, err := db.CreateTask(s.db, s.currentProjectID(), title)
		return t.ID, err
	}
	st, err := db.CreateSubtask(s.db, s.selectedTaskID(), title)
	return st.ID, err
}

func (s *tasksScreen) deleteItem(kind paneKind, id int64) {
	if kind == kindTask {
		db.DeleteTask(s.db, id)
		return
	}
	db.DeleteSubtask(s.db, id)
}

func (s *tasksScreen) resize(w, h int) {
	s.midH = max(h, 3)
	listW, descW, infoW := 0, 0, 0
	if w >= 110 {
		// три колонки 2:2:1 на всю ширину
		u := (w - 4) / 5
		listW, descW, infoW = 2*u, 2*u, u
		listW += (w - 4) - 5*u
	} else if w >= 70 {
		// list + info (2:1)
		listW = (w - 2) * 2 / 3
		infoW = w - 2 - listW
	} else {
		listW = w
	}
	s.listW, s.descW, s.infoW = listW, descW, infoW
	s.list.SetWidth(listW - 2)
	s.list.SetHeight(s.midH - 2)
	s.descV.Width = max(descW-4, 1)
	s.descV.Height = max(s.midH-2, 1)
	s.descText.SetWidth(max(descW-4, 1))
	s.descText.SetHeight(max(s.midH-2, 1))
	s.journalText.SetWidth(max(descW-4, 1))
	s.journalText.SetHeight(10)
	s.refreshDesc()
}

func (s *tasksScreen) header(w int) string {
	h := headerStyle.Render("Tasky")
	if s.projIdx >= 0 && s.projIdx < len(s.projects) {
		h += "  " + faint("проект: ") + s.projects[s.projIdx].Name
	}
	if s.searchQuery != "" {
		h += "  " + faint("поиск: ") + s.searchQuery
	}
	return padW(h, w)
}

func (s *tasksScreen) footer(w int) string {
	if s.mode == taskDescEdit {
		return padW(faint("Ctrl+S — сохранить · Esc — отмена"), w)
	}
	if s.focus == taskFocusDesc {
		return padW(faint("↑/↓ скролл · e — описание · l — ссылка · o — ссылки · Ctrl+J — запись · j — изменить запись · / — поиск · Tab — список"), w)
	}
	hint := "↑/↓ выбор · Enter раскрыть · n задача · a подзадача · d удалить · Ctrl+L старт/пауза · x/z статус · c — все статусы · / — поиск · [ / ] проект · Tab — описание · q выход"
	if s.searchQuery != "" {
		hint = "Поиск: «" + s.searchQuery + "» — / — изменить · Esc — сбросить"
	}
	return padW(faint(hint), w)
}

func (s *tasksScreen) view(w, h int) string {
	leftStyle := dimBox
	if s.focus == taskFocusList {
		leftStyle = focusBox
	}
	var left string
	if len(s.projects) == 0 {
		left = fixedBox(dimBox, "Нет проектов.\nНажмите p и создайте проект.", s.listW, s.midH)
	} else if s.searchQuery != "" && len(s.items) == 0 {
		left = fixedBox(dimBox, "Ничего не найдено по запросу\n«"+s.searchQuery+"».", s.listW, s.midH)
	} else if len(s.tasks) == 0 {
		left = fixedBox(dimBox, "Задач в проекте нет.", s.listW, s.midH)
	} else {
		// bubbles/list не дополняет строки до ширины — паддинг вручную
		left = leftStyle.Render(padLines(s.list.View(), max(s.listW-4, 0), max(s.midH-2, 0)))
	}

	cols := []string{left}
	if s.descW > 0 {
		cols = append(cols, "  ", s.descBox())
	}
	if s.infoW > 0 {
		cols = append(cols, "  ", s.infoBox())
	}
	body := lipgloss.JoinHorizontal(lipgloss.Top, cols...)
	return padH(body, w, h)
}

func (s *tasksScreen) dialog() (string, bool) {
	switch s.mode {
	case taskInput:
		kind := "задача"
		if s.inputKind == kindSubtask {
			kind = "подзадача"
		}
		body := s.input.View()
		if s.lastErr != nil {
			body += "\n\n" + errorStyle.Render("Ошибка: "+s.lastErr.Error())
		}
		d := dialog{title: "Новая " + kind, body: body,
			primary: "Enter — сохранить", esc: "Esc — отмена"}
		return d.render(), true
	case taskConfirm:
		name := ""
		if s.confirmKind == kindTask {
			for _, t := range s.tasks {
				if t.ID == s.confirmID {
					name = t.Title
				}
			}
		} else {
			for _, st := range s.subs {
				if st.ID == s.confirmID {
					name = st.Title
				}
			}
		}
		var d dialog
		if s.confirmKind == kindTask {
			d = dialog{title: "Удаление задачи",
				body:    fmt.Sprintf("Удалить задачу «%s» с подзадачами и временем?", name),
				primary: "y — да", esc: "n — нет"}
		} else {
			d = dialog{title: "Удаление подзадачи",
				body:    fmt.Sprintf("Удалить подзадачу «%s» и её время?", name),
				primary: "y — да", esc: "n — нет"}
		}
		return d.render(), true
	case taskDescEdit:
		return "", false
	case taskLinkInput:
		body := s.linkName.View() + "\n" + s.linkInput.View()
		if s.lastErr != nil {
			body += "\n\n" + errorStyle.Render("Ошибка: "+s.lastErr.Error())
		}
		d := dialog{title: "Добавить ссылку", body: body,
			primary: "Enter — добавить · Tab — поле", esc: "Esc — отмена"}
		return d.render(), true
	case taskLinks:
		body := s.linkList.View()
		if s.lastErr != nil {
			body += "\n\n" + errorStyle.Render("Не удалось открыть: "+s.lastErr.Error())
		}
		d := dialog{title: "Ссылки",
			body:    body,
			primary: "Enter — открыть · d — удалить", esc: "Esc — закрыть"}
		return d.render(), true
	case taskLinkConfirm:
		label := ""
		for _, l := range s.links {
			if l.ID == s.confirmLinkID {
				label = l.Name
				if label == "" {
					label = l.URL
				}
			}
		}
		d := dialog{title: "Удаление ссылки",
			body:    fmt.Sprintf("Удалить ссылку «%s»?", label),
			primary: "y — удалить", esc: "n — нет"}
		return d.render(), true
	case taskJournal:
		title := "Запись в журнал"
		if s.journalEditID != 0 {
			title = "Изменение записи"
		}
		body := s.journalText.View()
		if s.lastErr != nil {
			body += "\n\n" + errorStyle.Render("Ошибка: "+s.lastErr.Error())
		}
		d := dialog{title: title, body: body,
			primary: "Ctrl+S — сохранить", esc: "Esc — отмена"}
		return d.render(), true
	case taskStatusPick:
		d := dialog{title: "Статус",
			body:    s.statusPick.view(),
			primary: "Enter — выбрать", esc: "Esc — отмена"}
		return d.render(), true
	case taskStatusNote:
		title := "Заметка"
		if s.statusTarget != nil && s.statusTarget.NotePrompt != "" {
			title = s.statusTarget.Name + " — " + s.statusTarget.NotePrompt
		}
		body := s.statusNote.View()
		if s.lastErr != nil {
			body += "\n\n" + errorStyle.Render("Ошибка: "+s.lastErr.Error())
		}
		d := dialog{title: title, body: body,
			primary: "Ctrl+S — применить", esc: "Esc — отмена"}
		return d.render(), true
	case taskSearch:
		body := s.searchInput.View()
		if s.searchQuery != "" {
			body += "\n\n" + faint("Найдено: "+fmt.Sprint(len(s.items))+" элементов")
		}
		d := dialog{title: "Поиск", body: body,
			primary: "Enter — применить", esc: "Esc — отмена"}
		return d.render(), true
	}
	return "", false
}

// descBox — средняя колонка: описание, ссылки и журнал выбранного элемента
// (прокручиваемый viewport); при редактировании описания вместо контента —
// textarea.
func (s *tasksScreen) descBox() string {
	if s.mode == taskDescEdit {
		return focusBox.Render(padLines(s.descText.View(), max(s.descW-4, 0), max(s.midH-2, 0)))
	}
	style := dimBox
	if s.focus == taskFocusDesc {
		style = focusBox
	}
	return style.Render(s.descV.View())
}

// infoBox — правая колонка: выбранный элемент (на всю высоту) + общая
// информация (по высоте контента).
func (s *tasksScreen) infoBox() string {
	bot := s.infoBottom()
	botH := len(strings.Split(bot, "\n"))
	topH := s.midH - botH - 1
	if topH < 3 {
		topH = 3
	}
	return s.infoTop(topH) + "\n\n" + bot
}

func (s *tasksScreen) infoTop(topH int) string {
	kind, id := s.selectedKindID()
	var body []string
	switch {
	case kind == kindTask && id == 0:
		body = append(body, faint("Выберите задачу или подзадачу."))
	case kind == kindSubtask:
		var st *db.SubtaskWithTime
		for i := range s.subs {
			if s.subs[i].ID == id {
				st = &s.subs[i]
			}
		}
		if st == nil {
			body = append(body, faint("Подзадача не найдена."))
			break
		}
		body = append(body, st.Title)
		body = append(body, faint("Статус: ")+s.statusText(st.Status))
		total := time.Duration(st.TotalSeconds) * time.Second
		if st.ActiveSince != nil {
			total += s.now.Sub(time.Unix(*st.ActiveSince, 0))
		}
		body = append(body, faint("Время всего: ")+fmtDur(total))
		body = append(body, "")
		for _, e := range s.entries {
			body = append(body, entryLine(e, s.now))
		}
		if len(s.entries) == 0 {
			body = append(body, faint("Записей нет."))
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
			body = append(body, faint("Задача не найдена."))
			break
		}
		body = append(body, t.Title)
		body = append(body, faint("Статус: ")+s.statusText(t.Status))
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
		body = append(body, faint(fmt.Sprintf("%d %s, всего: %s", t.SubCount, plural, fmtDur(sum))))
		body = append(body, s.historyLines()...)
	default:
		body = append(body, faint("Выберите задачу или подзадачу."))
	}
	inner := strings.Join(body, "\n")
	inner = padLines(inner, max(s.infoW-4, 1), topH-2)
	return boxStyle.Render(inner)
}

// historyLines — последние 6 переходов статусов выбранного элемента: штамп
// дня отдельной строкой, сам переход и заметка wrapText.
func (s *tasksScreen) historyLines() []string {
	var body []string
	if len(s.history) == 0 {
		return body
	}
	body = append(body, "", faint("История статусов:"))
	start := 0
	if len(s.history) > 6 {
		start = len(s.history) - 6
	}
	w := max(s.infoW-4, 1)
	for _, h := range s.history[start:] {
		body = append(body, faint(h.CreatedAt.Format("2006-01-02 15:04")))
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
		return start + "–… · " + faint("идет "+fmtElapsed(now.Sub(e.StartedAt)))
	}
	d := time.Duration(e.EndedAt.Sub(e.StartedAt))
	return start + "–" + e.EndedAt.Format("15:04") + " · " + fmtDur(d)
}

func (s *tasksScreen) infoBottom() string {
	body := []string{
		faint("За сегодня: ") + fmtDur(s.today),
		faint("Неделя (Пн–Вс): ") + fmtDur(s.weekly),
		"",
	}
	if s.run != nil {
		elapsed := s.now.Sub(time.Unix(*s.run.ActiveSince, 0))
		body = append(body, "Сейчас: "+s.run.Title)
		body = append(body, faint("идет "+fmtElapsed(elapsed)))
	} else {
		body = append(body, faint("Ничего не запущено."))
	}
	for i := range body {
		body[i] = padW(body[i], max(s.infoW-4, 1))
	}
	return boxStyle.Render(strings.Join(body, "\n"))
}

func fmtDur(d time.Duration) string {
	d = d.Round(time.Second)
	if d < 0 {
		d = 0
	}
	h := int(d / time.Hour)
	m := int(d%time.Hour) / int(time.Minute)
	sec := int(d%time.Minute) / int(time.Second)
	if h > 0 {
		return fmt.Sprintf("%dч %dм", h, m)
	}
	if m > 0 {
		return fmt.Sprintf("%dм %dс", m, sec)
	}
	return fmt.Sprintf("%dс", sec)
}

func fmtElapsed(d time.Duration) string {
	d = d.Round(time.Second)
	if d < 0 {
		d = 0
	}
	m := int(d / time.Minute)
	sec := int(d%time.Minute) / int(time.Second)
	if m > 0 {
		return fmt.Sprintf("%dм %dс", m, sec)
	}
	return fmt.Sprintf("%dс", sec)
}

func (m *model) updateTasks(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	s := m.tasks

	switch s.mode {
	case taskInput:
		var cmd tea.Cmd
		s.input, cmd = s.input.Update(msg)
		switch msg.String() {
		case "enter":
			title := strings.TrimSpace(s.input.Value())
			if title != "" {
				created, err := s.createItem(s.inputKind, title)
				s.lastErr = err
				if err == nil {
					if s.inputKind == kindSubtask {
						s.expanded[s.selectedTaskID()] = true
					}
					s.loadData()
					s.selectByKindID(s.inputKind, created)
					s.loadInfo()
					s.loadDesc()
					s.descV.GotoTop()
				}
			}
			s.input.SetValue("")
			s.input.Blur()
			s.mode = taskBrowse
		case "esc":
			s.input.SetValue("")
			s.input.Blur()
			s.mode = taskBrowse
		}
		return m, cmd
	case taskConfirm:
		switch msg.String() {
		case "y", "enter":
			s.deleteItem(s.confirmKind, s.confirmID)
			s.mode = taskBrowse
			s.loadData()
		case "n", "esc":
			s.mode = taskBrowse
		}
		return m, nil
	case taskDescEdit:
		var cmd tea.Cmd
		s.descText, cmd = s.descText.Update(msg)
		switch msg.String() {
		case "ctrl+s":
			kind, id := s.selectedKindID()
			if kind == kindTask {
				db.UpdateTaskDescription(s.db, id, s.descText.Value())
			} else {
				db.UpdateSubtaskDescription(s.db, id, s.descText.Value())
			}
			s.descText.Blur()
			s.mode = taskBrowse
			s.loadDesc()
		case "esc":
			s.descText.Blur()
			s.mode = taskBrowse
		}
		return m, cmd
	case taskLinkInput:
		var cmd tea.Cmd
		if s.linkName.Focused() {
			s.linkName, cmd = s.linkName.Update(msg)
		} else {
			s.linkInput, cmd = s.linkInput.Update(msg)
		}
		switch msg.String() {
		case "tab":
			if s.linkName.Focused() {
				s.linkName.Blur()
				s.linkInput.Focus()
			} else {
				s.linkInput.Blur()
				s.linkName.Focus()
			}
		case "enter":
			if s.linkInput.Focused() {
				url := strings.TrimSpace(s.linkInput.Value())
				if url != "" {
					kind, id := s.selectedKindID()
					var err error
					if kind == kindTask {
						_, err = db.CreateTaskLink(s.db, id, strings.TrimSpace(s.linkName.Value()), url)
					} else {
						_, err = db.CreateSubtaskLink(s.db, id, strings.TrimSpace(s.linkName.Value()), url)
					}
					s.lastErr = err
					if err == nil {
						s.loadDesc()
					}
				}
				s.linkName.SetValue("")
				s.linkName.Blur()
				s.linkInput.SetValue("")
				s.linkInput.Blur()
				s.mode = taskBrowse
			} else {
				// Enter на названии — перейти к адресу
				s.linkName.Blur()
				s.linkInput.Focus()
			}
		case "esc":
			s.linkName.SetValue("")
			s.linkName.Blur()
			s.linkInput.SetValue("")
			s.linkInput.Blur()
			s.mode = taskBrowse
		}
		return m, cmd
	case taskLinks:
		var cmd tea.Cmd
		s.linkList, cmd = s.linkList.Update(msg)
		switch msg.String() {
		case "enter":
			if item, ok := s.linkList.SelectedItem().(linkItem); ok {
				if err := openURL(item.l.URL); err != nil {
					s.lastErr = err
				}
			}
			return m, nil
		case "d":
			if item, ok := s.linkList.SelectedItem().(linkItem); ok {
				s.confirmLinkID = item.l.ID
				s.mode = taskLinkConfirm
			}
			return m, nil
		case "esc":
			s.mode = taskBrowse
			s.lastErr = nil
		}
		return m, cmd
	case taskLinkConfirm:
		switch msg.String() {
		case "y", "enter":
			kind, _ := s.selectedKindID()
			if kind == kindTask {
				db.DeleteTaskLink(s.db, s.confirmLinkID)
			} else {
				db.DeleteSubtaskLink(s.db, s.confirmLinkID)
			}
			s.mode = taskLinks
			s.loadDesc()
		case "n", "esc":
			s.mode = taskLinks
		}
		return m, nil
	case taskJournal:
		var cmd tea.Cmd
		s.journalText, cmd = s.journalText.Update(msg)
		switch msg.String() {
		case "ctrl+s":
			text := strings.TrimSpace(s.journalText.Value())
			kind, id := s.selectedKindID()
			if kind == kindSubtask && id != 0 && text != "" {
				if s.journalEditID != 0 {
					db.UpdateJournalEntry(s.db, s.journalEditID, text)
				} else {
					db.CreateJournalEntry(s.db, id, text)
				}
			}
			s.journalText.Blur()
			s.mode = taskBrowse
			s.loadDesc()
			s.descV.GotoBottom()
		case "esc":
			s.journalText.Blur()
			s.mode = taskBrowse
		}
		return m, cmd
	case taskStatusPick:
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
				for _, st := range s.statuses {
					if st.ID == it.value {
						s.applyStatus(s.statusKind, s.statusID, st)
						break
					}
				}
			}
		case "esc":
			s.mode = taskBrowse
		}
		return m, nil
	case taskStatusNote:
		var cmd tea.Cmd
		s.statusNote, cmd = s.statusNote.Update(msg)
		switch msg.String() {
		case "ctrl+s":
			note := strings.TrimSpace(s.statusNote.Value())
			if s.statusTarget != nil {
				if err := db.SetStatus(s.db, dbOwner(s.statusKind), s.statusID,
					s.statusTarget.Name, note, time.Now()); err == nil {
					s.now = time.Now()
					s.loadData()
				} else {
					s.lastErr = err
				}
			}
			s.statusNote.Blur()
			s.mode = taskBrowse
		case "esc":
			s.statusNote.Blur()
			s.mode = taskBrowse
		}
		return m, cmd
	case taskSearch:
		var cmd tea.Cmd
		s.searchInput, cmd = s.searchInput.Update(msg)
		switch msg.String() {
		case "enter":
			s.searchQuery = strings.TrimSpace(s.searchInput.Value())
			s.searchInput.Blur()
			s.mode = taskBrowse
			s.buildItems()
			s.loadInfo()
			s.loadDesc()
		case "esc":
			s.searchQuery = ""
			s.searchInput.Blur()
			s.mode = taskBrowse
			s.buildItems()
			s.loadInfo()
			s.loadDesc()
		default:
			// живой фильтр по мере ввода
			s.searchQuery = strings.TrimSpace(s.searchInput.Value())
			s.buildItems()
			s.loadInfo()
			s.loadDesc()
		}
		return m, cmd
	}

	switch msg.String() {
	case "tab":
		if s.descW > 0 {
			if s.focus == taskFocusList {
				s.focus = taskFocusDesc
			} else {
				s.focus = taskFocusList
			}
		}
		return m, nil
	case "e":
		if s.focus == taskFocusDesc {
			s.lastErr = nil
			s.descText.SetValue(s.desc)
			s.mode = taskDescEdit
			s.descText.Focus()
			return m, nil
		}
	case "l":
		if s.focus == taskFocusDesc {
			s.lastErr = nil
			s.linkName.SetValue("")
			s.linkInput.SetValue("")
			s.mode = taskLinkInput
			s.linkName.Focus()
			s.linkInput.Blur()
			return m, nil
		}
	case "o":
		if s.focus == taskFocusDesc {
			s.lastErr = nil
			s.mode = taskLinks
			return m, nil
		}
	case "ctrl+j":
		if s.focus == taskFocusDesc {
			kind, id := s.selectedKindID()
			if kind == kindSubtask && id != 0 {
				s.lastErr = nil
				s.journalText.SetValue("")
				s.journalEditID = 0
				s.mode = taskJournal
				s.journalText.Focus()
			}
			return m, nil
		}
	case "j":
		if s.focus == taskFocusDesc {
			kind, id := s.selectedKindID()
			if kind == kindSubtask && id != 0 {
				// редактировать можно самую свежую запись текущего дня
				var target *db.JournalEntry
				for i := range s.journal {
					e := &s.journal[i]
					if sameDay(e.CreatedAt, s.now) {
						target = e
					}
				}
				if target == nil {
					s.lastErr = fmt.Errorf("редактировать можно только записи текущего дня")
					return m, nil
				}
				s.lastErr = nil
				s.journalEditID = target.ID
				s.journalText.SetValue(target.Text)
				s.journalText.CursorEnd()
				s.mode = taskJournal
				s.journalText.Focus()
			}
			return m, nil
		}
	case "/":
		s.lastErr = nil
		s.searchInput.SetValue(s.searchQuery)
		s.searchInput.Focus()
		s.mode = taskSearch
		return m, nil
	}

	if s.focus == taskFocusDesc {
		switch msg.String() {
		case "up", "down", "pgup", "pgdown", "home", "end":
			s.descV, _ = s.descV.Update(msg)
			return m, nil
		}
		return m, nil
	}

	switch msg.String() {
	case "enter", "right":
		s.toggleExpand()
		return m, nil
	case "esc":
		if s.searchQuery != "" {
			s.searchQuery = ""
			s.buildItems()
			s.loadInfo()
			s.loadDesc()
		}
		return m, nil
	case "ctrl+l":
		s.toggleTimer()
		return m, nil
	case "[":
		s.switchProject(-1)
		return m, nil
	case "]":
		s.switchProject(1)
		return m, nil
	case "n":
		if s.canCreate(kindTask) {
			s.inputKind = kindTask
			s.lastErr = nil
			s.mode = taskInput
			s.input.Focus()
		}
		return m, nil
	case "a":
		if s.canCreate(kindSubtask) {
			s.inputKind = kindSubtask
			s.lastErr = nil
			s.mode = taskInput
			s.input.Focus()
		}
		return m, nil
	case "d":
		if s.canDelete() {
			s.confirmKind = s.selectedKind()
			switch s.confirmKind {
			case kindTask:
				if item, ok := s.list.SelectedItem().(taskItem); ok {
					s.confirmID = item.t.ID
				}
			case kindSubtask:
				if item, ok := s.list.SelectedItem().(subItem); ok {
					s.confirmID = item.st.ID
				}
			}
			s.mode = taskConfirm
		}
		return m, nil
	case "x":
		s.shiftStatus(1)
		return m, nil
	case "z":
		s.shiftStatus(-1)
		return m, nil
	case "c":
		kind, id := s.selectedKindID()
		if id == 0 {
			return m, nil
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
		return m, nil
	}
	var cmd tea.Cmd
	beforeKind, beforeID := s.selectedKindID()
	s.list, cmd = s.list.Update(msg)
	afterKind, afterID := s.selectedKindID()
	if beforeKind != afterKind || beforeID != afterID {
		s.loadInfo()
		s.loadDesc()
		s.descV.GotoTop()
	}
	return m, cmd
}

// sameDay проверяет, что два момента времени приходятся на один календарный
// день (локальное время).
func sameDay(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}
