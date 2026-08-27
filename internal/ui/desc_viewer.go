package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"

	"github.com/detrenasama/tasky/internal/ui/theme"
)

// descViewer — просмотр описания с vim-подобным визуальным выделением.
// cursor и anchor — индексы рун в исходном тексте; выделение охватывает
// диапазон [anchor, cursor]. При anchor < 0 выделения нет (копируется весь
// текст).
type descViewer struct {
	text    string
	cursor  int
	anchor  int
	colHint int
	width   int
	height  int
	scroll  int  // верхняя видимая строка в режиме plain (просмотр)
	plain   bool // true — простой просмотр (без курсора/выделения)
}

type dvLine struct{ start, end int }

func newDescViewer(text string, w, h int) *descViewer {
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	return &descViewer{text: text, cursor: 0, anchor: -1, width: w, height: h}
}

func (v *descViewer) runes() []rune { return []rune(v.text) }

// layout разбивает текст на визуальные строки (перенос по ширине width),
// возвращая диапазоны рун [start, end).
func (v *descViewer) layout() []dvLine {
	rs := v.runes()
	if v.width <= 0 {
		v.width = 1
	}
	var lines []dvLine
	start := 0
	x := 0
	flush := func(end int) {
		lines = append(lines, dvLine{start: start, end: end})
	}
	for i, r := range rs {
		if r == '\n' {
			flush(i)
			start = i + 1
			x = 0
			continue
		}
		rw := runewidth.RuneWidth(r)
		if x+rw > v.width && x > 0 {
			flush(i)
			start = i
			x = rw
			continue
		}
		x += rw
	}
	flush(len(rs))
	return lines
}

// colInLine возвращает столбец курсора (по числу рун от начала строки).
func (v *descViewer) colInLine() int {
	lines := v.layout()
	ci := v.lineOfCursor(lines)
	if ci < 0 || ci >= len(lines) {
		return 0
	}
	ln := lines[ci]
	c := 0
	for i := ln.start; i < v.cursor && i < ln.end; i++ {
		c++
	}
	return c
}

func (v *descViewer) lineOfCursor(lines []dvLine) int {
	for i, ln := range lines {
		if v.cursor >= ln.start && v.cursor <= ln.end {
			return i
		}
	}
	return 0
}

func clampCol(ln dvLine, col int) int {
	c := ln.start + col
	if c < ln.start {
		c = ln.start
	}
	if c > ln.end {
		c = ln.end
	}
	return c
}

func (v *descViewer) moveLeft() {
	if v.cursor > 0 {
		v.cursor--
	}
	v.colHint = v.colInLine()
}

func (v *descViewer) moveRight() {
	rs := v.runes()
	if v.cursor < len(rs) {
		v.cursor++
	}
	v.colHint = v.colInLine()
}

func (v *descViewer) moveUp() {
	lines := v.layout()
	ci := v.lineOfCursor(lines)
	if ci > 0 {
		ci--
		v.cursor = clampCol(lines[ci], v.colHint)
	}
}

func (v *descViewer) moveDown() {
	lines := v.layout()
	ci := v.lineOfCursor(lines)
	if ci < len(lines)-1 {
		ci++
		v.cursor = clampCol(lines[ci], v.colHint)
	}
}

func (v *descViewer) toggleSelect() {
	if v.anchor < 0 {
		v.anchor = v.cursor
	} else {
		v.anchor = -1
	}
}

// scrollUp/scrollDown сдвигают окно просмотра по вертикали (режим plain).
func (v *descViewer) scrollUp(n int) {
	if n < 1 {
		n = 1
	}
	v.scroll -= n
	if v.scroll < 0 {
		v.scroll = 0
	}
}

func (v *descViewer) scrollDown(n int) {
	if n < 1 {
		n = 1
	}
	lines := v.layout()
	maxTop := len(lines) - v.height
	if maxTop < 0 {
		maxTop = 0
	}
	v.scroll += n
	if v.scroll > maxTop {
		v.scroll = maxTop
	}
}

func (v *descViewer) selectedText() string {
	if v.anchor < 0 {
		return v.text
	}
	rs := v.runes()
	a, b := v.anchor, v.cursor
	if a > b {
		a, b = b, a
	}
	if a < 0 {
		a = 0
	}
	if b > len(rs) {
		b = len(rs)
	}
	return string(rs[a:b])
}

var dvCursorStyle = lipgloss.NewStyle().Reverse(true)

// view рендерит текст с подсветкой выделения и курсора, с окном по высоте
// вокруг строки курсора (режим выделения) либо начиная с v.scroll (режим
// plain-просмотра, без курсора).
func (v *descViewer) view() string {
	rs := v.runes()
	lines := v.layout()
	a, b := v.cursor, v.cursor
	hasSel := v.anchor >= 0
	if hasSel {
		a, b = v.anchor, v.cursor
		if a > b {
			a, b = b, a
		}
	}
	curLine := v.lineOfCursor(lines)

	var out []string
	for li, ln := range lines {
		var sb strings.Builder
		for i := ln.start; i < ln.end; i++ {
			s := string(rs[i])
			switch {
			case hasSel && i >= a && i < b:
				sb.WriteString(theme.SelectionStyle.Render(s))
			case !v.plain && i == v.cursor && !hasSel:
				sb.WriteString(dvCursorStyle.Render(s))
			default:
				sb.WriteString(s)
			}
		}
		if !v.plain && v.cursor == ln.end && li == curLine {
			sb.WriteString(dvCursorStyle.Render(" "))
		}
		out = append(out, sb.String())
	}

	// окно: в режиме plain — начиная с v.scroll; в режиме выделения — вокруг
	// строки курсора.
	top := 0
	if v.plain {
		top = v.scroll
	} else if len(out) > v.height {
		top = curLine - v.height/2
	}
	if top < 0 {
		top = 0
	}
	if top > len(out)-v.height {
		top = len(out) - v.height
	}
	if top < 0 {
		top = 0
	}
	if top+v.height > len(out) {
		// pad
		for len(out) < top+v.height {
			out = append(out, "")
		}
	}
	return strings.Join(out[top:top+v.height], "\n")
}
