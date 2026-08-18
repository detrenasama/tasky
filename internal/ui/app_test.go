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

func TestTabsLine(t *testing.T) {
	lipgloss.SetColorProfile(termenv.ANSI256)
	line := tabsLine(screenProjects, 100)
	if got := lipgloss.Width(line); got != 100 {
		t.Errorf("видимая ширина %d, ожидалось 100", got)
	}
	plain := stripANSI(line)
	for _, tb := range tabs {
		label := tb.title + " <" + tb.key + ">"
		if !strings.Contains(plain, label) {
			t.Errorf("вкладка %q отсутствует в %q", label, plain)
		}
	}
	active := theme.HeaderStyle.Render("Проекты <p>")
	if !strings.Contains(line, active) {
		t.Error("текущая вкладка не выделена (HeaderStyle)")
	}
	if n := strings.Count(line, theme.Faint("Задачи <t>")); n != 1 {
		t.Errorf("muted-вкладка «Задачи»: %d, ожидалась 1", n)
	}
	if n := strings.Count(line, theme.Faint("Отчеты <r>")); n != 1 {
		t.Errorf("muted-вкладка «Отчеты»: %d, ожидалась 1", n)
	}
	if n := strings.Count(line, theme.Faint("Настройки <s>")); n != 1 {
		t.Errorf("muted-вкладка «Настройки»: %d, ожидалась 1", n)
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
