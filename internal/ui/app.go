package ui

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/detrenasama/tasky/changelog"
	"github.com/detrenasama/tasky/internal/feed"
	"github.com/detrenasama/tasky/internal/store"
	"github.com/detrenasama/tasky/internal/ui/theme"
	"github.com/detrenasama/tasky/internal/update"
)

type screen int

const (
	screenFeed screen = iota
	screenTasks
	screenProjects
	screenReports
	screenSettings
)

// sideW — ширина левой вертикальной панели вкладок; rightW — ширина правой
// колонки (всегда видима на всех экранах).
const sideW = 5
const rightW = 42

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
	{screenFeed, "\uf1ea"},     // nf-fa-newspaper
	{screenTasks, "\uf0ae"},    // nf-fa-tasks
	{screenProjects, "\uf07b"}, // nf-fa-folder
	{screenReports, "\uf080"},  // nf-fa-bar_chart
	{screenSettings, "\uf013"}, // nf-fa-cog
}

// sidebarView рендерит левую вертикальную панель вкладок: ширина 3, высота h.
// Каждая вкладка — блок 5×3 со значком по центру (по 2 пробела с каждой
// стороны от одноширинного значка); активная залита фоном Selection,
// остальные — фоном Panel; строки ниже последней вкладки продолжают панель
// тем же фоном. unread — число непрочитанных в ленте, рисует красную точку
// в правом верхнем углу таба ленты.
func sidebarView(cur screen, h int, unread int) string {
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
			// красная точка на ленте
			if it.scr == screenFeed && unread > 0 && r == top {
				// правый верхний угол (колонка 4)
				cell = "    " + theme.ErrorStyle.Render("●")
			}
			lines[r] = st.Render(cell)
		}
	}
	return strings.Join(lines, "\n")
}

