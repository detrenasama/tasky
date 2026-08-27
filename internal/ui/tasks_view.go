package ui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/detrenasama/tasky/internal/ui/theme"
	"strings"
)

func (s *tasksScreen) footer(w int) string {
	// подсказки и заметки (успех/ошибка) рисуются внутри модалки описания
	// (descModalFooter / renderDescModal), в общей панели подсказок не дублируем
	return ""
}

// descTitle возвращает заголовок выбранной задачи/подзадачи для контекста
// в модалке описания.
func (s *tasksScreen) descTitle() string {
	kind, id := s.selectedKindID()
	if kind == kindTask {
		for _, t := range s.tasks {
			if t.ID == id {
				return t.Title
			}
		}
	} else {
		for _, st := range s.subs {
			if st.ID == id {
				return st.Title
			}
		}
	}
	return ""
}

func (s *tasksScreen) descModalFooter() string {
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

// renderDescModal строит крупную модалку описания (просмотр/выделение/
// правка/подтверждение отмены). Возвращаемая строка уже имеет размер
// модалки и центрируется через overlay.
func (s *tasksScreen) renderDescModal() string {
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
	b.WriteString(padW(styleHints(footer), innerW))
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

func (s *tasksScreen) view(w, h int) string {
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
	titleLine := theme.HeaderStyle.Render("Задачи")
	if name := s.currentProjectName(); name != "" {
		projNameStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#ffffff"))
		right := projNameStyle.Render(name)
		gap := W - lipgloss.Width(titleLine) - lipgloss.Width(right)
		if gap < 1 {
			right = projNameStyle.Render(truncateWEnd(name, max(W-lipgloss.Width(titleLine)-1, 1)))
			gap = 1
		}
		titleLine = titleLine + strings.Repeat(" ", gap) + right
	}
	titleTxt := padW(titleLine, W)
	var body string
	switch {
	case len(s.projects) == 0:
		body = renderPane(leftStyle, padLines("Нет проектов.\nНажмите p и создайте проект.", W, max(H-2, 1)))
	case s.searchQuery != "" && len(s.items) == 0:
		body = renderPane(leftStyle, padLines("Ничего не найдено по запросу\n«"+s.searchQuery+"».", W, max(H-2, 1)))
	case len(s.tasks) == 0:
		body = renderPane(leftStyle, padLines("Задач в проекте нет.", W, max(H-2, 1)))
	default:
		// bubbles/list «развёрнут» на все элементы; listView() режет
		// видимое окно с плавной прокруткой и отступом listScrollOff.
		body = s.listView()
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

// rightContent возвращает содержимое правой колонки (без панели) заданной
// высоты; панель и «Tasky vX» снизу добавляет app.rightRail.
func (s *tasksScreen) rightContent(h int) string {
	bot := s.infoBottom()
	botH := len(strings.Split(bot, "\n"))
	topH := h - botH - 1
	if topH < 3 {
		topH = 3
	}
	return s.infoTop(topH) + "\n\n" + bot
}

func (s *tasksScreen) descBox() string {
	style := theme.Pane(false)
	// модалка описания (taskDescModal) рисуется поверх через overlay;
	// здесь колонка показывает только read-only описание.
	return renderPane(style, s.descV.View())
}
