package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

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

// updateTasksMsg прогоняет клавишу через updateTasks тестового экрана.
func (s *tasksScreen) updateTasksMsg(msg tea.KeyMsg) {
	(&model{tasks: s}).updateTasks(msg)
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

func TestProjectsResizeColumns(t *testing.T) {
	conn, err := db.Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if _, err := db.CreateProject(conn, "P"); err != nil {
		t.Fatal(err)
	}
	s := newProjectsScreen(conn)
	s.load()
	cases := []struct{ w, list, desc, info int }{
		{150, 30, 87, 29},
		{110, 22, 63, 21},
		{109, 53, 54, 0},
		{60, 29, 29, 0},
		{59, 59, 0, 0},
	}
	for _, c := range cases {
		s.resize(c.w, 26)
		if s.listW != c.list || s.descW != c.desc || s.infoW != c.info {
			t.Errorf("w=%d: list=%d desc=%d info=%d, ожидались %d/%d/%d",
				c.w, s.listW, s.descW, s.infoW, c.list, c.desc, c.info)
		}
	}
}

func TestProjectsViewFillsWidth(t *testing.T) {
	conn, err := db.Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if _, err := db.CreateProject(conn, "P"); err != nil {
		t.Fatal(err)
	}
	s := newProjectsScreen(conn)
	s.load()
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

func TestWrapText(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"", ""},
		{"одно слово", "одно\nслово"},
		{"aaa bbb", "aaa bbb"},
		{"aaa bbb ccc", "aaa bbb\nccc"},
		{"оченьдлинноеслово", "оченьдл\nинноесл\nово"},
		{"aaa\nbbb", "aaa\nbbb"},
	}
	for _, c := range cases {
		got := wrapText(c.in, 7)
		if got != c.want {
			t.Errorf("wrapText(%q, 7) = %q, ожидалось %q", c.in, got, c.want)
		}
	}
}

