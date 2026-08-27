package ui

import (
	"database/sql"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbletea"
	"github.com/detrenasama/tasky/internal/db"
	"github.com/detrenasama/tasky/internal/store"
)

func newTestTasksScreen(t *testing.T) *tasksScreen {
	t.Helper()
	conn, err := db.Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	s := newTasksScreen(store.NewSQLite(conn))
	s.load()
	return s
}

// updateTasksMsg прогоняет клавишу через updateTasks тестового экрана.

func (s *tasksScreen) updateTasksMsg(msg tea.KeyMsg) {
	(&model{tasks: s}).updateTasks(msg)
}

func (s *projectsScreen) updateProjectsMsg(msg tea.KeyMsg) {
	(&model{proj: s}).updateProjects(msg)
}

func indexRune(runes []rune, r rune) int {
	for i, rr := range runes {
		if rr == r {
			return i
		}
	}
	return -1
}

func tasksSeedProject(t *testing.T) (*sql.DB, *tasksScreen, db.Task, db.SubtaskWithTime) {
	t.Helper()
	conn, err := db.Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	p, err := db.CreateProject(conn, "P")
	if err != nil {
		t.Fatal(err)
	}
	task, err := db.CreateTask(conn, p.ID, "T")
	if err != nil {
		t.Fatal(err)
	}
	st, err := db.CreateSubtask(conn, task.ID, "S")
	if err != nil {
		t.Fatal(err)
	}
	s := newTasksScreen(store.NewSQLite(conn))
	s.load()
	s.resize(150, 26)
	return conn, s, task, st
}

// selectFirstSubtask раскрывает первую задачу (→) и переводит курсор на
// первую подзадачу.

func selectFirstSubtask(m *model) {
	m.updateTasks(tea.KeyMsg{Type: tea.KeyRight})
	m.updateTasks(tea.KeyMsg{Type: tea.KeyDown})
}

func searchTitles(s *tasksScreen) []string {
	var out []string
	for _, item := range s.items {
		switch it := item.(type) {
		case taskItem:
			out = append(out, it.t.Title)
		case subItem:
			out = append(out, it.st.Title)
		}
	}
	return out
}

// projectTitles собирает названия элементов списка проектов.

func projectTitles(s *projectsScreen) []string {
	var out []string
	for _, item := range s.items {
		if it, ok := item.(projectItem); ok {
			out = append(out, it.p.Name)
		}
	}
	return out
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// TestProjectSearch — / открывает модалку поиска проектов, живой фильтр по
// названию/описанию/ссылкам, Enter применяет, Esc отменяет, esc в браузе
// (через main.go) сбрасывает и не уводит на экран задач.

func reportsSeedProject(t *testing.T) (*sql.DB, db.Task, db.SubtaskWithTime) {
	t.Helper()
	conn, err := db.Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	p, err := db.CreateProject(conn, "P")
	if err != nil {
		t.Fatal(err)
	}
	task, err := db.CreateTask(conn, p.ID, "T")
	if err != nil {
		t.Fatal(err)
	}
	st, err := db.CreateSubtask(conn, task.ID, "S")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	if err := db.StartSession(conn, st.ID, now.Add(-2*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := db.StopSession(conn, st.ID, now.Add(-1*time.Hour)); err != nil {
		t.Fatal(err)
	}
	return conn, task, st
}

// newReportsModel собирает полную модель с отчётами и настройками для
// тестов клавиатуры.

func newReportsModel(conn *sql.DB) model {
	st := store.NewSQLite(conn)
	m := model{store: st, screen: screenTasks}
	m.paletteInput = textinput.New()
	m.tasks = newTasksScreen(st)
	m.proj = newProjectsScreen(st)
	repCfg := &reportConfig{period: periodToday, saveDir: "reports"}
	m.reports = newReportsScreen(st, repCfg)
	m.settings = newSettingsScreen(st, repCfg)
	m.tasks.load()
	m.proj.load()
	m.reports.load()
	m.tasks.resize(150, 27)
	m.proj.resize(150, 27)
	m.reports.resize(150, 27)
	m.width, m.height = 150, 30
	return m
}

// paletteNav открывает палитру команд (Ctrl+P) и выполняет навигацию по
// сочетанию клавиш (напр. tea.KeyCtrlR → «Отчеты»), имитируя переход между
// страницами, который теперь доступен только из палитры. Возвращает tea.Model,
// совместимо с последующим m = mm.(model) в тестах.
func paletteNav(t *testing.T, m model, ctrlKey tea.KeyType) model {
	t.Helper()
	mm, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlP})
	m2, ok := mm.(model)
	if !ok || !m2.paletteOpen {
		t.Fatal("палитра не открылась по Ctrl+P")
	}
	mm2, _ := m2.Update(tea.KeyMsg{Type: ctrlKey})
	m3, ok := mm2.(model)
	if !ok {
		t.Fatal("навигация через палитру не вернула модель")
	}
	return m3
}

// upd — вспомогательная отправка KeyMsg модели в тестах: возвращает модель
// без лишних промежуточных переменных.
func upd(m model, msg tea.KeyMsg) model {
	mm, _ := m.Update(msg)
	return mm.(model)
}

// TestReportsScreenRender — отчёт за сегодня: переход по r, заголовок
// периода, задачи с подзадачами и общее время.
