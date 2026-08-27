package ui

import (
	"strings"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/detrenasama/tasky/internal/ui/theme"
)

// cmdItem — команда палитры: название, необязательное короткое описание,
// сочетание клавиш (справа) и действие.
type cmdItem struct {
	title string
	desc  string
	keys  string
	run   func(m *model)
}

// cmdGroup — группа команд палитры с заголовком.
type cmdGroup struct {
	name string
	cmds []cmdItem
}

// paletteRow — строка палитры: либо заголовок группы (header), либо команда
// (cmd); header и cmd взаимоисключающие.
type paletteRow struct {
	header string
	cmd    *cmdItem
}

// paletteGroups собирает группы команд для текущего экрана: навигация
// доступна всегда, действия — только на экранах задач и проектов.
func (m *model) paletteGroups() []cmdGroup {
	nav := cmdGroup{name: "Навигация", cmds: []cmdItem{
		{title: "Задачи", keys: "ctrl+t", run: func(m *model) { m.switchScreen(screenTasks) }},
		{title: "Проекты", keys: "ctrl+p", run: func(m *model) { m.switchScreen(screenProjects) }},
		{title: "Отчеты", keys: "ctrl+r", run: func(m *model) {
			if run, err := m.store.RunningSession(); err == nil && run != nil {
				m.reportConfirm = true
				m.reportTitle = run.Title
				return
			}
			m.switchScreen(screenReports)
		}},
		{title: "Настройки", keys: "ctrl+s", run: func(m *model) { m.switchScreen(screenSettings) }},
	}}
	groups := []cmdGroup{nav}
	switch m.screen {
	case screenTasks:
		groups = append(groups, cmdGroup{name: "Действия", cmds: []cmdItem{
			{title: "Новая задача", keys: "n", run: func(m *model) { m.tasks.startNewTask() }},
			{title: "Новая подзадача", keys: "a", run: func(m *model) { m.tasks.startNewSubtask() }},
			{title: "Изменить название", keys: "e", run: func(m *model) { m.tasks.startEditTitle() }},
			{title: "Изменить описание", keys: "enter", run: func(m *model) { m.tasks.startEditDescription() }},
			{title: "Удалить задачу", keys: "d", run: func(m *model) { m.tasks.startDelete() }},
			{title: "Чеклист", keys: "i", run: func(m *model) { m.tasks.openChecklist() }},
			{title: "Статус: следующий", keys: "x", run: func(m *model) { m.tasks.shiftStatus(1) }},
			{title: "Статус: предыдущий", keys: "z", run: func(m *model) { m.tasks.shiftStatus(-1) }},
			{title: "Статус: выбрать", keys: "c", run: func(m *model) { m.tasks.openStatusPick() }},
			{title: "Запустить/пауза", keys: "ctrl+l", run: func(m *model) { m.tasks.toggleTimer() }},
			{title: "Время", keys: "t", run: func(m *model) { m.tasks.openTimeEntries() }},
			{title: "Проект ←", keys: "[", run: func(m *model) { m.tasks.switchProject(-1) }},
			{title: "Проект →", keys: "]", run: func(m *model) { m.tasks.switchProject(1) }},
			{title: "Переместить вверх", keys: "ctrl+up", run: func(m *model) { m.tasks.moveSelected(-1) }},
			{title: "Переместить вниз", keys: "ctrl+down", run: func(m *model) { m.tasks.moveSelected(1) }},
			{title: "Теги", keys: "g", run: func(m *model) { m.tasks.openTags() }},
			{title: "Открыть ссылки", keys: "o", run: func(m *model) { m.tasks.openLinks() }},
			{title: "Запись журнала", keys: "ctrl+j", run: func(m *model) { m.tasks.startJournal() }},
			{title: "Изменить запись", keys: "j", run: func(m *model) { m.tasks.editTodayJournal() }},
			{title: "Поиск", keys: "/", run: func(m *model) { m.tasks.startSearch() }},
		}})
	case screenProjects:
		groups = append(groups, cmdGroup{name: "Действия", cmds: []cmdItem{
			{title: "Новый проект", keys: "n", run: func(m *model) { m.proj.startNew() }},
			{title: "Изменить описание", keys: "enter", run: func(m *model) { m.proj.startEditDescription() }},
			{title: "Удалить проект", keys: "d", run: func(m *model) { m.proj.startDelete() }},
			{title: "Открыть ссылки", keys: "o", run: func(m *model) { m.proj.openLinks() }},
			{title: "Поиск", keys: "/", run: func(m *model) { m.proj.startSearch() }},
		}})
	}
	return groups
}

