package ui

import (
	"strings"
	"unicode/utf8"

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
	combined := strings.ReplaceAll(d.primary, "— ", "")
	if d.esc != "" {
		combined += " · " + strings.ReplaceAll(d.esc, "— ", "")
	}
	var content string
	if combined != "" {
		content = title + "\n\n" + d.body + "\n\n" + styleHints(combined)
	} else {
		content = title + "\n\n" + d.body
	}
	return renderPane(theme.ModalStyle, content)
}

// overlay кладёт dialog поверх base, центрируя его по середине холста
// w×h и затемняя фон. Фрагменты фона, накрытые диалогом, обрезаются по
// границе диалога; фоновые строки вне диалога затемняются целиком.
// overlay кладёт dialog поверх base, центрируя его по середине холста
// w×h и затемняя фон. maxW ограничивает ширину модалки (чтобы она не
// растягивалась во всю ширину экрана и не превращалась в серую полосу);
// строки длиннее maxW переносятся. maxW<=0 — без ограничения (палитра).
func overlay(base, dialog string, w, h, maxW int) string {
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
	if maxW > 0 {
		var wrapped []string
		for _, l := range dialogLines {
			if lipgloss.Width(l) > maxW && !lineHasBox(l) {
				wrapped = append(wrapped, wrapStyled(l, maxW)...)
			} else {
				wrapped = append(wrapped, l)
			}
		}
		dialogLines = wrapped
	}
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
		baseLines[row0+r] = theme.Dim(left) + dialogLines[r] + theme.Dim(right)
	}
	for i := 0; i < len(baseLines); i++ {
		if i < row0 || i >= row0+dh {
			baseLines[i] = theme.Dim(baseLines[i])
		}
	}
	return strings.Join(baseLines, "\n")
}

// dialogMaxW — максимальная ширина диалоговой модалки: всегда уже экрана,
// чтобы оставались боковые поля и модалка не выглядела серой полосой.
func dialogMaxW(w int) int {
	c := w * 2 / 3
	if c < 40 {
		return 40
	}
	return c
}

// lineHasBox — содержит ли строка box-drawing символы (рамка/разделители).
// Такие строки переносить нельзя: перенос разорвёт рамку модалки.
func lineHasBox(s string) bool {
	for _, r := range s {
		switch r {
		case '┌', '┐', '└', '┘', '│', '─',
			'├', '┤', '┬', '┴', '┼',
			'╭', '╮', '╰', '╯', '║', '═',
			'╔', '╗', '╚', '╝':
			return true
		}
	}
	return false
}

// wrapStyled переносит строку с ANSI-кодами по видимой ширине width,
// разбивая СТРОГО по пробелам (слово никогда не разрывается посередине);
// состояние стиля сохраняется внутри слов.
func wrapStyled(s string, width int) []string {
	if width <= 0 {
		return []string{s}
	}
	type tok struct {
		text    string
		isSpace bool
	}
	var toks []tok
	var style strings.Builder
	var cur strings.Builder
	curSpace := false
	flush := func() {
		if cur.Len() == 0 {
			style.Reset()
			return
		}
		toks = append(toks, tok{text: style.String() + cur.String(), isSpace: curSpace})
		cur.Reset()
		style.Reset()
	}
	i := 0
	for i < len(s) {
		if s[i] == '\x1b' {
			end := ansiEnd(s, i)
			style.WriteString(s[i : end+1])
			i = end + 1
			continue
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		sp := r == ' '
		if sp != curSpace && cur.Len() > 0 {
			flush()
		}
		curSpace = sp
		cur.WriteString(string(r))
		i += size
	}
	flush()

	var lines []string
	var line strings.Builder
	lineW := 0
	for _, tk := range toks {
		w := lipgloss.Width(tk.text)
		if tk.isSpace {
			if lineW > 0 {
				line.WriteString(tk.text)
				lineW += w
			}
			continue
		}
		if lineW > 0 && lineW+w > width {
			lines = append(lines, strings.TrimRight(line.String(), " "))
			line.Reset()
			lineW = 0
		}
		line.WriteString(tk.text)
		lineW += w
	}
	t := strings.TrimRight(line.String(), " ")
	if t != "" || len(lines) == 0 {
		lines = append(lines, t)
	}
	if len(lines) == 0 {
		lines = []string{""}
	}
	return lines
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
