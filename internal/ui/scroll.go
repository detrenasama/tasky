package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/list"
)

// listScrollOff — число элементов, которые должны оставаться видимыми выше и
// ниже курсора при прокрутке (аналог vim 'scrolloff'). Используется во всех
// списках приложения.
const listScrollOff = 4

// listStep возвращает число строк, занимаемых одним элементом списка
// (заголовок + описание + отступ между элементами).
func listStep(d list.ItemDelegate) int {
	step := d.Height() + d.Spacing()
	if step < 1 {
		return 1
	}
	return step
}

// sizeListHeight завышает высоту bubbles/list так, чтобы он рендерил ВСЕ
// элементы одной «страницей» (внутренний Paginator не срабатывает и не
// прыгает на целую страницу). minH — минимальная высота (видимое окно).
func sizeListHeight(l *list.Model, d list.ItemDelegate, count, minH int) {
	need := count * listStep(d)
	h := minH
	if need > h {
		h = need
	}
	l.SetHeight(h)
}

// syncListTop удерживает курсор в пределах [top+off, top+visible-1-off],
// где off = listScrollOff*step. Сдвигает top (верхнюю видимую строку) при
// необходимости, не давая курсору прилипать к краям окна.
func syncListTop(l *list.Model, top *int, d list.ItemDelegate, count, visible int) {
	if count == 0 {
		*top = 0
		return
	}
	if visible < 1 {
		visible = 1
	}
	step := listStep(d)
	spacing := d.Spacing()
	totalRows := count*step - spacing
	if totalRows < 0 {
		totalRows = 0
	}
	idxRow := l.Index() * step
	off := listScrollOff * step
	t := *top
	if idxRow < t+off {
		t = idxRow - off
	}
	if idxRow > t+visible-1-off {
		t = idxRow - visible + 1 + off
	}
	if t < 0 {
		t = 0
	}
	maxTop := totalRows - visible
	if maxTop < 0 {
		maxTop = 0
	}
	if t > maxTop {
		t = maxTop
	}
	*top = t
}

// clipList возвращает только строки видимого окна списка (начиная со строки
// top, длиной visible), дополняя недостающие пустыми строками. Список должен
// быть предварительно «развёрнут» через sizeListHeight, чтобы View() отдавал
// все элементы.
func clipList(l list.Model, top, visible int) string {
	if visible < 1 {
		visible = 1
	}
	if top < 0 {
		top = 0
	}
	rows := strings.Split(l.View(), "\n")
	end := top + visible
	if end > len(rows) {
		end = len(rows)
	}
	slice := rows[top:end]
	for len(slice) < visible {
		slice = append(slice, "")
	}
	return strings.Join(slice, "\n")
}
