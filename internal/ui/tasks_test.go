package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/detrenasama/tasky/internal/db"
	"github.com/detrenasama/tasky/internal/store"
	"github.com/detrenasama/tasky/internal/ui/theme"
	"github.com/muesli/termenv"
)

func TestResizeColumns(t *testing.T) {
	s := newTestTasksScreen(t)
	cases := []struct{ w, list, desc int }{
		{150, 74, 74},
		{110, 54, 54},
		{109, 53, 54},
		{90, 90, 0},
		{69, 69, 0},
	}
	for _, c := range cases {
		s.resize(c.w, 26)
		if s.listW != c.list || s.descW != c.desc {
			t.Errorf("w=%d: list=%d desc=%d, ожидались %d/%d",
				c.w, s.listW, s.descW, c.list, c.desc)
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
	lipgloss.SetColorProfile(termenv.ANSI256)
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
	s := newTasksScreen(store.NewSQLite(conn))
	s.load()
	s.resize(150, 26)
	// две колонки равной ширины: 74 + 2 + 74 = 150; заголовок страницы «Задачи»
	// сверху слева, каждая панель ровно своей ширины, разделитель — 2 пробела.
	out := s.view(150, 26)
	rows := strings.Split(out, "\n")
	if len(rows) != 26 {
		t.Fatalf("строк %d, ожидалось 26", len(rows))
	}
	for i, r := range rows {
		if lipgloss.Width(r) != 150 {
			t.Errorf("строка %d шириной %d, ожидалось 150", i, lipgloss.Width(r))
		}
	}
	if !strings.Contains(out, "Задачи") {
		t.Error("нет заголовка страницы «Задачи»")
	}
	if s.listW+s.descW+2 != 150 {
		t.Errorf("сумма колонок %d, ожидалось 150", s.listW+s.descW+2)
	}
}

func TestInfoBottomVisible(t *testing.T) {
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
	s := newTasksScreen(store.NewSQLite(conn))
	s.load()
	s.resize(150, 26)
	// правая колонка (ранее info-колонка) рендерится отдельно через
	// rightContent; проверяем её содержимое непосредственно.
	rc := s.rightContent(24)
	if !strings.Contains(stripANSI(rc), "Ничего не запущено.") {
		t.Error("в правой колонке внизу нет строки о запущенной подзадаче")
	}
}

// tasksSeedProject создаёт проект с задачей и подзадачей и возвращает
// инициализированный tasksScreen.

// TestTasksDescBox — колонка описания: для задачи описание и ссылки, для
// подзадачи — блоки «Описание» и «Журнал» с записью.
// TestHideCompletedTasks — задачи в завершённом статусе скрываются спустя
// N дней (настройка hide_days), поиск находит скрытые, выход из done
// возвращает задачу, подзадачи не фильтруются.

func TestHideCompletedTasks(t *testing.T) {
	conn, err := db.Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	p, err := db.CreateProject(conn, "P")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	mk := func(title string) db.Task {
		tk, err := db.CreateTask(conn, p.ID, title)
		if err != nil {
			t.Fatal(err)
		}
		return tk
	}
	old := mk("Старая")
	recent := mk("Свежая")
	reopened := mk("Возвращена")
	if err := db.SetStatus(conn, db.OwnerTask, old.ID, "Выполнена", "", now.Add(-8*24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := db.SetStatus(conn, db.OwnerTask, recent.ID, "Выполнена", "", now.Add(-2*24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := db.SetStatus(conn, db.OwnerTask, reopened.ID, "Выполнена", "", now.Add(-8*24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := db.SetStatus(conn, db.OwnerTask, reopened.ID, "Новая", "", now.Add(-1*time.Hour)); err != nil {
		t.Fatal(err)
	}

	s := newTasksScreen(store.NewSQLite(conn))
	s.now = now
	s.load()
	byID := map[int64]db.Task{}
	for _, t2 := range s.tasks {
		byID[t2.ID] = t2
	}
	if !s.hiddenDue(byID[old.ID]) {
		t.Error("задача done 8 дней назад не помечена скрытой")
	}
	if s.hiddenDue(byID[recent.ID]) {
		t.Error("задача done 2 дня назад помечена скрытой")
	}
	if s.hiddenDue(byID[reopened.ID]) {
		t.Error("задача, вернувшаяся из done, помечена скрытой")
	}
	tt := searchTitles(s)
	if containsString(tt, "Старая") {
		t.Errorf("скрытая задача в списке: %v", tt)
	}
	if !containsString(tt, "Свежая") || !containsString(tt, "Возвращена") {
		t.Errorf("видимые задачи пропали: %v", tt)
	}

	// поиск находит скрытую задачу
	s.searchQuery = "Старая"
	s.buildItems()
	if !containsString(searchTitles(s), "Старая") {
		t.Error("поиск не нашёл скрытую задачу")
	}

	// hideDays=0 — скрытие выключено
	s.searchQuery = ""
	s.hideDays = 0
	s.buildItems()
	if !containsString(searchTitles(s), "Старая") {
		t.Error("при hideDays=0 задача осталась скрытой")
	}

	// подзадачи не фильтруются: done-подзадача под видимой задачей
	st, err := db.CreateSubtask(conn, recent.ID, "Подзадача")
	if err != nil {
		t.Fatal(err)
	}
	if err := db.SetStatus(conn, db.OwnerSubtask, st.ID, "Выполнена", "", now.Add(-30*24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	s.searchQuery = ""
	s.hideDays = 7
	s.expanded[recent.ID] = true
	s.loadData()
	if !containsString(searchTitles(s), "Подзадача") {
		t.Error("done-подзадача пропала из списка")
	}

	// граница: ровно N дней — уже скрыта, чуть меньше — видна
	s.hideDays = 7
	bound := db.Task{
		Title:       "Граница",
		Status:      "Выполнена",
		CompletedAt: ptrTime(now.Add(-7 * 24 * time.Hour)),
	}
	s.now = now
	if !s.hiddenDue(bound) {
		t.Error("ровно N дней назад не скрыто")
	}
	bound.CompletedAt = ptrTime(now.Add(-7*24*time.Hour + time.Hour))
	if s.hiddenDue(bound) {
		t.Error("меньше N дней скрыто")
	}
	// статус не завершённого типа — не скрывается
	bound.Status = "В работе"
	bound.CompletedAt = nil
	if s.hiddenDue(bound) {
		t.Error("задача в работе скрыта")
	}
}

func containsString(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}

func ptrTime(t time.Time) *time.Time { return &t }

// TestTaskSearch — / открывает модалку поиска, фильтр по журналу/описанию/
// названию применяется по мере ввода, Enter оставляет фильтр, Esc сбрасывает.
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
	m.updateTasks(tea.KeyMsg{Type: tea.KeyRight}) // раскрыть задачу
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

// TestTasksExpandCollapse — → раскрывает задачу (видна подзадача), ← сворачивает.

func TestTasksExpandCollapse(t *testing.T) {
	_, s, _, _ := tasksSeedProject(t)

	if containsString(searchTitles(s), "S") {
		t.Fatal("подзадача видна до раскрытия")
	}
	// → раскрывает
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyRight})
	if !containsString(searchTitles(s), "S") {
		t.Fatal("→ не раскрыл задачу (нет подзадачи в списке)")
	}
	// ← сворачивает
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyLeft})
	if containsString(searchTitles(s), "S") {
		t.Fatal("← не свернул задачу (подзадача осталась в списке)")
	}
}

// TestTasksDescKeysOnlyInDescFocus — e/ctrl+j/l/o работают только в фокусе
// описания; инлайн-редактирование сохраняется по Ctrl+S, отменяется по Esc.

func TestTasksDescEdit(t *testing.T) {
	conn, s, task, _ := tasksSeedProject(t)
	m := &model{tasks: s, proj: newProjectsScreen(store.NewSQLite(conn))}
	runes := func(r rune) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}} }

	// e открывает модалку изменения названия (из списка, без фокуса)
	m.updateTasks(runes('e'))
	if s.mode != taskTitleEdit {
		t.Fatal("e не открыл изменение названия")
	}
	m.updateTasks(tea.KeyMsg{Type: tea.KeyEsc})
	if s.mode != taskBrowse {
		t.Fatal("Esc не закрыл модалку названия")
	}
	// ctrl+j для выбранной задачи (не подзадачи) — ничего не открывает
	before := s.mode
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlJ})
	if cmd != nil || s.mode != before {
		t.Error("ctrl+j для задачи открыл модалку или вернул команду")
	}

	// Enter открывает крупную модалку описания
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyEnter})
	if s.mode != taskDescModal || s.dmState != dmView {
		t.Fatal("Enter не открыл модалку описания")
	}
	s.updateTasksMsg(key('e'))
	if s.dmState != dmEdit {
		t.Fatal("e не вошёл в правку описания")
	}
	s.descText.SetValue("новое описание")
	dlg, open := s.dialog()
	if !open || !strings.Contains(dlg, "новое описание") {
		t.Error("при правке в модалке не виден textarea")
	}
	if strings.Contains(dlg, "Несохранённые") {
		t.Error("правка описания не должна открывать подтверждение")
	}
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyCtrlS})
	if s.mode != taskDescModal || s.dmState != dmView {
		t.Fatal("Ctrl+S должен остаться в модалке")
	}
	if got, _ := db.TaskDescription(conn, task.ID); got != "новое описание" {
		t.Errorf("описание в БД = %q", got)
	}

	// Esc с несохранёнными изменениями — подтверждение, а не выход
	s.updateTasksMsg(key('e'))
	s.descText.SetValue("не сохранять")
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyEsc})
	if s.dmState != dmDiscard {
		t.Fatalf("Esc с изменениями должен открыть подтверждение, dm=%d", s.dmState)
	}
	if got, _ := db.TaskDescription(conn, task.ID); got != "новое описание" {
		t.Errorf("подтверждение не должно сохранять: %q", got)
	}
	// Esc в подтверждении — возврат к правке
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyEsc})
	if s.dmState != dmEdit {
		t.Fatalf("Esc в подтверждении должен вернуть к правке, dm=%d", s.dmState)
	}
	// неизменённое описание — Esc из правки в просмотр, затем выход
	s.descText.SetValue("новое описание")
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyEsc})
	if s.dmState != dmView {
		t.Fatalf("Esc без изменений должен выйти в просмотр, dm=%d", s.dmState)
	}
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyEsc})
	if s.mode != taskBrowse {
		t.Fatalf("Esc без изменений должен выйти, mode=%d", s.mode)
	}

	// скролл описания по PgDn в обзоре
	if err := db.UpdateTaskDescription(conn, task.ID, strings.Repeat("строка длинного описания ", 100)); err != nil {
		t.Fatal(err)
	}
	s.loadDesc()
	y0 := s.descV.YOffset
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyPgDown})
	if s.descV.YOffset <= y0 {
		t.Errorf("PgDn не проскроллил описание: %d → %d", y0, s.descV.YOffset)
	}
}

