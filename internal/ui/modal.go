package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/detrenasama/tasky/internal/ui/theme"
)

type dialog struct {
	title   string
	body    string
	primary string
	esc     string
}

func (d dialog) render() string {
	title := theme.HeaderStyle.Render(d.title)
	buttons := theme.Faint("[") + " " + theme.AccentBtn(d.primary) + "  " + theme.Faint("]  [") + " " +
		theme.EscBtn(d.esc) + " " + theme.Faint("]")
	return theme.BoxStyle.Render(title + "\n\n" + d.body + "\n\n" + buttons)
}

// overlay кладёт dialog поверх base, центрируя его по середине холста
// w×h и затемняя фон. Фрагменты фона, накрытые диалогом, обрезаются по
// границе диалога; фоновые строки вне диалога затемняются целиком.
func overlay(base, dialog string, w, h int) string {
	baseLines := strings.Split(strings.ReplaceAll(base, "\r\n", "\n"), "\n")
	if len(baseLines) > h {
		baseLines = baseLines[:h]
	}
	for i := range baseLines {
		baseLines[i] = padW(baseLines[i], w)
	}
	for len(baseLines) < h {
		baseLines = append(baseLines, strings.Repeat(" ", w))
	}

	dialogLines := strings.Split(strings.ReplaceAll(dialog, "\r\n", "\n"), "\n")
	dw := 0
	for _, l := range dialogLines {
		if lipgloss.Width(l) > dw {
			dw = lipgloss.Width(l)
		}
	}
	dh := len(dialogLines)
	for i, l := range dialogLines {
		dialogLines[i] = padW(l, dw)
	}

	row0 := (h - dh) / 2
	if row0 < 0 {
		row0 = 0
	}
	col0 := (w - dw) / 2
	if col0 < 0 {
		col0 = 0
	}

	for r := 0; r < dh && row0+r < h; r++ {
		plain := stripANSI(baseLines[row0+r])
		runes := []rune(plain)
		left := string(runes[:min(col0, len(runes))])
		right := string(runes[min(col0+dw, len(runes)):])
		baseLines[row0+r] = theme.FaintStyle.Render(left) + dialogLines[r] + theme.FaintStyle.Render(right)
	}
	for i := 0; i < len(baseLines); i++ {
		if i < row0 || i >= row0+dh {
			baseLines[i] = theme.FaintStyle.Render(baseLines[i])
		}
	}
	return strings.Join(baseLines, "\n")
}

// stripANSI удаляет ESC-последовательности (CSI, OSC, двухсимвольные).
func stripANSI(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] != '\x1b' {
			b.WriteByte(s[i])
			continue
		}
		i = ansiEnd(s, i)
		if i >= len(s) {
			break
		}
	}
	return b.String()
}

// ansiEnd возвращает индекс последнего байта ESC-последовательности,
// начинающейся в s[start] (start указывает на байт ESC).
func ansiEnd(s string, start int) int {
	if start+1 >= len(s) {
		return start
	}
	switch s[start+1] {
	case ']': // OSC: до BEL или ST
		for j := start + 2; j < len(s); j++ {
			if s[j] == '\a' {
				return j
			}
			if s[j] == '\x1b' && j+1 < len(s) && s[j+1] == '\\' {
				return j + 1
			}
		}
		return len(s) - 1
	case '[': // CSI: параметры до финального символа (0x40–0x7E)
		for j := start + 2; j < len(s); j++ {
			c := s[j]
			if c >= '@' && c <= '~' {
				return j
			}
			if !(c >= '0' && c <= '9' || c == ';' || c == '?' || c == '>' || c == '<' || c == '=' || c == ' ') {
				return j - 1
			}
		}
		return len(s) - 1
	}
	return start + 1 // двухсимвольная последовательность ESC + символ
}
