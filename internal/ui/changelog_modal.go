package ui

import "github.com/charmbracelet/bubbletea"

// changelogModal — прокручиваемое окно «Что нового». Тело рендерится
// plain-текстом через descViewer (режим plain: без курсора и выделения).
type changelogModal struct {
	title   string
	viewer  *descViewer
	onClose func()
}

func newChangelogModal(title, body string, w, h int, onClose func()) *changelogModal {
	v := newDescViewer(body, w, h)
	v.plain = true
	return &changelogModal{title: title, viewer: v, onClose: onClose}
}

func (c *changelogModal) Init() tea.Cmd { return nil }

func (c *changelogModal) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if sz, ok := msg.(tea.WindowSizeMsg); ok {
		c.viewer.width, c.viewer.height = changelogModalSize(sz.Width, sz.Height)
		return c, nil
	}
	k, ok := msg.(tea.KeyMsg)
	if !ok {
		return c, nil
	}
	switch k.String() {
	case "up":
		c.viewer.scrollUp(1)
	case "down":
		c.viewer.scrollDown(1)
	case "pgup":
		c.viewer.scrollUp(c.viewer.height)
	case "pgdown":
		c.viewer.scrollDown(c.viewer.height)
	case "enter", "esc", "q", "ctrl+c":
		if c.onClose != nil {
			c.onClose()
		}
		return nil, nil
	}
	return c, nil
}

func (c *changelogModal) View() string {
	d := dialog{
		title:   c.title,
		body:    c.viewer.view(),
		primary: "Enter — закрыть",
		esc:     "↑/↓ — прокрутка",
	}
	return d.render()
}
