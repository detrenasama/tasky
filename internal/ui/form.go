package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbletea"

	"github.com/detrenasama/tasky/internal/ui/theme"
)

// Field — одно текстовое поле декларативной формы.
type Field struct {
	Key         string
	Label       string
	Value       string
	Placeholder string
	Width       int
	Required    bool
}

type formField struct {
	def   Field
	input textinput.Model
}

// Form — декларативная модалка с набором текстовых полей. Enter на
// непоследнем поле переходит к следующему; Enter на последнем (или Ctrl+S)
// отправляет значения в onSubmit. onSubmit возвращает (stayOpen, err):
// при err != nil форма остаётся открытой и показывает ошибку; при
// stayOpen == true — тоже остаётся (для live-логики без ошибки).
// Esc закрывает форму (вызывает onCancel, если задан); Tab/↑/↓ перемещают
// фокус. Добавление формы = один вызов NewForm в одном месте.
type Form struct {
	title    string
	fields   []formField
	focus    int
	err      error
	onSubmit func(values map[string]string) (stayOpen bool, err error)
	onCancel func()
}

// NewForm создаёт форму из полей и обработчика submit. Первое поле в фокусе.
func NewForm(title string, fields []Field, onSubmit func(values map[string]string) (bool, error)) *Form {
	ff := make([]formField, len(fields))
	for i, f := range fields {
		ti := textinput.New()
		ti.Placeholder = f.Placeholder
		ti.Prompt = "> "
		ti.CharLimit = 512
		ti.Width = f.Width
		ti.SetValue(f.Value)
		ff[i] = formField{def: f, input: ti}
	}
	f := &Form{title: title, fields: ff, onSubmit: onSubmit}
	if len(ff) > 0 {
		f.fields[0].input.Focus()
	}
	return f
}

func (f *Form) values() map[string]string {
	v := make(map[string]string, len(f.fields))
	for _, ff := range f.fields {
		v[ff.def.Key] = ff.input.Value()
	}
	return v
}

func (f *Form) focusInput() {
	for i := range f.fields {
		if i == f.focus {
			f.fields[i].input.Focus()
		} else {
			f.fields[i].input.Blur()
		}
	}
}

// SetValue устанавливает значение поля по ключу (используется live-логикой
// и тестами).
func (f *Form) SetValue(key, val string) {
	for i := range f.fields {
		if f.fields[i].def.Key == key {
			f.fields[i].input.SetValue(val)
		}
	}
}

// Value возвращает значение поля по ключу.
func (f *Form) Value(key string) string {
	for i := range f.fields {
		if f.fields[i].def.Key == key {
			return f.fields[i].input.Value()
		}
	}
	return ""
}

// FocusedKey возвращает ключ сфокусированного поля.
func (f *Form) FocusedKey() string {
	if f.focus >= 0 && f.focus < len(f.fields) {
		return f.fields[f.focus].def.Key
	}
	return ""
}

func (f *Form) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok {
		switch k.String() {
		case "esc":
			if f.onCancel != nil {
				f.onCancel()
			}
			return nil, nil
		case "ctrl+s":
			return f.submit()
		case "enter":
			if f.focus < len(f.fields)-1 {
				f.focus++
				f.focusInput()
				return f, nil
			}
			return f.submit()
		case "tab", "down":
			if len(f.fields) > 1 {
				f.focus = (f.focus + 1) % len(f.fields)
				f.focusInput()
			}
			return f, nil
		case "up":
			if len(f.fields) > 1 {
				f.focus = (f.focus - 1 + len(f.fields)) % len(f.fields)
				f.focusInput()
			}
			return f, nil
		}
	}
	var cmd tea.Cmd
	f.fields[f.focus].input, cmd = f.fields[f.focus].input.Update(msg)
	return f, cmd
}

func (f *Form) submit() (tea.Model, tea.Cmd) {
	values := f.values()
	for _, ff := range f.fields {
		if ff.def.Required && strings.TrimSpace(values[ff.def.Key]) == "" {
			f.err = fmt.Errorf("поле «%s» не может быть пустым", ff.def.Label)
			return f, nil
		}
	}
	if f.onSubmit == nil {
		return nil, nil
	}
	stay, err := f.onSubmit(values)
	f.err = err
	if stay || err != nil {
		return f, nil
	}
	return nil, nil
}

// Init заглушка для удовлетворенияtea.Model.
func (f *Form) Init() tea.Cmd { return nil }

func (f *Form) View() string {
	var b strings.Builder
	for i, ff := range f.fields {
		label := "  " + ff.def.Label
		if i == f.focus {
			label = theme.HeaderStyle.Render("▸ ") + ff.def.Label
		}
		b.WriteString(label + "\n")
		b.WriteString(ff.input.View() + "\n")
	}
	if f.err != nil {
		b.WriteString("\n" + theme.ErrorStyle.Render("Ошибка: "+f.err.Error()))
	}
	d := dialog{
		title:   f.title,
		body:    b.String(),
		primary: "Enter — сохранить · Tab — поле",
		esc:     "Esc — отмена",
	}
	return d.render()
}
