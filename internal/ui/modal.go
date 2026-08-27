package ui

import (
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"

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
	combined := d.primary
	if d.esc != "" {
		combined += " · " + d.esc
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
		left, right := splitStyledAtWidth(baseLines[row0+r], col0, dw)
		baseLines[row0+r] = theme.Dim(left) + dialogLines[r] + theme.Dim(right)
	}
	for i := 0; i < len(baseLines); i++ {
		if i < row0 || i >= row0+dh {
			baseLines[i] = theme.Dim(baseLines[i])
		}
	}
	return strings.Join(baseLines, "\n")
}

// splitStyledAtWidth режет строку s с ANSI-кодами на левую и правую части по
// видимой ширине: левая — [0,col0), правая — [col0+boxW, конец). Колонки под
// модалкой ([col0,col0+boxW)) пропускаются (их перекроет диалог). Слова,
// пересекающие границы модалки, полностью прячутся (заменяются пробелами с
// сохранением цвета фона), чтобы на краях модалки слова не разрывались
// пополам. Цвета фона (SGR) сохраняются по обе стороны и переносятся в правую
// часть, поэтому бэкдроп остаётся «живым», а не плоским блоком.
func splitStyledAtWidth(s string, col0, boxW int) (left, right string) {
	plain := []rune(stripANSI(s))
	rb := col0 + boxW
	blankL := -1
	if col0 > 0 && col0 <= len(plain) && plain[col0-1] != ' ' {
		st, _ := wordRange(plain, col0-1)
		blankL = st
	}
	blankR := -1
	if rb >= 0 && rb < len(plain) && plain[rb] != ' ' {
		_, en := wordRange(plain, rb)
		blankR = en
	}

	var lb, rb2 strings.Builder
	var active string
	rightStarted := false
	pos := 0
	ri := 0
	i := 0
	for i < len(s) {
		if s[i] == '\x1b' {
			end := ansiEnd(s, i)
			seq := s[i : end+1]
			if isSGReset(seq) {
				active = ""
			} else {
				active += seq
			}
			switch {
			case pos < col0:
				lb.WriteString(seq)
			case pos >= rb:
				if !rightStarted {
					rb2.WriteString(active)
					rightStarted = true
				}
				rb2.WriteString(seq)
			}
			i = end + 1
			continue
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		rw := runewidth.RuneWidth(r)
		switch {
		case pos < col0:
			if blankL >= 0 && ri >= blankL && ri < col0 {
				lb.WriteRune(' ')
			} else {
				lb.WriteRune(r)
			}
		case pos >= rb:
			if !rightStarted {
				rb2.WriteString(active)
				rightStarted = true
			}
			if blankR >= 0 && ri < blankR {
				rb2.WriteRune(' ')
			} else {
				rb2.WriteRune(r)
			}
		}
		// область под модалкой (col0<=pos<rb) — не пишем, перекроет диалог
		pos += rw
		ri++
		i += size
	}
	if !rightStarted {
		rb2.WriteString(active)
	}
	return lb.String(), rb2.String()
}

// wordRange возвращает [start,end) слова (непрерывная последовательность
// не-пробелов), содержащего руну at. Если at — пробел, возвращает at,at.
func wordRange(plain []rune, at int) (int, int) {
	s := at
	for s > 0 && plain[s-1] != ' ' {
		s--
	}
	e := at
	for e < len(plain) && plain[e] != ' ' {
		e++
	}
	return s, e
}

// isSGReset — является ли SGR-последовательность сбросом (\x1b[0m / \x1b[m).
func isSGReset(seq string) bool {
	if !strings.HasPrefix(seq, "\x1b[") || !strings.HasSuffix(seq, "m") {
		return false
	}
	body := seq[2 : len(seq)-1]
	return body == "0" || body == ""
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