// TestTasksLinkAddFlow — добавление ссылки задачи: одна модалка с двумя
// инпутами (название → адрес), Tab/Enter, Esc — отмена.
func TestTasksTitleEdit(t *testing.T) {
	conn, s, task, st := tasksSeedProject(t)
	m := &model{tasks: s}
	runes := func(r rune) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}} }
	titleOf := func() string {
		var title string
		if err := conn.QueryRow("SELECT title FROM tasks WHERE id = ?", task.ID).Scan(&title); err != nil {
			t.Fatal(err)
		}
		return title
	}
	subtitleOf := func() string {
		var title string
		if err := conn.QueryRow("SELECT title FROM subtasks WHERE id = ?", st.ID).Scan(&title); err != nil {
			t.Fatal(err)
		}
		return title
	}

	// e в фокусе списка открывает модалку с префилленным названием
	s.updateTasksMsg(runes('e'))
	if s.mode != taskTitleEdit {
		t.Fatalf("e не открыл изменение названия (mode=%d)", s.mode)
	}
	if _, open := s.dialog(); !open {
		t.Error("изменение названия не рендерится как модалка")
	}
	if s.input.Value() != task.Title {
		t.Errorf("input не префиллен: %q", s.input.Value())
	}

	// Esc отменяет — название не меняется
	s.input.SetValue("Не сохранять")
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyEsc})
	if s.mode != taskBrowse {
		t.Fatalf("Esc не закрыл модалку (mode=%d)", s.mode)
	}
	if titleOf() != "T" {
		t.Errorf("Esc сохранил название: %q", titleOf())
	}

	// пустое название — ошибка, модалка остаётся
	s.updateTasksMsg(runes('e'))
	s.input.SetValue("   ")
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyEnter})
	if s.mode != taskTitleEdit {
		t.Fatalf("Enter с пустым названием закрыл модалку (mode=%d)", s.mode)
	}
	if s.lastErr == nil {
		t.Error("для пустого названия не выставлена ошибка")
	}
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyEsc})

	// Enter сохраняет новое название задачи
	s.updateTasksMsg(runes('e'))
	s.input.SetValue("Переименованная")
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyEnter})
	if s.mode != taskBrowse {
		t.Fatalf("Enter не сохранил название (mode=%d)", s.mode)
	}
	if titleOf() != "Переименованная" {
		t.Errorf("название в БД = %q", titleOf())
	}
	if !containsString(searchTitles(s), "Переименованная") {
		t.Errorf("список не обновился: %v", searchTitles(s))
	}

	// подзадача: раскрыть задачу, на подзадачу, e → Enter
	selectFirstSubtask(m)
	s.updateTasksMsg(runes('e'))
	if s.inputKind != kindSubtask || s.input.Value() != "S" {
		t.Fatalf("модалка подзадачи: kind=%d value=%q", s.inputKind, s.input.Value())
	}
	s.input.SetValue("Переименованная подзадача")
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyEnter})
	if subtitleOf() != "Переименованная подзадача" {
		t.Errorf("название подзадачи в БД = %q", subtitleOf())
	}
	if !containsString(searchTitles(s), "Переименованная подзадача") {
		t.Errorf("список не обновился: %v", searchTitles(s))
	}
}

