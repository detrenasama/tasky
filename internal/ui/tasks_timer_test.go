package ui

import (
	"testing"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/detrenasama/tasky/internal/db"
)

// TestCtrlLTaskStopsRunning — ctrl+l при выделенной задаче останавливает
// текущую сессию (запускать на задаче нечего).
func TestCtrlLTaskStopsRunning(t *testing.T) {
	conn, s, _, st := tasksSeedProject(t)
	if err := db.StartSession(conn, st.ID, time.Now().Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	s.loadData()

	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyCtrlL})

	run, err := db.RunningSession(conn)
	if err != nil {
		t.Fatal(err)
	}
	if run != nil {
		t.Fatalf("сессия не остановлена: %+v", run)
	}
	entries, _ := db.TimeEntriesBySubtask(conn, st.ID)
	if len(entries) != 1 || entries[0].EndedAt == nil {
		t.Fatalf("запись времени: %+v", entries)
	}
}

// TestCtrlLTaskWithoutSession — без активной сессии ctrl+l на задаче ничего не
// делает и не создаёт записей.
func TestCtrlLTaskWithoutSession(t *testing.T) {
	conn, s, _, st := tasksSeedProject(t)

	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyCtrlL})

	run, _ := db.RunningSession(conn)
	if run != nil {
		t.Fatalf("неожиданно запущена сессия: %+v", run)
	}
	entries, _ := db.TimeEntriesBySubtask(conn, st.ID)
	if len(entries) != 0 {
		t.Fatalf("созданы записи времени: %+v", entries)
	}
}

// TestCtrlLSubtaskToggle — на подзадаче ctrl+l по-прежнему запускает и
// останавливает её сессию.
func TestCtrlLSubtaskToggle(t *testing.T) {
	conn, s, _, _ := tasksSeedProject(t)
	selectFirstSubtask(&model{tasks: s})

	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyCtrlL})
	if run, _ := db.RunningSession(conn); run == nil {
		t.Fatal("сессия не запущена")
	}

	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyCtrlL})
	if run, _ := db.RunningSession(conn); run != nil {
		t.Fatalf("сессия не остановлена: %+v", run)
	}
}

// TestCtrlKStopsAnySelection — ctrl+k останавливает текущую сессию, даже если
// выделена не та подзадача, которая идёт (здесь — задача).
func TestCtrlKStopsAnySelection(t *testing.T) {
	conn, s, _, st := tasksSeedProject(t)
	if err := db.StartSession(conn, st.ID, time.Now().Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	s.loadData()

	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyCtrlK})

	if run, _ := db.RunningSession(conn); run != nil {
		t.Fatalf("сессия не остановлена: %+v", run)
	}
}

// TestCtrlKWithoutSession — без активной сессии ctrl+k ничего не делает.
func TestCtrlKWithoutSession(t *testing.T) {
	conn, s, _, st := tasksSeedProject(t)

	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyCtrlK})

	if run, _ := db.RunningSession(conn); run != nil {
		t.Fatalf("неожиданно запущена сессия: %+v", run)
	}
	if entries, _ := db.TimeEntriesBySubtask(conn, st.ID); len(entries) != 0 {
		t.Fatalf("созданы записи времени: %+v", entries)
	}
}

// TestCtrlKInModalIgnored — в модалке поиска ctrl+k не останавливает сессию:
// клавиша уходит в поле ввода.
func TestCtrlKInModalIgnored(t *testing.T) {
	conn, s, _, st := tasksSeedProject(t)
	if err := db.StartSession(conn, st.ID, time.Now().Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	s.loadData()
	s.searchInput.Focus()
	s.mode = taskSearch

	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyCtrlK})

	if run, _ := db.RunningSession(conn); run == nil {
		t.Fatal("ctrl+k в модалке остановил сессию")
	}
}

// TestPaletteStopTimer — команда остановки есть в палитре экрана задач.
func TestPaletteStopTimer(t *testing.T) {
	_, s, _, _ := tasksSeedProject(t)
	m := &model{screen: screenTasks, tasks: s}
	for _, g := range m.paletteGroups() {
		for _, c := range g.cmds {
			if c.keys == "ctrl+k" {
				if c.title != "Остановить таймер" {
					t.Fatalf("название команды: %q", c.title)
				}
				return
			}
		}
	}
	t.Fatal("в палитре нет команды ctrl+k")
}
