// Темы Tasky: набор цветов, загружаемый из JSON. Встроенные темы лежат в
// themes/*.json (//go:embed), пользовательские — в <configDir>/themes/*.json.
// Активная тема хранится в БД (ключ settings.theme) и переключается на
// странице настроек.
package theme

import (
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

//go:embed themes/*.json
var themesFS embed.FS

// Colors — палитра темы. Все цвета опциональны: не заданные в файле падают
// на цвета темы по умолчанию (opencode).
type Colors struct {
	Accent       string `json:"accent"`
	Error        string `json:"error"`
	Success      string `json:"success"`
	Warning      string `json:"warning"`
	Text         string `json:"text"`
	Muted        string `json:"muted"`
	Background   string `json:"background"`
	Panel        string `json:"panel"`
	Tabs         string `json:"tabs"`
	Content      string `json:"content"`
	Element      string `json:"element"`
	Selection    string `json:"selection"`
	Border       string `json:"border"`
	BorderSubtle string `json:"borderSubtle"`
	Modal        string `json:"modal"`
}

// Theme — готовая тема: имя и палитра.
type Theme struct {
	Name   string
	Colors Colors
}

type themeFile struct {
	Name   string `json:"name"`
	Colors Colors `json:"colors"`
}

// active — текущая тема; applyTheme пересобирает package-стили из неё.
var active = defaultTheme()

// userThemes — пользовательские темы из <configDir>/themes/*.json.
var userThemes = map[string]Theme{}

// themesDir — каталог пользовательских тем (запоминается в Init).
var themesDir string

// ThemesDir возвращает каталог пользовательских тем (configDir/themes).
func ThemesDir() string { return themesDir }

func init() {
	applyTheme(active)
}

// defaultColors — палитра темы по умолчанию (OpenCode, тёмный вариант).
func defaultColors() Colors {
	return Colors{
		Accent:       "#5c9cf5",
		Error:        "#e06c75",
		Success:      "#7fd88f",
		Warning:      "#f5a742",
		Text:         "#eeeeee",
		Muted:        "#808080",
		Background:   "#0a0a0a",
		Panel:        "#1e1e1e",
		Tabs:         "#141414",
		Content:      "#141414",
		Element:      "#1e1e1e",
		Selection:    "#282828",
		Border:       "#484848",
		BorderSubtle: "#3c3c3c",
		Modal:        "#161616",
	}
}

func defaultTheme() Theme { return Theme{Name: DefaultName, Colors: defaultColors()} }

// Init загружает активную тему при старте: имя из БД (передаётся параметром),
// переопределение — env TASKY_THEME. Сканирует configDir на пользовательские
// темы. Неизвестная тема — fallback на дефолтную.
func Init(configDir, name string) {
	themesDir = filepath.Join(configDir, "themes")
	userThemes = loadUserThemes(configDir)
	if v := os.Getenv("TASKY_THEME"); v != "" {
		name = v
	}
	if name == "" {
		name = DefaultName
	}
	if t, ok := findTheme(name); ok {
		applyTheme(t)
	} else {
		applyTheme(defaultTheme())
	}
}

// Apply применяет тему по имени (для переключения в настройках). Ошибка,
// если тема не найдена.
func Apply(name string) error {
	t, ok := findTheme(name)
	if !ok {
		return fmt.Errorf("тема «%s» не найдена", name)
	}
	applyTheme(t)
	return nil
}

// ActiveName — имя активной темы.
func ActiveName() string { return active.Name }

// Themes — имена всех доступных тем: встроенные + пользовательские (сортировка).
func Themes() []string {
	names := map[string]bool{}
	entries, err := themesFS.ReadDir("themes")
	if err == nil {
		for _, e := range entries {
			if strings.HasSuffix(e.Name(), ".json") {
				names[strings.TrimSuffix(e.Name(), ".json")] = true
			}
		}
	}
	for n := range userThemes {
		names[n] = true
	}
	out := make([]string, 0, len(names))
	for n := range names {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

func findTheme(name string) (Theme, bool) {
	if t, ok := userThemes[name]; ok {
		return t, true
	}
	return loadBuiltin(name)
}

func loadBuiltin(name string) (Theme, bool) {
	data, err := themesFS.ReadFile("themes/" + name + ".json")
	if err != nil {
		return Theme{}, false
	}
	return parseTheme(data, name)
}

func loadUserThemes(configDir string) map[string]Theme {
	out := map[string]Theme{}
	dir := filepath.Join(configDir, "themes")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return out
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		t, ok := parseTheme(data, strings.TrimSuffix(e.Name(), ".json"))
		if !ok {
			continue
		}
		if _, has := out[t.Name]; has {
			continue
		}
		out[t.Name] = t
	}
	return out
}

// parseTheme разбирает JSON темы: имя из файла (поле name) или имени файла;
// недостающие цвета заполняются цветами по умолчанию.
func parseTheme(data []byte, fileBase string) (Theme, bool) {
	var f themeFile
	if err := json.Unmarshal(data, &f); err != nil {
		return Theme{}, false
	}
	name := strings.TrimSpace(f.Name)
	if name == "" {
		name = fileBase
	}
	if name == "" {
		return Theme{}, false
	}
	c := defaultColors()
	if v := f.Colors.Accent; v != "" {
		c.Accent = v
	}
	if v := f.Colors.Error; v != "" {
		c.Error = v
	}
	if v := f.Colors.Success; v != "" {
		c.Success = v
	}
	if v := f.Colors.Warning; v != "" {
		c.Warning = v
	}
	if v := f.Colors.Text; v != "" {
		c.Text = v
	}
	if v := f.Colors.Muted; v != "" {
		c.Muted = v
	}
	if v := f.Colors.Background; v != "" {
		c.Background = v
	}
	if v := f.Colors.Panel; v != "" {
		c.Panel = v
	}
	if v := f.Colors.Tabs; v != "" {
		c.Tabs = v
	}
	if v := f.Colors.Content; v != "" {
		c.Content = v
	}
	if v := f.Colors.Element; v != "" {
		c.Element = v
	}
	// tabs/content по умолчанию наследуют цвет panel (правая панель),
	// если явно не заданы в теме.
	if f.Colors.Tabs == "" {
		c.Tabs = c.Panel
	}
	if f.Colors.Content == "" {
		c.Content = c.Panel
	}
	if v := f.Colors.Selection; v != "" {
		c.Selection = v
	}
	if v := f.Colors.Border; v != "" {
		c.Border = v
	}
	if v := f.Colors.BorderSubtle; v != "" {
		c.BorderSubtle = v
	}
	if v := f.Colors.Modal; v != "" {
		c.Modal = v
	}
	return Theme{Name: name, Colors: c}, true
}

// applyTheme сохраняет активную тему и пересобирает package-стили.
func applyTheme(t Theme) {
	active = t
	acc := lipgloss.Color(t.Colors.Accent)
	muted := lipgloss.Color(t.Colors.Muted)
	Accent = acc
	HeaderStyle = lipgloss.NewStyle().Bold(true).Foreground(acc)
	FaintStyle = lipgloss.NewStyle().Foreground(muted)
	MutedStyle = lipgloss.NewStyle().Foreground(muted)
	DimStyle = lipgloss.NewStyle().Faint(true)
	ErrorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(t.Colors.Error))
	SaveOKStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(t.Colors.Success))
	TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(t.Colors.Text))
	SelectionStyle = lipgloss.NewStyle().
		Background(lipgloss.Color(t.Colors.Selection)).
		Foreground(lipgloss.Color(t.Colors.Accent))
	linkStyle = lipgloss.NewStyle().Underline(true).Foreground(acc)
	ModalStyle = lipgloss.NewStyle().
		Background(lipgloss.Color(t.Colors.Modal)).
		Padding(1, 2)
}
