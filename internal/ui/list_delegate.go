package ui

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"

	"github.com/detrenasama/tasky/internal/ui/theme"
)

// selDelegate — делегат списка, закрашивающий каждую строку фоном целиком:
// выделенный элемент — серым (ListSelectionStyle), остальные — фоном контента.
// Это исправляет штатное поведение bubbles/list, где фон выделения не
// переживает встроенные сбросы \x1b[0m внутри Title() (например, цветную
// полосу статуса) и не заполняет всю ширину строки.
type selDelegate struct {
	list.DefaultDelegate
}

// newListDelegate создаёт делегат со стилями активной темы.
func newListDelegate() selDelegate {
	d := list.NewDefaultDelegate()
	theme.ApplyToDelegate(&d)
	return selDelegate{DefaultDelegate: d}
}

// Render рисует элемент. Для выделенного (и не фильтруемого) элемента каждая
// строка оборачивается в renderPane с серым фоном, который восстанавливается
// после внутренних \x1b[0m; невыделенные строки получают фон контента.
func (d selDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	var buf bytes.Buffer
	d.DefaultDelegate.Render(&buf, m, index, item)
	selected := index == m.Index() && m.FilterState() != list.Filtering
	bg := theme.ContentBgStyle()
	if selected {
		bg = theme.ListSelectionStyle()
	}
	raw := strings.TrimRight(buf.String(), "\n")
	lines := strings.Split(raw, "\n")
	out := make([]string, len(lines))
	for i, line := range lines {
		out[i] = renderPane(bg, padW(line, m.Width()))
	}
	fmt.Fprint(w, strings.Join(out, "\n"))
}
