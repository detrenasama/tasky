package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbletea"
	"github.com/detrenasama/tasky/internal/db"
	"os"

	"github.com/detrenasama/tasky/internal/store"
)

func key(r rune) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}} }

// TestTasksDescModalOpen — Enter открывает крупную модалку с текстом,
// Esc без изменений закрывает её.
func TestTasksDescModalOpen(t *testing.T) {
	conn, s, task, _ := tasksSeedProject(t)
	if err := db.UpdateTaskDescription(conn, task.ID, "текст описания"); err != nil {
		t.Fatal(err)
	}
	s.list.Select(0)
	s.loadDesc()
	s.startEditDescription()
	if s.mode != taskDescModal || s.dmState != dmView {
		t.Fatalf("модалка не открылась: mode=%d dm=%d", s.mode, s.dmState)
	}
	dlg, open := s.dialog()
	if !open || dlg == "" {
		t.Fatal("dialog модалки не открыт")
	}
	if !strings.Contains(dlg, "текст описания") {
		t.Errorf("модалка не показывает текст описания")
	}
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyEsc})
	if s.mode != taskBrowse {
		t.Fatalf("Esc без изменений должен выйти, mode=%d", s.mode)
	}
}

// TestTasksDescEditSave — в модалке e открывает правку, Ctrl+S сохраняет
// и остаётся в модалке (БД обновляется).
func TestTasksDescEditSave(t *testing.T) {
	conn, s, task, _ := tasksSeedProject(t)
	if err := db.UpdateTaskDescription(conn, task.ID, "исходное"); err != nil {
		t.Fatal(err)
	}
	s.list.Select(0)
	s.loadDesc()
	s.startEditDescription()
	s.updateTasksMsg(key('e'))
	if s.dmState != dmEdit {
		t.Fatalf("e не вошёл в правку, dm=%d", s.dmState)
	}
	s.descText.SetValue("изменённое")
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyCtrlS})
	if s.mode != taskDescModal || s.dmState != dmView {
		t.Fatalf("Ctrl+S должен сохранить и остаться в модалке, mode=%d dm=%d", s.mode, s.dmState)
	}
	if got, _ := db.TaskDescription(conn, task.ID); got != "изменённое" {
		t.Errorf("Ctrl+S не сохранил изменения: %q", got)
	}
}

// TestTasksDescDiscardByY — изменение + Esc открывает подтверждение, y
// отбрасывает изменения, но остаётся в модалке (БД не меняется).
func TestTasksDescDiscardByY(t *testing.T) {
	conn, s, task, _ := tasksSeedProject(t)
	if err := db.UpdateTaskDescription(conn, task.ID, "исходное"); err != nil {
		t.Fatal(err)
	}
	s.list.Select(0)
	s.loadDesc()
	s.startEditDescription()
	s.updateTasksMsg(key('e'))
	s.descText.SetValue("изменённое")
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyEsc})
	if s.dmState != dmDiscard {
		t.Fatalf("Esc с изменениями должен открыть подтверждение, dm=%d", s.dmState)
	}
	if d, open := s.dialog(); !open || !strings.Contains(d, "Несохранённые") {
		t.Errorf("подтверждение должно показываться, open=%v", open)
	}
	s.updateTasksMsg(key('y'))
	if s.mode != taskDescModal || s.dmState != dmView {
		t.Fatalf("y должен остаться в модалке, mode=%d dm=%d", s.mode, s.dmState)
	}
	if s.descWork != "исходное" {
		t.Errorf("y должен отбросить изменения, descWork=%q", s.descWork)
	}
	if got, _ := db.TaskDescription(conn, task.ID); got != "исходное" {
		t.Errorf("y не должен сохранять изменения: %q", got)
	}
}

// TestTasksDescDiscardCancel — n в подтверждении возвращает к правке.
func TestTasksDescDiscardCancel(t *testing.T) {
	conn, s, task, _ := tasksSeedProject(t)
	if err := db.UpdateTaskDescription(conn, task.ID, "исходное"); err != nil {
		t.Fatal(err)
	}
	s.list.Select(0)
	s.loadDesc()
	s.startEditDescription()
	s.updateTasksMsg(key('e'))
	s.descText.SetValue("изменённое")
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyEsc})
	s.updateTasksMsg(key('n'))
	if s.dmState != dmEdit {
		t.Fatalf("n должен вернуть к правке, dm=%d", s.dmState)
	}
	if s.mode != taskDescModal {
		t.Fatalf("n не должен выходить из модалки, mode=%d", s.mode)
	}
}

