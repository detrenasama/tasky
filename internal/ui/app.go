package ui

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbletea"

	"github.com/detrenasama/tasky/internal/store"
	"github.com/detrenasama/tasky/internal/ui/theme"
	"github.com/detrenasama/tasky/internal/update"
)

type screen int

const (
	screenTasks screen = iota
	screenProjects
	screenReports
	screenSettings
)

// sidebarItem — вкладка левой вертикальной панели: страница и значок
// (Nerd Font, одинарной ширины).
type sidebarItem struct {
	scr   screen
	glyph string
}

// sidebarItems — порядок вкладок слева направо сверху вниз. Значки требуют
// патченый шрифт Nerd Font; при его отсутствии glyph можно заменить на
// буквы T/P/R/S.
var sidebarItems = []sidebarItem{
	{screenTasks, "\uf0ae"},    // nf-fa-tasks
	{screenProjects, "\uf07b"}, // nf-fa-folder
	{screenReports, "\uf080"},  // nf-fa-bar_chart
	{screenSettings, "\uf013"}, // nf-fa-cog
}

// sidebarView рендерит левую вертикальную панель вкладок: ширина 3, высота h.
// Каждая вкладка — блок 5×3 со значком по центру (по 2 пробела с каждой
// стороны от одноширинного значка); активная залита фоном Selection,
// остальные — фоном Panel; строки ниже последней вкладки продолжают панель
// тем же фоном.
func sidebarView(cur screen, h int) string {
	const sideW = 5
	lines := make([]string, max(h, 0))
	for i := range lines {
		lines[i] = theme.SidebarInactive().Render(strings.Repeat(" ", sideW))
	}
	for k, it := range sidebarItems {
		top := k * 3
		if top >= h {
			break
		}
		st := theme.SidebarInactive()
		if it.scr == cur {
			st = theme.SidebarActive()
		}
		end := min(top+3, h)
		for r := top; r < end; r++ {
			cell := strings.Repeat(" ", sideW)
			if r == top+1 {
				cell = "  " + it.glyph + "  "
			}
			lines[r] = st.Render(cell)
		}
	}
	return strings.Join(lines, "\n")
}

// joinSidebar склеивает левую панель (ширина 5) и контент построчно.
func joinSidebar(sidebar, content string, w, h int) string {
	const sideW = 5
	cw := w - sideW
	sLines := strings.Split(sidebar, "\n")
	cLines := strings.Split(content, "\n")
	out := make([]string, max(h, 0))
	for i := 0; i < h; i++ {
		var s, c string
		if i < len(sLines) {
			s = sLines[i]
		}
		if i < len(cLines) {
			c = cLines[i]
		}
		out[i] = padW(s, sideW) + padW(c, cw)
	}
	return strings.Join(out, "\n")
}

// model — корневая модель приложения: переключение экранов (вкладки),
// тик 1с, хром (шапка/подвал/вкладки) и общие диалоги выхода/отчётов.
type model struct {
	store  store.Store
	screen screen
	width  int
	height int

	tasks    *tasksScreen
	proj     *projectsScreen
	reports  *reportsScreen
	settings *settingsScreen

	// quitting — подтверждение выхода при запущенном учёте времени;
	// quitTitle — название подзадачи с идущим временем.
	quitting  bool
	quitTitle string

	// reportConfirm — подтверждение перехода к отчётам при запущенном учёте
	// времени; reportTitle — название подзадачи с идущим временем.
	reportConfirm bool
	reportTitle   string

	// version — версия сборки (из -ldflags); updateVer — доступное обновление.
	version   string
	updateVer string

	// palette — палитра команд (ctrl+p): поисковая строка и список групп.
	paletteOpen   bool
	paletteInput  textinput.Model
	paletteRows   []paletteRow
	paletteSel    int
	paletteScroll int
}

