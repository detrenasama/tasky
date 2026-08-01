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

type paneFocus bool

const (
	focusTasks paneFocus = true
	focusSubs  paneFocus = false
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

var (
	boxStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1)
	focusBox = boxStyle.Copy().BorderForeground(accent)
	dimBox   = boxStyle.Copy().Faint(true)
)

type taskItem struct{ t db.Task }

func (i taskItem) FilterValue() string { return i.t.Title }

func (i taskItem) Title() string { return i.t.Title }

func (i taskItem) Description() string {
	plural := "подзадач"
	if i.t.SubCount == 1 {
		plural = "подзадача"
	}
	return fmt.Sprintf("%s · %d %s", statusRU(i.t.Status), i.t.SubCount, plural)
}

type subItem struct {
	st  db.SubtaskWithTime
	scr *tasksScreen
}

func (i subItem) FilterValue() string { return i.st.Title }

func (i subItem) Title() string { return i.st.Title }

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
	taskList list.Model
	subs     []db.SubtaskWithTime
	subList  list.Model
	focus    paneFocus
	weekly   time.Duration
	now      time.Time

	mode        taskMode
	input       textinput.Model
	inputKind   paneKind
	confirmKind paneKind
	confirmID   int64
	lastErr     error
}

func newTasksScreen(conn *sql.DB) *tasksScreen {
	s := &tasksScreen{db: conn, focus: focusTasks}

	td := list.NewDefaultDelegate()
	td.ShowDescription = true
	s.taskList = list.New(nil, td, 40, 20)
	s.taskList.Title = "Задачи"
	s.taskList.SetShowHelp(false)
	s.taskList.SetShowPagination(false)
	s.taskList.SetShowStatusBar(false)

	sd := list.NewDefaultDelegate()
	sd.ShowDescription = true
	s.subList = list.New(nil, sd, 40, 20)
	s.subList.Title = "Подзадачи"
	s.subList.SetShowHelp(false)
	s.subList.SetShowPagination(false)
	s.subList.SetShowStatusBar(false)

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
	if len(s.projects) > 0 {
		s.loadTasks()
	} else {
		s.tasks = nil
		s.subs = nil
		s.taskList.SetItems(nil)
		s.subList.SetItems(nil)
	}
	s.weekly, _ = db.WeeklyTotal(s.db, s.now)
}

func (s *tasksScreen) currentProjectID() int64 {
	if s.projIdx < 0 || s.projIdx >= len(s.projects) {
		return 0
	}
	return s.projects[s.projIdx].ID
}

func (s *tasksScreen) loadTasks() {
	s.tasks, _ = db.TasksByProject(s.db, s.currentProjectID())
	items := make([]list.Item, len(s.tasks))
	for i, t := range s.tasks {
		items[i] = taskItem{t}
	}
	s.taskList.SetItems(items)
	if len(items) > 0 {
		s.taskList.Select(0)
	}
	s.loadSubs()
}

func (s *tasksScreen) loadSubs() {
	s.subs = nil
	if s.taskList.Index() < 0 {
		s.subList.SetItems(nil)
		return
	}
	ti, ok := s.taskList.SelectedItem().(taskItem)
	if !ok {
		s.subList.SetItems(nil)
		return
	}
	s.subs, _ = db.SubtasksWithTime(s.db, ti.t.ID)
	items := make([]list.Item, len(s.subs))
	for i, st := range s.subs {
		items[i] = subItem{st: st, scr: s}
	}
	s.subList.SetItems(items)
	if len(items) > 0 {
		idx := s.subList.Index()
		if idx >= len(items) {
			idx = len(items) - 1
		}
		s.subList.Select(idx)
	}
}

func (s *tasksScreen) switchProject(dir int) {
	if len(s.projects) < 2 {
		return
	}
	s.projIdx = (s.projIdx + dir + len(s.projects)) % len(s.projects)
	s.loadTasks()
}

func (s *tasksScreen) focusedKind() paneKind {
	if s.focus == focusSubs {
		return kindSubtask
	}
	return kindTask
}

func (s *tasksScreen) selectedTaskID() int64 {
	ti, ok := s.taskList.SelectedItem().(taskItem)
	if !ok {
		return 0
	}
	return ti.t.ID
}

func (s *tasksScreen) canCreate() bool {
	if s.focusedKind() == kindTask {
		return s.projIdx >= 0 && s.projIdx < len(s.projects)
	}
	return s.selectedTaskID() != 0
}

func (s *tasksScreen) canDelete() bool {
	if s.focusedKind() == kindTask {
		return s.taskList.Index() >= 0 && len(s.tasks) > 0
	}
	return s.subList.Index() >= 0 && len(s.subs) > 0
}

func (s *tasksScreen) selectTask(id int64) {
	for i, t := range s.tasks {
		if t.ID == id {
			s.taskList.Select(i)
			break
		}
	}
}

func (s *tasksScreen) selectSubtask(id int64) {
	for i, st := range s.subs {
		if st.ID == id {
			s.subList.Select(i)
			break
		}
	}
}

func (s *tasksScreen) createItem(kind paneKind, title string) error {
	if kind == kindTask {
		_, err := db.CreateTask(s.db, s.currentProjectID(), title)
		return err
	}
	_, err := db.CreateSubtask(s.db, s.selectedTaskID(), title)
	return err
}

func (s *tasksScreen) deleteItem(kind paneKind, id int64) {
	if kind == kindTask {
		db.DeleteTask(s.db, id)
		return
	}
	db.DeleteSubtask(s.db, id)
}

