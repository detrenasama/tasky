package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/kalpamer/tasky/internal/db"
)

const dbPath = "tasky.db"

var (
	accent      = lipgloss.Color("212")
	headerStyle = lipgloss.NewStyle().Bold(true).Foreground(accent)
	faintStyle  = lipgloss.NewStyle().Faint(true)
	errorStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	boxStyle    = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1)
	focusBox    = boxStyle.Copy().BorderForeground(accent)
	dimBox      = boxStyle.Copy().Faint(true)
)

func faint(s string) string { return faintStyle.Render(s) }

type screen int

const (
	screenTasks screen = iota
	screenProjects
)

type model struct {
	db     *sql.DB
	screen screen
	width  int
	height int

	tasks *tasksScreen
	proj  *projectsScreen
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
		if m.height < 3 {
			m.height = 3
		}
		m.tasks.resize(m.width, m.height-2)
		m.proj.resize(m.width, m.height-2)
		return m, nil
	case tea.KeyMsg:
		if m.screen == screenProjects && m.proj.mode == projInput {
			return m.updateProjects(msg)
		}
		if m.screen == screenTasks && m.tasks.mode == taskInput {
			return m.updateTasks(msg)
		}
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}
		if m.screen == screenTasks {
			return m.updateTasks(msg)
		}
		return m.updateProjects(msg)
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

func (m model) View() string {
	w, h := m.width, m.height
	if h < 3 {
		h = 3
	}
	midH := h - 2

	var header, mid, footer string
	var dlg string
	var modalOpen bool
	if m.screen == screenProjects {
		header, footer = m.proj.header(w), m.proj.footer(w)
		mid = m.proj.view(w, midH)
		dlg, modalOpen = m.proj.dialog()
	} else {
		header, footer = m.tasks.header(w), m.tasks.footer(w)
		mid = m.tasks.view(w, midH)
		dlg, modalOpen = m.tasks.dialog()
	}

	full := header + "\n" + mid + "\n" + footer
	if modalOpen {
		full = overlay(full, dlg, w, h)
	}
	return full
}

func main() {
	conn, err := db.Open(dbPath)
	if err != nil {
		log.Fatalf("не удалось открыть базу данных: %v", err)
	}
	defer conn.Close()

	m := model{db: conn}
	m.tasks = newTasksScreen(conn)
	m.tasks.now = time.Now()
	m.tasks.load()
	m.proj = newProjectsScreen(conn)
	m.proj.load()

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "ошибка запуска: ", err)
		os.Exit(1)
	}
}
