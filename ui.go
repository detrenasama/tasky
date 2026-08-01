package main

import (
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

// padW обрезает строку до width (по видимой ширине, с учётом ANSI-кодов)
// и дополняет пробелами.
func padW(line string, width int) string {
	line = truncateW(line, width)
	return line + strings.Repeat(" ", width-lipgloss.Width(line))
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

// fixedBox рендерит content в рамке style ровно w×h (контент дополняется
// пробелами по ширине и высоте, лишние строки обрезаются). Хром рамки
// boxStyle = 4 (бордер 2 + паддинг 2), поэтому контент паддится до w-4.
func fixedBox(style lipgloss.Style, content string, w, h int) string {
	if h < 2 {
		h = 2
	}
	inner := padLines(content, max(w-4, 1), h-2)
	return style.Render(inner)
}

// padLines приводит многострочный текст к холсту w×h, дополняя пробелами
// и обрезая лишние строки.
func padLines(content string, w, h int) string {
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