func (s *tasksScreen) toggleTimer() {
	if s.subList.Index() < 0 || len(s.subs) == 0 {
		return
	}
	sub := s.subs[s.subList.Index()]
	now := time.Now()
	if sub.ActiveSince != nil {
		db.StopSession(s.db, sub.ID, now)
	} else {
		db.StartSession(s.db, sub.ID, now)
	}
	s.now = now
	s.loadSubs()
	s.weekly, _ = db.WeeklyTotal(s.db, now)
}

func (s *tasksScreen) resize(w, h int) {
	bodyH := h - 5
	if bodyH < 3 {
		bodyH = 3
	}
	leftW := w*2/5 - 4
	if leftW < 20 {
		leftW = 20
	}
	rightW := w - leftW - 6
	if rightW < 20 {
		rightW = 20
	}
	s.taskList.SetWidth(leftW)
	s.taskList.SetHeight(bodyH)

	subH := bodyH - 6
	if subH < 3 {
		subH = 3
	}
	s.subList.SetWidth(rightW)
	s.subList.SetHeight(subH)
}

func (s *tasksScreen) view() string {
	header := headerStyle.Render("Tasky")
	if s.projIdx >= 0 && s.projIdx < len(s.projects) {
		header += "  " + faint("проект: ") + s.projects[s.projIdx].Name
	}

	body := lipgloss.JoinHorizontal(lipgloss.Top,
		s.taskPane(),
		"  ",
		lipgloss.JoinVertical(lipgloss.Left, s.subPane(), "", s.weeklyBox()),
	)

	switch s.mode {
	case taskInput:
		kind := "задачи"
		if s.inputKind == kindSubtask {
			kind = "подзадачи"
		}
		body += "\n\n" + boxStyle.Render("Название "+kind+":\n"+s.input.View())
		if s.lastErr != nil {
			body += "\n" + errorStyle.Render("Ошибка: "+s.lastErr.Error())
		}
	case taskConfirm:
		name := ""
		if s.confirmKind == kindTask {
			for _, t := range s.tasks {
				if t.ID == s.confirmID {
					name = t.Title
				}
			}
			body += "\n\n" + boxStyle.Render(
				fmt.Sprintf("Удалить задачу «%s» с подзадачами и временем? (y/n)", name))
		} else {
			for _, st := range s.subs {
				if st.ID == s.confirmID {
					name = st.Title
				}
			}
			body += "\n\n" + boxStyle.Render(
				fmt.Sprintf("Удалить подзадачу «%s» и её время? (y/n)", name))
		}
	}

	footer := faint("↑/↓ выбор · Tab фокус · n создать · d удалить · Ctrl+L старт/пауза · [ / ] проект · p проекты · q выход")
	return header + "\n\n" + body + "\n\n" + footer
}

func (s *tasksScreen) taskPane() string {
	if len(s.projects) == 0 {
		return dimBox.Render("Нет проектов.\nНажмите p и создайте проект.")
	}
	if len(s.tasks) == 0 {
		return dimBox.Render("Задач в проекте нет.")
	}
	view := s.taskList.View()
	if s.focus != focusTasks {
		view = faintStyle.Render(view)
	}
	return focusOrDim(s.taskList, view)
}

func (s *tasksScreen) subPane() string {
	if len(s.projects) == 0 || len(s.tasks) == 0 {
		return dimBox.Render("Подзадачи появятся здесь.")
	}
	if len(s.subs) == 0 {
		return dimBox.Render("Подзадач нет.")
	}
	view := s.subList.View()
	if s.focus != focusSubs {
		view = faintStyle.Render(view)
	}
	return focusOrDim(s.subList, view)
}

func focusOrDim(l list.Model, view string) string {
	if l.Index() < 0 {
		return dimBox.Render(view)
	}
	return focusBox.Render(view)
}

func (s *tasksScreen) weeklyBox() string {
	body := headerStyle.Render("Неделя (Пн–Вс)") + "\n" +
		faint("по всем проектам: ") + headerStyle.Render(fmtDur(s.weekly))
	return dimBox.Render(body)
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
	if h > 0 {
		return fmt.Sprintf("%dч %dм", h, m)
	}
	if m > 0 {
		return fmt.Sprintf("%dм", m)
	}
	return "0м"
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
				var created int64
				if s.inputKind == kindTask {
					t, err := db.CreateTask(s.db, s.currentProjectID(), title)
					s.lastErr = err
					created = t.ID
				} else {
					st, err := db.CreateSubtask(s.db, s.selectedTaskID(), title)
					s.lastErr = err
					created = st.ID
				}
				if s.lastErr == nil {
					s.loadTasks()
					if s.inputKind == kindTask {
						s.selectTask(created)
					} else {
						s.selectSubtask(created)
					}
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
			s.loadTasks()
		case "n", "esc":
			s.mode = taskBrowse
		}
		return m, nil
	}

	switch msg.String() {
	case "tab":
		if s.projIdx >= 0 && len(s.tasks) > 0 {
			s.focus = !s.focus
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
		if s.canCreate() {
			s.inputKind = s.focusedKind()
			s.lastErr = nil
			s.mode = taskInput
			s.input.Focus()
		}
		return m, nil
	case "d":
		if s.canDelete() {
			s.confirmKind = s.focusedKind()
			if s.confirmKind == kindTask {
				s.confirmID = s.tasks[s.taskList.Index()].ID
			} else {
				s.confirmID = s.subs[s.subList.Index()].ID
			}
			s.mode = taskConfirm
		}
		return m, nil
	case "p":
		m.screen = screenProjects
		m.proj.load()
		return m, nil
	}
	if s.focus == focusTasks {
		before := s.taskList.Index()
		var cmd tea.Cmd
		s.taskList, cmd = s.taskList.Update(msg)
		if s.taskList.Index() != before {
			s.loadSubs()
		}
		return m, cmd
	}
	var cmd tea.Cmd
	s.subList, cmd = s.subList.Update(msg)
	return m, cmd
}
