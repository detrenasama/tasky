package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/detrenasama/tasky/internal/db"
	"github.com/detrenasama/tasky/internal/store"
	"github.com/detrenasama/tasky/internal/ui/theme"
	"github.com/muesli/termenv"
)

func TestSidebarView(t *testing.T) {
	lipgloss.SetColorProfile(termenv.ANSI256)
	const h = 30
	view := sidebarView(screenProjects, h)
	lines := strings.Split(view, "\n")
	if len(lines) != h {
		t.Errorf("строк %d, ожидалось %d", len(lines), h)
	}
	for i, l := range lines {
		if w := lipgloss.Width(l); w != 5 {
			t.Errorf("строка %d: видимая ширина %d, ожидалось 5", i, w)
		}
	}
	plain := stripANSI(view)
	for _, it := range sidebarItems {
		if !strings.Contains(plain, it.glyph) {
			t.Errorf("значок вкладки %v отсутствует в панели", it.scr)
		}
	}
	// активная вкладка (Проекты) залита фоном Selection, неактивные — Panel
	var activeGlyph string
	for _, it := range sidebarItems {
		if it.scr == screenProjects {
			activeGlyph = it.glyph
		}
	}
	activeLine := theme.SidebarActive().Render("  " + activeGlyph + "  ")
	if !strings.Contains(view, activeLine) {
		t.Error("активная вкладка не выделена (SidebarActive)")
	}
	inactiveLine := theme.SidebarInactive().Render("     ")
	if !strings.Contains(view, inactiveLine) {
		t.Error("панель не содержит фон неактивной вкладки (SidebarInactive)")
	}
}

func TestQuitConfirmRunningSession(t *testing.T) {
	conn, err := db.Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
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
	if err := db.StartSession(conn, st.ID, time.Now()); err != nil {
		t.Fatal(err)
	}
	m := model{store: store.NewSQLite(conn), tasks: newTasksScreen(store.NewSQLite(conn)), proj: newProjectsScreen(store.NewSQLite(conn)), screen: screenTasks}
	m.tasks.load()
	m.tasks.resize(150, 27)
	m.proj.resize(150, 27)
	m.width, m.height = 150, 30
	q := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}

	// q не выходит сразу, а показывает предупреждение
	mm, cmd := m.Update(q)
	m = mm.(model)
	if cmd != nil {
		t.Fatal("q с запущенной сессией не должен выходить сразу")
	}
	if !m.quitting || !strings.Contains(m.quitTitle, "S") {
		t.Errorf("предупреждение не выставлено: quitting=%v title=%q", m.quitting, m.quitTitle)
	}
	view := m.View()
	if !strings.Contains(view, "Остановить и выйти") {
		t.Error("в предупреждении нет вопроса об остановке")
	}

	// Esc отменяет, сессия продолжает идти
	mm, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = mm.(model)
	if m.quitting {
		t.Error("Esc не отменил предупреждение")
	}
	if run, _ := db.RunningSession(conn); run == nil {
		t.Error("отмена выхода остановила сессию")
	}

	// q → Enter: сессия останавливается, приложение выходит
	mm, _ = m.Update(q)
	m = mm.(model)
	if !m.quitting {
		t.Fatal("q не показал предупреждение повторно")
	}
	mm, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = mm.(model)
	if cmd == nil {
		t.Fatal("Enter не вернул команду выхода")
	}
	if run, _ := db.RunningSession(conn); run != nil {
		t.Error("выход не остановил сессию")
	}
}

// TestQuitNoSession — без запущенной сессии выход сразу, без предупреждения.

func TestQuitNoSession(t *testing.T) {
	conn, err := db.Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	p, err := db.CreateProject(conn, "P")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.CreateTask(conn, p.ID, "T"); err != nil {
		t.Fatal(err)
	}
	m := model{store: store.NewSQLite(conn), tasks: newTasksScreen(store.NewSQLite(conn)), proj: newProjectsScreen(store.NewSQLite(conn)), screen: screenTasks}
	m.tasks.load()

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Fatal("q без запущенной сессии должен выходить сразу")
	}
	if m.quitting {
		t.Error("предупреждение показано без запущенной сессии")
	}
}

// reportsSeedProject создаёт проект с задачей и подзадачей и закрытой
// записью времени за последний час (в пределах сегодняшнего дня).
