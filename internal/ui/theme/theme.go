// Пакет theme — стили и палитры приложения. Единая точка, где объявлены
// цвета и стили lipgloss; при появлении настроек темы здесь будет жить
// переключение палитр (загрузка/сохранение тем).
package theme

import "github.com/charmbracelet/lipgloss"

// Хрома по умолчанию (фиксированная).
var (
	Accent      = lipgloss.Color("212")
	HeaderStyle = lipgloss.NewStyle().Bold(true).Foreground(Accent)
	FaintStyle  = lipgloss.NewStyle().Faint(true)
	ErrorStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	BoxStyle    = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1)
	FocusBox    = BoxStyle.Copy().BorderForeground(Accent)
	DimBox      = BoxStyle.Copy().Faint(true)
	SaveOKStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
)

func Faint(s string) string { return FaintStyle.Render(s) }

func AccentBtn(label string) string {
	return lipgloss.NewStyle().Bold(true).Foreground(Accent).Render(label)
}

func EscBtn(label string) string {
	return FaintStyle.Render(label)
}

// StatusPalette — палитра цветов статусов (приглушённые, видны на тёмном
// фоне). Хранится индексом в pickList.
var StatusPalette = []string{
	"#6a9955", "#4f7942", "#569cd6", "#c586c0", "#d7ba7d",
	"#ce9178", "#8a8a8a", "#8f3e3e", "#4ec9b0", "#d47a9e",
}

var PaletteNames = []string{
	"зелёный", "тёмно-зелёный", "синий", "фиолетовый", "жёлтый",
	"оранжевый", "серый", "тёмно-красный", "голубой", "розовый",
}
