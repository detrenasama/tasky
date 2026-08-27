package ui

import (
	"os"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/detrenasama/tasky/internal/db"
	"github.com/detrenasama/tasky/internal/store"
)

func TestProjectsResizeColumns(t *testing.T) {
	conn, err := db.Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if _, err := db.CreateProject(conn, "P"); err != nil {
		t.Fatal(err)
	}
	s := newProjectsScreen(store.NewSQLite(conn))
	s.load()
	cases := []struct{ w, list, desc int }{
		{150, 74, 74},
		{110, 54, 54},
		{109, 53, 54},
		{60, 60, 0},
		{59, 59, 0},
	}
	for _, c := range cases {
		s.resize(c.w, 26)
		if s.listW != c.list || s.descW != c.desc {
			t.Errorf("w=%d: list=%d desc=%d, ожидались %d/%d",
				c.w, s.listW, s.descW, c.list, c.desc)
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
	s := newProjectsScreen(store.NewSQLite(conn))
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
	s := newProjectsScreen(store.NewSQLite(conn))
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

func TestProjectsDescEdit(t *testing.T) {
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
	m := model{proj: newProjectsScreen(store.NewSQLite(conn))}
	m.proj.load()
	m.proj.resize(150, 26)

	// e — без действия (описание теперь открывается по Enter)
	before := m.proj.mode
	m.updateProjects(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	if m.proj.mode != before {
		t.Error("e не должен открывать редактирование")
	}

	// Enter открывает крупную модалку описания
	m.updateProjects(tea.KeyMsg{Type: tea.KeyEnter})
	if m.proj.mode != projDescModal || m.proj.dmState != dmView {
		t.Error("Enter должен открыть модалку описания")
	}

	// правка в модалке: e открывает textarea
	m.updateProjects(key('e'))
	if m.proj.dmState != dmEdit {
		t.Error("e должен открыть правку описания")
	}
	m.proj.descText.SetValue("текст в textarea")
	dlg, open := m.proj.dialog()
	if !open || !strings.Contains(dlg, "текст в textarea") {
		t.Error("при правке в модалке не виден textarea")
	}
	if strings.Contains(dlg, "Несохранённые") {
		t.Error("правка не должна открывать подтверждение")
	}

	// Ctrl+S сохраняет и остаётся в модалке
	m.proj.descText.SetValue("новое описание")
	m.updateProjects(tea.KeyMsg{Type: tea.KeyCtrlS})
	if m.proj.mode != projDescModal || m.proj.dmState != dmView {
		t.Error("Ctrl+S должен остаться в модалке")
	}
	got, _ := db.ProjectDescription(conn, p.ID)
	if got != "новое описание" {
		t.Errorf("описание в БД = %q, ожидалось «новое описание»", got)
	}

	// Esc с несохранёнными изменениями — подтверждение
	m.updateProjects(key('e'))
	m.proj.descText.SetValue("не сохранять")
	m.updateProjects(tea.KeyMsg{Type: tea.KeyEsc})
	if m.proj.dmState != dmDiscard {
		t.Fatalf("Esc с изменениями должен открыть подтверждение, dm=%d", m.proj.dmState)
	}
	if got, _ := db.ProjectDescription(conn, p.ID); got != "новое описание" {
		t.Errorf("подтверждение не должно сохранять: %q", got)
	}
	// Esc в подтверждении — возврат к правке
	m.updateProjects(tea.KeyMsg{Type: tea.KeyEsc})
	if m.proj.dmState != dmEdit {
		t.Fatalf("Esc в подтверждении должен вернуть к правке, dm=%d", m.proj.dmState)
	}
	// неизменённое описание — Esc в просмотр, затем выход
	m.proj.descText.SetValue("новое описание")
	m.updateProjects(tea.KeyMsg{Type: tea.KeyEsc})
	if m.proj.dmState != dmView {
		t.Fatalf("Esc без изменений должен выйти в просмотр, dm=%d", m.proj.dmState)
	}
	m.updateProjects(tea.KeyMsg{Type: tea.KeyEsc})
	if m.proj.mode != projBrowse {
		t.Fatalf("Esc без изменений должен выйти, mode=%d", m.proj.mode)
	}

	// снова длинное описание, чтобы был скролл в обзоре
	if err := db.UpdateProjectDescription(conn, p.ID, strings.Repeat("строка длинного описание ", 100)); err != nil {
		t.Fatal(err)
	}
	m.proj.loadDesc()
	y0 := m.proj.descV.YOffset
	m.updateProjects(tea.KeyMsg{Type: tea.KeyPgDown})
	if m.proj.descV.YOffset <= y0 {
		t.Errorf("PgDn не проскроллил описание: %d → %d", y0, m.proj.descV.YOffset)
	}
}

// TestProjectLinksEscDoesNotQuit — регрессия: bubbles/list по умолчанию
// привязывает Quit к esc и q; из модалки ссылок Esc должен закрывать модалку,
// а не выходить из приложения.
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
	s := newProjectsScreen(store.NewSQLite(conn))
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

func TestProjectSearch(t *testing.T) {
	conn, err := db.Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	p1, err := db.CreateProject(conn, "GetJet")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.CreateProject(conn, "Верстка"); err != nil {
		t.Fatal(err)
	}
	if err := db.UpdateProjectDescription(conn, p1.ID, "адаптивные страницы"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.CreateProjectLink(conn, p1.ID, "Доки", "https://example.com"); err != nil {
		t.Fatal(err)
	}
	s := newProjectsScreen(store.NewSQLite(conn))
	s.load()
	m := &model{proj: s}
	runes := func(r rune) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}} }

	// / открывает модалку поиска
	m.updateProjects(runes('/'))
	if s.mode != projSearch {
		t.Fatalf("/ не открыл поиск (mode=%d)", s.mode)
	}
	if _, open := s.dialog(); !open {
		t.Error("поиск не рендерится как модалка")
	}

	// поиск по названию: «верст» — только «Верстка»
	for _, r := range "верст" {
		m.updateProjects(runes(r))
	}
	if s.searchQuery != "верст" {
		t.Fatalf("запрос = %q", s.searchQuery)
	}
	if got := projectTitles(s); !equalStrings(got, []string{"Верстка"}) {
		t.Errorf("после «верст»: %v", got)
	}

	// Enter применяет и закрывает модалку, фильтр остаётся
	m.updateProjects(tea.KeyMsg{Type: tea.KeyEnter})
	if s.mode != projBrowse || s.searchQuery != "верст" {
		t.Fatalf("Enter: mode=%d query=%q", s.mode, s.searchQuery)
	}
	if got := projectTitles(s); !equalStrings(got, []string{"Верстка"}) {
		t.Errorf("фильтр после Enter: %v", got)
	}

	// поиск по описанию (модалка открывается с прошлым запросом — очищаем)
	m.updateProjects(runes('/'))
	s.searchInput.SetValue("")
	for _, r := range "адаптив" {
		m.updateProjects(runes(r))
	}
	if got := projectTitles(s); !equalStrings(got, []string{"GetJet"}) {
		t.Errorf("по описанию: %v", got)
	}

	// поиск по ссылке
	s.searchInput.SetValue("")
	m.updateProjects(tea.KeyMsg{Type: tea.KeyEnter})
	m.updateProjects(runes('/'))
	s.searchInput.SetValue("example")
	m.updateProjects(runes(' '))
	if got := projectTitles(s); !equalStrings(got, []string{"GetJet"}) {
		t.Errorf("по ссылке: %v", got)
	}

	// Esc в модалке отменяет поиск целиком
	m.updateProjects(tea.KeyMsg{Type: tea.KeyEsc})
	if s.mode != projBrowse || s.searchQuery != "" {
		t.Fatalf("Esc в модалке: mode=%d query=%q", s.mode, s.searchQuery)
	}
	if got := projectTitles(s); !equalStrings(got, []string{"GetJet", "Верстка"}) {
		t.Errorf("после отмены: %v", got)
	}

	// esc в браузе при активном поиске: сброс через main.go, экран не меняется
	s.searchQuery = "верст"
	s.buildItems()
	m2 := model{store: store.NewSQLite(conn), tasks: newTasksScreen(store.NewSQLite(conn)), proj: s, screen: screenProjects}
	m2.tasks.load()
	mm, _ := m2.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m2 = mm.(model)
	if s.searchQuery != "" || m2.screen != screenProjects {
		t.Errorf("esc при поиске: query=%q screen=%d", s.searchQuery, m2.screen)
	}
	if got := projectTitles(s); !equalStrings(got, []string{"GetJet", "Верстка"}) {
		t.Errorf("после esc: %v", got)
	}
}

// TestProjectsDescModalEditorUnchanged — если $EDITOR вернул тот же текст
// (просто закрыли без правок), БД не меняется и заметка не показывается.
func TestProjectsDescModalEditorUnchanged(t *testing.T) {
	conn, err := db.Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	p, err := db.CreateProject(conn, "P")
	if err != nil {
		t.Fatal(err)
	}
	if err := db.UpdateProjectDescription(conn, p.ID, "исходное"); err != nil {
		t.Fatal(err)
	}
	m := model{proj: newProjectsScreen(store.NewSQLite(conn))}
	m.proj.load()
	m.proj.resize(150, 26)
	m.proj.list.Select(0)
	m.proj.loadDesc()
	m.proj.startEditDescription()
	m.updateProjects(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'E'}})
	if m.proj.extEditMode != 2 {
		t.Fatalf("extEditMode должен быть 2, =%d", m.proj.extEditMode)
	}
	f, err := os.CreateTemp("", "tasky-ed-*.md")
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString("исходное")
	f.Close()
	m.proj.applyExternalEdit(editReturnMsg{path: f.Name(), err: nil})
	if m.proj.dmState != dmView {
		t.Fatalf("после редактора должен быть просмотр, dm=%d", m.proj.dmState)
	}
	if m.proj.notice != "" {
		t.Errorf("при отсутствии изменений заметка не нужна: %q", m.proj.notice)
	}
	if got, _ := db.ProjectDescription(conn, p.ID); got != "исходное" {
		t.Errorf("БД не должна меняться при отсутствии правок: %q", got)
	}
}
