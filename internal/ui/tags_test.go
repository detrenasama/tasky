package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbletea"
	"github.com/detrenasama/tasky/internal/db"
	"github.com/detrenasama/tasky/internal/store"
)

// TestTaskTagsFlow — клавиша g открывает модалку тегов задачи, n создаёт тег
// (тип по умолчанию + значение), Ctrl+S сохраняет, метка появляется в
// заголовке списка, Enter редактирует, d удаляет с подтверждением.
func TestTaskTagsFlow(t *testing.T) {
	conn, s, task, _ := tasksSeedProject(t)
	runes := func(r rune) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}} }

	// g открывает модалку тегов выбранной задачи
	s.updateTasksMsg(runes('g'))
	if s.mode != taskTags {
		t.Fatalf("g не открыл теги (mode=%d)", s.mode)
	}
	dlg, open := s.dialog()
	if !open || !strings.Contains(dlg, "Теги") {
		t.Error("модалка тегов не отрендерилась")
	}
	if !strings.Contains(dlg, "Тегов нет") {
		t.Error("пустой список не показывает подсказку")
	}
	if s.tagTaskID != task.ID {
		t.Errorf("tagTaskID = %d, ожидался %d", s.tagTaskID, task.ID)
	}

	// n — новый тег, тип по умолчанию — первый из каталога
	s.updateTasksMsg(runes('n'))
	if s.mode != taskTagEdit || s.tagEditID != 0 {
		t.Fatalf("n не открыл редактор (mode=%d id=%d)", s.mode, s.tagEditID)
	}
	if len(s.tagTypes) == 0 {
		t.Fatal("каталог типов пуст")
	}
	if s.tagEditType != s.tagTypes[0].ID {
		t.Errorf("тип по умолчанию %d, ожидался %d", s.tagEditType, s.tagTypes[0].ID)
	}
	s.tagEditText.SetValue("GW-567")
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyCtrlS})
	if s.mode != taskTags {
		t.Fatalf("Ctrl+S не сохранил тег (mode=%d)", s.mode)
	}
	tags, _ := db.TaskTags(conn, task.ID)
	if len(tags) != 1 || tags[0].Text != "GW-567" || tags[0].TypeName != s.tagTypes[0].Name {
		t.Fatalf("тег не создан: %+v", tags)
	}

	// метка тега появилась в заголовке списка
	s.loadData()
	title := s.list.Items()[0].(taskItem).Title()
	if !strings.Contains(title, "[GW-567]") {
		t.Errorf("метка тега не в заголовке: %q", title)
	}

	// выбор типа через модалку: Enter на строке «Тип» → список типов
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyEsc}) // закрыть теги
	s.updateTasksMsg(runes('g'))
	s.updateTasksMsg(runes('n'))
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyEnter})
	if s.mode != taskTagTypePick {
		t.Fatalf("Enter на типе не открыл модалку типов (mode=%d)", s.mode)
	}
	if _, open := s.dialog(); !open {
		t.Error("модалка типов не отрендерилась")
	}
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyEnter})
	if s.mode != taskTagEdit {
		t.Fatalf("выбор типа не вернул в редактор (mode=%d)", s.mode)
	}
	if s.tagEditType != s.tagTypes[0].ID {
		t.Errorf("выбранный тип %d, ожидался %d", s.tagEditType, s.tagTypes[0].ID)
	}
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyEsc}) // отмена редактора

	// Enter на теге — редактирование с префиллом
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyEnter})
	if s.mode != taskTagEdit || s.tagEditID != tags[0].ID {
		t.Fatalf("Enter не открыл редактор (mode=%d id=%d)", s.mode, s.tagEditID)
	}
	if s.tagEditText.Value() != "GW-567" {
		t.Errorf("префилл значения: %q", s.tagEditText.Value())
	}
	s.tagEditText.SetValue("GW-999")
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyCtrlS})
	tags, _ = db.TaskTags(conn, task.ID)
	if tags[0].Text != "GW-999" {
		t.Errorf("значение не обновилось: %+v", tags[0])
	}

	// удаление: d → подтверждение → y
	s.updateTasksMsg(runes('d'))
	if s.mode != taskTagConfirm {
		t.Fatalf("d не открыл подтверждение (mode=%d)", s.mode)
	}
	dlg, _ = s.dialog()
	if !strings.Contains(dlg, "Удалить тег") {
		t.Error("подтверждение удаления не отрендерилось")
	}
	s.updateTasksMsg(runes('y'))
	if s.mode != taskTags || len(s.tags) != 0 {
		t.Fatalf("тег не удалён (mode=%d tags=%+v)", s.mode, s.tags)
	}

	// Esc закрывает модалку тегов
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyEsc})
	if s.mode != taskBrowse {
		t.Fatalf("Esc не закрыл теги (mode=%d)", s.mode)
	}
}

// TestTaskTagsFromSubtask — g на подзадаче открывает теги родительской задачи.
func TestTaskTagsFromSubtask(t *testing.T) {
	conn, s, task, _ := tasksSeedProject(t)
	m := &model{tasks: s}
	selectFirstSubtask(m)
	types, _ := db.TagTypes(conn)
	if _, err := db.CreateTag(conn, task.ID, types[0].ID, "4455", ""); err != nil {
		t.Fatal(err)
	}
	s.loadData()
	runes := func(r rune) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}} }
	s.updateTasksMsg(runes('g'))
	if s.mode != taskTags || s.tagTaskID != task.ID {
		t.Fatalf("g на подзадаче: mode=%d tagTaskID=%d, ожидался теги родителя %d",
			s.mode, s.tagTaskID, task.ID)
	}
	if len(s.tags) != 1 || s.tags[0].Text != "4455" {
		t.Errorf("теги родителя: %+v", s.tags)
	}
}