// New создаёт корневую модель: экраны, общий конфиг отчётов, первичная
// загрузка данных. dataDir — корень данных (база, отчёты по умолчанию);
// version — версия сборки.
func New(st store.Store, dataDir, version string) *model {
	m := &model{store: st, version: version}
	m.tasks = newTasksScreen(st)
	m.tasks.version = version
	m.tasks.now = time.Now()
	m.tasks.load()
	m.proj = newProjectsScreen(st)
	m.proj.load()
	repCfg := &reportConfig{period: periodToday, saveDir: filepath.Join(dataDir, "reports")}
	m.reports = newReportsScreen(st, repCfg)
	m.reports.load()
	m.settings = newSettingsScreen(st, repCfg)
	pi := textinput.New()
	pi.Placeholder = "поиск команд…"
	pi.Prompt = "> "
	pi.CharLimit = 64
	pi.Width = 40
	m.paletteInput = pi
	return m
}

type tickMsg time.Time

func tickCmd(t time.Time) tea.Msg { return tickMsg(t) }

// updateMsg — результат проверки последней версии (пустая строка — ошибка).
type updateMsg string

func checkUpdateCmd() tea.Cmd {
	return func() tea.Msg {
		v, err := update.LatestVersion()
		if err != nil {
			return updateMsg("")
		}
		return updateMsg(v)
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(tea.Tick(time.Second, tickCmd), checkUpdateCmd())
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		if m.height < 4 {
			m.height = 4
		}
		m.tasks.resize(m.width-5, m.height-2)
		m.proj.resize(m.width-5, m.height-2)
		m.reports.resize(m.width-5, m.height-2)
		m.settings.resize(m.width-5, m.height-2)
		return m, nil
	case tea.KeyMsg:
		if m.quitting {
			switch msg.String() {
			case "y", "enter":
				if run, err := m.store.RunningSession(); err == nil && run != nil {
					m.store.StopSession(run.ID, time.Now())
				}
				return m, tea.Quit
			case "n", "esc", "q", "ctrl+c":
				m.quitting = false
			}
			return m, nil
		}
		if m.reportConfirm {
			switch msg.String() {
			case "y", "enter":
				if run, err := m.store.RunningSession(); err == nil && run != nil {
					m.store.StopSession(run.ID, time.Now())
				}
				m.reportConfirm = false
				m.switchScreen(screenReports)
				return m, nil
			case "n", "esc", "r", "q", "ctrl+c":
				m.reportConfirm = false
			}
			return m, nil
		}
		if m.paletteOpen {
			return m.updatePalette(msg)
		}
		if m.screen == screenProjects && m.proj.mode == projInput {
			return m.updateProjects(msg)
		}
		if m.screen == screenTasks && m.tasks.mode == taskInput {
			return m.updateTasks(msg)
		}
		if m.screen == screenProjects &&
			(m.proj.mode == projDescEdit || m.proj.mode == projLinkInput || m.proj.mode == projLinks || m.proj.mode == projLinkConfirm ||
				m.proj.mode == projSearch) {
			return m.updateProjects(msg)
		}
		if m.screen == screenTasks &&
			(m.tasks.mode == taskDescEdit || m.tasks.mode == taskLinkInput || m.tasks.mode == taskLinks ||
				m.tasks.mode == taskLinkConfirm || m.tasks.mode == taskJournal ||
				m.tasks.mode == taskStatusPick || m.tasks.mode == taskStatusNote ||
				m.tasks.mode == taskSearch || m.tasks.mode == taskTags ||
				m.tasks.mode == taskTagEdit || m.tasks.mode == taskTagTypePick ||
				m.tasks.mode == taskTagConfirm || m.tasks.mode == taskTitleEdit ||
				m.tasks.mode == taskChecklist || m.tasks.mode == taskChecklistConfirm) {
			return m.updateTasks(msg)
		}
		if m.screen == screenSettings && m.settings.mode != settingsBrowse {
			return m.updateSettings(msg)
		}
		// на экране отчётов ctrl+s сохраняет отчёт в файл
		if m.screen == screenReports && msg.String() == "ctrl+s" {
			m.reports.save()
			return m, nil
		}
		switch msg.String() {
		case "q", "ctrl+c":
			if run, err := m.store.RunningSession(); err == nil && run != nil {
				m.quitting = true
				m.quitTitle = run.Title
				return m, nil
			}
			return m, tea.Quit
		}
		// на экране проектов esc при активном поиске сбрасывает его (иначе
		// глобальный esc увёл бы на экран задач)
		if m.screen == screenProjects && m.proj.searchQuery != "" && msg.String() == "esc" {
			m.proj.clearSearch()
			return m, nil
		}
		if !m.modalOpen() {
			switch msg.String() {
			case "ctrl+p":
				m.openPalette()
				return m, nil
			}
		}
		if m.screen == screenTasks {
			return m.updateTasks(msg)
		}
		if m.screen == screenProjects {
			return m.updateProjects(msg)
		}
		if m.screen == screenSettings {
			return m.updateSettings(msg)
		}
		return m, nil
	case tickMsg:
		now := time.Time(msg)
		m.tasks.now = now
		if m.screen == screenTasks {
			m.tasks.today, _ = m.store.TodayTotal(now)
			m.tasks.weekly, _ = m.store.WeeklyTotal(now)
		}
		return m, tea.Tick(time.Second, tickCmd)
	case updateMsg:
		if msg != "" && m.version != "" && m.version != "dev" &&
			update.Compare(string(msg), m.version) > 0 {
			m.updateVer = string(msg)
			m.tasks.updateVer = string(msg)
		}
		return m, nil
	}
	return m, nil
}

