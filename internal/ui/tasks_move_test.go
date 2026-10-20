package ui

import (
	"testing"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/detrenasama/tasky/internal/db"
	"github.com/detrenasama/tasky/internal/store"
)

// TestMoveTaskSkipsHidden — ctrl+up/down переставляет задачу на одну видимую
// позицию: скрытые (завершённые) соседи пропускаются.
func TestMoveTaskSkipsHidden(t *testing.T) {
	conn, err := db.Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	p, err := db.CreateProject(conn, "P")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	t1, err := db.CreateTask(conn, p.ID, "T1")
	if err != nil {
		t.Fatal(err)
	}
	hidden, err := db.CreateTask(conn, p.ID, "Скрытая")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.CreateTask(conn, p.ID, "T3"); err != nil {
		t.Fatal(err)
	}
	if err := db.SetStatus(conn, db.OwnerTask, hidden.ID, "Выполнена", "", now.Add(-8*24*time.Hour)); err != nil {
		t.Fatal(err)
	}

	s := newTasksScreen(store.NewSQLite(conn))
	s.now = now
	s.load()

	if got := searchTitles(s); !equalStrings(got, []string{"T1", "T3"}) {
		t.Fatalf("видимый список: %v", got)
	}

	// вниз через скрытую задачу
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyCtrlDown})
	if got := searchTitles(s); !equalStrings(got, []string{"T3", "T1"}) {
		t.Fatalf("после ctrl+down: %v", got)
	}
	if kind, id := s.selectedKindID(); kind != kindTask || id != t1.ID {
		t.Fatalf("выделение не сохранено: kind=%d id=%d", kind, id)
	}

	// вверх возвращает на место
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyCtrlUp})
	if got := searchTitles(s); !equalStrings(got, []string{"T1", "T3"}) {
		t.Fatalf("после ctrl+up: %v", got)
	}
}

// TestMoveTaskAllHiddenNoop — если в направлении нет видимого соседа, порядок
// не меняется.
func TestMoveTaskAllHiddenNoop(t *testing.T) {
	conn, err := db.Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	p, err := db.CreateProject(conn, "P")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	upper, err := db.CreateTask(conn, p.ID, "Верхняя скрытая")
	if err != nil {
		t.Fatal(err)
	}
	mid, err := db.CreateTask(conn, p.ID, "Видимая")
	if err != nil {
		t.Fatal(err)
	}
	lower, err := db.CreateTask(conn, p.ID, "Нижняя скрытая")
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []int64{upper.ID, lower.ID} {
		if err := db.SetStatus(conn, db.OwnerTask, id, "Выполнена", "", now.Add(-8*24*time.Hour)); err != nil {
			t.Fatal(err)
		}
	}
	s := newTasksScreen(store.NewSQLite(conn))
	s.now = now
	s.load()

	order := func() []int64 {
		tasks, err := db.TasksByProject(conn, p.ID)
		if err != nil {
			t.Fatal(err)
		}
		ids := make([]int64, 0, len(tasks))
		for _, tk := range tasks {
			ids = append(ids, tk.ID)
		}
		return ids
	}
	want := []int64{upper.ID, mid.ID, lower.ID}

	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyCtrlUp})
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyCtrlDown})
	if got := order(); len(got) != len(want) || got[0] != want[0] || got[1] != want[1] || got[2] != want[2] {
		t.Fatalf("порядок изменился: %v, ожидался %v", got, want)
	}
}

// TestMoveSubtaskSkipsHidden — то же правило для подзадач.
func TestMoveSubtaskSkipsHidden(t *testing.T) {
	conn, err := db.Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	p, err := db.CreateProject(conn, "P")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	task, err := db.CreateTask(conn, p.ID, "T")
	if err != nil {
		t.Fatal(err)
	}
	s1, err := db.CreateSubtask(conn, task.ID, "S1")
	if err != nil {
		t.Fatal(err)
	}
	hidden, err := db.CreateSubtask(conn, task.ID, "Скрытая подзадача")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.CreateSubtask(conn, task.ID, "S3"); err != nil {
		t.Fatal(err)
	}
	if err := db.SetStatus(conn, db.OwnerSubtask, hidden.ID, "Выполнена", "", now.Add(-30*24*time.Hour)); err != nil {
		t.Fatal(err)
	}

	s := newTasksScreen(store.NewSQLite(conn))
	s.now = now
	s.load()
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyRight}) // раскрыть T
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyDown})  // на S1

	if got := searchTitles(s); !equalStrings(got, []string{"T", "S1", "S3"}) {
		t.Fatalf("видимый список: %v", got)
	}

	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyCtrlDown})
	if got := searchTitles(s); !equalStrings(got, []string{"T", "S3", "S1"}) {
		t.Fatalf("после ctrl+down: %v", got)
	}
	if kind, id := s.selectedKindID(); kind != kindSubtask || id != s1.ID {
		t.Fatalf("выделение не сохранено: kind=%d id=%d", kind, id)
	}

	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyCtrlUp})
	if got := searchTitles(s); !equalStrings(got, []string{"T", "S1", "S3"}) {
		t.Fatalf("после ctrl+up: %v", got)
	}
}

// TestMoveBlockedDuringSearch — при активном поиске перемещение недоступно.
func TestMoveBlockedDuringSearch(t *testing.T) {
	conn, err := db.Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	p, err := db.CreateProject(conn, "P")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.CreateTask(conn, p.ID, "Alpha"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.CreateTask(conn, p.ID, "Beta"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.CreateTask(conn, p.ID, "Alpha два"); err != nil {
		t.Fatal(err)
	}
	s := newTasksScreen(store.NewSQLite(conn))
	s.load()

	s.searchQuery = "alpha"
	s.buildItems()
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyCtrlDown})
	s.moveSelected(1)

	tasks, err := db.TasksByProject(conn, p.ID)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"Alpha", "Beta", "Alpha два"}
	for i, tk := range tasks {
		if tk.Title != want[i] {
			t.Fatalf("порядок изменился при поиске: %v", tasks)
		}
	}
}

// TestPaletteHidesMoveDuringSearch — при активном поиске команды перемещения
// не предлагаются в палитре.
func TestPaletteHidesMoveDuringSearch(t *testing.T) {
	s := newTestTasksScreen(t)
	m := &model{screen: screenTasks, tasks: s}
	hasMove := func() bool {
		for _, g := range m.paletteGroups() {
			for _, c := range g.cmds {
				if c.keys == "ctrl+up" || c.keys == "ctrl+down" {
					return true
				}
			}
		}
		return false
	}
	if !hasMove() {
		t.Fatal("без поиска команды перемещения отсутствуют")
	}
	s.searchQuery = "x"
	if hasMove() {
		t.Error("во время поиска команды перемещения всё ещё в палитре")
	}
}