func TestProjectsDescBox(t *testing.T) {
	conn, err := db.Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	p, err := db.CreateProject(conn, "P")
	if err != nil {
		t.Fatal(err)
	}
	if err := db.UpdateProjectDescription(conn, p.ID, "Описание проекта"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.CreateProjectLink(conn, p.ID, "Доки", "https://example.com"); err != nil {
		t.Fatal(err)
	}
	s := newProjectsScreen(conn)
	s.load()
	s.resize(150, 26)

	view := s.view(150, 26)
	plain := stripANSI(view)
	if !strings.Contains(plain, "Описание проекта") {
		t.Error("описание не отображается в колонке описания")
	}
	if !strings.Contains(plain, "Доки") {
		t.Error("название ссылки не отображается в колонке описания")
	}
	if strings.Contains(plain, "https://example.com") {
		t.Error("адрес ссылки не должен показываться в колонке, если есть название")
	}
	for i, r := range strings.Split(view, "\n") {
		if lipgloss.Width(r) != 150 {
			t.Errorf("строка %d шириной %d, ожидалось 150", i, lipgloss.Width(r))
		}
	}
}

func TestProjectsFocusTab(t *testing.T) {
	conn, err := db.Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if _, err := db.CreateProject(conn, "P"); err != nil {
		t.Fatal(err)
	}
	m := model{proj: newProjectsScreen(conn)}
	m.proj.load()
	m.proj.resize(150, 26)
	key := func(t tea.KeyType) tea.KeyMsg { return tea.KeyMsg{Type: t} }

	if m.proj.focus != projFocusList {
		t.Fatalf("начальный фокус %d, ожидался список", m.proj.focus)
	}
	m.updateProjects(key(tea.KeyTab))
	if m.proj.focus != projFocusDesc {
		t.Fatal("Tab не переключил фокус на описание")
	}
	m.updateProjects(key(tea.KeyTab))
	if m.proj.focus != projFocusList {
		t.Fatal("Tab не вернул фокус на список")
	}
}

func TestProjectsDescKeysOnlyInDescFocus(t *testing.T) {
	conn, err := db.Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	p, err := db.CreateProject(conn, "P")
	if err != nil {
		t.Fatal(err)
	}
	if err := db.UpdateProjectDescription(conn, p.ID, strings.Repeat("строка длинного описания ", 100)); err != nil {
		t.Fatal(err)
	}
	m := model{proj: newProjectsScreen(conn)}
	m.proj.load()
	m.proj.resize(150, 26)
	key := func(t tea.KeyType) tea.KeyMsg { return tea.KeyMsg{Type: t} }
	_ = key

	// e в фокусе списка: режим остаётся browse
	before := m.proj.mode
	m.updateProjects(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	if m.proj.mode != before {
		t.Error("e в фокусе списка не должен открывать редактирование")
	}

	// e в фокусе описания открывает редактирование
	m.updateProjects(tea.KeyMsg{Type: tea.KeyTab})
	m.updateProjects(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	if m.proj.mode != projDescEdit {
		t.Error("e в фокусе описания должен открыть редактирование")
	}

	// редактирование инлайн: колонка показывает textarea, модалки нет
	m.proj.descText.SetValue("текст в textarea")
	if !strings.Contains(m.proj.descBox(), "текст в textarea") {
		t.Error("при редактировании в колонке не виден textarea")
	}
	if _, open := m.proj.dialog(); open {
		t.Error("редактирование не должно открывать модалку")
	}

	// Ctrl+S сохраняет
	m.proj.descText.SetValue("новое описание")
	m.updateProjects(tea.KeyMsg{Type: tea.KeyCtrlS})
	if m.proj.mode != projBrowse {
		t.Error("Ctrl+S не закрыл редактирование")
	}
	got, _ := db.ProjectDescription(conn, p.ID)
	if got != "новое описание" {
		t.Errorf("описание в БД = %q, ожидалось «новое описание»", got)
	}

	// Esc отменяет без сохранения
	m.updateProjects(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	m.proj.descText.SetValue("не сохранять")
	m.updateProjects(tea.KeyMsg{Type: tea.KeyEsc})
	if m.proj.mode != projBrowse {
		t.Error("Esc не закрыл редактирование")
	}
	if got, _ := db.ProjectDescription(conn, p.ID); got != "новое описание" {
		t.Errorf("Esc сохранил изменения: %q", got)
	}

	// снова длинное описание, чтобы был скролл: e → SetValue → Ctrl+S
	m.updateProjects(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	m.proj.descText.SetValue(strings.Repeat("строка длинного описания ", 100))
	m.updateProjects(tea.KeyMsg{Type: tea.KeyCtrlS})

	// скролл: после Ctrl+S фокус остался на описании, down скроллит viewport
	y0 := m.proj.descV.YOffset
	m.updateProjects(tea.KeyMsg{Type: tea.KeyDown})
	if m.proj.descV.YOffset != y0+1 {
		t.Errorf("down не проскроллил описание: %d → %d", y0, m.proj.descV.YOffset)
	}
}

// TestProjectLinksEscDoesNotQuit — регрессия: bubbles/list по умолчанию
// привязывает Quit к esc и q; из модалки ссылок Esc должен закрывать модалку,
// а не выходить из приложения.
func TestProjectLinksEscDoesNotQuit(t *testing.T) {
	conn, err := db.Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	p, err := db.CreateProject(conn, "P")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.CreateProjectLink(conn, p.ID, "", "https://example.com"); err != nil {
		t.Fatal(err)
	}
	m := model{proj: newProjectsScreen(conn), tasks: newTasksScreen(conn), screen: screenProjects}
	m.proj.load()
	m.proj.resize(150, 26)

	m.updateProjects(tea.KeyMsg{Type: tea.KeyTab})
	m.updateProjects(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
	if m.proj.mode != projLinks {
		t.Fatalf("o не открыл модалку ссылок (mode=%d)", m.proj.mode)
	}

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd != nil {
		t.Error("Esc из модалки ссылок вернул команду (выход из приложения)")
	}
	if m.proj.mode != projBrowse {
		t.Error("Esc не закрыл модалку ссылок")
	}
}

// TestProjectLinkAddFlow — добавление ссылки одной модалкой: два инпута
// (название → адрес), переключение по Tab/Enter, Esc — отмена.
func TestProjectLinkAddFlow(t *testing.T) {
	conn, err := db.Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	p, err := db.CreateProject(conn, "P")
	if err != nil {
		t.Fatal(err)
	}
	m := model{proj: newProjectsScreen(conn)}
	m.proj.load()
	m.proj.resize(150, 26)
	runes := func(r rune) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}} }

	m.updateProjects(tea.KeyMsg{Type: tea.KeyTab})
	m.updateProjects(runes('l'))
	if m.proj.mode != projLinkInput {
		t.Fatalf("l не открыл модалку ссылки (mode=%d)", m.proj.mode)
	}
	if !m.proj.linkName.Focused() || m.proj.linkInput.Focused() {
		t.Error("фокус должен быть на названии")
	}

	// Enter на названии — переходит к адресу, не сохраняет
	m.proj.linkName.SetValue("Доки")
	m.updateProjects(tea.KeyMsg{Type: tea.KeyEnter})
	if !m.proj.linkInput.Focused() || m.proj.linkName.Focused() {
		t.Error("Enter не перевёл фокус на адрес")
	}
	if links, _ := db.ProjectLinks(conn, p.ID); len(links) != 0 {
		t.Fatalf("Enter на названии сохранил ссылку: %d", len(links))
	}

	// Tab обратно на название, потом снова на адрес
	m.updateProjects(tea.KeyMsg{Type: tea.KeyTab})
	if !m.proj.linkName.Focused() {
		t.Error("Tab не вернул фокус на название")
	}
	m.updateProjects(tea.KeyMsg{Type: tea.KeyTab})
	if !m.proj.linkInput.Focused() {
		t.Error("Tab не перевёл фокус на адрес")
	}

	// Enter на адресе сохраняет оба поля
	m.proj.linkInput.SetValue("https://example.com")
	m.updateProjects(tea.KeyMsg{Type: tea.KeyEnter})
	if m.proj.mode != projBrowse {
		t.Fatalf("Enter не сохранил ссылку (mode=%d)", m.proj.mode)
	}
	links, _ := db.ProjectLinks(conn, p.ID)
	if len(links) != 1 || links[0].Name != "Доки" || links[0].URL != "https://example.com" {
		t.Errorf("сохранённая ссылка = %+v", links)
	}

	// Esc отменяет
	m.updateProjects(runes('l'))
	m.proj.linkName.SetValue("Имя")
	m.proj.linkInput.SetValue("https://example.org")
	m.updateProjects(tea.KeyMsg{Type: tea.KeyEsc})
	if m.proj.mode != projBrowse {
		t.Fatalf("Esc не отменил ввод (mode=%d)", m.proj.mode)
	}
	if links, _ := db.ProjectLinks(conn, p.ID); len(links) != 1 {
		t.Errorf("отменённый ввод создал ссылку: %d", len(links))
	}

	// Enter на адресе с пустым URL — модалка закрывается, ссылка не создаётся
	m.updateProjects(runes('l'))
	m.updateProjects(tea.KeyMsg{Type: tea.KeyTab})
	m.updateProjects(tea.KeyMsg{Type: tea.KeyEnter})
	if m.proj.mode != projBrowse {
		t.Fatalf("Enter с пустым URL не закрыл модалку (mode=%d)", m.proj.mode)
	}
	if links, _ := db.ProjectLinks(conn, p.ID); len(links) != 1 {
		t.Errorf("пустой URL создал ссылку: %d", len(links))
	}
}

// TestProjectLinkDeleteConfirm — удаление ссылки из списка требует
// подтверждения: d → подтверждение, y удаляет, n/esc отменяет.
func TestProjectLinkDeleteConfirm(t *testing.T) {
	conn, err := db.Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	p, err := db.CreateProject(conn, "P")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.CreateProjectLink(conn, p.ID, "Доки", "https://example.com"); err != nil {
		t.Fatal(err)
	}
	m := model{proj: newProjectsScreen(conn)}
	m.proj.load()
	m.proj.resize(150, 26)

	m.updateProjects(tea.KeyMsg{Type: tea.KeyTab})
	m.updateProjects(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
	m.updateProjects(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	if m.proj.mode != projLinkConfirm {
		t.Fatalf("d не открыл подтверждение (mode=%d)", m.proj.mode)
	}
	if _, open := m.proj.dialog(); !open {
		t.Error("подтверждение удаления не рендерится как модалка")
	}
	if links, _ := db.ProjectLinks(conn, p.ID); len(links) != 1 {
		t.Fatalf("подтверждение удалило ссылку до y")
	}

	// n — отмена: возврат в список, ссылка на месте
	m.updateProjects(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	if m.proj.mode != projLinks {
		t.Fatalf("n не вернул в список ссылок (mode=%d)", m.proj.mode)
	}
	if links, _ := db.ProjectLinks(conn, p.ID); len(links) != 1 {
		t.Errorf("n удалил ссылку: %d", len(links))
	}

	// d → esc — тоже отмена
	m.updateProjects(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	m.updateProjects(tea.KeyMsg{Type: tea.KeyEsc})
	if m.proj.mode != projLinks {
		t.Fatalf("esc не вернул в список ссылок (mode=%d)", m.proj.mode)
	}

	// d → y — удаление, возврат в список
	m.updateProjects(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	m.updateProjects(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if m.proj.mode != projLinks {
		t.Fatalf("y не вернул в список ссылок (mode=%d)", m.proj.mode)
	}
	if links, _ := db.ProjectLinks(conn, p.ID); len(links) != 0 {
		t.Errorf("y не удалил ссылку: %d", len(links))
	}
}

// TestProjectsDescBoxTruncatesLongLink — длинное название ссылки в колонке
// обрезается многоточием.
func TestProjectsDescBoxTruncatesLongLink(t *testing.T) {
	conn, err := db.Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	p, err := db.CreateProject(conn, "P")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.CreateProjectLink(conn, p.ID, strings.Repeat("оченьдлинноеназвание", 10), "https://example.com"); err != nil {
		t.Fatal(err)
	}
	s := newProjectsScreen(conn)
	s.load()
	s.resize(150, 26)

	box := stripANSI(s.descBox())
	for _, line := range strings.Split(box, "\n") {
		if strings.Contains(line, "оченьдлинноеназвание") {
			if !strings.Contains(line, "…") {
				t.Errorf("длинная ссылка не обрезана многоточием: %q", line)
			}
			break
		}
	}
}

func TestTruncateWEnd(t *testing.T) {
	cases := []struct {
		in, want string
		w        int
	}{
		{"abc", "abc", 5},
		{"abcdef", "abcd…", 5},
		{"оченьдлинноеназвание", "оченьд…", 7},
		{"", "", 4},
		{"abc", "…", 1},
	}
	for _, c := range cases {
		got := truncateWEnd(c.in, c.w)
		if got != c.want {
			t.Errorf("truncateWEnd(%q, %d) = %q, ожидалось %q", c.in, c.w, got, c.want)
		}
	}
}

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
	if !strings.Contains(line, "38;5;212") {
		t.Error("текущая вкладка не выделена цветом")
	}
	if n := strings.Count(line, "\x1b[2m"); n != 3 {
		t.Errorf("dim-вкладок: %d, ожидалось 3", n)
	}
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

// tasksSeedProject создаёт проект с задачей и подзадачей и возвращает
// инициализированный tasksScreen.
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
	s := newTasksScreen(conn)
	s.load()
	s.resize(150, 26)
	return conn, s, task, st
}

// selectFirstSubtask раскрывает первую задачу и переводит курсор на первую
// подзадачу.
func selectFirstSubtask(m *model) {
	m.updateTasks(tea.KeyMsg{Type: tea.KeyEnter})
	m.updateTasks(tea.KeyMsg{Type: tea.KeyDown})
}

func TestTasksFocusTab(t *testing.T) {
	_, s, _, _ := tasksSeedProject(t)
	if s.focus != taskFocusList {
		t.Fatalf("начальный фокус %d, ожидался список", s.focus)
	}
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyTab})
	if s.focus != taskFocusDesc {
		t.Fatal("Tab не переключил фокус на описание")
	}
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyTab})
	if s.focus != taskFocusList {
		t.Fatal("Tab не вернул фокус на список")
	}
}

// TestTasksDescBox — колонка описания: для задачи описание и ссылки, для
// подзадачи — блоки «Описание» и «Журнал» с записью.
func TestTasksDescBox(t *testing.T) {
	conn, s, task, st := tasksSeedProject(t)
	if err := db.UpdateTaskDescription(conn, task.ID, "описание задачи"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.CreateTaskLink(conn, task.ID, "Доки", "https://example.com"); err != nil {
		t.Fatal(err)
	}
	if err := db.UpdateSubtaskDescription(conn, st.ID, "описание подзадачи"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.CreateSubtaskLink(conn, st.ID, "", "https://example.org"); err != nil {
		t.Fatal(err)
	}
	entry, err := db.CreateJournalEntry(conn, st.ID, "первая запись")
	if err != nil {
		t.Fatal(err)
	}
	s.load()

	// задача: описание + ссылка
	plain := stripANSI(s.descBox())
	if !strings.Contains(plain, "описание задачи") || !strings.Contains(plain, "Доки") {
		t.Errorf("в колонке задачи нет описания/ссылки: %q", plain)
	}
	if strings.Contains(plain, "Журнал") {
		t.Error("у задачи не должно быть журнала")
	}

	// подзадача: описание + ссылка + журнал
	m := &model{tasks: s}
	m.updateTasks(tea.KeyMsg{Type: tea.KeyEnter}) // раскрыть задачу
	m.updateTasks(tea.KeyMsg{Type: tea.KeyDown})  // на подзадачу
	plain = stripANSI(s.descBox())
	for _, want := range []string{"описание подзадачи", "https://example.org",
		"Журнал", entry.CreatedAt.Format("02.01.2006 15:04"), "первая запись"} {
		if !strings.Contains(plain, want) {
			t.Errorf("в колонке подзадачи нет %q: %q", want, plain)
		}
	}
	for i, r := range strings.Split(s.descBox(), "\n") {
		if lipgloss.Width(r) != s.descW {
			t.Errorf("строка %d колонки шириной %d, ожидалось %d", i, lipgloss.Width(r), s.descW)
		}
	}
}

// TestTasksDescKeysOnlyInDescFocus — e/ctrl+j/l/o работают только в фокусе
// описания; инлайн-редактирование сохраняется по Ctrl+S, отменяется по Esc.
func TestTasksDescKeysOnlyInDescFocus(t *testing.T) {
	conn, s, task, _ := tasksSeedProject(t)
	m := &model{tasks: s, proj: newProjectsScreen(conn)}
	runes := func(r rune) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}} }

	// e в фокусе списка — ничего не открывает
	m.updateTasks(runes('e'))
	if s.mode != taskBrowse {
		t.Fatal("e в фокусе списка открыл редактирование")
	}
	// ctrl+j в фокусе списка — ничего не открывает
	before := s.mode
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlJ})
	if cmd != nil || s.mode != before {
		t.Error("ctrl+j в фокусе списка открыл модалку или вернул команду")
	}

	// e в фокусе описания открывает инлайн-редактирование
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyTab})
	s.updateTasksMsg(runes('e'))
	if s.mode != taskDescEdit {
		t.Fatal("e в фокусе описания не открыл редактирование")
	}
	s.descText.SetValue("новое описание")
	if !strings.Contains(s.descBox(), "новое описание") {
		t.Error("при редактировании в колонке не виден textarea")
	}
	if _, open := s.dialog(); open {
		t.Error("редактирование описания не должно открывать модалку")
	}
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyCtrlS})
	if s.mode != taskBrowse {
		t.Fatal("Ctrl+S не закрыл редактирование")
	}
	if got, _ := db.TaskDescription(conn, task.ID); got != "новое описание" {
		t.Errorf("описание в БД = %q", got)
	}

	// Esc отменяет
	s.updateTasksMsg(runes('e'))
	s.descText.SetValue("не сохранять")
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyEsc})
	if got, _ := db.TaskDescription(conn, task.ID); got != "новое описание" {
		t.Errorf("Esc сохранил изменения: %q", got)
	}

	// скролл viewport в фокусе описания
	s.updateTasksMsg(runes('e'))
	s.descText.SetValue(strings.Repeat("строка длинного описания ", 100))
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyCtrlS})
	y0 := s.descV.YOffset
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyDown})
	if s.descV.YOffset != y0+1 {
		t.Errorf("down не проскроллил описание: %d → %d", y0, s.descV.YOffset)
	}
}

