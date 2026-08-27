package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/detrenasama/tasky/internal/ui/theme"
)

func (s *projectsScreen) footer(w int) string {
	// подсказки и заметки (успех/ошибка) рисуются внутри модалки описания
	// (descModalFooter / renderDescModal), в общей панели подсказок не дублируем
	return ""
}

// descTitle возвращает название выбранного проекта для контекста модалки.
func (s *projectsScreen) descTitle() string {
	item, ok := s.list.SelectedItem().(projectItem)
	if !ok {
		return ""
	}
	return item.p.Name
}

func (s *projectsScreen) descModalFooter() string {
	switch s.dmState {
	case dmView:
		return "↑↓/PgUp/PgDn — прокрутка · e — править · E — редактор · v — выделение · y — копировать · Esc — выйти"
	case dmSelect:
		return "←→↑↓/hjkl — движение · Space — начать выделение · Enter — копировать · v/Esc — выход · d — удалить"
	case dmEdit:
		return "Ctrl+S — сохранить · Esc — отмена"
	}
	return ""
}

// renderDescModal строит крупную модалку описания проекта. Возвращаемая
// строка уже имеет размер модалки и центрируется через overlay.
func (s *projectsScreen) renderDescModal() string {
	mW, mH := s.descModalDims()
	style := theme.ModalStyle
	hFrame := style.GetHorizontalFrameSize()
	vFrame := style.GetVerticalFrameSize()
	innerW := max(mW-hFrame, 1)
	innerH := max(mH-vFrame, 1)
	bodyH := max(innerH-3, 1)
	noticeLine := ""
	if s.notice != "" {
		bodyH = max(innerH-4, 1)
		st := theme.SaveOKStyle
		if strings.HasPrefix(s.notice, "Не удалось") {
			st = theme.ErrorStyle
		}
		noticeLine = "\n" + padW(st.Render(s.notice), innerW)
	}

	s.descViewer.width = innerW
	s.descViewer.height = bodyH
	s.descText.SetWidth(innerW)
	s.descText.SetHeight(bodyH)

	title := theme.HeaderStyle.Render("Описание")
	if t := s.descTitle(); t != "" {
		title += " · " + theme.Faint(t)
	}
	var body string
	if s.dmState == dmEdit {
		body = s.descText.View()
	} else {
		body = s.descViewer.view()
	}
	footer := s.descModalFooter()

	var b strings.Builder
	b.WriteString(padW(title, innerW))
	b.WriteString("\n")
	b.WriteString(padW("", innerW))
	b.WriteString("\n")
	b.WriteString(body)
	b.WriteString(noticeLine)
	b.WriteString("\n")
	b.WriteString(padW(footer, innerW))
	inner := padLines(b.String(), innerW, innerH)
	box := renderPane(style, inner)

	if s.dmState == dmDiscard {
		d := dialog{title: "Несохранённые изменения",
			body:    "Описание изменено, но не сохранено.",
			primary: "y — отбросить", esc: "Esc — остаться"}
		box = overlay(box, d.render(), mW, mH, 0)
	}
	return box
}

func (s *projectsScreen) view(w, h int) string {
	leftStyle := theme.Pane(false)
	// colW — полная ширина левой колонки (контент + 2 пробела-разделителя),
	// равна ширине списка; заголовок/отступ/дополнение рендерятся через
	// theme.Pane (её фрейм 2 уже входит в colW), поэтому их ширина совпадает.
	colW := s.listW
	if s.descW > 0 {
		colW += 2
	}
	W := max(colW-leftStyle.GetHorizontalFrameSize(), 0)
	H := max(s.midH, 1)
	titleTxt := padW(theme.HeaderStyle.Render("Проекты"), W)
	var body string
	switch {
	case len(s.projects) == 0 && s.mode == projBrowse:
		body = renderPane(leftStyle, padLines("Проектов нет.\nНажмите n для создания.", W, max(H-2, 1)))
	case s.searchQuery != "" && len(s.items) == 0:
		body = renderPane(leftStyle, padLines("Ничего не найдено по запросу\n«"+s.searchQuery+"».", W, max(H-2, 1)))
	default:
		// bubbles/list не дополняет строки до ширины — паддинг внутри
		// делегата (selDelegate закрашивает каждую строку целиком)
		body = s.list.View()
	}
	// заголовок страницы (фон контента) + отступ 1 строка + тело списка
	// (строки несут собственный фон: серый для выделенной, контент для
	// остальных), затем дополнение до H строк фоном контента.
	left := renderPane(leftStyle, titleTxt) + "\n" + renderPane(leftStyle, padW("", W)) + "\n" + body
	lines := strings.Split(left, "\n")
	for len(lines) < H {
		lines = append(lines, renderPane(leftStyle, padW("", W)))
	}
	if len(lines) > H {
		lines = lines[:H]
	}
	left = strings.Join(lines, "\n")

	cols := []string{left}
	if s.descW > 0 {
		cols = append(cols, s.descBox())
	}
	bodyOut := lipgloss.JoinHorizontal(lipgloss.Top, cols...)
	return padH(bodyOut, w, h)
}

// rightContent возвращает содержимое правой колонки (без панели); панель и
// «Tasky vX» снизу добавляет app.rightRail.
func (s *projectsScreen) rightContent(h int) string {
	return "Сводка и настройки\n\nКоэффициент времени:\n1.0"
}

// descBox — средняя колонка: описание проекта (read-only). Модалка описания
// (projDescModal) рисуется поверх через overlay.
func (s *projectsScreen) descBox() string {
	style := theme.Pane(false)
	return renderPane(style, s.descV.View())
}

// infoBox removed: правая колонка рендерится через rightRail/rightContent.
