package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbletea"
	"github.com/detrenasama/tasky/internal/db"
)

func TestTasksLinkAddFlow(t *testing.T) {
	conn, s, task, _ := tasksSeedProject(t)
	runes := func(r rune) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}} }

	s.updateTasksMsg(runes('o'))
	s.updateTasksMsg(runes('n'))
	if s.mode != taskLinkEdit {
		t.Fatalf("o+n не открыл модалку ссылки (mode=%d)", s.mode)
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
	if s.mode != taskLinks {
		t.Fatalf("Enter не сохранил ссылку (mode=%d)", s.mode)
	}
	links, _ := db.TaskLinks(conn, task.ID)
	if len(links) != 1 || links[0].Name != "Доки" || links[0].URL != "https://example.com" {
		t.Errorf("сохранённая ссылка = %+v", links)
	}
	if !strings.Contains(stripANSI(s.descBox()), "Доки") {
		t.Error("ссылка не отобразилась в колонке описания")
	}

	// Esc из формы новой ссылки возвращает в список ссылок (а не закрывает всю модалку).
	// На этом этапе мы уже в taskLinks (после шага с пустым URL), поэтому
	// открываем форму через n.
	s.updateTasksMsg(runes('n'))
	if s.mode != taskLinkEdit || s.editLinkID != 0 {
		t.Fatalf("n не открыл форму новой ссылки (mode=%d id=%d)", s.mode, s.editLinkID)
	}
	s.linkInput.SetValue("https://example.org")
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyEsc})
	if s.mode != taskLinks {
		t.Fatalf("Esc не вернул в список ссылок (mode=%d)", s.mode)
	}
	// повторный Esc уже закрывает список
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyEsc})
	if s.mode != taskBrowse {
		t.Fatalf("повторный Esc не закрыл список (mode=%d)", s.mode)
	}
	if links, _ := db.TaskLinks(conn, task.ID); len(links) != 1 {
		t.Errorf("отменённый ввод создал ссылку: %d", len(links))
	}

	// пустой URL закрывает модалку без создания
	s.updateTasksMsg(runes('o'))
	s.updateTasksMsg(runes('n'))
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyTab})
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyEnter})
	if s.mode != taskLinks {
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

// TestTasksLinkEditFlow — из списка ссылок (o) клавиша e открывает форму с
// префиллом выбранной ссылки, изменение сохраняется (Update), а n создаёт
// новую ссылку; Enter в списке по-прежнему открывает URL.
func TestTasksLinkEditFlow(t *testing.T) {
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

	// e открывает форму с префиллом
	s.updateTasksMsg(runes('e'))
	if s.mode != taskLinkEdit || s.editLinkID == 0 {
		t.Fatalf("e не открыл редактор (mode=%d id=%d)", s.mode, s.editLinkID)
	}
	if s.linkName.Value() != "Доки" || s.linkInput.Value() != "https://example.com" {
		t.Errorf("префилл: name=%q url=%q", s.linkName.Value(), s.linkInput.Value())
	}
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyTab})
	if !s.linkInput.Focused() {
		t.Error("Tab не перевёл фокус на адрес")
	}
	s.linkInput.SetValue("https://changed.org")
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyEnter})
	if s.mode != taskLinks {
		t.Fatalf("Enter не вернул в список (mode=%d)", s.mode)
	}
	links, _ := db.TaskLinks(conn, task.ID)
	if len(links) != 1 || links[0].URL != "https://changed.org" || links[0].Name != "Доки" {
		t.Errorf("ссылка не обновилась: %+v", links)
	}

	// n создаёт новую ссылку
	s.updateTasksMsg(runes('n'))
	if s.mode != taskLinkEdit || s.editLinkID != 0 {
		t.Fatalf("n не открыл новую ссылку (mode=%d id=%d)", s.mode, s.editLinkID)
	}
	s.linkName.SetValue("Второй")
	s.linkInput.SetValue("https://second.org")
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyTab})
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyEnter})
	links, _ = db.TaskLinks(conn, task.ID)
	if len(links) != 2 {
		t.Errorf("n не создал ссылку: %+v", links)
	}
}

// TestTasksLinkEmptyState — при отсутствии ссылок список не показывает
// подсказок правки/удаления и выводит плейсхолдер «Ссылок нет».
func TestTasksLinkEmptyState(t *testing.T) {
	conn, s, task, _ := tasksSeedProject(t)
	_ = conn
	_ = task
	runes := func(r rune) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}} }

	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyTab})
	s.updateTasksMsg(runes('o'))
	if s.mode != taskLinks {
		t.Fatalf("o не открыл список ссылок (mode=%d)", s.mode)
	}
	dlg, open := s.dialog()
	if !open {
		t.Fatal("модалка списка ссылок не отрендерилась")
	}
	if !strings.Contains(dlg, "Ссылок нет") {
		t.Error("нет плейсхолдера для пустого списка")
	}
	if strings.Contains(dlg, "изменить") || strings.Contains(dlg, "удалить") {
		t.Error("в пустом списке не должно быть подсказок правки/удаления")
	}

	// n открывает форму новой ссылки, Esc возвращает в список (не в обзор)
	s.updateTasksMsg(runes('n'))
	if s.mode != taskLinkEdit || s.editLinkID != 0 {
		t.Fatalf("n не открыл новую ссылку (mode=%d id=%d)", s.mode, s.editLinkID)
	}
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyEsc})
	if s.mode != taskLinks {
		t.Fatalf("Esc из формы не вернул в список (mode=%d)", s.mode)
	}
}

// TestTaskJournalAddAndEdit — Ctrl+J добавляет запись (Ctrl+S сохраняет,
// Esc отменяет), j редактирует самую свежую запись текущего дня.
