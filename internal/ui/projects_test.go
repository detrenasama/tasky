package ui

import (
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

	// Enter открывает инлайн-редактирование описания
	m.updateProjects(tea.KeyMsg{Type: tea.KeyEnter})
	if m.proj.mode != projDescEdit {
		t.Error("Enter должен открыть редактирование описания")
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
	m.updateProjects(tea.KeyMsg{Type: tea.KeyEnter})
	m.proj.descText.SetValue("не сохранять")
	m.updateProjects(tea.KeyMsg{Type: tea.KeyEsc})
	if m.proj.mode != projBrowse {
		t.Error("Esc не закрыл редактирование")
	}
	if got, _ := db.ProjectDescription(conn, p.ID); got != "новое описание" {
		t.Errorf("Esc сохранил изменения: %q", got)
	}

	// снова длинное описание, чтобы был скролл: Enter → SetValue → Ctrl+S
	m.updateProjects(tea.KeyMsg{Type: tea.KeyEnter})
	m.proj.descText.SetValue(strings.Repeat("строка длинного описание ", 100))
	m.updateProjects(tea.KeyMsg{Type: tea.KeyCtrlS})

	// скролл описания по PgDn в browse
	y0 := m.proj.descV.YOffset
	m.updateProjects(tea.KeyMsg{Type: tea.KeyPgDown})
	if m.proj.descV.YOffset <= y0 {
		t.Errorf("PgDn не проскроллил описание: %d → %d", y0, m.proj.descV.YOffset)
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
	m := model{proj: newProjectsScreen(store.NewSQLite(conn)), tasks: newTasksScreen(store.NewSQLite(conn)), screen: screenProjects}
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
	m := model{proj: newProjectsScreen(store.NewSQLite(conn))}
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
	m := model{proj: newProjectsScreen(store.NewSQLite(conn))}
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
