package ui

import (
	"fmt"
	"strings"

	"github.com/detrenasama/tasky/internal/ui/theme"
)

func (s *settingsScreen) dialog() (string, bool) {
	switch s.mode {
	case settingsDirInput:
		body := s.dirInput.View()
		if s.esc.Armed() {
			body += "\n" + theme.AccentText(escHintInput)
		}
		d := dialog{title: "Каталог сохранения отчётов",
			body:    body,
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
		if s.esc.Armed() {
			body += "\n" + theme.AccentText(escHintInput)
		}
		if s.lastErr != nil {
			body += "\n\n" + theme.ErrorStyle.Render("Ошибка: "+s.lastErr.Error())
		}
		d := dialog{title: "Свой период",
			body:    body,
			primary: "Enter — применить", esc: "Esc — отмена"}
		return d.render(), true
	case settingsHideInput:
		body := s.hideInput.View()
		if s.esc.Armed() {
			body += "\n" + theme.AccentText(escHintInput)
		}
		if s.lastErr != nil {
			body += "\n\n" + theme.ErrorStyle.Render("Ошибка: "+s.lastErr.Error())
		}
		d := dialog{title: "Скрытие завершённых задач",
			body:    body,
			primary: "Enter — сохранить", esc: "Esc — отмена"}
		return d.render(), true
	case settingsFeedInput:
		body2 := s.feedInput.View()
		if s.esc.Armed() {
			body2 += "\n" + theme.AccentText(escHintInput)
		}
		if s.lastErr != nil {
			body2 += "\n\n" + theme.ErrorStyle.Render("Ошибка: "+s.lastErr.Error())
		}
		d2 := dialog{title: "Час генерации ленты",
			body:    body2,
			primary: "Enter — сохранить · 0..23", esc: "Esc — отмена"}
		return d2.render(), true
	case settingsStatusList:
		body := s.statusPick.view()
		if s.lastErr != nil {
			body += "\n\n" + theme.ErrorStyle.Render("Ошибка: "+s.lastErr.Error())
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
			"Цвет:      " + colorPreview(theme.StatusPalette[s.editColor]) + " " + theme.PaletteNames[s.editColor],
			"Быстрая цепочка: " + boolWord(s.editQuick),
			"Подсказка: " + s.editNote.View(),
		}
		var body []string
		for i, l := range lines {
			if i == s.editFocus {
				body = append(body, theme.HeaderStyle.Render("▸ "+l))
			} else {
				body = append(body, "  "+l)
			}
		}
		body = append(body, "", theme.Faint("Тип — Enter, цвет — Enter, цепочка — Enter, Ctrl+S — сохранить"))
		inner := strings.Join(body, "\n")
		if s.esc.Armed() {
			inner += "\n" + theme.AccentText(escHintChanges)
		}
		if s.lastErr != nil {
			inner += "\n\n" + theme.ErrorStyle.Render("Ошибка: "+s.lastErr.Error())
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
	case settingsTagTypeList:
		body := s.tagTypePick.view()
		if s.lastErr != nil {
			body += "\n\n" + theme.ErrorStyle.Render("Ошибка: "+s.lastErr.Error())
		}
		d := dialog{title: "Типы тегов",
			body:    body,
			primary: "Enter — изменить · n — новый · d — удалить", esc: "Esc — назад"}
		return d.render(), true
	case settingsTagTypeEdit:
		title := "Новый тип тега"
		if s.tagTypeEditID != 0 {
			title = "Тип тега"
		}
		lines := []string{
			"Имя:   " + s.editName.View(),
			"Тип:   " + tagKindNames[s.editKind],
			"Цвет:  " + colorPreview(theme.StatusPalette[s.editColor]) + " " + theme.PaletteNames[s.editColor],
		}
		var body []string
		for i, l := range lines {
			if i == s.editFocus {
				body = append(body, theme.HeaderStyle.Render("▸ "+l))
			} else {
				body = append(body, "  "+l)
			}
		}
		body = append(body, "", theme.Faint("Тип — Enter, цвет — Enter, Ctrl+S — сохранить"))
		inner := strings.Join(body, "\n")
		if s.esc.Armed() {
			inner += "\n" + theme.AccentText(escHintChanges)
		}
		if s.lastErr != nil {
			inner += "\n\n" + theme.ErrorStyle.Render("Ошибка: "+s.lastErr.Error())
		}
		d := dialog{title: title, body: inner,
			primary: "Ctrl+S — сохранить", esc: "Esc — отмена"}
		return d.render(), true
	case settingsTagTypeConfirm:
		name := ""
		for _, tt := range s.tagTypes {
			if tt.ID == s.tagTypeDelID {
				name = tt.Name
			}
		}
		d := dialog{title: "Удаление типа тега",
			body:    fmt.Sprintf("Удалить тип тега «%s»?", name),
			primary: "y — удалить", esc: "n — нет"}
		return d.render(), true
	case settingsThemeList:
		body := s.themePick.view()
		if s.lastErr != nil {
			body += "\n\n" + theme.ErrorStyle.Render("Ошибка: "+s.lastErr.Error())
		}
		body += "\n\n" + theme.Faint("темы: "+theme.ThemesDir())
		d := dialog{title: "Тема",
			body:    body,
			primary: "Enter — выбрать", esc: "Esc — отмена"}
		return d.render(), true
	}
	return "", false
}
