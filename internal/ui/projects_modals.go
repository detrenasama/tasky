package ui

import (
	"fmt"

	"github.com/detrenasama/tasky/internal/ui/theme"
)

func (s *projectsScreen) startNew() {
	s.mode = projInput
	s.input.Focus()
}

// startDelete открывает подтверждение удаления выбранного проекта.
func (s *projectsScreen) startDelete() {
	item, ok := s.list.SelectedItem().(projectItem)
	if !ok {
		return
	}
	s.confirmID = item.p.ID
	s.mode = projConfirm
}

// startEditDescription открывает инлайн-редактирование описания проекта
// (Enter в списке — то же, что раньше «e» в колонке описания).
func (s *projectsScreen) startEditDescription() {
	s.descText.SetValue(s.desc)
	s.mode = projDescEdit
	s.descText.Focus()
}

// startLinkEdit открывает модалку ссылки проекта: id=0 — новая, иначе —
// правка существующей (поля префилливаются из списка ссылок).
func (s *projectsScreen) startLinkEdit(id int64) {
	s.lastErr = nil
	s.editLinkID = id
	s.linkName.SetValue("")
	s.linkInput.SetValue("")
	for _, l := range s.links {
		if l.ID == id {
			s.linkName.SetValue(l.Name)
			s.linkInput.SetValue(l.URL)
		}
	}
	s.mode = projLinkEdit
	s.linkName.Focus()
	s.linkInput.Blur()
}

// openLinks открывает список ссылок проекта.
func (s *projectsScreen) openLinks() {
	s.lastErr = nil
	s.mode = projLinks
}

// startSearch открывает модалку поиска по проектам.
func (s *projectsScreen) startSearch() {
	s.lastErr = nil
	s.searchInput.SetValue(s.searchQuery)
	s.searchInput.Focus()
	s.mode = projSearch
}

// refreshDesc собирает контент viewport колонки описания.
func (s *projectsScreen) dialog() (string, bool) {
	switch s.mode {
	case projInput:
		body := s.input.View()
		if s.lastErr != nil {
			body += "\n\n" + theme.ErrorStyle.Render("Ошибка: "+s.lastErr.Error())
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
	case projDescEdit:
		return "", false
	case projLinkEdit:
		title := "Добавить ссылку"
		if s.editLinkID != 0 {
			title = "Изменить ссылку"
		}
		body := s.linkName.View() + "\n" + s.linkInput.View()
		if s.lastErr != nil {
			body += "\n\n" + theme.ErrorStyle.Render("Ошибка: "+s.lastErr.Error())
		}
		d := dialog{title: title, body: body,
			primary: "Enter — сохранить · Tab — поле", esc: "Esc — отмена"}
		return d.render(), true
	case projLinks:
		var body string
		if len(s.links) == 0 {
			body = theme.Faint("Ссылок проекта нет. n — добавить.")
		} else {
			body = s.linkList.View()
		}
		if s.lastErr != nil {
			body += "\n\n" + theme.ErrorStyle.Render("Не удалось открыть: "+s.lastErr.Error())
		}
		hint := "n — новая · Esc — закрыть"
		if len(s.links) > 0 {
			hint = "Enter — открыть · e — изменить · d — удалить · " + hint
		}
		d := dialog{title: "Ссылки проекта",
			body:    body,
			primary: hint}
		return d.render(), true
	case projLinkConfirm:
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
	case projSearch:
		body := s.searchInput.View()
		if s.searchQuery != "" {
			body += "\n\n" + theme.Faint(fmt.Sprintf("Найдено: %d элементов", len(s.items)))
		}
		d := dialog{title: "Поиск", body: body,
			primary: "Enter — применить", esc: "Esc — отмена"}
		return d.render(), true
	}
	return "", false
}