// TestTasksJournalEscDoesNotQuit — регрессия: Esc из модалки журнала/ссылок
// закрывает модалку, а не выходит из приложения.
func TestTaskCreateRefreshesInfo(t *testing.T) {
	conn, s, task, _ := tasksSeedProject(t)
	// у первой задачи есть история статусов
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	if plain := stripANSI(s.infoTop(20)); !strings.Contains(plain, "Новая → В работе") {
		t.Fatalf("нет истории у первой задачи: %q", plain)
	}

	// создаём новую задачу — курсор переходит на неё
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	s.input.SetValue("Новая задача")
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyEnter})
	kind, id := s.selectedKindID()
	if kind != kindTask || id == task.ID {
		t.Fatalf("не выбрана новая задача: kind=%d id=%d", kind, id)
	}
	plain := stripANSI(s.infoTop(20))
	if strings.Contains(plain, "Новая → В работе") {
		t.Errorf("info-панель показывает историю старой задачи: %q", plain)
	}
	hist, _ := db.StatusHistory(conn, db.OwnerTask, id)
	if len(hist) != 0 {
		t.Errorf("история новой задачи: %+v", hist)
	}

	// создаём подзадачу под задачей с историей
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	s.input.SetValue("Новая подзадача")
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyEnter})
	kind, id = s.selectedKindID()
	if kind != kindSubtask {
		t.Fatalf("не выбрана новая подзадача: kind=%d id=%d", kind, id)
	}
	if plain = stripANSI(s.infoTop(20)); strings.Contains(plain, "Новая → В работе") {
		t.Errorf("info-панель подзадачи показывает историю задачи: %q", plain)
	}
}

