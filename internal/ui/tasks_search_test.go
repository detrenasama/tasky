package ui

import (
	"testing"

	"github.com/charmbracelet/bubbletea"
	"github.com/detrenasama/tasky/internal/db"
	"github.com/detrenasama/tasky/internal/store"
)

func TestTaskSearch(t *testing.T) {
	conn, err := db.Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	p, err := db.CreateProject(conn, "P")
	if err != nil {
		t.Fatal(err)
	}
	t1, err := db.CreateTask(conn, p.ID, "SEO страницы")
	if err != nil {
		t.Fatal(err)
	}
	st1, err := db.CreateSubtask(conn, t1.ID, "Мета-теги")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.CreateSubtask(conn, t1.ID, "Картинки"); err != nil {
		t.Fatal(err)
	}
	t2, err := db.CreateTask(conn, p.ID, "Отчёт")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.CreateSubtask(conn, t2.ID, "Сборка"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.CreateJournalEntry(conn, st1.ID, "работал над мета-тегами"); err != nil {
		t.Fatal(err)
	}
	if err := db.UpdateTaskDescription(conn, t1.ID, "оптимизация скорости"); err != nil {
		t.Fatal(err)
	}
	if err := db.UpdateTaskDescription(conn, t2.ID, "ежемесячная сводка"); err != nil {
		t.Fatal(err)
	}
	s := newTasksScreen(store.NewSQLite(conn))
	s.load()
	runes := func(r rune) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}} }

	// / открывает модалку поиска
	s.updateTasksMsg(runes('/'))
	if s.mode != taskSearch {
		t.Fatalf("/ не открыл поиск (mode=%d)", s.mode)
	}
	if _, open := s.dialog(); !open {
		t.Error("поиск не рендерится как модалка")
	}

	// поиск по журналу: «мета» — подзадача «Мета-теги» из журнала st1
	for _, r := range "мета" {
		s.updateTasksMsg(runes(r))
	}
	if s.searchQuery != "мета" {
		t.Fatalf("запрос = %q", s.searchQuery)
	}
	if got := searchTitles(s); !equalStrings(got, []string{"SEO страницы", "Мета-теги"}) {
		t.Errorf("после «мета»: %v", got)
	}

	// Enter применяет запрос и закрывает модалку
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyEnter})
	if s.mode != taskBrowse {
		t.Fatalf("Enter не закрыл поиск (mode=%d)", s.mode)
	}
	if s.searchQuery != "мета" {
		t.Errorf("Enter стёр запрос: %q", s.searchQuery)
	}
	if got := searchTitles(s); !equalStrings(got, []string{"SEO страницы", "Мета-теги"}) {
		t.Errorf("фильтр после Enter: %v", got)
	}

	// Esc в браузе сбрасывает поиск
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyEsc})
	if s.searchQuery != "" {
		t.Errorf("Esc не сбросил запрос: %q", s.searchQuery)
	}
	if got := searchTitles(s); !equalStrings(got, []string{"SEO страницы", "Отчёт"}) {
		t.Errorf("после сброса: %v", got)
	}

	// поиск по описанию задачи: совпала задача — видны все её подзадачи
	s.updateTasksMsg(runes('/'))
	for _, r := range "оптимиз" {
		s.updateTasksMsg(runes(r))
	}
	if got := searchTitles(s); !equalStrings(got, []string{"SEO страницы", "Мета-теги", "Картинки"}) {
		t.Errorf("по описанию: %v", got)
	}

	// поиск по названию подзадачи: видна только совпавшая подзадача
	s.searchInput.SetValue("")
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyEnter})
	s.updateTasksMsg(runes('/'))
	s.searchInput.SetValue("сборка")
	s.updateTasksMsg(runes(' '))
	if got := searchTitles(s); !equalStrings(got, []string{"Отчёт", "Сборка"}) {
		t.Errorf("по названию подзадачи: %v", got)
	}

	// Esc внутри модалки отменяет поиск целиком
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyEsc})
	if s.mode != taskBrowse || s.searchQuery != "" {
		t.Fatalf("Esc в модалке: mode=%d query=%q", s.mode, s.searchQuery)
	}
	if got := searchTitles(s); !equalStrings(got, []string{"SEO страницы", "Отчёт"}) {
		t.Errorf("после отмены: %v", got)
	}
}
