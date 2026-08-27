package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/detrenasama/tasky/internal/db"
	"github.com/detrenasama/tasky/internal/ui/theme"
)

// linkItem — элемент списка ссылок: название (или URL) и URL в описании.
type linkItem struct{ l db.Link }

func (i linkItem) FilterValue() string {
	if i.l.Name != "" {
		return i.l.Name + " " + i.l.URL
	}
	return i.l.URL
}

func (i linkItem) Title() string {
	if i.l.Name != "" {
		return i.l.Name
	}
	return i.l.URL
}

func (i linkItem) Description() string {
	if i.l.Name == "" {
		return ""
	}
	return i.l.URL
}

var linkStyle = lipgloss.NewStyle().Underline(true).Foreground(theme.Accent)

// pickItem — элемент списка выбора в модалках настроек (value — id проекта
// или код периода).
type pickItem struct {
	value int64
	label string
}

// pickList — простой скроллируемый список: курсор ▸, при выходе за видимую
// область содержимое плавно прокручивается (без постраничных прыжков).
type pickList struct {
	items   []pickItem
	sel     int
	scroll  int
	visible int
}

func (p *pickList) setVisible(n int) {
	p.visible = max(n, 1)
	if len(p.items) > 0 && p.sel >= len(p.items) {
		p.sel = len(p.items) - 1
	}
	p.clampScroll()
}

func (p *pickList) clampScroll() {
	// отступ сверху/снизу (как vim scrolloff) — держим listScrollOff
	// элементов видимыми вокруг курсора, если окно достаточно велико.
	off := listScrollOff
	if p.visible <= 2*off {
		off = 0
	}
	if p.sel < p.scroll+off {
		p.scroll = p.sel - off
	}
	if p.sel > p.scroll+p.visible-1-off {
		p.scroll = p.sel - p.visible + 1 + off
	}
	if p.scroll < 0 {
		p.scroll = 0
	}
	maxScroll := len(p.items) - p.visible
	if maxScroll < 0 {
		maxScroll = 0
	}
	if p.scroll > maxScroll {
		p.scroll = maxScroll
	}
}

// move сдвигает курсор по кругу и прокручивает список при необходимости.
func (p *pickList) move(d int) {
	n := len(p.items)
	if n == 0 {
		return
	}
	p.sel = (p.sel + d + n) % n
	p.clampScroll()
}

func (p *pickList) view() string {
	var lines []string
	end := min(len(p.items), p.scroll+p.visible)
	for i := p.scroll; i < end; i++ {
		label := "  " + p.items[i].label
		if i == p.sel {
			label = theme.HeaderStyle.Render("▸ ") + p.items[i].label
		}
		lines = append(lines, label)
	}
	return strings.Join(lines, "\n")
}

func (p *pickList) selected() (pickItem, bool) {
	if p.sel < 0 || p.sel >= len(p.items) {
		return pickItem{}, false
	}
	return p.items[p.sel], true
}

// colorPreview — цветной квадрат-превью для палитры.
func colorPreview(color string) string {
	return lipgloss.NewStyle().Background(lipgloss.Color(color)).Render("  ")
}