// TestSettingsStatusesManage — настройки статусов: просмотр, создание,
// удаление неиспользуемого и запрет удаления используемого.

// TestPaneBgNotClipped — после цветного фрагмента контента (muted-подпись,
// статус, выделенный элемент) фон панели восстанавливается: между внутренним
// \x1b[0m и концом строки не должно быть «голых» пробелов.
func TestPaneBgNotClipped(t *testing.T) {
	lipgloss.SetColorProfile(termenv.ANSI256)
	probe := lipgloss.NewStyle().Background(theme.Pane(false).GetBackground()).Render("§")
	bgSeq := strings.Split(probe, "§")[0]
	if bgSeq == "" {
		t.Fatal("фон панели не эмитит ANSI (нужен цветовой профиль)")
	}
	pad := strings.Repeat(" ", 10)
	out := renderPane(theme.Pane(false), theme.Faint("метка")+pad)
	// паддинг после цветного фрагмента должен остаться под фоном панели,
	// а не «оголиться» до цвета терминала
	if !strings.Contains(out, bgSeq+pad) {
		t.Errorf("паддинг без фона после внутреннего сброса: %q", out)
	}
	// каждый внутренний сброс сопровождается восстановлением фона
	if n := strings.Count(out, "\x1b[0m"+bgSeq); n < 2 {
		t.Errorf("фон не восстановлен после внутренних сбросов: %d, строка %q", n, out)
	}
	// метка (5) + паддинг (10) + паддинг панели (2)
	if got := lipgloss.Width(out); got != 17 {
		t.Errorf("ширина %d, ожидалось 17 (метка + паддинг + хром панели)", got)
	}
}