// TestTasksLinkAddFlow — добавление ссылки задачи: одна модалка с двумя
// инпутами (название → адрес), Tab/Enter, Esc — отмена.
func TestTasksLinkAddFlow(t *testing.T) {
	conn, s, task, _ := tasksSeedProject(t)
	runes := func(r rune) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}} }

	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyTab})
	s.updateTasksMsg(runes('l'))
	if s.mode != taskLinkInput {
		t.Fatalf("l не открыл модалку ссылки (mode=%d)", s.mode)
	}
	if !s.linkName.Focused() || s.linkInput.Focused() {
		t.Error("фокус должен быть на названии")
	}

	s.linkName.SetValue("Доки")
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyEnter})
	if !s.linkInput.Focused() || s.linkName.Focused() {
		t.Error("Enter не перевёл фокус на адрес")
	}
	if links, _ := db.TaskLinks(conn, task.ID); len(links) != 0 {
		t.Fatalf("Enter на названии сохранил ссылку: %d", len(links))
	}

	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyTab})
	if !s.linkName.Focused() {
		t.Error("Tab не вернул фокус на название")
	}
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyTab})
	if !s.linkInput.Focused() {
		t.Error("Tab не перевёл фокус на адрес")
	}

	s.linkInput.SetValue("https://example.com")
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyEnter})
	if s.mode != taskBrowse {
		t.Fatalf("Enter не сохранил ссылку (mode=%d)", s.mode)
	}
	links, _ := db.TaskLinks(conn, task.ID)
	if len(links) != 1 || links[0].Name != "Доки" || links[0].URL != "https://example.com" {
		t.Errorf("сохранённая ссылка = %+v", links)
	}
	if !strings.Contains(stripANSI(s.descBox()), "Доки") {
		t.Error("ссылка не отобразилась в колонке описания")
	}

	// Esc отменяет
	s.updateTasksMsg(runes('l'))
	s.linkInput.SetValue("https://example.org")
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyEsc})
	if s.mode != taskBrowse {
		t.Fatalf("Esc не отменил ввод (mode=%d)", s.mode)
	}
	if links, _ := db.TaskLinks(conn, task.ID); len(links) != 1 {
		t.Errorf("отменённый ввод создал ссылку: %d", len(links))
	}

	// пустой URL закрывает модалку без создания
	s.updateTasksMsg(runes('l'))
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyTab})
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyEnter})
	if s.mode != taskBrowse {
		t.Fatalf("Enter с пустым URL не закрыл модалку (mode=%d)", s.mode)
	}
	if links, _ := db.TaskLinks(conn, task.ID); len(links) != 1 {
		t.Errorf("пустой URL создал ссылку: %d", len(links))
	}
}

