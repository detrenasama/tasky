package ui

import (
	"github.com/charmbracelet/bubbletea"
)

// Confirm — простая модалка подтверждения (да/нет). onYes вызывается при
// y/enter, onNo — при n/esc/q/ctrl+c; после вызова модалка закрывается.
// Заменяет разбросанные *Confirm-режимы экранов единым вызовом
// m.modal.Show(kit.NewConfirm(...)).
type Confirm struct {
	title    string
	body     string
	yesLabel string
	noLabel  string
	onYes    func() tea.Cmd
	onNo     func() tea.Cmd
}

// NewConfirm создаёт модалку подтверждения; onYes — действие при «да».
func NewConfirm(title, body string, onYes func() tea.Cmd) *Confirm {
	return &Confirm{
		title:    title,
		body:     body,
		yesLabel: "y — да",
		noLabel:  "n — нет",
		onYes:    onYes,
	}
}

func (c *Confirm) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok {
		switch k.String() {
		case "y", "enter":
			var cmd tea.Cmd
			if c.onYes != nil {
				cmd = c.onYes()
			}
			return nil, cmd
		case "n", "esc", "q", "ctrl+c":
			var cmd tea.Cmd
			if c.onNo != nil {
				cmd = c.onNo()
			}
			return nil, cmd
		}
	}
	return c, nil
}

// Init заглушка для удовлетворения tea.Model.
func (c *Confirm) Init() tea.Cmd { return nil }

func (c *Confirm) View() string {
	d := dialog{
		title:   c.title,
		body:    c.body,
		primary: c.yesLabel,
		esc:     c.noLabel,
	}
	return d.render()
}