// TestTasksMoveSelected — Ctrl+↑/↓ перемещает выбранную задачу/подзадачу и
// сохраняет выделение; на границах порядок не меняется.
func TestTasksMoveSelected(t *testing.T) {
	conn, err := db.Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	p, err := db.CreateProject(conn, "P")
	if err != nil {
		t.Fatal(err)
	}
	t1, err := db.CreateTask(conn, p.ID, "T1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.CreateTask(conn, p.ID, "T2"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.CreateTask(conn, p.ID, "T3"); err != nil {
		t.Fatal(err)
	}
	s := newTasksScreen(store.NewSQLite(conn))
	s.load()

	// первая задача вниз: порядок меняется, выделение сохраняется
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyCtrlDown})
	if got := searchTitles(s); !equalStrings(got, []string{"T2", "T1", "T3"}) {
		t.Fatalf("после ctrl+down: %v", got)
	}
	if kind, id := s.selectedKindID(); kind != kindTask || id != t1.ID {
		t.Fatalf("выделение не сохранено: kind=%d id=%d", kind, id)
	}

	// вверх возвращает на место
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyCtrlUp})
	if got := searchTitles(s); !equalStrings(got, []string{"T1", "T2", "T3"}) {
		t.Fatalf("после ctrl+up: %v", got)
	}

	// граница: первая вверх — no-op
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyCtrlUp})
	if got := searchTitles(s); !equalStrings(got, []string{"T1", "T2", "T3"}) {
		t.Fatalf("граница изменила порядок: %v", got)
	}

	// подзадачи: раскрыть T1, выбрать первую подзадачу и опустить
	st1, err := db.CreateSubtask(conn, t1.ID, "S1")
	if err != nil {
		t.Fatal(err)
	}
	st2, err := db.CreateSubtask(conn, t1.ID, "S2")
	if err != nil {
		t.Fatal(err)
	}
	s.load()
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyRight}) // раскрыть T1
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyDown})  // на S1
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyCtrlDown})
	var got []int64
	for _, st := range s.subs {
		if st.TaskID == t1.ID {
			got = append(got, st.ID)
		}
	}
	if len(got) != 2 || got[0] != st2.ID || got[1] != st1.ID {
		t.Fatalf("порядок подзадач: %v", got)
	}
	if kind, id := s.selectedKindID(); kind != kindSubtask || id != st1.ID {
		t.Fatalf("выделение подзадачи не сохранено: kind=%d id=%d", kind, id)
	}
}