func (m *model) modalOpen() bool {
	if m.settings != nil && m.settings.mode != settingsBrowse {
		return true
	}
	return m.tasks.mode != taskBrowse || m.proj.mode != projBrowse
}

// retheme пересобирает стили делегатов списков после смены темы (панели и
// модалки читают стили на рендере, им обновление не нужно).
func (m *model) retheme() {
	m.tasks.retheme()
	m.proj.retheme()
}

// switchScreen переключает страницу и подгружает её данные.
func (m *model) switchScreen(s screen) {
	m.screen = s
	switch s {
	case screenTasks:
		m.tasks.load()
	case screenProjects:
		m.proj.load()
	case screenReports:
		m.reports.load()
	case screenSettings:
		m.settings.load()
	}
}

func (m model) View() string {
	sideW := 5
	w, h := m.width, m.height
	if w < sideW+1 {
		w = sideW + 1
	}
	if h < 4 {
		h = 4
	}
	cw := w - sideW
	midH := h - 2

	var header, mid, footer string
	var dlg string
	var modalOpen bool
	switch m.screen {
	case screenProjects:
		header, footer = m.proj.header(cw), m.proj.footer(cw)
		mid = m.proj.view(cw, midH)
		dlg, modalOpen = m.proj.dialog()
	case screenReports:
		header, footer = m.reports.header(cw), m.reports.footer(cw)
		mid = m.reports.view(cw, midH)
		dlg, modalOpen = m.reports.dialog()
	case screenSettings:
		header, footer = m.settings.header(cw), m.settings.footer(cw)
		mid = m.settings.view(cw, midH)
		dlg, modalOpen = m.settings.dialog()
	default:
		header, footer = m.tasks.header(cw), m.tasks.footer(cw)
		mid = m.tasks.view(cw, midH)
		dlg, modalOpen = m.tasks.dialog()
	}

	content := header + "\n" + mid + "\n" + footer
	full := joinSidebar(sidebarView(m.screen, h), content, w, h)
	if modalOpen {
		full = overlay(full, dlg, w, h)
	}
	if m.paletteOpen {
		d, _ := m.paletteDialog()
		if d != "" {
			full = overlay(full, d, w, h)
		}
	}
	if m.quitting {
		d := dialog{
			title:   "Учёт времени запущен",
			body:    fmt.Sprintf("На подзадаче «%s» идёт учёт времени.\nОстановить и выйти?", m.quitTitle),
			primary: "Enter — остановить и выйти",
			esc:     "Esc — отмена",
		}
		full = overlay(full, d.render(), w, h)
	}
	if m.reportConfirm {
		d := dialog{
			title:   "Учёт времени запущен",
			body:    fmt.Sprintf("На подзадаче «%s» идёт учёт времени.\nОстановить и сформировать отчёт?", m.reportTitle),
			primary: "Enter — остановить и сформировать отчёт",
			esc:     "Esc — отмена",
		}
		full = overlay(full, d.render(), w, h)
	}
	return full
}
