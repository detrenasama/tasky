package main

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/kalpamer/tasky/internal/db"
)

func newTestTasksScreen(t *testing.T) *tasksScreen {
	t.Helper()
	conn, err := db.Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	s := newTasksScreen(conn)
	s.load()
	return s
}

func TestResizeColumns(t *testing.T) {
	s := newTestTasksScreen(t)
	cases := []struct{ w, list, desc, info int }{
		{150, 59, 58, 29},
		{110, 43, 42, 21},
		{109, 71, 0, 36},
		{90, 58, 0, 30},
		{69, 69, 0, 0},
	}
	for _, c := range cases {
		s.resize(c.w, 26)
		if s.listW != c.list || s.descW != c.desc || s.infoW != c.info {
			t.Errorf("w=%d: list=%d desc=%d info=%d, ожидались %d/%d/%d",
				c.w, s.listW, s.descW, s.infoW, c.list, c.desc, c.info)
		}
	}
}

func TestViewFillsWidth(t *testing.T) {
	s := newTestTasksScreen(t)
	for _, w := range []int{150, 90, 60} {
		s.resize(w, 26)
		out := s.view(w, 26)
		rows := strings.Split(out, "\n")
		if len(rows) != 26 {
			t.Errorf("w=%d: строк %d, ожидалось 26", w, len(rows))
		}
		for i, r := range rows {
			if lipgloss.Width(r) != w {
				t.Errorf("w=%d: строка %d шириной %d, ожидалось %d", w, i, lipgloss.Width(r), w)
			}
		}
	}
}

func TestViewColumnsWithTasks(t *testing.T) {
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
	s := newTasksScreen(conn)
	s.load()
	s.resize(150, 26)
	// три колонки: 59 + 2 + 58 + 2 + 29 = 150
	want := []struct {
		name string
		w    int
	}{
		{"list", s.listW}, {"desc", s.descW}, {"info", s.infoW},
	}
	row := []rune(stripANSI(strings.Split(s.view(150, 26), "\n")[0]))
	for _, c := range want {
		start := indexRune(row, '╭')
		if start < 0 {
			t.Fatalf("%s: не найден ╭", c.name)
		}
		end := indexRune(row, '╮')
		if end < 0 {
			t.Fatalf("%s: не найден ╮", c.name)
		}
		if got := end - start + 1; got != c.w {
			t.Errorf("%s: ширина %d, ожидалось %d", c.name, got, c.w)
		}
		t.Logf("%s: %d", c.name, end-start+1)
		if end+2 < len(row) {
			row = row[end+2:] // пропустить рамку и разделитель
		}
	}
	if s.listW+s.descW+s.infoW+4 != 150 {
		t.Errorf("сумма колонок %d, ожидалось 150", s.listW+s.descW+s.infoW+4)
	}
}

func indexRune(runes []rune, r rune) int {
	for i, rr := range runes {
		if rr == r {
			return i
		}
	}
	return -1
}

func TestInfoBottomBorderVisible(t *testing.T) {
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
	now := time.Unix(1_700_000_000, 0)
	if err := db.StartSession(conn, st.ID, now); err != nil {
		t.Fatal(err)
	}
	if err := db.StopSession(conn, st.ID, now.Add(2*time.Second)); err != nil {
		t.Fatal(err)
	}
	s := newTasksScreen(conn)
	s.load()
	s.resize(150, 26)
	rows := strings.Split(s.view(150, 26), "\n")
	if len(rows) != 26 {
		t.Fatalf("строк %d, ожидалось 26", len(rows))
	}
	last := stripANSI(rows[25])
	if n := strings.Count(last, "╰"); n != 3 {
		t.Errorf("на последней строке %d нижних бордеров, ожидалось 3 (list/desc/info)", n)
	}
}
