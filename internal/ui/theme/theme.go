// Пакет theme — стили и палитры приложения. Активная тема (набор цветов)
// хранится в пакете и может переключаться в рантайме (раздел «Тема» в
// настройках): стили пересобираются из активной темы при каждом рендере.
package theme

import (
	"strings"

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
	// TextStyle — основной (белый) цвет текста: явный foreground, чтобы
	// сочетания клавиш в подсказках были белыми независимо от умолчаний терминала.
	TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(active.Colors.Text))
	// SelectionStyle — выделенная строка списка: фон Selection + акцентный
	// текст (используется в палитре команд).
	SelectionStyle = lipgloss.NewStyle().
			Background(lipgloss.Color(active.Colors.Selection)).
			Foreground(lipgloss.Color(active.Colors.Accent))
	linkStyle = lipgloss.NewStyle().Underline(true).Foreground(Accent)
)

// ListSelectionStyle — фон выделенного элемента списка задач/проектов:
// серый (Selection), только фон, чтобы встроенные цвета (статус, теги)
// оставались видимыми. Читается на рендере, поэтому учитывает смену темы.
func ListSelectionStyle() lipgloss.Style {
	return lipgloss.NewStyle().Background(lipgloss.Color(active.Colors.Selection))
}

// ContentBgStyle — фон контента (content) без отступов: им закрашиваются
// невыделенные строки списка, чтобы колонка имела сплошной фон.
func ContentBgStyle() lipgloss.Style {
	return lipgloss.NewStyle().Background(lipgloss.Color(active.Colors.Content))
}

// Pane возвращает стиль плоской панели без рамок для контента: фон content.
// Фокуса на панелях контента больше нет, поэтому стиль одинаков для обеих
// колонок (список и описание) независимо от аргумента focused.
func Pane(focused bool) lipgloss.Style {
	_ = focused
	return lipgloss.NewStyle().Padding(0, 1).Background(lipgloss.Color(active.Colors.Content))
}

// PanelColor — цвет фона правой панели (panel).
func PanelColor() lipgloss.Color { return lipgloss.Color(active.Colors.Panel) }

// ModalStyle — стиль модального диалога: плоская панель без рамок.
var ModalStyle = lipgloss.NewStyle().
	Background(lipgloss.Color("#1e1e1e")).
	Padding(1, 2)

// SidebarInactive — неактивная вкладка левой вертикальной панели и фон
// самой панели: фон tabs, приглушённый текст.
func SidebarInactive() lipgloss.Style {
	return lipgloss.NewStyle().
		Background(lipgloss.Color(active.Colors.Tabs)).
		Foreground(lipgloss.Color(active.Colors.Muted))
}

// SidebarActive — активная вкладка левой вертикальной панели: фон Selection,
// акцентный текст (консистентно с выделением в палитре команд).
func SidebarActive() lipgloss.Style {
	return lipgloss.NewStyle().
		Background(lipgloss.Color(active.Colors.Selection)).
		Foreground(lipgloss.Color(active.Colors.Accent))
}

func Faint(s string) string { return MutedStyle.Render(s) }

// Text — основной (белый) цвет текста: явный foreground, чтобы сочетания
// клавиш в подсказках были белыми независимо от умолчаний терминала.
func Text(s string) string { return TextStyle.Render(s) }

// Muted — вторичный текст цветом muted темы (подсказки, штампы, метки).
func Muted(s string) string { return MutedStyle.Render(s) }

// Dim — затемнение фона под модалкой. Чтобы затемнение покрывало всю
// строку (включая цветные сегменты с собственными сбросами стиля), faint
// включается повторно после каждого \x1b[0m — иначе он сбрасывается после
// первого сегмента и фон затемняется только в полосе высоты модалки.
func Dim(s string) string {
	const faint = "\x1b[2m"
	const reset = "\x1b[0m"
	out := faint + s
	out = strings.ReplaceAll(out, reset, reset+faint)
	return out + reset
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
	// Без левой рамки: иначе у выделенного элемента появляется видимый
	// символ «│» в начале строки (в отличие от невыделенного, где слева 2
	// невидимых пробела), и статус/текст визуально сдвигаются вправо.
	// Отступ слева (2) совпадает с NormalTitle, поэтому выравнивание
	// идентично; фон выделения задаётся снаружи (selDelegate / список ссылок).
	d.Styles.SelectedTitle = lipgloss.NewStyle().
		Background(sel).
		Foreground(acc).
		Padding(0, 0, 0, 2)
	d.Styles.SelectedDesc = lipgloss.NewStyle().
		Foreground(acc).
		Padding(0, 0, 0, 2)
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