// TestTasksLinkDeleteConfirm — удаление ссылки задачи с подтверждением.
func TestTasksLinkDeleteConfirm(t *testing.T) {
	conn, s, task, _ := tasksSeedProject(t)
	if _, err := db.CreateTaskLink(conn, task.ID, "Доки", "https://example.com"); err != nil {
		t.Fatal(err)
	}
	s.loadDesc()
	runes := func(r rune) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}} }

	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyTab})
	s.updateTasksMsg(runes('o'))
	if s.mode != taskLinks {
		t.Fatalf("o не открыл список ссылок (mode=%d)", s.mode)
	}
	s.updateTasksMsg(runes('d'))
	if s.mode != taskLinkConfirm {
		t.Fatalf("d не открыл подтверждение (mode=%d)", s.mode)
	}
	if _, open := s.dialog(); !open {
		t.Error("подтверждение не рендерится как модалка")
	}

	s.updateTasksMsg(runes('n'))
	if s.mode != taskLinks {
		t.Fatalf("n не вернул в список ссылок (mode=%d)", s.mode)
	}
	s.updateTasksMsg(runes('d'))
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyEsc})
	if s.mode != taskLinks {
		t.Fatalf("esc не вернул в список ссылок (mode=%d)", s.mode)
	}
	if links, _ := db.TaskLinks(conn, task.ID); len(links) != 1 {
		t.Errorf("отменённое удаление удалило ссылку: %d", len(links))
	}

	s.updateTasksMsg(runes('d'))
	s.updateTasksMsg(runes('y'))
	if s.mode != taskLinks {
		t.Fatalf("y не вернул в список ссылок (mode=%d)", s.mode)
	}
	if links, _ := db.TaskLinks(conn, task.ID); len(links) != 0 {
		t.Errorf("y не удалил ссылку: %d", len(links))
	}
}

