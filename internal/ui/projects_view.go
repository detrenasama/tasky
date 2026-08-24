package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/detrenasama/tasky/internal/ui/theme"
)

func (s *projectsScreen) footer(w int) string {
	if s.mode == projDescEdit {
		return "Ctrl+S — сохранить · Esc — отмена"
	}
	// подсказки действий перенесены в палитру команд (Ctrl+P); здесь не выводим
	return ""
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

// descBox — средняя колонка: описание проекта (переносимое, прокручиваемое)
// и блок «Ссылки»; при редактировании вместо контента — textarea.
func (s *projectsScreen) descBox() string {
	style := theme.Pane(false)
	if s.mode == projDescEdit {
		return renderPane(style, padLines(s.descText.View(), max(s.descW-style.GetHorizontalFrameSize(), 0), max(s.midH-style.GetVerticalFrameSize(), 0)))
	}
	return renderPane(style, s.descV.View())
}

// infoBox removed: правая колонка рендерится через rightRail/rightContent.