// TestTasksDescModalSelectDelete — v открывает выделение, d удаляет
// выделенное из рабочей копии (несохранённое), Esc подтверждает отмену.
func TestTasksDescModalSelectDelete(t *testing.T) {
	conn, s, task, _ := tasksSeedProject(t)
	if err := db.UpdateTaskDescription(conn, task.ID, "абвгде"); err != nil {
		t.Fatal(err)
	}
	s.list.Select(0)
	s.loadDesc()
	s.startEditDescription()
	s.updateTasksMsg(key('v'))
	if s.dmState != dmSelect {
		t.Fatalf("v должен войти в выделение, dm=%d", s.dmState)
	}
	s.descViewer.cursor = 1
	s.descViewer.anchor = 1
	s.descViewer.moveRight()
	s.descViewer.moveRight()
	if got := s.descViewer.selectedText(); got != "бв" {
		t.Errorf("выделение = %q, ожидалось «бв»", got)
	}
	s.updateTasksMsg(key('d'))
	if s.descWork != "агде" {
		t.Errorf("после удаления descWork = %q, ожидалось «агде»", s.descWork)
	}
	if s.dmState != dmView {
		t.Fatalf("после удаления dm=%d", s.dmState)
	}
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyEsc})
	if s.dmState != dmDiscard {
		t.Fatalf("Esc с несохранённым должен подтверждать, dm=%d", s.dmState)
	}
	s.updateTasksMsg(key('n'))
	if s.mode != taskDescModal {
		t.Fatalf("n не должен выходить, mode=%d", s.mode)
	}
	if got, _ := db.TaskDescription(conn, task.ID); got != "абвгде" {
		t.Errorf("БД не должна меняться без Ctrl+S: %q", got)
	}
}

// TestTasksDescModalEditor — Shift+E (rune 'E') открывает $EDITOR (mode 2);
// возврат записывает текст в рабочую копию; Ctrl+S сохраняет.
func TestTasksDescModalEditor(t *testing.T) {
	conn, s, task, _ := tasksSeedProject(t)
	if err := db.UpdateTaskDescription(conn, task.ID, "исходное"); err != nil {
		t.Fatal(err)
	}
	s.list.Select(0)
	s.loadDesc()
	s.startEditDescription()
	m := model{tasks: s}
	_, cmd := m.updateTasks(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'E'}})
	if cmd == nil {
		t.Fatal("ожидалась команда запуска редактора")
	}
	if s.extEditMode != 2 {
		t.Fatalf("extEditMode должен быть 2, =%d", s.extEditMode)
	}
	f, err := os.CreateTemp("", "tasky-ed-*.md")
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString("из редактора")
	f.Close()
	s.applyExternalEdit(editReturnMsg{path: f.Name(), err: nil})
	if s.descWork != "из редактора" {
		t.Errorf("после редактора descWork = %q", s.descWork)
	}
	if s.dmState != dmView {
		t.Fatalf("после редактора должен быть просмотр модалки, dm=%d", s.dmState)
	}
	if got, _ := db.TaskDescription(conn, task.ID); got != "из редактора" {
		t.Errorf("$EDITOR должен сразу сохранить текст в БД: %q", got)
	}
}

// TestTasksDescModalEditorUnchanged — если $EDITOR вернул тот же текст
// (просто закрыли без правок), БД не меняется и заметка не показывается.
func TestTasksDescModalEditorUnchanged(t *testing.T) {
	conn, s, task, _ := tasksSeedProject(t)
	if err := db.UpdateTaskDescription(conn, task.ID, "исходное"); err != nil {
		t.Fatal(err)
	}
	s.list.Select(0)
	s.loadDesc()
	s.startEditDescription()
	m := model{tasks: s}
	m.updateTasks(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'E'}})
	if s.extEditMode != 2 {
		t.Fatalf("extEditMode должен быть 2, =%d", s.extEditMode)
	}
	f, err := os.CreateTemp("", "tasky-ed-*.md")
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString("исходное")
	f.Close()
	s.applyExternalEdit(editReturnMsg{path: f.Name(), err: nil})
	if s.dmState != dmView {
		t.Fatalf("после редактора должен быть просмотр, dm=%d", s.dmState)
	}
	if s.notice != "" {
		t.Errorf("при отсутствии изменений заметка не нужна: %q", s.notice)
	}
	if got, _ := db.TaskDescription(conn, task.ID); got != "исходное" {
		t.Errorf("БД не должна меняться при отсутствии правок: %q", got)
	}
}