// TestTaskJournalAddAndEdit — Ctrl+J добавляет запись (Ctrl+S сохраняет,
// Esc отменяет), j редактирует самую свежую запись текущего дня.
func TestTaskJournalAddAndEdit(t *testing.T) {
	conn, s, _, st := tasksSeedProject(t)
	m := &model{tasks: s}
	selectFirstSubtask(m)
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyTab})

	// Ctrl+J в фокусе описания на подзадаче открывает модалку записи
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyCtrlJ})
	if s.mode != taskJournal {
		t.Fatalf("ctrl+j не открыл журнал (mode=%d)", s.mode)
	}
	if _, open := s.dialog(); !open {
		t.Error("запись журнала не рендерится как модалка")
	}

	// Esc отменяет
	s.journalText.SetValue("не сохранять")
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyEsc})
	if s.mode != taskBrowse {
		t.Fatalf("Esc не отменил запись (mode=%d)", s.mode)
	}
	if entries, _ := db.JournalEntries(conn, st.ID); len(entries) != 0 {
		t.Errorf("отменённая запись создалась: %d", len(entries))
	}

	// Ctrl+J → Ctrl+S сохраняет
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyCtrlJ})
	s.journalText.SetValue("работал над задачей")
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyCtrlS})
	if s.mode != taskBrowse {
		t.Fatalf("Ctrl+S не сохранил запись (mode=%d)", s.mode)
	}
	entries, _ := db.JournalEntries(conn, st.ID)
	if len(entries) != 1 || entries[0].Text != "работал над задачей" {
		t.Fatalf("записи = %+v", entries)
	}
	plain := stripANSI(s.descBox())
	if !strings.Contains(plain, "работал над задачей") {
		t.Error("запись не отобразилась в колонке описания")
	}
	if !strings.Contains(plain, entries[0].CreatedAt.Format("02.01.2006 15:04")) {
		t.Error("в колонке нет даты/времени записи")
	}

	// j — редактирование свежей записи текущего дня
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if s.mode != taskJournal || s.journalEditID != entries[0].ID {
		t.Fatalf("j не открыл редактирование записи (mode=%d, id=%d)", s.mode, s.journalEditID)
	}
	if s.journalText.Value() != "работал над задачей" {
		t.Errorf("textarea не заполнен текстом записи: %q", s.journalText.Value())
	}
	s.journalText.SetValue("доработал")
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyCtrlS})
	entries, _ = db.JournalEntries(conn, st.ID)
	if len(entries) != 1 || entries[0].Text != "доработал" {
		t.Errorf("запись не обновилась: %+v", entries)
	}
}

// TestTaskJournalEditOnlyToday — редактировать можно только записи текущего
// дня: для вчерашней записи j показывает ошибку.
func TestTaskJournalEditOnlyToday(t *testing.T) {
	conn, s, _, st := tasksSeedProject(t)
	m := &model{tasks: s}
	if _, err := conn.Exec(
		"INSERT INTO journal_entries (subtask_id, created_at, text) VALUES (?, ?, 'вчера')",
		st.ID, time.Now().Add(-24*time.Hour).Unix()); err != nil {
		t.Fatal(err)
	}
	s.loadDesc()
	selectFirstSubtask(m)
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyTab})

	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if s.mode != taskBrowse {
		t.Fatalf("j открыл редактирование вчерашней записи (mode=%d)", s.mode)
	}
	if s.lastErr == nil {
		t.Error("для вчерашней записи не выставлена ошибка")
	}
}

// TestTasksJournalEscDoesNotQuit — регрессия: Esc из модалки журнала/ссылок
// закрывает модалку, а не выходит из приложения.
func TestTasksJournalEscDoesNotQuit(t *testing.T) {
	_, s, _, _ := tasksSeedProject(t)
	m := &model{tasks: s, screen: screenTasks}
	selectFirstSubtask(m)
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyTab})
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyCtrlJ})
	if s.mode != taskJournal {
		t.Fatalf("ctrl+j не открыл журнал (mode=%d)", s.mode)
	}
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd != nil {
		t.Error("Esc из модалки журнала вернул команду (выход из приложения)")
	}
	if s.mode != taskBrowse {
		t.Error("Esc не закрыл модалку журнала")
	}
}

