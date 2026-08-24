package ui

import (
	"testing"

	"github.com/charmbracelet/bubbletea"
	"github.com/detrenasama/tasky/internal/db"
	"github.com/detrenasama/tasky/internal/store"
)

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

	m.updateProjects(runes('o'))
	m.updateProjects(runes('n'))
	if m.proj.mode != projLinkEdit {
		t.Fatalf("o+n не открыл модалку ссылки (mode=%d)", m.proj.mode)
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
	if m.proj.mode != projLinks {
		t.Fatalf("Enter не сохранил ссылку (mode=%d)", m.proj.mode)
	}
	links, _ := db.ProjectLinks(conn, p.ID)
	if len(links) != 1 || links[0].Name != "Доки" || links[0].URL != "https://example.com" {
		t.Errorf("сохранённая ссылка = %+v", links)
	}

	// Esc из формы новой ссылки возвращает в список ссылок (а не закрывает всю модалку).
	// На этом этапе мы уже в projLinks (после шага с пустым URL), поэтому
	// открываем форму через n.
	m.updateProjects(runes('n'))
	if m.proj.mode != projLinkEdit || m.proj.editLinkID != 0 {
		t.Fatalf("n не открыл форму новой ссылки (mode=%d id=%d)", m.proj.mode, m.proj.editLinkID)
	}
	m.proj.linkName.SetValue("Имя")
	m.proj.linkInput.SetValue("https://example.org")
	m.updateProjects(tea.KeyMsg{Type: tea.KeyEsc})
	if m.proj.mode != projLinks {
		t.Fatalf("Esc не вернул в список ссылок (mode=%d)", m.proj.mode)
	}
	// повторный Esc уже закрывает список
	m.updateProjects(tea.KeyMsg{Type: tea.KeyEsc})
	if m.proj.mode != projBrowse {
		t.Fatalf("повторный Esc не закрыл список (mode=%d)", m.proj.mode)
	}
	if links, _ := db.ProjectLinks(conn, p.ID); len(links) != 1 {
		t.Errorf("отменённый ввод создал ссылку: %d", len(links))
	}

	// Enter на адресе с пустым URL — модалка закрывается, ссылка не создаётся
	m.updateProjects(runes('o'))
	m.updateProjects(runes('n'))
	m.updateProjects(tea.KeyMsg{Type: tea.KeyTab})
	m.updateProjects(tea.KeyMsg{Type: tea.KeyEnter})
	if m.proj.mode != projLinks {
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

// TestProjectLinkEditFlow — из списка ссылок (o) клавиша e открывает форму с
// префиллом выбранной ссылки, изменение сохраняется (Update), а n создаёт
// новую ссылку; Enter в списке по-прежнему открывает URL.
func TestProjectLinkEditFlow(t *testing.T) {
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
	runes := func(r rune) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}} }

	m.updateProjects(tea.KeyMsg{Type: tea.KeyTab})
	m.updateProjects(runes('o'))
	if m.proj.mode != projLinks {
		t.Fatalf("o не открыл список ссылок (mode=%d)", m.proj.mode)
	}

	m.updateProjects(runes('e'))
	if m.proj.mode != projLinkEdit || m.proj.editLinkID == 0 {
		t.Fatalf("e не открыл редактор (mode=%d id=%d)", m.proj.mode, m.proj.editLinkID)
	}
	if m.proj.linkName.Value() != "Доки" || m.proj.linkInput.Value() != "https://example.com" {
		t.Errorf("префилл: name=%q url=%q", m.proj.linkName.Value(), m.proj.linkInput.Value())
	}
	m.updateProjects(tea.KeyMsg{Type: tea.KeyTab})
	if !m.proj.linkInput.Focused() {
		t.Error("Tab не перевёл фокус на адрес")
	}
	m.proj.linkInput.SetValue("https://changed.org")
	m.updateProjects(tea.KeyMsg{Type: tea.KeyEnter})
	if m.proj.mode != projLinks {
		t.Fatalf("Enter не вернул в список (mode=%d)", m.proj.mode)
	}
	links, _ := db.ProjectLinks(conn, p.ID)
	if len(links) != 1 || links[0].URL != "https://changed.org" || links[0].Name != "Доки" {
		t.Errorf("ссылка не обновилась: %+v", links)
	}

	m.updateProjects(runes('n'))
	if m.proj.mode != projLinkEdit || m.proj.editLinkID != 0 {
		t.Fatalf("n не открыл новую ссылку (mode=%d id=%d)", m.proj.mode, m.proj.editLinkID)
	}
	m.proj.linkName.SetValue("Второй")
	m.proj.linkInput.SetValue("https://second.org")
	m.updateProjects(tea.KeyMsg{Type: tea.KeyTab})
	m.updateProjects(tea.KeyMsg{Type: tea.KeyEnter})
	links, _ = db.ProjectLinks(conn, p.ID)
	if len(links) != 2 {
		t.Errorf("n не создал ссылку: %+v", links)
	}
}

// TestProjectsDescBoxTruncatesLongLink — длинное название ссылки в колонке
// обрезается многоточием.