// model — корневая модель приложения: переключение экранов (вкладки),
// тик 1с, хром (шапка/подвал/вкладки) и общие диалоги выхода/отчётов.
type model struct {
	store  store.Store
	screen screen
	width  int
	height int

	feed     *feedScreen
	tasks    *tasksScreen
	proj     *projectsScreen
	reports  *reportsScreen
	settings *settingsScreen

	unreadFeed int
	modal      ModalHost

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

	// pendingChangelog — изменения, которые нужно показать на первом
	// WindowSizeMsg (когда уже известны размеры окна).
	pendingChangelog []changelog.Entry

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
	m := &model{store: st, version: version, screen: screenFeed}
	m.feed = newFeedScreen(st)
	m.feed.load()
	m.tasks = newTasksScreen(st)
	m.tasks.now = time.Now()
	m.tasks.load()
	m.proj = newProjectsScreen(st)
	m.proj.load()
	repCfg := &reportConfig{period: periodToday, saveDir: filepath.Join(dataDir, "reports")}
	m.reports = newReportsScreen(st, repCfg)
	m.reports.load()
	m.settings = newSettingsScreen(st, repCfg)
	if n, _ := st.FeedUnreadCount(); n >= 0 {
		m.unreadFeed = n
	}
	pi := textinput.New()
	pi.Placeholder = "поиск команд…"
	pi.Prompt = "> "
	pi.CharLimit = 64
	pi.Width = 40
	m.paletteInput = pi
	m.initChangelog()
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
	cmds := []tea.Cmd{tea.Tick(time.Second, tickCmd)}
	if update.IsRelease(m.version) {
		cmds = append(cmds, checkUpdateCmd())
	}
	return tea.Batch(cmds...)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		if m.height < 4 {
			m.height = 4
		}
		// Внутренние размеры центральной области: view() рендерится в boxStyle
		// (горизонтальный паддинг 4, вертикальный — 4 с учётом подвала),
		// поэтому экраны должны раскладываться ровно в эти размеры, иначе
		// нижние/правые строки колонок (в т.ч. описания) усекаются.
		innerW := m.width - sideW - rightW - 4
		innerH := m.height - 5
		if m.feed != nil {
			m.feed.resize(innerW, innerH)
		}
		if m.tasks != nil {
			m.tasks.resize(innerW, innerH)
		}
		if m.proj != nil {
			m.proj.resize(innerW, innerH)
		}
		if m.reports != nil {
			m.reports.resize(innerW, innerH)
		}
		if m.settings != nil {
			m.settings.resize(innerW, innerH)
		}
		var cmd tea.Cmd
		if m.modal.Active() {
			cmd = m.modal.Update(msg)
		}
		m.openPendingChangelog()
		return m, cmd
	case editReturnMsg:
		if m.screen == screenProjects {
			m.proj.applyExternalEdit(msg)
		} else {
			m.tasks.applyExternalEdit(msg)
		}
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
		if m.modal.Active() {
			cmd := m.modal.Update(msg)
			return m, cmd
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
			(m.proj.mode == projDescModal || m.proj.mode == projLinkEdit || m.proj.mode == projLinks || m.proj.mode == projLinkConfirm ||
				m.proj.mode == projSearch) {
			return m.updateProjects(msg)
		}
		if m.screen == screenTasks &&
			(m.tasks.mode == taskDescModal || m.tasks.mode == taskLinkEdit || m.tasks.mode == taskLinks ||
				m.tasks.mode == taskLinkConfirm || m.tasks.mode == taskJournal ||
				m.tasks.mode == taskStatusPick || m.tasks.mode == taskStatusNote ||
				m.tasks.mode == taskSearch || m.tasks.mode == taskTags ||
				m.tasks.mode == taskTagEdit || m.tasks.mode == taskTagTypePick ||
				m.tasks.mode == taskTagConfirm || m.tasks.mode == taskTitleEdit ||
				m.tasks.mode == taskChecklist || m.tasks.mode == taskChecklistConfirm ||
				m.tasks.mode == taskJournalDiscard ||
				m.tasks.mode == taskTimeList || m.tasks.mode == taskTimeEdit ||
				m.tasks.mode == taskTimeDelete) {
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
			case "alt+1":
				m.switchScreen(screenFeed)
				return m, nil
			case "alt+2":
				m.switchScreen(screenTasks)
				return m, nil
			case "alt+3":
				m.switchScreen(screenProjects)
				return m, nil
			case "alt+4":
				if run, err := m.store.RunningSession(); err == nil && run != nil {
					m.reportConfirm = true
					m.reportTitle = run.Title
					return m, nil
				}
				m.switchScreen(screenReports)
				return m, nil
			case "alt+5":
				m.switchScreen(screenSettings)
				return m, nil
			}
		}
		if m.screen == screenFeed {
			return m.updateFeed(msg)
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
		if m.tasks != nil {
			m.tasks.now = now
		}
		if m.screen == screenTasks && m.tasks != nil {
			m.tasks.today, _ = m.store.TodayTotal(now)
			m.tasks.weekly, _ = m.store.WeeklyTotal(now)
		}
		// feed генерация (раз в секунду проверяем NeedsGenerate)
		if s, ok := m.store.(*store.SQLite); ok {
			if feed.NeedsGenerate(s.DB(), now) {
				_, _ = feed.GenerateDaily(s.DB(), now)
			}
		}
		if n, _ := m.store.FeedUnreadCount(); n != m.unreadFeed {
			m.unreadFeed = n
			if m.screen == screenFeed && m.feed != nil {
				m.feed.load()
			}
		}
		return m, tea.Tick(time.Second, tickCmd)
	case updateMsg:
		if msg != "" && update.IsRelease(m.version) &&
			update.Compare(string(msg), m.version) > 0 {
			m.updateVer = string(msg)
			body := changelog.Render(changelog.Current(string(msg)))
			if s, ok := m.store.(*store.SQLite); ok {
				_, _ = feed.GenerateUpdate(s.DB(), string(msg), body, time.Now())
			}
			if n, _ := m.store.FeedUnreadCount(); n >= 0 {
				m.unreadFeed = n
			}
		}
		return m, nil
	default:
		// Прочие сообщения (например, прогресс обновления) получает активная
		// модалка — иначе они теряются и канал обновления переполняется.
		if m.modal.Active() {
			return m, m.modal.Update(msg)
		}
	}
	return m, nil
}

func (m *model) updateFeed(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.feed == nil {
		return m, nil
	}
	// delegate
	_, cmd := m.feed.updateFeed(msg)
	// after update, check unread changed
	if sel := m.feed.selected(); sel != nil && !sel.IsRead {
		// selected changed inside updateFeed already marked read
	}
	if n, _ := m.store.FeedUnreadCount(); n >= 0 {
		m.unreadFeed = n
	}
	return m, cmd
}