// TestQuitConfirmRunningSession — при выходе с запущенным учётом времени
// показывается предупреждение: Enter останавливает подзадачу и выходит,
// Esc отменяет выход.
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
	m := model{db: conn, tasks: newTasksScreen(conn), proj: newProjectsScreen(conn), screen: screenTasks}
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
	m := model{db: conn, tasks: newTasksScreen(conn), proj: newProjectsScreen(conn), screen: screenTasks}
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
	m := model{db: conn, screen: screenTasks}
	m.tasks = newTasksScreen(conn)
	m.proj = newProjectsScreen(conn)
	repCfg := &reportConfig{period: periodToday, saveDir: "reports"}
	m.reports = newReportsScreen(conn, repCfg)
	m.settings = newSettingsScreen(conn, repCfg)
	m.tasks.load()
	m.proj.load()
	m.reports.load()
	m.tasks.resize(150, 27)
	m.proj.resize(150, 27)
	m.reports.resize(150, 27)
	m.width, m.height = 150, 30
	return m
}

// TestReportsScreenRender — отчёт за сегодня: переход по r, заголовок
// периода, задачи с подзадачами и общее время.
func TestReportsScreenRender(t *testing.T) {
	conn, task, _ := reportsSeedProject(t)
	m := newReportsModel(conn)

	mm, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	m = mm.(model)
	if cmd != nil {
		t.Fatal("переход на отчёты не должен возвращать команду")
	}
	if m.screen != screenReports {
		t.Fatalf("экран %d, ожидался screenReports", m.screen)
	}
	view := m.View()
	for _, want := range []string{"Отчет за сегодня", "Общее время:", task.Title, "S · "} {
		if !strings.Contains(view, want) {
			t.Errorf("в отчёте нет %q", want)
		}
	}
}

// TestReportsEmptyDay — без записей за сегодня отчёт показывает подсказку.
func TestReportsEmptyDay(t *testing.T) {
	conn, err := db.Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	m := newReportsModel(conn)

	mm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	m = mm.(model)
	if !strings.Contains(m.View(), "Времени за период ещё не учтено") {
		t.Error("пустой отчёт не показывает подсказку")
	}
}

// TestReportsSwitchWithRunningSession — при запущенном учёте времени переход
// на отчёты показывает предупреждение: Enter останавливает подзадачу и
// переходит, Esc отменяет.
func TestReportsSwitchWithRunningSession(t *testing.T) {
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
	if err := db.StartSession(conn, st.ID, time.Now().Add(-30*time.Minute)); err != nil {
		t.Fatal(err)
	}
	m := newReportsModel(conn)
	r := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}}

	mm, cmd := m.Update(r)
	m = mm.(model)
	if cmd != nil {
		t.Fatal("r с запущенной сессией не должен переключать сразу")
	}
	if !m.reportConfirm || m.screen != screenTasks {
		t.Errorf("предупреждение не выставлено: confirm=%v screen=%d", m.reportConfirm, m.screen)
	}
	if !strings.Contains(m.View(), "сформировать отчёт") {
		t.Error("в предупреждении нет вопроса о формировании отчёта")
	}

	mm, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = mm.(model)
	if m.reportConfirm {
		t.Error("Esc не отменил предупреждение")
	}
	if run, _ := db.RunningSession(conn); run == nil {
		t.Error("отмена остановила сессию")
	}

	mm, _ = m.Update(r)
	m = mm.(model)
	if !m.reportConfirm {
		t.Fatal("r не показал предупреждение повторно")
	}
	mm, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = mm.(model)
	if m.reportConfirm || m.screen != screenReports {
		t.Errorf("Enter не перешёл к отчётам: confirm=%v screen=%d", m.reportConfirm, m.screen)
	}
	if run, _ := db.RunningSession(conn); run != nil {
		t.Error("переход к отчётам не остановил сессию")
	}
	if !strings.Contains(m.View(), "S · ") {
		t.Error("остановленная сессия не попала в отчёт")
	}
}

// TestReportsPeriods — период «вчера» показывает данные за вчера и свой
// заголовок; «неделя» и «месяц» — свои заголовки.
func TestReportsPeriods(t *testing.T) {
	conn, err := db.Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	p, err := db.CreateProject(conn, "P")
	if err != nil {
		t.Fatal(err)
	}
	task, err := db.CreateTask(conn, p.ID, "T вчера")
	if err != nil {
		t.Fatal(err)
	}
	st, err := db.CreateSubtask(conn, task.ID, "S вчера")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	if err := db.StartSession(conn, st.ID, now.Add(-26*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := db.StopSession(conn, st.ID, now.Add(-25*time.Hour)); err != nil {
		t.Fatal(err)
	}
	m := newReportsModel(conn)
	r := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}}

	// сегодня: записей нет
	mm, _ := m.Update(r)
	m = mm.(model)
	if !strings.Contains(m.View(), "Времени за период ещё не учтено") {
		t.Error("за сегодня отчёт не пуст")
	}

	// вчера
	m.reports.cfg.period = periodYesterday
	mm, _ = m.Update(r)
	m = mm.(model)
	view := m.View()
	for _, want := range []string{"Отчет за вчера", "T вчера", "S вчера"} {
		if !strings.Contains(view, want) {
			t.Errorf("за вчера нет %q", want)
		}
	}

	// неделя и месяц — заголовки
	m.reports.cfg.period = periodWeek
	mm, _ = m.Update(r)
	m = mm.(model)
	if !strings.Contains(m.View(), "Отчет за неделю") {
		t.Error("нет заголовка недели")
	}
	m.reports.cfg.period = periodMonth
	mm, _ = m.Update(r)
	m = mm.(model)
	if !strings.Contains(m.View(), "Отчет за "+monthNames[time.Now().Month()-1]) {
		t.Error("нет заголовка месяца")
	}
}

