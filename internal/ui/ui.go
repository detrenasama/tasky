package ui

import (
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

// padW обрезает строку до width (по видимой ширине, с учётом ANSI-кодов)
// и дополняет пробелами.
func padW(line string, width int) string {
	if width < 0 {
		width = 0
	}
	line = truncateW(line, width)
	return line + strings.Repeat(" ", max(width-lipgloss.Width(line), 0))
}

// truncateW обрезает строку до видимой ширины width, сохраняя ANSI-коды
// внутри усечённой части.
func truncateW(line string, width int) string {
	var b strings.Builder
	visible := 0
	for i := 0; i < len(line); {
		if line[i] == '\x1b' {
			end := ansiEnd(line, i)
			b.WriteString(line[i:min(end+1, len(line))])
			i = end + 1
			continue
		}
		r, size := utf8.DecodeRuneInString(line[i:])
		rw := runewidth.RuneWidth(r)
		if visible+rw > width {
			break
		}
		b.WriteString(line[i : i+size])
		visible += rw
		i += size
	}
	return b.String()
}

// truncateWEnd обрезает строку до видимой ширины width, сохраняя ANSI-коды;
// если строка не влезла, хвост заменяется многоточием.
func truncateWEnd(line string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(line) <= width {
		return line
	}
	return truncateW(line, width-1) + "…"
}

// padH приводит body к холсту w×h: обрезает лишние строки, дополняет
// недостающие пробелами.
func padH(body string, w, h int) string {
	lines := strings.Split(strings.ReplaceAll(body, "\r\n", "\n"), "\n")
	if len(lines) > h {
		lines = lines[:h]
	}
	for i := range lines {
		lines[i] = padW(lines[i], w)
	}
	for len(lines) < h {
		lines = append(lines, strings.Repeat(" ", w))
	}
	return strings.Join(lines, "\n")
}

// fixedBox рендерит content в панели style ровно w×h (контент дополняется
// пробелами по ширине и высоте, лишние строки обрезаются). Размер хрома
// берётся из самого стиля (рамки/паддинга), поэтому работает и для плоских
// панелей без рамок, и для классических рамок.
func fixedBox(style lipgloss.Style, content string, w, h int) string {
	if h < 2 {
		h = 2
	}
	inner := padLines(content, max(w-style.GetHorizontalFrameSize(), 1), h-style.GetVerticalFrameSize())
	return renderPane(style, inner)
}

// padLines приводит многострочный текст к холсту w×h, дополняя пробелами
// и обрезая лишние строки.
func padLines(content string, w, h int) string {
	if h <= 0 {
		return ""
	}
	if w < 0 {
		w = 0
	}
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	if len(lines) > h {
		lines = lines[:h]
	}
	for i := range lines {
		lines[i] = padW(lines[i], w)
	}
	for len(lines) < h {
		lines = append(lines, strings.Repeat(" ", w))
	}
	return strings.Join(lines, "\n")
}

// renderPane рендерит content в панель style, восстанавливая фон после
// внутренних \x1b[0m контента: липглосс оборачивает строку одной SGR-посылкой,
// и сброс внутри цветного фрагмента (статус, muted-подпись, выделенный
// элемент списка) гасит фон до конца строки.
func renderPane(style lipgloss.Style, content string) string {
	out := style.Render(content)
	// Точная SGR-последовательность фона при текущем профиле (ANSI256/TrueColor).
	probe := lipgloss.NewStyle().Background(style.GetBackground()).Render("§")
	bg := strings.Split(probe, "§")[0]
	if bg == "" {
		return out
	}
	body, ok := strings.CutSuffix(out, "\x1b[0m")
	if !ok {
		return strings.ReplaceAll(out, "\x1b[0m", "\x1b[0m"+bg)
	}
	return strings.ReplaceAll(body, "\x1b[0m", "\x1b[0m"+bg) + "\x1b[0m"
}