func (m *model) openTaskDates() {
	kind, id := m.tasks.selectedKindID()
	if id == 0 {
		return
	}
	var dueStr, schedStr string
	if kind == kindTask {
		for _, t := range m.tasks.tasks {
			if t.ID == id {
				if t.DueDate != nil {
					dueStr = t.DueDate.Format("02.01.2006")
				}
				if t.ScheduleDate != nil {
					schedStr = t.ScheduleDate.Format("02.01.2006")
				}
				break
			}
		}
	} else {
		for _, st := range m.tasks.subs {
			if st.ID == id {
				if st.DueDate != nil {
					dueStr = st.DueDate.Format("02.01.2006")
				}
				if st.ScheduleDate != nil {
					schedStr = st.ScheduleDate.Format("02.01.2006")
				}
				break
			}
		}
	}
	form := NewForm("Срок и план", []Field{
		{Key: "due", Label: "Срок (ДД.ММ.ГГГГ)", Value: dueStr, Placeholder: "пусто — без срока", Width: 30},
		{Key: "sched", Label: "Запланировано (ДД.ММ.ГГГГ)", Value: schedStr, Placeholder: "пусто — без плана", Width: 30},
	}, func(vals map[string]string) (bool, error) {
		due, err := parseDateField(vals["due"])
		if err != nil {
			return true, err
		}
		sched, err := parseDateField(vals["sched"])
		if err != nil {
			return true, err
		}
		var e error
		if kind == kindTask {
			e = m.store.SetTaskDates(id, due, sched)
		} else {
			e = m.store.SetSubtaskDates(id, due, sched)
		}
		if e != nil {
			return true, e
		}
		m.tasks.load()
		return false, nil
	})
	m.modal.Show(form)
}

func parseDateField(s string) (*time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	// layout "2.1.2006" accepts without leading zeros, like reports custom period
	t, err := time.Parse("2.1.2006", s)
	if err != nil {
		return nil, fmt.Errorf("неверный формат даты %q, ожидается ДД.ММ.ГГГГ", s)
	}
	// normalize to 00:00 local
	nd := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local)
	return &nd, nil
}

func (m *model) modalOpen() bool {
	if m.modal.Active() {
		return true
	}
	if m.settings != nil && m.settings.mode != settingsBrowse {
		return true
	}
	if m.tasks != nil && m.tasks.mode != taskBrowse {
		return true
	}
	if m.proj != nil && m.proj.mode != projBrowse {
		return true
	}
	return false
}

// retheme пересобирает стили делегатов списков после смены темы (панели и
// модалки читают стили на рендере, им обновление не нужно).
func (m *model) retheme() {
	if m.feed != nil {
		m.feed.retheme()
	}
	if m.tasks != nil {
		m.tasks.retheme()
	}
	if m.proj != nil {
		m.proj.retheme()
	}
}

// switchScreen переключает страницу и подгружает её данные.
func (m *model) switchScreen(s screen) {
	m.screen = s
	switch s {
	case screenFeed:
		if m.feed != nil {
			m.feed.load()
		}
	case screenTasks:
		if m.tasks != nil {
			m.tasks.load()
		}
	case screenProjects:
		if m.proj != nil {
			m.proj.load()
		}
	case screenReports:
		if m.reports != nil {
			m.reports.load()
		}
	case screenSettings:
		if m.settings != nil {
			m.settings.load()
		}
	}
	if n, _ := m.store.FeedUnreadCount(); n >= 0 {
		m.unreadFeed = n
	}
}

