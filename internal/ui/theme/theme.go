// Пакет theme — стили и палитры приложения. Активная тема (набор цветов)
// хранится в пакете и может переключаться в рантайме (раздел «Тема» в
// настройках): стили пересобираются из активной темы при каждом рендере.
package theme

import (
	"strconv"
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
	Background(lipgloss.Color(active.Colors.Modal)).
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

// dimFactor — во сколько раз затемняются цвета фона бэкдропа (0.55 ≈ на 45%
// темнее). Текст дополнительно гасится faint в Dim; вместе это даёт заметно
// более тёмный, но сохраняющий контент фон вокруг модалки.
const dimFactor = 0.4

// Dim — затемнение фона под модалкой. Чтобы затемнение покрывало всю
// строку (включая цветные сегменты с собственными сбросами стиля), faint
// включается повторно после каждого \x1b[0m — иначе он сбрасывается после
// первого сегмента и фон затемняется только в полосе высоты модалки.
// Поверх этого цвета фона строки дополнительно потемняются к чёрному
// (dimBackgrounds), чтобы бэкдроп был ещё тусклее, но контент оставался виден.
func Dim(s string) string {
	const faint = "\x1b[2m"
	const reset = "\x1b[0m"
	out := faint + s
	out = strings.ReplaceAll(out, reset, reset+faint)
	out = dimBackgrounds(out, dimFactor)
	return out + reset
}

// dimBackgrounds проходит по строке и в каждой SGR-последовательности с фоном
// (48;2;r;g;b или 48;5;n) умножает цвет на factor, делая фон темнее. Прочие
// последовательности (текст, faint, reset) оставляются как есть.
func dimBackgrounds(s string, factor float64) string {
	var b strings.Builder
	i := 0
	for i < len(s) {
		if s[i] != '\x1b' {
			b.WriteByte(s[i])
			i++
			continue
		}
		end := ansiEndTheme(s, i)
		seq := s[i : end+1]
		if strings.HasSuffix(seq, "m") && strings.Contains(seq, "48") {
			seq = dimBgInSeq(seq, factor)
		}
		b.WriteString(seq)
		i = end + 1
	}
	return b.String()
}

// dimBgInSeq затемняет фоновый цвет внутри одной SGR-последовательности.
func dimBgInSeq(seq string, factor float64) string {
	if len(seq) < 4 || seq[0] != '\x1b' || seq[1] != '[' || seq[len(seq)-1] != 'm' {
		return seq
	}
	body := seq[2 : len(seq)-1]
	parts := strings.Split(body, ";")
	for k := 0; k+1 < len(parts); k++ {
		if parts[k] != "48" {
			continue
		}
		if k+4 < len(parts) && parts[k+1] == "2" {
			for off := 2; off <= 4; off++ {
				n, err := strconv.Atoi(parts[k+off])
				if err != nil {
					return seq
				}
				parts[k+off] = strconv.Itoa(int(float64(n) * factor))
			}
			return "\x1b[" + strings.Join(parts, ";") + "m"
		}
		if k+2 < len(parts) && parts[k+1] == "5" {
			idx, err := strconv.Atoi(parts[k+2])
			if err != nil {
				return seq
			}
			r, g, b := rgbFrom256(idx)
			r, g, b = int(float64(r)*factor), int(float64(g)*factor), int(float64(b)*factor)
			// Перевыпускаем как truecolor, чтобы затемнение применилось
			// независимо от цветового профиля терминала.
			return "\x1b[48;2;" + strconv.Itoa(r) + ";" + strconv.Itoa(g) + ";" + strconv.Itoa(b) + "m"
		}
		// 16-цветные формы 4n/10n оставляем без изменений.
		break
	}
	return "\x1b[" + strings.Join(parts, ";") + "m"
}

// rgbFrom256 — перевод индекса 256-цветовой палитры xterm в RGB.
func rgbFrom256(n int) (r, g, b int) {
	switch {
	case n < 16:
		// Системные 16 цветов: грубая аппроксимация (точная палитра
		// зависит от терминала). Для фонов при темизации не используются.
		return 128, 128, 128
	case n <= 231:
		code := n - 16
		levels := []int{0, 95, 135, 175, 215, 255}
		return levels[code/36], levels[(code/6)%6], levels[code%6]
	default: // 232..255 — градации серого
		v := 8 + (n-232)*10
		return v, v, v
	}
}

// ansiEndTheme возвращает индекс последнего байта ESC-последовательности,
// начинающейся в s[start] (start указывает на байт ESC). Локальная копия, т.к.
// пакет theme не может импортировать ui (цикл зависимостей).
func ansiEndTheme(s string, start int) int {
	if start+1 >= len(s) {
		return start
	}
	switch s[start+1] {
	case ']':
		for j := start + 2; j < len(s); j++ {
			if s[j] == '\a' {
				return j
			}
			if s[j] == '\x1b' && j+1 < len(s) && s[j+1] == '\\' {
				return j + 1
			}
		}
		return len(s) - 1
	case '[':
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
	return start + 1
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
