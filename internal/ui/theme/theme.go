// Пакет theme — стили и палитры приложения. Активная тема (набор цветов)
// хранится в пакете и может переключаться в рантайме (раздел «Тема» в
// настройках): стили пересобираются из активной темы при каждом рендере.
package theme

import (
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
)

// DefaultName — тема по умолчанию (встроенная, стиль OpenCode).
const DefaultName = "opencode"

// Package-level стили, производные от активной темы. Читаются на рендере,
// поэтому переключение темы применяется живой перерисовкой.
var (
	Accent      = lipgloss.Color("212")
	HeaderStyle = lipgloss.NewStyle().Bold(true).Foreground(Accent)
	FaintStyle  = lipgloss.NewStyle().Faint(true)
	MutedStyle  = lipgloss.NewStyle().Faint(true)
	DimStyle    = lipgloss.NewStyle().Faint(true)
	ErrorStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	SaveOKStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	linkStyle   = lipgloss.NewStyle().Underline(true).Foreground(Accent)
)

// Pane возвращает стиль плоской панели без рамок: неактивная — фон-заливка
// панели; активная (в фокусе) — светлее + левая акцентная полоса.
func Pane(focused bool) lipgloss.Style {
	s := lipgloss.NewStyle().Padding(0, 1)
	if focused {
		return s.
			Background(lipgloss.Color(active.Colors.Element)).
			Border(lipgloss.NormalBorder(), false, false, false, true).
			BorderForeground(lipgloss.Color(active.Colors.Accent))
	}
	return s.Background(lipgloss.Color(active.Colors.Panel))
}

// ModalStyle — стиль модального диалога: плоская панель с лёгкой рамкой.
var ModalStyle = lipgloss.NewStyle().
	Background(lipgloss.Color("#1e1e1e")).
	Border(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("#3c3c3c")).
	Padding(1, 2)

func Faint(s string) string { return MutedStyle.Render(s) }

// Muted — вторичный текст цветом muted темы (подсказки, штампы, метки).
func Muted(s string) string { return MutedStyle.Render(s) }

// Dim — затемнение фона под модалкой (атрибут faint, цвета сохраняются).
func Dim(s string) string { return DimStyle.Render(s) }

func AccentBtn(label string) string {
	return lipgloss.NewStyle().Bold(true).Foreground(Accent).Render(label)
}

func EscBtn(label string) string {
	return MutedStyle.Render(label)
}

// ApplyToDelegate стилизует делегат списка bubbles по активной теме:
// выделенный элемент — акцентная левая полоса + фон выделения + акцентный
// текст; обычный — текст темы, вторичный — muted.
func ApplyToDelegate(d *list.DefaultDelegate) {
	if d == nil {
		return
	}
	acc := lipgloss.Color(active.Colors.Accent)
	muted := lipgloss.Color(active.Colors.Muted)
	text := lipgloss.Color(active.Colors.Text)
	sel := lipgloss.Color(active.Colors.Selection)
	d.Styles.NormalTitle = lipgloss.NewStyle().Foreground(text).Padding(0, 0, 0, 2)
	d.Styles.NormalDesc = lipgloss.NewStyle().Foreground(muted).Padding(0, 0, 0, 2)
	d.Styles.SelectedTitle = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(acc).
		Background(sel).
		Foreground(acc).
		Padding(0, 0, 0, 1)
	d.Styles.SelectedDesc = lipgloss.NewStyle().
		Foreground(acc).
		Padding(0, 0, 0, 1)
	d.Styles.DimmedTitle = lipgloss.NewStyle().Foreground(muted).Padding(0, 0, 0, 2)
	d.Styles.DimmedDesc = lipgloss.NewStyle().Foreground(muted).Padding(0, 0, 0, 2)
	d.Styles.FilterMatch = lipgloss.NewStyle().Underline(true)
}

// StatusPalette — палитра цветов статусов (приглушённые, видны на тёмном
// фоне). Хранится индексом в pickList. Цвета статусов — данные пользователя
// из БД, не тема.
var StatusPalette = []string{
	"#6a9955", "#4f7942", "#569cd6", "#c586c0", "#d7ba7d",
	"#ce9178", "#8a8a8a", "#8f3e3e", "#4ec9b0", "#d47a9e",
}

var PaletteNames = []string{
	"зелёный", "тёмно-зелёный", "синий", "фиолетовый", "жёлтый",
	"оранжевый", "серый", "тёмно-красный", "голубой", "розовый",
}
