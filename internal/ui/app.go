package ui

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbletea"

	"github.com/kalpamer/tasky/internal/db"
	"github.com/kalpamer/tasky/internal/ui/theme"
)

type screen int

const (
	screenTasks screen = iota
	screenProjects
	screenReports
	screenSettings
)

type tab struct {
	title string
	key   string
	scr   screen
}

var tabs = []tab{
	{"Задачи", "t", screenTasks},
	{"Проекты", "p", screenProjects},
	{"Отчеты", "r", screenReports},
	{"Настройки", "s", screenSettings},
}

// tabsLine — строка вкладок страниц: текущая выделена цветом, остальные dim.
func tabsLine(cur screen, w int) string {
	parts := make([]string, 0, len(tabs))
	for _, t := range tabs {
		label := t.title + " <" + t.key + ">"
		if t.scr == cur {
			parts = append(parts, theme.HeaderStyle.Render(label))
		} else {
			parts = append(parts, theme.Faint(label))
		}
	}
	return padW(strings.Join(parts, "  "), w)
}

// model — корневая модель приложения: переключение экранов (вкладки),
// тик 1с, хром (шапка/подвал/вкладки) и общие диалоги выхода/отчётов.
type model struct {
	db     *sql.DB
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
}

// New создаёт корневую модель: экраны, общий конфиг отчётов, первичная
// загрузка данных.
func New(conn *sql.DB) *model {
	m := &model{db: conn}
	m.tasks = newTasksScreen(conn)
	m.tasks.now = time.Now()
	m.tasks.load()
	m.proj = newProjectsScreen(conn)
	m.proj.load()
	repCfg := &reportConfig{period: periodToday, saveDir: "reports"}
	m.reports = newReportsScreen(conn, repCfg)
	m.reports.load()
	m.settings = newSettingsScreen(conn, repCfg)
	return m
}

type tickMsg time.Time

func tickCmd(t time.Time) tea.Msg { return tickMsg(t) }

func (m model) Init() tea.Cmd {
	return tea.Tick(time.Second, tickCmd)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		if m.height < 4 {
			m.height = 4
		}
		m.tasks.resize(m.width, m.height-3)
		m.proj.resize(m.width, m.height-3)
		m.reports.resize(m.width, m.height-3)
		m.settings.resize(m.width, m.height-3)
		return m, nil
	case tea.KeyMsg:
		if m.quitting {
			switch msg.String() {
			case "y", "enter":
				if run, err := db.RunningSession(m.db); err == nil && run != nil {
					db.StopSession(m.db, run.ID, time.Now())
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
				if run, err := db.RunningSession(m.db); err == nil && run != nil {
					db.StopSession(m.db, run.ID, time.Now())
				}
				m.reportConfirm = false
				m.switchScreen(screenReports)
				return m, nil
			case "n", "esc", "r", "q", "ctrl+c":
				m.reportConfirm = false
			}
			return m, nil
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
				m.tasks.mode == taskTagConfirm) {
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
			if run, err := db.RunningSession(m.db); err == nil && run != nil {
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
			case "t":
				m.switchScreen(screenTasks)
				return m, nil
			case "p":
				m.switchScreen(screenProjects)
				return m, nil
			case "r":
				if run, err := db.RunningSession(m.db); err == nil && run != nil {
					m.reportConfirm = true
					m.reportTitle = run.Title
					return m, nil
				}
				m.switchScreen(screenReports)
				return m, nil
			case "s":
				m.switchScreen(screenSettings)
				return m, nil
			case "esc":
				if m.screen != screenTasks {
					m.switchScreen(screenTasks)
					return m, nil
				}
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
			m.tasks.today, _ = db.TodayTotal(m.db, now)
			m.tasks.weekly, _ = db.WeeklyTotal(m.db, now)
		}
		return m, tea.Tick(time.Second, tickCmd)
	}
	return m, nil
}

func (m *model) modalOpen() bool {
	if m.settings != nil && m.settings.mode != settingsBrowse {
		return true
	}
	return m.tasks.mode != taskBrowse || m.proj.mode != projBrowse
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
	w, h := m.width, m.height
	if h < 4 {
		h = 4
	}
	midH := h - 3

	var header, mid, footer string
	var dlg string
	var modalOpen bool
	switch m.screen {
	case screenProjects:
		header, footer = m.proj.header(w), m.proj.footer(w)
		mid = m.proj.view(w, midH)
		dlg, modalOpen = m.proj.dialog()
	case screenReports:
		header, footer = m.reports.header(w), m.reports.footer(w)
		mid = m.reports.view(w, midH)
		dlg, modalOpen = m.reports.dialog()
	case screenSettings:
		header, footer = m.settings.header(w), m.settings.footer(w)
		mid = m.settings.view(w, midH)
		dlg, modalOpen = m.settings.dialog()
	default:
		header, footer = m.tasks.header(w), m.tasks.footer(w)
		mid = m.tasks.view(w, midH)
		dlg, modalOpen = m.tasks.dialog()
	}

	full := tabsLine(m.screen, w) + "\n" + header + "\n" + mid + "\n" + footer
	if modalOpen {
		full = overlay(full, dlg, w, h)
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
