package main

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbletea"

	"github.com/kalpamer/tasky/internal/db"
)

type projMode int

const (
	projBrowse projMode = iota
	projInput
	projConfirm
)

type projectItem struct{ p db.Project }

func (i projectItem) FilterValue() string { return i.p.Name }

func (i projectItem) Title() string { return i.p.Name }

func (i projectItem) Description() string {
	return "создан " + i.p.CreatedAt.Format("02.01.2006")
}

type projectsScreen struct {
	db        *sql.DB
	projects  []db.Project
	list      list.Model
	mode      projMode
	input     textinput.Model
	confirmID int64
	lastErr   error
}

func newProjectsScreen(conn *sql.DB) *projectsScreen {
	s := &projectsScreen{db: conn, mode: projBrowse}
	d := list.NewDefaultDelegate()
	d.ShowDescription = true
	s.list = list.New(nil, d, 40, 20)
	s.list.Title = "Проекты"
	s.list.SetShowHelp(false)
	s.list.SetShowPagination(false)
	s.list.SetShowStatusBar(false)

	s.input = textinput.New()
	s.input.Placeholder = "Имя проекта"
	s.input.Prompt = "> "
	s.input.CharLimit = 64
	s.input.Width = 40
	return s
}

func (s *projectsScreen) load() {
	s.projects, _ = db.Projects(s.db)
	items := make([]list.Item, len(s.projects))
	for i, p := range s.projects {
		items[i] = projectItem{p}
	}
	s.list.SetItems(items)
	s.lastErr = nil
}

func (s *projectsScreen) resize(w, h int) {
	s.list.SetWidth(w - 2)
	s.list.SetHeight(h - 2)
}

func (s *projectsScreen) header(w int) string {
	return padW(headerStyle.Render("Tasky")+"  "+faint("Проекты"), w)
}

func (s *projectsScreen) footer(w int) string {
	return padW(faint("n — создать · d — удалить · esc — назад · q — выход"), w)
}

func (s *projectsScreen) view(w, h int) string {
	var body string
	if len(s.projects) == 0 && s.mode == projBrowse {
		body = dimBox.Render("Проектов нет.\nНажмите n для создания.")
	} else {
		body = focusBox.Render(s.list.View())
	}
	return padH(body, w, h)
}

func (s *projectsScreen) dialog() (string, bool) {
	switch s.mode {
	case projInput:
		body := s.input.View()
		if s.lastErr != nil {
			body += "\n\n" + errorStyle.Render("Ошибка: "+s.lastErr.Error())
		}
		d := dialog{title: "Новый проект", body: body,
			primary: "Enter — создать", esc: "Esc — отмена"}
		return d.render(), true
	case projConfirm:
		name := ""
		for _, p := range s.projects {
			if p.ID == s.confirmID {
				name = p.Name
			}
		}
		d := dialog{title: "Удаление проекта",
			body:    fmt.Sprintf("Удалить проект «%s» и всё его время?", name),
			primary: "y — удалить", esc: "n — нет"}
		return d.render(), true
	}
	return "", false
}

func (m *model) updateProjects(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	s := m.proj
	switch s.mode {
	case projInput:
		var cmd tea.Cmd
		s.input, cmd = s.input.Update(msg)
		switch msg.String() {
		case "enter":
			name := strings.TrimSpace(s.input.Value())
			if name != "" {
				_, err := db.CreateProject(s.db, name)
				s.lastErr = err
				if err == nil {
					s.load()
				}
			}
			s.input.SetValue("")
			s.input.Blur()
			s.mode = projBrowse
		case "esc":
			s.input.SetValue("")
			s.input.Blur()
			s.mode = projBrowse
		}
		return m, cmd
	case projConfirm:
		switch msg.String() {
		case "y", "enter":
			db.DeleteProject(s.db, s.confirmID)
			s.mode = projBrowse
			s.load()
		case "n", "esc":
			s.mode = projBrowse
		}
		return m, nil
	default:
		switch msg.String() {
		case "n":
			s.mode = projInput
			s.input.Focus()
			return m, nil
		case "d":
			item, ok := s.list.SelectedItem().(projectItem)
			if ok {
				s.confirmID = item.p.ID
				s.mode = projConfirm
			}
			return m, nil
		case "esc":
			m.screen = screenTasks
			m.tasks.load()
			return m, nil
		}
		var cmd tea.Cmd
		s.list, cmd = s.list.Update(msg)
		return m, cmd
	}
}