// TestReportsSave — клавиша s на экране отчётов сохраняет файл с заголовком,
// задачами и общим временем; на экране появляется подтверждение.
func TestReportsSave(t *testing.T) {
	conn, task, _ := reportsSeedProject(t)
	m := newReportsModel(conn)
	dir := t.TempDir()
	m.settings.cfg.saveDir = dir
	r := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}}
	sv := tea.KeyMsg{Type: tea.KeyCtrlS}

	mm, _ := m.Update(r)
	m = mm.(model)
	mm, _ = m.Update(sv)
	m = mm.(model)

	name := m.reports.saveFileName()
	data, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Fatalf("файл отчёта не создан: %v", err)
	}
	for _, want := range []string{"Отчет за сегодня", task.Title, "Общее время:"} {
		if !strings.Contains(string(data), want) {
			t.Errorf("в файле отчёта нет %q", want)
		}
	}
	if !strings.Contains(m.View(), "Отчёт сохранён") {
		t.Error("на экране нет подтверждения сохранения")
	}
}

// TestReportsSaveCreatesDir — каталог сохранения создаётся автоматически.
func TestReportsSaveCreatesDir(t *testing.T) {
	conn, _, _ := reportsSeedProject(t)
	m := newReportsModel(conn)
	dir := t.TempDir() + "/deep/reports"
	m.settings.cfg.saveDir = dir

	mm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	m = mm.(model)
	mm, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	m = mm.(model)

	if _, err := os.Stat(filepath.Join(dir, m.reports.saveFileName())); err != nil {
		t.Fatalf("файл в созданном каталоге: %v", err)
	}
}

// TestReportsJournalInReport — при включённом журнале записи за период
// показываются под подзадачей и попадают в сохраняемый файл.
func TestReportsJournalInReport(t *testing.T) {
	conn, _, st := reportsSeedProject(t)
	if _, err := db.CreateJournalEntry(conn, st.ID, "запись в журнале"); err != nil {
		t.Fatal(err)
	}
	m := newReportsModel(conn)
	m.settings.cfg.includeJournal = true
	dir := t.TempDir()
	m.settings.cfg.saveDir = dir

	mm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	m = mm.(model)
	if !strings.Contains(m.View(), "запись в журнале") {
		t.Error("запись журнала не показана в отчёте")
	}

	mm, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	m = mm.(model)
	data, err := os.ReadFile(filepath.Join(dir, m.reports.saveFileName()))
	if err != nil {
		t.Fatalf("файл отчёта не создан: %v", err)
	}
	if !strings.Contains(string(data), "запись в журнале") {
		t.Error("запись журнала не попала в файл")
	}
}

