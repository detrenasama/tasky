package ui

import (
	"strconv"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbletea"
	"github.com/detrenasama/tasky/internal/db"
	"github.com/detrenasama/tasky/internal/store"
)

// scrollInvariants проверяет плавную прокрутку: курсор видим в окне и не
// прилипает к краю (удерживается в отступе ~listScrollOff, кроме начала/конца
// списка, где докрутить дальше нельзя).
func scrollInvariants(t *testing.T, s *tasksScreen) {
	t.Helper()
	step := listStep(s.listDelegate)
	visible := s.listH
	if visible < 1 {
		visible = 1
	}
	off := listScrollOff * step
	n := len(s.items)
	totalRows := n*step - s.listDelegate.Spacing()
	if totalRows < 0 {
		totalRows = 0
	}
	maxTop := totalRows - visible
	if maxTop < 0 {
		maxTop = 0
	}
	idxRow := s.list.Index() * step
	topMargin := idxRow - s.listTop
	botMargin := (s.listTop + visible - 1) - idxRow
	// в середине списка курсор держим в отступе ~scrolloff от краёв; у самого
	// начала/конца докрутить дальше нельзя — там меньший отступ допустим.
	minMargin := off - 1
	if s.listTop > 0 && topMargin < minMargin {
		t.Errorf("курсор слишком близко к верху: idxRow=%d top=%d off=%d", idxRow, s.listTop, off)
	}
	if s.listTop < maxTop && botMargin < minMargin {
		t.Errorf("курсор слишком близко к низу: idxRow=%d top=%d maxTop=%d off=%d", idxRow, s.listTop, maxTop, off)
	}
	if s.listTop > 0 && idxRow > s.listTop+visible-1 {
		t.Errorf("курсор ниже окна: idxRow=%d top=%d visible=%d", idxRow, s.listTop, visible)
	}
	if len(s.items) > 0 {
		sel := s.items[s.list.Index()]
		var title string
		switch it := sel.(type) {
		case taskItem:
			title = it.t.Title
		}
		if title != "" && !strings.Contains(s.listView(), title) {
			t.Errorf("выбранная задача %q не видна в окне списка", title)
		}
	}
}

func TestTasksListScroll(t *testing.T) {
	conn, err := db.Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	p, err := db.CreateProject(conn, "P")
	if err != nil {
		t.Fatal(err)
	}
	const n = 30
	for i := 0; i < n; i++ {
		if _, err := db.CreateTask(conn, p.ID, "задача-"+strconv.Itoa(i)); err != nil {
			t.Fatal(err)
		}
	}
	s := newTasksScreen(store.NewSQLite(conn))
	s.load()
	s.resize(150, 26)

	if s.list.Index() != 0 || s.listTop != 0 {
		t.Fatalf("начальное состояние: index=%d top=%d", s.list.Index(), s.listTop)
	}
	step := listStep(s.listDelegate)

	// прокручиваем по одному — рывков на целую страницу быть не должно
	prevIdx, prevTop := s.list.Index(), s.listTop
	for i := 0; i < n-1; i++ {
		s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyDown})
		if s.list.Index() != prevIdx+1 {
			t.Fatalf("шаг %d: index %d, ожидался %d (не по одному!)", i, s.list.Index(), prevIdx+1)
		}
		if delta := s.listTop - prevTop; delta > step || delta < -step {
			t.Fatalf("шаг %d: скачок прокрутки listTop на %d (больше одного элемента %d)", i, delta, step)
		}
		scrollInvariants(t, s)
		prevIdx, prevTop = s.list.Index(), s.listTop
	}
	if s.list.Index() != n-1 {
		t.Fatalf("после прокрутки index=%d, ожидалось %d", s.list.Index(), n-1)
	}

	// Home — в начало
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyHome})
	if s.list.Index() != 0 || s.listTop != 0 {
		t.Errorf("Home: index=%d top=%d", s.list.Index(), s.listTop)
	}
	scrollInvariants(t, s)

	// End — в конец, верх окна прижат к максимуму
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyEnd})
	if s.list.Index() != n-1 {
		t.Errorf("End: index=%d", s.list.Index())
	}
	scrollInvariants(t, s)

	// в середине списка курсор не прилипает к краю окна
	s.list.Select(n / 2)
	s.syncScroll()
	idxRow := s.list.Index() * step
	if s.listTop > 0 && idxRow-s.listTop < 1 {
		t.Errorf("середина: курсор прилип к верху окна (top=%d idxRow=%d)", s.listTop, idxRow)
	}
	scrollInvariants(t, s)
}

func TestProjectsListScroll(t *testing.T) {
	conn, err := db.Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	const n = 25
	for i := 0; i < n; i++ {
		if _, err := db.CreateProject(conn, "проект-"+strconv.Itoa(i)); err != nil {
			t.Fatal(err)
		}
	}
	s := newProjectsScreen(store.NewSQLite(conn))
	s.load()
	s.resize(150, 26)
	step := listStep(s.listDelegate)
	visible := s.listH
	off := listScrollOff * step

	prevIdx, prevTop := s.list.Index(), s.listTop
	for i := 0; i < n-1; i++ {
		s.updateProjectsMsg(tea.KeyMsg{Type: tea.KeyDown})
		if s.list.Index() != prevIdx+1 {
			t.Fatalf("шаг %d: index %d, ожидался %d", i, s.list.Index(), prevIdx+1)
		}
		if delta := s.listTop - prevTop; delta > step || delta < -step {
			t.Fatalf("шаг %d: скачок прокрутки listTop на %d", i, delta)
		}
		idxRow := s.list.Index() * step
		topMargin := idxRow - s.listTop
		botMargin := (s.listTop + visible - 1) - idxRow
		minMargin := off - 1
		if s.listTop > 0 && topMargin < minMargin {
			t.Fatalf("задача %d: курсор у верха top=%d idxRow=%d off=%d", i, s.listTop, idxRow, off)
		}
		if s.listTop < (n*step-s.listDelegate.Spacing())-visible && botMargin < minMargin {
			t.Fatalf("задача %d: курсор у низа top=%d idxRow=%d off=%d", i, s.listTop, idxRow, off)
		}
		prevIdx, prevTop = s.list.Index(), s.listTop
	}
}
