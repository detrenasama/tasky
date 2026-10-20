package ui

import (
	"database/sql"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/detrenasama/tasky/internal/db"
	"github.com/detrenasama/tasky/internal/store"
)

// hideSeedProject создаёт проект с задачей, завершённой 8 дней назад, и
// возвращает экран задач с прочитанными данными.
func hideSeedProject(t *testing.T) (*sql.DB, *tasksScreen, db.Task) {
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
	task, err := db.CreateTask(conn, p.ID, "Старая")
	if err != nil {
		t.Fatal(err)
	}
	at := time.Now().Add(-8 * 24 * time.Hour)
	if err := db.SetStatus(conn, db.OwnerTask, task.ID, "Выполнена", "", at); err != nil {
		t.Fatal(err)
	}
	task.Status = "Выполнена"
	task.CompletedAt = &at
	s := newTasksScreen(store.NewSQLite(conn))
	s.load()
	return conn, s, task
}

// TestToggleHideCompleted — ctrl+h показывает скрытые завершённые элементы,
// повторное нажатие возвращает фильтр; настройка в БД не меняется.
func TestToggleHideCompleted(t *testing.T) {
	conn, s, task := hideSeedProject(t)
	if !s.hiddenDue(task) {
		t.Fatal("задача не скрыта до переключения")
	}
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyCtrlH})
	if s.hideDays != 0 || s.hideSaved != defaultHideDays {
		t.Fatalf("после ctrl+h: hideDays=%d hideSaved=%d", s.hideDays, s.hideSaved)
	}
	if !containsString(searchTitles(s), "Старая") {
		t.Error("скрытая задача не появилась после ctrl+h")
	}
	if v, ok, err := db.GetSetting(conn, "hide_days"); err != nil || (ok && v != "7") {
		t.Errorf("ctrl+h записал настройку: %q ok=%v err=%v", v, ok, err)
	}

	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyCtrlH})
	if s.hideDays != defaultHideDays {
		t.Fatalf("после повторного ctrl+h: hideDays=%d", s.hideDays)
	}
	if containsString(searchTitles(s), "Старая") {
		t.Error("задача осталась видимой после включения скрытия")
	}
}

// TestToggleHideRestoresThreshold — возвращается порог, заданный в настройках.
func TestToggleHideRestoresThreshold(t *testing.T) {
	conn, s, _ := hideSeedProject(t)
	if err := db.SetSetting(conn, "hide_days", "14"); err != nil {
		t.Fatal(err)
	}
	s.load()
	if s.hideDays != 14 {
		t.Fatalf("порог из настроек: %d", s.hideDays)
	}
	s.toggleHide()
	s.toggleHide()
	if s.hideDays != 14 || s.hideSaved != 14 {
		t.Fatalf("порог не восстановлен: hideDays=%d hideSaved=%d", s.hideDays, s.hideSaved)
	}
}

// TestToggleHideSurvivesReload — выключенный режим живёт до перезапуска и не
// теряется при перезагрузке экрана (переключение страниц).
func TestToggleHideSurvivesReload(t *testing.T) {
	_, s, _ := hideSeedProject(t)
	s.toggleHide()
	s.load()
	if s.hideDays != 0 {
		t.Fatalf("после load: hideDays=%d", s.hideDays)
	}
	if !containsString(searchTitles(s), "Старая") {
		t.Error("выключенный режим потерялся при перезагрузке экрана")
	}
}

// TestToggleHideFromZeroSetting — если в настройках скрытие отключено (0),
// ctrl+h включает его с порогом по умолчанию.
func TestToggleHideFromZeroSetting(t *testing.T) {
	conn, s, task := hideSeedProject(t)
	if err := db.SetSetting(conn, "hide_days", "0"); err != nil {
		t.Fatal(err)
	}
	s.load()
	if s.hideDays != 0 {
		t.Fatalf("hideDays=%d, ожидался 0", s.hideDays)
	}
	s.toggleHide()
	if s.hideDays != defaultHideDays {
		t.Fatalf("hideDays=%d, ожидался %d", s.hideDays, defaultHideDays)
	}
	if !s.hiddenDue(task) {
		t.Error("задача не скрылась после включения")
	}
}

// TestStatusLabel — серый режим в заголовке: поиск важнее скрытия.
func TestStatusLabel(t *testing.T) {
	_, s, _ := hideSeedProject(t)
	if got := s.statusLabel(); got != "Активные" {
		t.Fatalf("режим по умолчанию: %q", got)
	}
	s.toggleHide()
	if got := s.statusLabel(); got != "Все" {
		t.Fatalf("после ctrl+h: %q", got)
	}
	s.searchQuery = "старая"
	if got := s.statusLabel(); got != `Поиск: "старая"` {
		t.Fatalf("при поиске: %q", got)
	}
}

// TestTasksHeaderShowsStatus — метка режима выводится в заголовке страницы.
func TestTasksHeaderShowsStatus(t *testing.T) {
	_, s, _ := hideSeedProject(t)
	s.resize(150, 26)
	if view := stripANSI(s.view(150, 26)); !strings.Contains(view, "(Активные)") {
		t.Error("в заголовке нет «(Активные)»")
	}
	s.toggleHide()
	if view := stripANSI(s.view(150, 26)); !strings.Contains(view, "(Все)") {
		t.Error("в заголовке нет «(Все)»")
	}
	s.searchQuery = "q"
	s.buildItems()
	if view := stripANSI(s.view(150, 26)); !strings.Contains(view, `(Поиск: "q")`) {
		t.Error("в заголовке нет режима поиска")
	}
}

// TestPaletteHideToggle — команда ctrl+h есть в палитре, название зависит от
// режима.
func TestPaletteHideToggle(t *testing.T) {
	_, s, _ := hideSeedProject(t)
	m := &model{screen: screenTasks, tasks: s}
	hideCmd := func() string {
		for _, g := range m.paletteGroups() {
			for _, c := range g.cmds {
				if c.keys == "ctrl+h" {
					return c.title
				}
			}
		}
		return ""
	}
	if got := hideCmd(); got != "Скрыть выполненные" {
		t.Fatalf("название команды: %q", got)
	}
	s.toggleHide()
	if got := hideCmd(); got != "Показать выполненные" {
		t.Fatalf("название команды после ctrl+h: %q", got)
	}
}