func (m model) View() string {
	w, h := m.width, m.height
	if w < sideW+rightW+20 {
		w = sideW + rightW + 20
	}
	if h < 8 {
		h = 8
	}
	centralW := w - sideW - rightW
	midH := h - 1

	var mid, footer, rightMid string
	var dlg string
	var modalOpen bool
	switch m.screen {
	case screenFeed:
		if m.feed != nil {
			mid = m.feed.view(centralW-4, midH-4)
			footer = m.feed.footer(centralW)
			rightMid = m.feed.rightContent(midH - 1)
			dlg, modalOpen = m.feed.dialog()
		} else {
			mid = m.tasks.view(centralW-4, midH-4)
			footer = m.tasks.footer(centralW)
			rightMid = m.tasks.rightContent(midH - 1)
			dlg, modalOpen = m.tasks.dialog()
		}
	case screenProjects:
		mid = m.proj.view(centralW-4, midH-4)
		footer = m.proj.footer(centralW)
		rightMid = m.proj.rightContent(midH - 1)
		dlg, modalOpen = m.proj.dialog()
	case screenReports:
		mid = m.reports.view(centralW-4, midH-4)
		footer = m.reports.footer(centralW)
		rightMid = m.reports.rightContent(midH - 1)
		dlg, modalOpen = m.reports.dialog()
	case screenSettings:
		mid = m.settings.view(centralW-4, midH-4)
		footer = m.settings.footer(centralW)
		rightMid = m.settings.rightContent(midH - 1)
		dlg, modalOpen = m.settings.dialog()
	default:
		mid = m.tasks.view(centralW-4, midH-4)
		footer = m.tasks.footer(centralW)
		rightMid = m.tasks.rightContent(midH - 1)
		dlg, modalOpen = m.tasks.dialog()
	}

	// центральная часть и нижняя панель подсказок — единый паддинг
	// (вертикальный 1, горизонтальный 2), фон panel, как у правой колонки.
	boxStyle := lipgloss.NewStyle().Padding(1, 2).Background(theme.Pane(false).GetBackground())
	// Простой рендер (без post-обработки reset'ов renderPane): фон задаётся
	// для отступов, а внутренние области (списки и т.п.) несут собственный
	// фон (контент или серый для выделенной строки) и сами восстанавливают
	// его после внутренних reset'ов.
	mid = boxStyle.Render(padLines(mid, max(centralW-4, 1), max(midH-4, 1)))

	// нижняя панель подсказок: клавиши белым, названия серым (как сейчас);
	// справа — «ctrl+p команды» с отступом 2, чтобы подсказки не перекрывались.
	footInnerW := max(centralW-4, 1)
	footRight := "  " + styleHints("ctrl+p команды")
	footLeftW := max(footInnerW-lipgloss.Width(footRight), 0)
	footLine := padW(truncateW(styleHints(footer), footLeftW), footLeftW) + footRight
	footPanel := renderPane(boxStyle, padLines(footLine, footInnerW, 1))

	sbLines := strings.Split(sidebarView(m.screen, h, m.unreadFeed), "\n")
	railLines := strings.Split(rightRail(m.version, m.updateVer, rightW, h, rightMid), "\n")
	midLines := strings.Split(mid, "\n")
	footLines := strings.Split(footPanel, "\n")

	out := make([]string, h)
	for i := 0; i < h; i++ {
		s := padW(sbLines[i], sideW)
		var c string
		if i < len(midLines) {
			c = padW(midLines[i], centralW)
		} else {
			idx := i - len(midLines)
			if idx < len(footLines) {
				c = padW(footLines[idx], centralW)
			} else {
				c = padW("", centralW)
			}
		}
		r := padW(railLines[i], rightW)
		out[i] = s + c + r
	}
	full := strings.Join(out, "\n")

	if modalOpen {
		// крупная модалка описания уже имеет нужный размер — не переносить
		modalMaxW := dialogMaxW(w)
		if m.tasks.mode == taskDescModal || m.proj.mode == projDescModal {
			modalMaxW = 0
		}
		full = overlay(full, dlg, w, h, modalMaxW)
	}
	if m.paletteOpen {
		d, _ := m.paletteDialog()
		if d != "" {
			full = overlay(full, d, w, h, 0)
		}
	}
	if m.modal.Active() {
		d := m.modal.View()
		full = overlay(full, d, w, h, dialogMaxW(w))
	}
	if m.quitting {
		d := dialog{
			title:   "Учёт времени запущен",
			body:    fmt.Sprintf("На подзадаче «%s» идёт учёт времени.\nОстановить и выйти?", m.quitTitle),
			primary: "Enter — остановить и выйти",
			esc:     "Esc — отмена",
		}
		full = overlay(full, d.render(), w, h, dialogMaxW(w))
	}
	if m.reportConfirm {
		d := dialog{
			title:   "Учёт времени запущен",
			body:    fmt.Sprintf("На подзадаче «%s» идёт учёт времени.\nОстановить и сформировать отчёт?", m.reportTitle),
			primary: "Enter — остановить и сформировать отчёт",
			esc:     "Esc — отмена",
		}
		full = overlay(full, d.render(), w, h, dialogMaxW(w))
	}
	return full
}

// rightRail — правая колонка (шириной width, высотой height) с общим
// паддингом (вертикальный 1, горизонтальный 2): сверху содержимое экрана
// (middle), снизу — «Tasky vX», а при наличии обновления под ней —
// «↑ доступно vY · tasky upgrade».
func rightRail(version, updateVer string, width, height int, middle string) string {
	innerW := max(width-4, 1)  // 2 пробела слева и справа
	innerH := max(height-2, 1) // 1 строка сверху и снизу
	versionLine := theme.HeaderStyle.Render("Tasky") + " " +
		theme.Faint("v"+strings.TrimPrefix(version, "v"))

	footer := make([]string, 0, 3)
	footer = append(footer, "")
	footer = append(footer, versionLine)
	if updateVer != "" {
		line := truncateW("↑ доступно "+updateVer+" · ctrl+p → Обновить", innerW)
		footer = append(footer, theme.AccentText(line))
	}
	if len(footer) > innerH {
		footer = footer[len(footer)-innerH:]
	}

	midH := innerH - len(footer)
	midLines := strings.Split(middle, "\n")
	lines := make([]string, 0, innerH)
	for i := 0; i < midH; i++ {
		if i < len(midLines) {
			lines = append(lines, midLines[i])
		} else {
			lines = append(lines, "")
		}
	}
	lines = append(lines, footer...)
	for len(lines) < innerH {
		lines = append(lines, "")
	}

	inner := strings.Join(lines, "\n")
	style := lipgloss.NewStyle().Padding(1, 2).Background(theme.PanelColor())
	return renderPane(style, padLines(inner, innerW, innerH))
}
