package main

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/kalpamer/tasky/internal/db"
)

type taskMode int

const (
	taskBrowse taskMode = iota
	taskInput
	taskConfirm
)

type paneKind int

const (
	kindTask paneKind = iota
	kindSubtask
)

type taskItem struct {
	t        db.Task
	expanded bool
}

func (i taskItem) FilterValue() string { return i.t.Title }

func (i taskItem) Title() string {
	marker := "▸"
	if i.expanded {
		marker = "▾"
	}
	return marker + " " + i.t.Title
}

func (i taskItem) Description() string {
	plural := "подзадач"
	if i.t.SubCount == 1 {
		plural = "подзадача"
	}
	return statusRU(i.t.Status) + " · " + fmt.Sprintf("%d %s", i.t.SubCount, plural)
}

type subItem struct {
	st  db.SubtaskWithTime
	scr *tasksScreen
}

func (i subItem) FilterValue() string { return i.st.Title }

func (i subItem) Title() string { return "  ├ " + i.st.Title }

func (i subItem) Description() string {
	text := statusRU(i.st.Status)
	total := time.Duration(i.st.TotalSeconds) * time.Second
	if i.st.ActiveSince != nil {
		elapsed := i.scr.now.Sub(time.Unix(*i.st.ActiveSince, 0))
		return text + " · идет " + fmtElapsed(elapsed)
	}
	return text + " · " + fmtDur(total)
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
}

func newTasksScreen(conn *sql.DB) *tasksScreen {
	s := &tasksScreen{db: conn, expanded: map[int64]bool{}}

	d := list.NewDefaultDelegate()
	d.ShowDescription = true
	s.list = list.New(nil, d, 80, 20)
	s.list.Title = "Задачи"
	s.list.SetShowHelp(false)
	s.list.SetShowPagination(false)
	s.list.SetShowStatusBar(false)

	s.input = textinput.New()
	s.input.Placeholder = "Название"
	s.input.Prompt = "> "
	s.input.CharLimit = 64
	s.input.Width = 40

	return s
}

func (s *tasksScreen) load() {
	s.projects, _ = db.Projects(s.db)
	if s.projIdx >= len(s.projects) {
		s.projIdx = 0
	}
	s.loadData()
	s.weekly, _ = db.WeeklyTotal(s.db, s.now)
}

func (s *tasksScreen) loadData() {
	if s.projIdx < 0 || s.projIdx >= len(s.projects) {
		s.tasks = nil
		s.subs = nil
		s.buildItems()
		s.loadInfo()
		return
	}
	s.tasks, _ = db.TasksByProject(s.db, s.currentProjectID())
	s.subs, _ = db.SubtasksByProject(s.db, s.currentProjectID())
	s.buildItems()
	s.loadInfo()
}

func (s *tasksScreen) loadInfo() {
	s.run, _ = db.RunningSession(s.db)
	s.entries = nil
	kind, id := s.selectedKindID()
	if kind == kindSubtask {
		s.entries, _ = db.TimeEntriesBySubtask(s.db, id)
	}
}

func (s *tasksScreen) currentProjectID() int64 {
	if s.projIdx < 0 || s.projIdx >= len(s.projects) {
		return 0
	}
	return s.projects[s.projIdx].ID
}

func (s *tasksScreen) buildItems() {
	s.items = []list.Item{}
	for _, t := range s.tasks {
		s.items = append(s.items, taskItem{t: t, expanded: s.expanded[t.ID]})
		if s.expanded[t.ID] {
			for _, st := range s.subs {
				if st.TaskID == t.ID {
					s.items = append(s.items, subItem{st: st, scr: s})
				}
			}
		}
	}
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
}

func (s *tasksScreen) header(w int) string {
	h := headerStyle.Render("Tasky")
	if s.projIdx >= 0 && s.projIdx < len(s.projects) {
		h += "  " + faint("проект: ") + s.projects[s.projIdx].Name
	}
	return padW(h, w)
}

func (s *tasksScreen) footer(w int) string {
	hint := "↑/↓ выбор · Enter раскрыть · n задача · a подзадача · d удалить · Ctrl+L старт/пауза · [ / ] проект · p проекты · q выход"
	return padW(faint(hint), w)
}

func (s *tasksScreen) view(w, h int) string {
	var left string
	if len(s.projects) == 0 {
		left = fixedBox(dimBox, "Нет проектов.\nНажмите p и создайте проект.", s.listW, s.midH)
	} else if len(s.tasks) == 0 {
		left = fixedBox(dimBox, "Задач в проекте нет.", s.listW, s.midH)
	} else {
		// bubbles/list не дополняет строки до ширины — паддинг вручную
		left = focusBox.Render(padLines(s.list.View(), s.listW-4, s.midH-2))
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
	}
	return "", false
}

// descBox — зарезервированная колонка под описание задачи/подзадачи.
func (s *tasksScreen) descBox() string {
	return fixedBox(dimBox, faint("Описание"), s.descW, s.midH)
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
		body = append(body, faint("Статус: ")+statusRU(st.Status))
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
		body = append(body, faint("Статус: ")+statusRU(t.Status))
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
	default:
		body = append(body, faint("Выберите задачу или подзадачу."))
	}
	inner := strings.Join(body, "\n")
	inner = padLines(inner, max(s.infoW-4, 1), topH-2)
	return boxStyle.Render(inner)
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

func statusRU(s string) string {
	switch s {
	case "todo":
		return "TODO"
	case "in_progress":
		return "в работе"
	case "done":
		return "готово"
	}
	return s
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
	}

	switch msg.String() {
	case "enter", "right":
		s.toggleExpand()
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
	case "p":
		m.screen = screenProjects
		m.proj.load()
		return m, nil
	}
	var cmd tea.Cmd
	beforeKind, beforeID := s.selectedKindID()
	s.list, cmd = s.list.Update(msg)
	afterKind, afterID := s.selectedKindID()
	if beforeKind != afterKind || beforeID != afterID {
		s.loadInfo()
	}
	return m, cmd
}