// TestTaskSearchByTag — поиск «/» находит задачу по тексту тега.
func TestTaskSearchByTag(t *testing.T) {
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
	if _, err := db.CreateTask(conn, p.ID, "Отчёт"); err != nil {
		t.Fatal(err)
	}
	types, _ := db.TagTypes(conn)
	if _, err := db.CreateTag(conn, t1.ID, types[0].ID, "GW-567", ""); err != nil {
		t.Fatal(err)
	}
	s := newTasksScreen(store.NewSQLite(conn))
	s.load()
	runes := func(r rune) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}} }

	s.updateTasksMsg(runes('/'))
	for _, r := range "gw-5" {
		s.updateTasksMsg(runes(r))
	}
	if got := searchTitles(s); !equalStrings(got, []string{"SEO страницы"}) {
		t.Errorf("по тегу: %v", got)
	}
}

// TestReportsTagsShown — теги задачи показываются в строке отчёта и попадают
// в сохраняемый файл (номера для поиска во внешних сервисах).
func TestReportsTagsShown(t *testing.T) {
	conn, task, _ := reportsSeedProject(t)
	types, _ := db.TagTypes(conn)
	if _, err := db.CreateTag(conn, task.ID, types[0].ID, "GW-567", ""); err != nil {
		t.Fatal(err)
	}
	m := newReportsModel(conn)
	mm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	m = mm.(model)
	if !strings.Contains(m.View(), "[GW-567]") {
		t.Error("в строке отчёта нет метки тега")
	}

	dir := t.TempDir()
	m.settings.cfg.saveDir = dir
	mm, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	m = mm.(model)
	data, err := os.ReadFile(filepath.Join(dir, m.reports.saveFileName()))
	if err != nil {
		t.Fatalf("файл отчёта не создан: %v", err)
	}
	if !strings.Contains(string(data), "[GW-567]") {
		t.Error("в файле отчёта нет метки тега")
	}
}

// TestSettingsTagTypesManage — раздел «Типы тегов» в настройках: список,
// создание, редактирование и удаление типа.
func TestSettingsTagTypesManage(t *testing.T) {
	conn, err := db.Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	m := newReportsModel(conn)
	if len(m.settings.tagTypes) != 2 {
		t.Fatalf("сид типов по умолчанию: %d, ожидалось 2", len(m.settings.tagTypes))
	}

	mm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	m = mm.(model)
	if !strings.Contains(m.View(), "Типы тегов: 2") {
		t.Error("нет строки «Типы тегов» в настройках")
	}
	for i := 0; i < 6; i++ {
		m.updateSettings(tea.KeyMsg{Type: tea.KeyDown})
	}
	if m.settings.sel != 6 {
		t.Fatalf("sel = %d, ожидался 6", m.settings.sel)
	}
	m.updateSettings(tea.KeyMsg{Type: tea.KeyEnter})
	if m.settings.mode != settingsTagTypeList {
		t.Fatalf("Enter не открыл список типов (mode=%d)", m.settings.mode)
	}
	if _, open := m.settings.dialog(); !open {
		t.Error("список типов не отрендерился")
	}

	// n — новый тип
	m.updateSettings(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	if m.settings.mode != settingsTagTypeEdit || m.settings.tagTypeEditID != 0 {
		t.Fatalf("n не открыл редактор (mode=%d)", m.settings.mode)
	}
	m.settings.editName.SetValue("Планфикс")
	m.updateSettings(tea.KeyMsg{Type: tea.KeyCtrlS})
	if m.settings.mode != settingsTagTypeList {
		t.Fatalf("Ctrl+S не сохранил тип (mode=%d)", m.settings.mode)
	}
	types, _ := db.TagTypes(conn)
	if len(types) != 3 || types[2].Name != "Планфикс" {
		t.Fatalf("тип не создан: %+v", types)
	}

	// Enter — редактирование выбранного (первого) типа
	m.updateSettings(tea.KeyMsg{Type: tea.KeyEnter})
	if m.settings.mode != settingsTagTypeEdit {
		t.Fatalf("Enter не открыл редактор (mode=%d)", m.settings.mode)
	}
	if m.settings.editName.Value() != types[0].Name {
		t.Errorf("префилл имени: %q", m.settings.editName.Value())
	}
	m.settings.editName.SetValue("Jira-2")
	m.updateSettings(tea.KeyMsg{Type: tea.KeyCtrlS})
	types, _ = db.TagTypes(conn)
	if types[0].Name != "Jira-2" {
		t.Errorf("имя не обновилось: %+v", types[0])
	}

	// d — удаление с подтверждением
	m.updateSettings(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	if m.settings.mode != settingsTagTypeConfirm {
		t.Fatalf("d не открыл подтверждение (mode=%d)", m.settings.mode)
	}
	m.updateSettings(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if m.settings.mode != settingsTagTypeList {
		t.Fatalf("y не вернул к списку (mode=%d)", m.settings.mode)
	}
	types, _ = db.TagTypes(conn)
	if len(types) != 2 {
		t.Errorf("тип не удалён: %d", len(types))
	}

	// Esc закрывает список
	m.updateSettings(tea.KeyMsg{Type: tea.KeyEsc})
	if m.settings.mode != settingsBrowse {
		t.Fatalf("Esc не закрыл список (mode=%d)", m.settings.mode)
	}
}