// TestProjectsDescModalOpen + save/discard (зеркало для проектов).
func TestProjectsDescDiscardByY(t *testing.T) {
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
	if m.proj.mode != projDescModal {
		t.Fatalf("модалка проекта не открылась: mode=%d", m.proj.mode)
	}
	m.updateProjects(key('e'))
	m.proj.descText.SetValue("изменённое")
	m.updateProjects(tea.KeyMsg{Type: tea.KeyEsc})
	if m.proj.dmState != dmDiscard {
		t.Fatalf("Esc должен подтверждать, dm=%d", m.proj.dmState)
	}
	m.updateProjects(key('y'))
	if m.proj.mode != projDescModal || m.proj.dmState != dmView {
		t.Fatalf("y должен остаться в модалке, mode=%d dm=%d", m.proj.mode, m.proj.dmState)
	}
	if m.proj.descWork != "исходное" {
		t.Errorf("y должен отбросить изменения, descWork=%q", m.proj.descWork)
	}
	if got, _ := db.ProjectDescription(conn, p.ID); got != "исходное" {
		t.Errorf("y не должен сохранять изменения: %q", got)
	}
}

// TestProjectsDescEditSave — Ctrl+S в модалке проекта сохраняет.
func TestProjectsDescEditSave(t *testing.T) {
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
	m.updateProjects(key('e'))
	m.proj.descText.SetValue("изменённое")
	m.updateProjects(tea.KeyMsg{Type: tea.KeyCtrlS})
	if m.proj.mode != projDescModal || m.proj.dmState != dmView {
		t.Fatalf("Ctrl+S должен сохранить и остаться, mode=%d dm=%d", m.proj.mode, m.proj.dmState)
	}
	if got, _ := db.ProjectDescription(conn, p.ID); got != "изменённое" {
		t.Errorf("Ctrl+S не сохранил: %q", got)
	}
}

// TestTasksDescModalCopyNotice — y в режиме просмотра копирует весь текст и
// устанавливает заметку (успех или ошибка инструмента буфера обмена).
func TestTasksDescModalCopyNotice(t *testing.T) {
	conn, s, task, _ := tasksSeedProject(t)
	if err := db.UpdateTaskDescription(conn, task.ID, "текст описания"); err != nil {
		t.Fatal(err)
	}
	s.list.Select(0)
	s.loadDesc()
	s.startEditDescription()
	if s.dmState != dmView {
		t.Fatal("модалка не открылась")
	}
	s.updateTasksMsg(key('y'))
	if s.notice == "" {
		t.Error("копирование должно установить заметку")
	}
	if s.dmState != dmView {
		t.Fatalf("в просмотре y не должен менять состояние, dm=%d", s.dmState)
	}
}

// TestTasksDescModalCopySelectionClears — v входит в выделение, Space ставит
// метку, движение расширяет, Enter копирует и сбрасывает выделение.
func TestTasksDescModalCopySelectionClears(t *testing.T) {
	conn, s, task, _ := tasksSeedProject(t)
	if err := db.UpdateTaskDescription(conn, task.ID, "абвгде"); err != nil {
		t.Fatal(err)
	}
	s.list.Select(0)
	s.loadDesc()
	s.startEditDescription()
	s.updateTasksMsg(key('v'))
	if s.dmState != dmSelect {
		t.Fatalf("v не вошёл в выделение, dm=%d", s.dmState)
	}
	if s.descViewer.anchor != -1 {
		t.Error("до Space выделения быть не должно (anchor == -1)")
	}
	s.updateTasksMsg(key(' '))
	if s.descViewer.anchor != s.descViewer.cursor {
		t.Error("Space должен поставить метку в позиции курсора")
	}
	s.updateTasksMsg(key('l'))
	if s.descViewer.selectedText() == "" {
		t.Error("после Space и движения выделение не должно быть пустым")
	}
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyEnter})
	if s.dmState != dmView {
		t.Fatalf("после копирования должен быть просмотр, dm=%d", s.dmState)
	}
	if s.notice == "" {
		t.Error("после копирования должна быть заметка")
	}
	if s.descViewer.anchor != -1 {
		t.Error("после копирования выделение должно сброситься")
	}
}

// TestTasksDescModalCopyYAlias — y остаётся алиасом копирования в режиме
// выделения (Space + движение + y копирует и очищает выделение).
func TestTasksDescModalCopyYAlias(t *testing.T) {
	conn, s, task, _ := tasksSeedProject(t)
	if err := db.UpdateTaskDescription(conn, task.ID, "абвгде"); err != nil {
		t.Fatal(err)
	}
	s.list.Select(0)
	s.loadDesc()
	s.startEditDescription()
	s.updateTasksMsg(key('v'))
	s.updateTasksMsg(key(' '))
	s.updateTasksMsg(key('l'))
	s.updateTasksMsg(key('y'))
	if s.dmState != dmView {
		t.Fatalf("после копирования y должен быть просмотр, dm=%d", s.dmState)
	}
	if s.notice == "" {
		t.Error("после копирования y должна быть заметка")
	}
	if s.descViewer.anchor != -1 {
		t.Error("после копирования y выделение должно сброситься")
	}
}