// TestSettingsForm — настройки: период и проект выбираются в модалках,
// журнал — toggle, каталог — ввод пути; значения пишутся в общий cfg.
func TestSettingsForm(t *testing.T) {
	conn, err := db.Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	p1, err := db.CreateProject(conn, "P1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.CreateProject(conn, "P2"); err != nil {
		t.Fatal(err)
	}
	m := newReportsModel(conn)

	mm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	m = mm.(model)
	if m.screen != screenSettings {
		t.Fatalf("s не открыл настройки (screen=%d)", m.screen)
	}
	view := m.View()
	for _, want := range []string{"Настройки отчёта", "Период:  сегодня", "все проекты", "Журнал:  выкл", "Каталог: reports"} {
		if !strings.Contains(view, want) {
			t.Errorf("в настройках нет %q", want)
		}
	}

	// период: Enter → модалка списка, ↓ Enter → вчера
	m.updateSettings(tea.KeyMsg{Type: tea.KeyEnter})
	if m.settings.mode != settingsPeriodList {
		t.Fatalf("Enter на периоде не открыл модалку (mode=%d)", m.settings.mode)
	}
	if !strings.Contains(m.View(), "Период отчёта") {
		t.Error("модалка периода не отрендерилась")
	}
	m.updateSettings(tea.KeyMsg{Type: tea.KeyDown})
	m.updateSettings(tea.KeyMsg{Type: tea.KeyEnter})
	if m.settings.cfg.period != periodYesterday || !strings.Contains(m.View(), "Период:  вчера") {
		t.Errorf("период не сменился: %d", m.settings.cfg.period)
	}

	// свой период: модалка (курсор на текущем — «вчера»), ↓×3 до «свой…»
	m.updateSettings(tea.KeyMsg{Type: tea.KeyEnter})
	if m.settings.mode != settingsPeriodList {
		t.Fatalf("Enter на периоде не открыл модалку (mode=%d)", m.settings.mode)
	}
	for i := 0; i < 3; i++ {
		m.updateSettings(tea.KeyMsg{Type: tea.KeyDown})
	}
	m.updateSettings(tea.KeyMsg{Type: tea.KeyEnter})
	if m.settings.mode != settingsPeriodInput {
		t.Fatalf("«свой…» не открыл ввод (mode=%d)", m.settings.mode)
	}

	// неверный формат — ошибка, режим не закрывается
	m.settings.periodInput.SetValue("abc")
	m.updateSettings(tea.KeyMsg{Type: tea.KeyEnter})
	if m.settings.mode != settingsPeriodInput || m.settings.lastErr == nil {
		t.Fatal("неверная дата не показала ошибку")
	}
	if !strings.Contains(m.View(), "нужен формат ДД.ММ.ГГГГ") {
		t.Error("нет подсказки про формат")
	}
	m.updateSettings(tea.KeyMsg{Type: tea.KeyEsc})
	if m.settings.mode != settingsBrowse || m.settings.cfg.period != periodYesterday {
		t.Error("Esc после ошибки вёл себя неверно")
	}

	// верный диапазон: снова модалка (курсор на «вчера»), ↓×3 до «свой…»
	m.updateSettings(tea.KeyMsg{Type: tea.KeyEnter})
	for i := 0; i < 3; i++ {
		m.updateSettings(tea.KeyMsg{Type: tea.KeyDown})
	}
	m.updateSettings(tea.KeyMsg{Type: tea.KeyEnter})
	m.settings.periodInput.SetValue("01.08.2026-03.08.2026")
	m.updateSettings(tea.KeyMsg{Type: tea.KeyEnter})
	if m.settings.cfg.period != periodCustom {
		t.Fatal("диапазон не применился")
	}
	wantFrom := time.Date(2026, 8, 1, 0, 0, 0, 0, time.Local)
	wantTo := time.Date(2026, 8, 4, 0, 0, 0, 0, time.Local)
	if !m.settings.cfg.customFrom.Equal(wantFrom) || !m.settings.cfg.customTo.Equal(wantTo) {
		t.Errorf("границы: %v — %v", m.settings.cfg.customFrom, m.settings.cfg.customTo)
	}
	if !strings.Contains(m.View(), "Период:  свой · 01.08.2026 – 03.08.2026") {
		t.Error("строка периода не показывает диапазон")
	}
	if m.reports.periodLabel() != "Отчет · 01.08.2026 – 03.08.2026" {
		t.Errorf("заголовок отчёта: %q", m.reports.periodLabel())
	}

	// одиночная дата: модалка (курсор уже на «свой…»), сразу Enter
	m.updateSettings(tea.KeyMsg{Type: tea.KeyEnter})
	m.updateSettings(tea.KeyMsg{Type: tea.KeyEnter})
	m.settings.periodInput.SetValue("15.08.2026")
	m.updateSettings(tea.KeyMsg{Type: tea.KeyEnter})
	if m.reports.periodLabel() != "Отчет за день · 15.08.2026" {
		t.Errorf("заголовок одиночного дня: %q", m.reports.periodLabel())
	}
	if m.reports.saveFileName() != "2026-08-15.txt" {
		t.Errorf("имя файла: %q", m.reports.saveFileName())
	}

	// проект: ↓ Enter → модалка списка, ↓ Enter → P1, Esc — отмена
	m.updateSettings(tea.KeyMsg{Type: tea.KeyDown})
	m.updateSettings(tea.KeyMsg{Type: tea.KeyEnter})
	if m.settings.mode != settingsProjList {
		t.Fatalf("Enter на проекте не открыл модалку (mode=%d)", m.settings.mode)
	}
	if !strings.Contains(m.View(), "Фильтр по проекту") {
		t.Error("модалка проекта не отрендерилась")
	}
	m.updateSettings(tea.KeyMsg{Type: tea.KeyDown})
	m.updateSettings(tea.KeyMsg{Type: tea.KeyEnter})
	if m.settings.cfg.projectID != p1.ID || !strings.Contains(m.View(), "Проект:  P1") {
		t.Errorf("фильтр не перешёл на первый проект: %d", m.settings.cfg.projectID)
	}
	m.updateSettings(tea.KeyMsg{Type: tea.KeyEnter})
	if m.settings.mode != settingsProjList {
		t.Fatal("Enter на проекте не открыл модалку повторно")
	}
	m.updateSettings(tea.KeyMsg{Type: tea.KeyEsc})
	if m.settings.mode != settingsBrowse || m.settings.cfg.projectID != p1.ID {
		t.Error("Esc не отменил выбор проекта")
	}

	// журнал: ↓ Enter → вкл
	m.updateSettings(tea.KeyMsg{Type: tea.KeyDown})
	m.updateSettings(tea.KeyMsg{Type: tea.KeyEnter})
	if !m.settings.cfg.includeJournal || !strings.Contains(m.View(), "Журнал:  вкл") {
		t.Error("журнал не включился")
	}

	// каталог: ↓ Enter → модалка, ввод пути, Enter
	m.updateSettings(tea.KeyMsg{Type: tea.KeyDown})
	m.updateSettings(tea.KeyMsg{Type: tea.KeyEnter})
	if m.settings.mode != settingsDirInput {
		t.Fatalf("Enter на каталоге не открыл модалку (mode=%d)", m.settings.mode)
	}
	if !strings.Contains(m.View(), "Каталог сохранения отчётов") {
		t.Error("модалка каталога не отрендерилась")
	}
	m.settings.dirInput.SetValue("out/reports")
	m.updateSettings(tea.KeyMsg{Type: tea.KeyEnter})
	if m.settings.mode != settingsBrowse {
		t.Fatal("Enter не закрыл модалку каталога")
	}
	if m.settings.cfg.saveDir != "out/reports" || !strings.Contains(m.View(), "Каталог: out/reports") {
		t.Errorf("каталог не сохранился: %q", m.settings.cfg.saveDir)
	}

	// Esc в модалке отменяет
	m.updateSettings(tea.KeyMsg{Type: tea.KeyEnter})
	m.settings.dirInput.SetValue("другой")
	m.updateSettings(tea.KeyMsg{Type: tea.KeyEsc})
	if m.settings.mode != settingsBrowse || m.settings.cfg.saveDir != "out/reports" {
		t.Error("Esc не отменил изменение каталога")
	}
}

// TestSettingsListScroll — при большом числе проектов модалка показывает
// лишь видимую часть и плавно прокручивается за курсором.
func TestSettingsListScroll(t *testing.T) {
	conn, err := db.Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	for i := 0; i < 15; i++ {
		if _, err := db.CreateProject(conn, fmt.Sprintf("Проект %02d", i)); err != nil {
			t.Fatal(err)
		}
	}
	m := newReportsModel(conn)
	m.settings.resize(100, 27) // visible = 12

	mm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	m = mm.(model)
	m.updateSettings(tea.KeyMsg{Type: tea.KeyDown}) // строка «Проект»
	m.updateSettings(tea.KeyMsg{Type: tea.KeyEnter})
	if m.settings.mode != settingsProjList {
		t.Fatalf("модалка проекта не открылась (mode=%d)", m.settings.mode)
	}
	view := m.View()
	if !strings.Contains(view, "все проекты") || strings.Contains(view, "Проект 14") {
		t.Error("в начале видны не те элементы списка")
	}

	for i := 0; i < 15; i++ {
		m.updateSettings(tea.KeyMsg{Type: tea.KeyDown})
	}
	view = m.View()
	if !strings.Contains(view, "Проект 14") || strings.Contains(view, "Проект 00") {
		t.Error("список не прокрутился за курсором до конца")
	}

	m.updateSettings(tea.KeyMsg{Type: tea.KeyEnter})
	if m.settings.cfg.projectID == 0 {
		t.Error("Enter не выбрал проект из прокрученного списка")
	}
}
