package ui

import (
	"fmt"
	"os"

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

// startEditDescription открывает модалку описания проекта в режиме просмотра
// (Enter в списке).
func (s *projectsScreen) startEditDescription() {
	s.openDescModal(false)
}

// openDescModal открывает крупную модалку описания проекта. sel=true — сразу
// в режиме визуального выделения.
func (s *projectsScreen) openDescModal(sel bool) {
	s.lastErr = nil
	s.notice = ""
	s.loadDesc()
	s.descWork = s.desc
	mW, mH := s.descModalDims()
	v := newDescViewer(s.descWork, mW-4, mH-4)
	v.plain = true
	v.scroll = 0
	if sel {
		v.plain = false
		v.cursor = 0
		v.anchor = 0
		s.dmState = dmSelect
	} else {
		s.dmState = dmView
	}
	s.descViewer = v
	s.mode = projDescModal
}

func (s *projectsScreen) descModalDims() (int, int) {
	fw, fh := s.fullW, s.fullH
	if fw <= 0 {
		fw = 150
	}
	if fh <= 0 {
		fh = 40
	}
	mW := max(fw*4/5, 60)
	if mW > fw-4 {
		mW = max(fw-4, 20)
	}
	mH := max(fh*4/5, 20)
	if mH > fh-4 {
		mH = max(fh-4, 10)
	}
	return mW, mH
}

func (s *projectsScreen) refreshDescViewer() {
	mW, mH := s.descModalDims()
	v := newDescViewer(s.descWork, mW-4, mH-4)
	v.plain = true
	v.scroll = 0
	s.descViewer = v
}

func (s *projectsScreen) deleteDescSelection() {
	rs := []rune(s.descWork)
	a, b := s.descViewer.anchor, s.descViewer.cursor
	if a > b {
		a, b = b, a
	}
	if a < 0 {
		a = 0
	}
	if b > len(rs) {
		b = len(rs)
	}
	s.descWork = string(append(rs[:a], rs[b:]...))
	s.refreshDescViewer()
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
	case projDescModal:
		return s.renderDescModal(), true
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

// applyExternalEdit вызывается после завершения внешнего редактора ($EDITOR):
// читает временный файл и либо подгружает текст в textarea (режим правки
// описания), либо сразу сохраняет в проект (режим просмотра).
func (s *projectsScreen) applyExternalEdit(msg editReturnMsg) {
	defer os.Remove(msg.path)
	if msg.err != nil {
		s.notice = "Ошибка редактора: " + msg.err.Error()
		return
	}
	data, err := os.ReadFile(msg.path)
	if err != nil {
		s.notice = "Не удалось прочитать файл: " + err.Error()
		return
	}
	text := string(data)
	if s.extEditMode == 1 {
		s.descText.SetValue(text)
		return
	}
	if s.extEditMode == 2 {
		// модалка описания: текст сразу сохраняется в БД (если изменился),
		// возврат в просмотр; без изменений — ничего не пишем
		s.descWork = text
		if text != s.desc {
			s.saveDescWork()
			s.notice = "Описание обновлено"
		}
		s.refreshDescViewer()
		s.dmState = dmView
		s.extEditMode = 0
		s.extEditPath = ""
		return
	}
	s.store.UpdateProjectDescription(s.selectedProjectID(), text)
	s.loadDesc()
	s.notice = "Описание обновлено"
}

// saveDescWork сохраняет рабочую копию описания выбранного проекта в БД.
func (s *projectsScreen) saveDescWork() {
	s.store.UpdateProjectDescription(s.selectedProjectID(), s.descWork)
	s.desc = s.descWork
	s.loadDesc()
}

// copyDescSelection копирует выделенное (или весь текст, если выделения нет)
// в буфер обмена и сбрасывает выделение, возвращаясь в режим просмотра.
func (s *projectsScreen) copyDescSelection() {
	text := s.descWork
	if s.dmState == dmSelect {
		text = s.descViewer.selectedText()
	}
	if text == "" {
		text = s.descWork
	}
	if err := copyToClipboard(text); err != nil {
		s.notice = "Не удалось скопировать: " + err.Error()
	} else {
		s.notice = "Скопировано в буфер обмена"
	}
	s.descViewer.plain = true
	s.descViewer.anchor = -1
	s.descViewer.scroll = s.descViewer.lineOfCursor(s.descViewer.layout())
	s.dmState = dmView
}