// paletteVisible — число видимых строк палитры: зависит от высоты экрана.
func (m *model) paletteVisible() int {
	return max(4, min(12, m.height-12))
}

// paletteRebuild пересобирает список строк по запросу фильтра (живой).
func (m *model) paletteRebuild() {
	m.paletteRows = nil
	q := strings.ToLower(strings.TrimSpace(m.paletteInput.Value()))
	for _, g := range m.paletteGroups() {
		matched := false
		for _, c := range g.cmds {
			if q != "" && !strings.Contains(strings.ToLower(c.title), q) &&
				!strings.Contains(strings.ToLower(c.desc), q) {
				continue
			}
			if !matched {
				m.paletteRows = append(m.paletteRows, paletteRow{header: g.name})
				matched = true
			}
			cc := c
			m.paletteRows = append(m.paletteRows, paletteRow{cmd: &cc})
		}
	}
	m.paletteSel = -1
	for i, row := range m.paletteRows {
		if row.cmd != nil {
			m.paletteSel = i
			break
		}
	}
	m.clampPaletteScroll()
}

// clampPaletteScroll удерживает выбранную строку в видимой области с отступом
// listScrollOff строк сверху и снизу (как vim scrolloff), если окно достаточно
// велико; иначе прижимает к краю без отступа.
func (m *model) clampPaletteScroll() {
	v := m.paletteVisible()
	off := listScrollOff
	if v <= 2*off {
		off = 0
	}
	if m.paletteSel < 0 {
		m.paletteScroll = 0
		return
	}
	if m.paletteSel < m.paletteScroll+off {
		m.paletteScroll = m.paletteSel - off
	}
	if m.paletteSel > m.paletteScroll+v-1-off {
		m.paletteScroll = m.paletteSel - v + 1 + off
	}
	if m.paletteScroll < 0 {
		m.paletteScroll = 0
	}
	if m.paletteScroll > max(len(m.paletteRows)-v, 0) {
		m.paletteScroll = max(len(m.paletteRows)-v, 0)
	}
}

// paletteMove сдвигает курсор на следующую команду (заголовки групп
// пропускаются, движение по кругу).
func (m *model) paletteMove(d int) {
	n := len(m.paletteRows)
	if n == 0 {
		return
	}
	step := 1
	if d < 0 {
		step = -1
	}
	for i := 0; i < n; i++ {
		m.paletteSel = (m.paletteSel + step + n) % n
		if m.paletteRows[m.paletteSel].cmd != nil {
			m.clampPaletteScroll()
			return
		}
	}
}

// execPaletteCmd закрывает палитру и выполняет команду.
func (m *model) execPaletteCmd(c *cmdItem) {
	if c == nil || c.run == nil {
		m.closePalette()
		return
	}
	run := c.run
	m.closePalette()
	run(m)
}

// paletteExec выполняет выбранную команду (Enter).
func (m *model) paletteExec() {
	if m.paletteSel < 0 || m.paletteSel >= len(m.paletteRows) {
		m.closePalette()
		return
	}
	m.execPaletteCmd(m.paletteRows[m.paletteSel].cmd)
}

// paletteExecKey сразу выполняет команду по её сочетанию клавиш. Команда
// ищется в нефильтрованных группах: хоткей работает даже при активном
// фильтре, скрывшем пункт навигации.
func (m *model) paletteExecKey(keys string) {
	for _, g := range m.paletteGroups() {
		for _, c := range g.cmds {
			if c.keys == keys {
				m.execPaletteCmd(&c)
				return
			}
		}
	}
}

func (m *model) openPalette() {
	m.paletteOpen = true
	m.paletteInput.SetValue("")
	m.paletteInput.Focus()
	m.paletteRebuild()
}

func (m *model) closePalette() {
	m.paletteOpen = false
	m.paletteInput.Blur()
	m.paletteInput.SetValue("")
	m.paletteRows = nil
	m.paletteSel = -1
	m.paletteScroll = 0
}