// TestTasksMoveSubtaskAfterOtherTasks — подзадачи задачи, стоящей в проекте не
// первой, перемещаются корректно (индексы в s.subs смещены подзадачами ранних
// задач; границы считаются только по сиблингам родителя).
func TestTasksMoveSubtaskAfterOtherTasks(t *testing.T) {
	conn, err := db.Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close() })
	p, err := db.CreateProject(conn, "P")
	if err != nil {
		t.Fatal(err)
	}
	t0, err := db.CreateTask(conn, p.ID, "T0")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.CreateSubtask(conn, t0.ID, "Z"); err != nil {
		t.Fatal(err)
	}
	t1, err := db.CreateTask(conn, p.ID, "T1")
	if err != nil {
		t.Fatal(err)
	}
	s1, err := db.CreateSubtask(conn, t1.ID, "S1")
	if err != nil {
		t.Fatal(err)
	}
	s2, err := db.CreateSubtask(conn, t1.ID, "S2")
	if err != nil {
		t.Fatal(err)
	}
	s := newTasksScreen(store.NewSQLite(conn))
	s.load()

	// T0 → T1, раскрыть, встать на S1, опустить
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyDown})
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyRight})
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyDown})
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyCtrlDown})
	// s.subs: [Z, S2, S1]; выделение на S1
	var got []int64
	for _, st := range s.subs {
		if st.TaskID == t1.ID {
			got = append(got, st.ID)
		}
	}
	if len(got) != 2 || got[0] != s2.ID || got[1] != s1.ID {
		t.Fatalf("после ctrl+down: %v, ожидались S2, S1", got)
	}
	if kind, id := s.selectedKindID(); kind != kindSubtask || id != s1.ID {
		t.Fatalf("выделение не сохранено: kind=%d id=%d", kind, id)
	}
	// подзадачи ранней задачи не затронуты
	var z []int64
	for _, st := range s.subs {
		if st.TaskID == t0.ID {
			z = append(z, st.ID)
		}
	}
	if len(z) != 1 {
		t.Fatalf("подзадачи T0 изменились: %v", z)
	}

	// вверх возвращает на место
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyCtrlUp})
	got = got[:0]
	for _, st := range s.subs {
		if st.TaskID == t1.ID {
			got = append(got, st.ID)
		}
	}
	if len(got) != 2 || got[0] != s1.ID || got[1] != s2.ID {
		t.Fatalf("после ctrl+up: %v, ожидались S1, S2", got)
	}

	// границы: первая вверх — no-op
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyCtrlUp})
	got = got[:0]
	for _, st := range s.subs {
		if st.TaskID == t1.ID {
			got = append(got, st.ID)
		}
	}
	if len(got) != 2 || got[0] != s1.ID || got[1] != s2.ID {
		t.Fatalf("первая вверх изменила порядок: %v", got)
	}
	// опустить S1 в самый низ, затем вниз — no-op
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyCtrlDown})
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyCtrlDown})
	got = got[:0]
	for _, st := range s.subs {
		if st.TaskID == t1.ID {
			got = append(got, st.ID)
		}
	}
	if len(got) != 2 || got[0] != s2.ID || got[1] != s1.ID {
		t.Fatalf("последняя вниз изменила порядок: %v", got)
	}
}