// TestProjectsDescModalCopySelectionClears — зеркало для проектов.
func TestProjectsDescModalCopySelectionClears(t *testing.T) {
	conn, err := db.Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	p, err := db.CreateProject(conn, "P")
	if err != nil {
		t.Fatal(err)
	}
	if err := db.UpdateProjectDescription(conn, p.ID, "проектное описание"); err != nil {
		t.Fatal(err)
	}
	m := model{proj: newProjectsScreen(store.NewSQLite(conn))}
	m.proj.load()
	m.proj.resize(150, 26)
	m.proj.list.Select(0)
	m.proj.loadDesc()
	m.proj.startEditDescription()
	m.updateProjects(key('v'))
	if m.proj.dmState != dmSelect {
		t.Fatalf("v не вошёл в выделение, dm=%d", m.proj.dmState)
	}
	if m.proj.descViewer.anchor != -1 {
		t.Error("до Space выделения быть не должно (anchor == -1)")
	}
	m.updateProjects(key(' '))
	if m.proj.descViewer.anchor != m.proj.descViewer.cursor {
		t.Error("Space должен поставить метку в позиции курсора")
	}
	m.updateProjects(key('l'))
	if m.proj.descViewer.selectedText() == "" {
		t.Error("после Space и движения выделение не должно быть пустым")
	}
	m.updateProjects(tea.KeyMsg{Type: tea.KeyEnter})
	if m.proj.dmState != dmView {
		t.Fatalf("после копирования должен быть просмотр, dm=%d", m.proj.dmState)
	}
	if m.proj.notice == "" {
		t.Error("после копирования должна быть заметка")
	}
	if m.proj.descViewer.anchor != -1 {
		t.Error("после копирования выделение должно сброситься")
	}
}

// TestDescViewerSelection — визуальное выделение возвращает подстроку.
func TestDescViewerSelection(t *testing.T) {
	v := newDescViewer("абв\nгде", 10, 5)
	if v.selectedText() != "абв\nгде" {
		t.Fatalf("без выделения копируется весь текст: %q", v.selectedText())
	}
	v.moveRight()
	v.toggleSelect()
	v.moveRight()
	v.moveRight()
	if got := v.selectedText(); got != "бв" {
		t.Errorf("выделение = %q, ожидалось «бв»", got)
	}
	v.toggleSelect()
	if v.selectedText() != "абв\nгде" {
		t.Errorf("после снятия выделения копируется весь текст: %q", v.selectedText())
	}
	if v.view() == "" {
		t.Error("view не должен быть пустым")
	}
	// длинный текст не должен паниковать при рендере
	long := newDescViewer(strings.Repeat("строка описания ", 200), 40, 10)
	_ = long.view()
}

// TestCopyClipboardEmpty — пустой текст даёт ошибку.
func TestCopyClipboardEmpty(t *testing.T) {
	if err := copyToClipboard(""); err == nil {
		t.Error("ожидалась ошибка для пустого текста")
	}
}

// TestDescriptionNotClipped — полный рендер (через app.View) показывает хвост
// длинного описания; регрессия усечения нижних строк колонки описания.
// Ресайз идёт через реальный путь WindowSizeMsg (как в app.go).
func TestDescriptionNotClipped(t *testing.T) {
	conn, task, _ := reportsSeedProject(t)
	// достаточно длинное описание, чтобы уйти за нижнюю границу видимой
	// области viewport и оказаться в зоне усечения
	long := strings.Repeat("описание задачи длинное ", 200) + "МАРКЕР_КОНЕЦ"
	if err := db.UpdateTaskDescription(conn, task.ID, long); err != nil {
		t.Fatal(err)
	}
	m := newReportsModel(conn)
	mm, _ := m.Update(tea.WindowSizeMsg{Width: 200, Height: 40})
	m = mm.(model)
	m.tasks.list.Select(0)
	m.tasks.loadDesc()
	m.tasks.descV.GotoBottom()
	out := m.View()
	if !strings.Contains(out, "МАРКЕР_КОНЕЦ") {
		t.Errorf("описание обрезается: хвост «МАРКЕР_КОНЕЦ» не виден в рендере")
	}
}
