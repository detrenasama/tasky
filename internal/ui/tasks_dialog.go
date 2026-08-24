package ui

import (
	"fmt"
	"strings"

	"github.com/detrenasama/tasky/internal/ui/theme"
)

func (s *tasksScreen) dialog() (string, bool) {
	switch s.mode {
	case taskInput:
		kind := "задача"
		if s.inputKind == kindSubtask {
			kind = "подзадача"
		}
		body := s.input.View()
		if s.lastErr != nil {
			body += "\n\n" + theme.ErrorStyle.Render("Ошибка: "+s.lastErr.Error())
		}
		d := dialog{title: "Новая " + kind, body: body,
			primary: "Enter — сохранить", esc: "Esc — отмена"}
		return d.render(), true
	case taskTitleEdit:
		kind := "задача"
		if s.inputKind == kindSubtask {
			kind = "подзадача"
		}
		body := s.input.View()
		if s.lastErr != nil {
			body += "\n\n" + theme.ErrorStyle.Render("Ошибка: "+s.lastErr.Error())
		}
		d := dialog{title: "Название " + kind, body: body,
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
	case taskLinkEdit:
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
	case taskLinks:
		var body string
		if len(s.links) == 0 {
			body = theme.Faint("Ссылок нет. n — добавить.")
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
		d := dialog{title: "Ссылки",
			body:    body,
			primary: hint}
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
			body += "\n\n" + theme.ErrorStyle.Render("Ошибка: "+s.lastErr.Error())
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
			body += "\n\n" + theme.ErrorStyle.Render("Ошибка: "+s.lastErr.Error())
		}
		d := dialog{title: title, body: body,
			primary: "Ctrl+S — применить", esc: "Esc — отмена"}
		return d.render(), true
	case taskSearch:
		body := s.searchInput.View()
		if s.searchQuery != "" {
			body += "\n\n" + theme.Faint("Найдено: "+fmt.Sprint(len(s.items))+" элементов")
		}
		d := dialog{title: "Поиск", body: body,
			primary: "Enter — применить", esc: "Esc — отмена"}
		return d.render(), true
	case taskTags:
		body := s.tagPick.view()
		if len(s.tags) == 0 {
			body = theme.Faint("Тегов нет. n — добавить.")
		}
		if s.lastErr != nil {
			body += "\n\n" + theme.ErrorStyle.Render("Ошибка: "+s.lastErr.Error())
		}
		d := dialog{title: "Теги",
			body:    body,
			primary: "Enter — изменить · n — новый · d — удалить", esc: "Esc — закрыть"}
		return d.render(), true
	case taskTagEdit:
		title := "Новый тег"
		if s.tagEditID != 0 {
			title = "Тег"
		}
		name, color := s.tagTypeName(s.tagEditType)
		lines := []string{
			"Тип:      " + colorPreview(color) + " " + name,
			"Значение: " + s.tagEditText.View(),
			"URL:      " + s.tagEditURL.View(),
		}
		var body []string
		for i, l := range lines {
			if i == s.tagEditFocus {
				body = append(body, theme.HeaderStyle.Render("▸ "+l))
			} else {
				body = append(body, "  "+l)
			}
		}
		body = append(body, "", theme.Faint("Тип — Enter, Ctrl+S — сохранить"))
		inner := strings.Join(body, "\n")
		if s.lastErr != nil {
			inner += "\n\n" + theme.ErrorStyle.Render("Ошибка: "+s.lastErr.Error())
		}
		d := dialog{title: title, body: inner,
			primary: "Ctrl+S — сохранить", esc: "Esc — отмена"}
		return d.render(), true
	case taskTagTypePick:
		d := dialog{title: "Тип тега",
			body:    s.tagTypePick.view(),
			primary: "Enter — выбрать", esc: "Esc — отмена"}
		return d.render(), true
	case taskTagConfirm:
		label := ""
		for _, t := range s.tags {
			if t.ID == s.tagConfirmID {
				label = t.Text
			}
		}
		d := dialog{title: "Удаление тега",
			body:    fmt.Sprintf("Удалить тег «%s»?", label),
			primary: "y — удалить", esc: "n — нет"}
		return d.render(), true

	case taskChecklist:
		var body []string
		if len(s.checklistPick.items) == 0 {
			body = append(body, theme.Faint("Список пуст. n — добавить пункт."))
		} else {
			for i := range s.checklistPick.items {
				label := "  " + s.checklistPick.items[i].label
				if i == s.checklistPick.sel {
					label = theme.HeaderStyle.Render("▸ ") + s.checklistPick.items[i].label
				}
				body = append(body, label)
			}
		}
		inner := strings.Join(body, "\n")
		if s.checklistNew {
			inner += "\n\n" + s.checklistInput.View()
		}
		if s.lastErr != nil {
			inner += "\n\n" + theme.ErrorStyle.Render("Ошибка: "+s.lastErr.Error())
		}
		primary := "Enter — переключить"
		if s.checklistNew {
			primary = "Enter — сохранить"
		}
		d := dialog{title: "Чек-лист",
			body:    inner,
			primary: primary,
			esc:     "Esc — закрыть"}
		return d.render(), true

	case taskChecklistConfirm:
		label := ""
		for _, ci := range s.checklistItems {
			if ci.ID == s.checklistConfirmID {
				label = ci.Text
			}
		}
		d := dialog{title: "Удаление пункта чек-листа",
			body:    fmt.Sprintf("Удалить пункт «%s»?", label),
			primary: "y — удалить", esc: "n — нет"}
		return d.render(), true
	}
	return "", false
}

// descBox — средняя колонка: описание, ссылки и журнал выбранного элемента
// (прокручиваемый viewport); при редактировании описания вместо контента —
// textarea.
