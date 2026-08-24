package ui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/detrenasama/tasky/internal/ui/theme"
	"strings"
)

func (s *tasksScreen) footer(w int) string {
	if s.mode == taskDescEdit {
		return "Ctrl+S — сохранить · Esc — отмена"
	}
	// подсказки действий перенесены в палитру команд (Ctrl+P); здесь не выводим
	return ""
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
	if s.mode == taskDescEdit {
		return renderPane(style, padLines(s.descText.View(), max(s.descW-style.GetHorizontalFrameSize(), 0), max(s.midH-style.GetVerticalFrameSize(), 0)))
	}
	return renderPane(style, s.descV.View())
}