// updatePalette обрабатывает клавиши палитры: навигация по списку, Enter —
// выполнить, Esc — закрыть, ctrl+t/p/r/s — выполнить команду навигации
// сразу; все остальные клавиши уходят в поисковую строку (живой фильтр).
func (m model) updatePalette(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up":
		m.paletteMove(-1)
		return m, nil
	case "down":
		m.paletteMove(1)
		return m, nil
	case "pgup":
		m.paletteMove(-m.paletteVisible())
		return m, nil
	case "pgdown":
		m.paletteMove(m.paletteVisible())
		return m, nil
	case "enter":
		m.paletteExec()
		return m, nil
	case "esc", "ctrl+c":
		m.closePalette()
		return m, nil
	case "ctrl+t", "ctrl+p", "ctrl+r", "ctrl+s":
		m.paletteExecKey(msg.String())
		return m, nil
	default:
		var cmd tea.Cmd
		m.paletteInput, cmd = m.paletteInput.Update(msg)
		m.paletteRebuild()
		return m, cmd
	}
}

// paletteDialog собирает модалку «Команды»: поисковая строка сверху, под ней
// список команд группами (заголовки не выбираются), справа — сочетания
// клавиш.
func (m *model) paletteDialog() (string, bool) {
	if !m.paletteOpen {
		return "", false
	}
	pw := m.paletteWidth()
	m.paletteInput.Width = max(pw-2, 10)
	visible := m.paletteVisible()
	var lines []string
	if len(m.paletteRows) == 0 {
		lines = append(lines, padW(theme.Faint("Ничего не найдено"), pw))
	}
	end := min(len(m.paletteRows), m.paletteScroll+visible)
	for i := m.paletteScroll; i < end; i++ {
		row := m.paletteRows[i]
		if row.header != "" {
			var topPadding = ""
			if i > 0 {
				topPadding = "\n"
			}
			lines = append(lines, padW(theme.Faint(topPadding+row.header), pw))
			continue
		}
		lines = append(lines, m.paletteCmdLine(row.cmd, pw, i == m.paletteSel))
	}
	body := padW(m.paletteInput.View(), pw) + "\n\n" + strings.Join(lines, "\n")
	d := dialog{title: "Команды", body: body,
		primary: "Enter — выполнить", esc: "Esc — отмена"}
	return d.render(), true
}

// paletteCmdLine — строка команды: название, описание и сочетание клавиш
// справа по правому краю; выбранная — залита фоном Selection на всю ширину.
func (m *model) paletteCmdLine(c *cmdItem, pw int, sel bool) string {
	content := c.title
	if c.desc != "" {
		content += " " + theme.Faint(c.desc)
	}
	if c.keys != "" {
		keys := theme.Faint(c.keys)
		content = padW(content, pw-2-lipgloss.Width(keys)) + keys
	} else {
		content = padW(content, pw-2)
	}
	if sel {
		return renderSelectionLine(padW("  "+content, pw))
	}
	return "  " + content
}

// renderSelectionLine заливает строку фоном SelectionStyle, восстанавливая его
// после внутренних \x1b[0m контента (faint-подписи описания и клавиш): диалог
// после этого подставит фон модалки после каждого \x1b[0m, но фон выделения
// идёт последним SGR и побеждает.
func renderSelectionLine(content string) string {
	out := theme.SelectionStyle.Render(content)
	probe := lipgloss.NewStyle().Background(theme.SelectionStyle.GetBackground()).Render("§")
	bg := strings.Split(probe, "§")[0]
	if bg == "" {
		return out
	}
	body, ok := strings.CutSuffix(out, "\x1b[0m")
	if !ok {
		return strings.ReplaceAll(out, "\x1b[0m", "\x1b[0m"+bg)
	}
	return strings.ReplaceAll(body, "\x1b[0m", "\x1b[0m"+bg) + "\x1b[0m"
}

// paletteWidth — ширина строк палитры: под самую длинную строку с запасом,
// ограничена шириной экрана.
func (m *model) paletteWidth() int {
	pw := 50
	for _, row := range m.paletteRows {
		w := 0
		if row.cmd != nil {
			c := row.cmd
			w = lipgloss.Width(c.title) + 2
			if c.desc != "" {
				w += 1 + lipgloss.Width(c.desc)
			}
			if c.keys != "" {
				w += lipgloss.Width(c.keys) + 2
			}
		} else {
			w = lipgloss.Width(row.header) + 2
		}
		if w > pw {
			pw = w
		}
	}
	pw += 6
	if m.width > 0 && pw > m.width-8 {
		pw = m.width - 8
	}
	if pw < 40 {
		pw = 40
	}
	return pw
}
