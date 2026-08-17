package ui

import (
	"database/sql"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/detrenasama/tasky/internal/db"
	"github.com/detrenasama/tasky/internal/ui/theme"
	"github.com/muesli/termenv"
)

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
	s := newTasksScreen(conn)
	s.load()
	s.resize(150, 26)
	// три колонки: 59 + 2 + 58 + 2 + 29 = 150; каждая панель ровно своей
	// ширины, разделители — 2 пробела
	row := strings.Split(s.view(150, 26), "\n")[0]
	if got := lipgloss.Width(row); got != 150 {
		t.Fatalf("ширина строки %d, ожидалось 150", got)
	}
	// граница панели: сброс ANSI + восстановленный фон, 2 пробела-разделителя,
	// фон следующей панели
	probe := lipgloss.NewStyle().Background(theme.Pane(false).GetBackground()).Render("§")
	bgSeq := strings.Split(probe, "§")[0]
	if bgSeq == "" {
		t.Fatal("фон панели не эмитит ANSI (нужен цветовой профиль)")
	}
	sep := bgSeq + "  " + bgSeq
	var idx []int
	from := 0
	for {
		i := strings.Index(row[from:], sep)
		if i < 0 {
			break
		}
		idx = append(idx, from+i)
		from += i + len(sep)
	}
	if len(idx) != 2 {
		t.Fatalf("границ панелей: %d, ожидалось 2", len(idx))
	}
	segs := []string{
		row[:idx[0]],
		row[idx[0]+len(bgSeq)+2 : idx[1]],
		row[idx[1]+len(bgSeq)+2:],
	}
	want := []int{s.listW, s.descW, s.infoW}
	for i, seg := range segs {
		if got := lipgloss.Width(seg); got != want[i] {
			t.Errorf("колонка %d: ширина %d, ожидалось %d", i, got, want[i])
		}
	}
	if s.listW+s.descW+s.infoW+4 != 150 {
		t.Errorf("сумма колонок %d, ожидалось 150", s.listW+s.descW+s.infoW+4)
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
	s := newTasksScreen(conn)
	s.load()
	s.resize(150, 26)
	rows := strings.Split(s.view(150, 26), "\n")
	if len(rows) != 26 {
		t.Fatalf("строк %d, ожидалось 26", len(rows))
	}
	if got := lipgloss.Width(rows[25]); got != 150 {
		t.Errorf("последняя строка шириной %d, ожидалось 150", got)
	}
	// подвал info-колонки (состояние запущенной подзадачи) виден внизу
	tail := stripANSI(strings.Join(rows[len(rows)-4:], "\n"))
	if !strings.Contains(tail, "Ничего не запущено.") {
		t.Error("в info-колонке внизу нет строки о запущенной подзадаче")
	}
}

// tasksSeedProject создаёт проект с задачей и подзадачей и возвращает
// инициализированный tasksScreen.

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

	s := newTasksScreen(conn)
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
	s := newTasksScreen(conn)
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

// searchTitles собирает названия элементов дерева списка задач.

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

	// e в фокусе списка открывает модалку изменения названия
	m.updateTasks(runes('e'))
	if s.mode != taskTitleEdit {
		t.Fatal("e в фокусе списка не открыл изменение названия")
	}
	m.updateTasks(tea.KeyMsg{Type: tea.KeyEsc})
	if s.mode != taskBrowse {
		t.Fatal("Esc не закрыл модалку названия")
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

// TestTasksTitleEdit — e в фокусе списка открывает модалку изменения названия:
// input префиллен текущим названием, Enter сохраняет (задачи и подзадачи),
// Esc отменяет, пустое название показывает ошибку и не закрывает модалку.

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

func TestTaskStatusQuickCycle(t *testing.T) {
	conn, s, task, _ := tasksSeedProject(t)
	if len(s.statuses) != 8 {
		t.Fatalf("статусов %d, ожидалось 8", len(s.statuses))
	}
	down := func() {
		s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	}
	statusOf := func() string {
		var st string
		if err := conn.QueryRow("SELECT status FROM tasks WHERE id = ?", task.ID).Scan(&st); err != nil {
			t.Fatal(err)
		}
		return st
	}

	down() // В работе
	if statusOf() != "В работе" {
		t.Fatalf("после x статус %q", statusOf())
	}
	down() // На проверке
	down() // Выполнена
	if statusOf() != "Выполнена" {
		t.Fatalf("после x×3 статус %q", statusOf())
	}
	var completed sql.NullInt64
	if err := conn.QueryRow("SELECT completed_at FROM tasks WHERE id = ?", task.ID).Scan(&completed); err != nil {
		t.Fatal(err)
	}
	if !completed.Valid {
		t.Error("completed_at не выставлен для Выполнена")
	}

	down() // без зацикливания: остаёмся на Выполнена
	if statusOf() != "Выполнена" {
		t.Fatalf("x с Выполнена зациклился: %q", statusOf())
	}

	// z назад: На проверке, completed_at очищен
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'z'}})
	if statusOf() != "На проверке" {
		t.Fatalf("после z статус %q", statusOf())
	}
	if err := conn.QueryRow("SELECT completed_at FROM tasks WHERE id = ?", task.ID).Scan(&completed); err != nil {
		t.Fatal(err)
	}
	if completed.Valid {
		t.Error("completed_at не очищен при выходе из Выполнена")
	}

	// статус вне цепочки: x прыгает на первый элемент
	db.SetStatus(conn, db.OwnerTask, task.ID, "Отменена", "", time.Now())
	s.loadData()
	down()
	if statusOf() != "Новая" {
		t.Fatalf("x из внецепочного статуса: %q", statusOf())
	}

	// полоса и цветной статус в списке
	plain := stripANSI(s.list.View())
	if !strings.Contains(plain, "Новая · ") {
		t.Errorf("в списке нет статуса: %q", plain)
	}

	// быстрый возврат к исходному статусу очищает историю («Новая → Новая»
	// не пишется)
	hist, _ := db.StatusHistory(conn, db.OwnerTask, task.ID)
	if len(hist) != 0 {
		t.Errorf("возврат к «Новой» оставил записи истории: %+v", hist)
	}
}

// TestTaskStatusPickAndNote — c открывает модалку выбора, «Делегирована»
// требует заметку (имя коллеги), переход пишется в журнал подзадачи.

func TestTaskStatusPickAndNote(t *testing.T) {
	conn, s, _, st := tasksSeedProject(t)
	m := &model{tasks: s}
	selectFirstSubtask(m)

	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	if s.mode != taskStatusPick {
		t.Fatalf("c не открыл выбор статуса (mode=%d)", s.mode)
	}
	if _, open := s.dialog(); !open {
		t.Error("выбор статуса не рендерится как модалка")
	}
	// преселект на текущем статусе («Новая»)
	if s.statusPick.sel != 0 {
		t.Errorf("преселект курсора: %d", s.statusPick.sel)
	}

	// ↓ до «Делегирована» (индекс 5) и Enter — модалка заметки
	for i := 0; i < 5; i++ {
		s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyDown})
	}
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyEnter})
	if s.mode != taskStatusNote {
		t.Fatalf("Делегирована не открыла заметку (mode=%d)", s.mode)
	}
	if _, open := s.dialog(); !open {
		t.Error("заметка не рендерится как модалка")
	}

	// Esc отменяет — статус не меняется
	s.statusNote.SetValue("Иван")
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyEsc})
	if s.mode != taskBrowse {
		t.Fatalf("Esc не отменил заметку (mode=%d)", s.mode)
	}
	var status string
	if err := conn.QueryRow("SELECT status FROM subtasks WHERE id = ?", st.ID).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "Новая" {
		t.Fatalf("отменённый переход применился: %q", status)
	}

	// снова: Делегирована + Ctrl+S с именем коллеги
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	for i := 0; i < 5; i++ {
		s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyDown})
	}
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyEnter})
	s.statusNote.SetValue("Иван Петров")
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyCtrlS})
	if s.mode != taskBrowse {
		t.Fatalf("Ctrl+S не применил статус (mode=%d)", s.mode)
	}
	if err := conn.QueryRow("SELECT status FROM subtasks WHERE id = ?", st.ID).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "Делегирована" {
		t.Fatalf("статус после заметки: %q", status)
	}
	hist, _ := db.StatusHistory(conn, db.OwnerSubtask, st.ID)
	if len(hist) != 1 || hist[0].From != "Новая" || hist[0].To != "Делегирована" ||
		hist[0].Note != "Иван Петров" {
		t.Errorf("запись истории: %+v", hist)
	}
	entries, _ := db.JournalEntries(conn, st.ID)
	if len(entries) != 0 {
		t.Errorf("журнал не должен содержать переход статуса: %+v", entries)
	}

	// повторный выбор того же статуса с другим именем — замена записи
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyEnter}) // преселект на «Делегирована»
	if s.mode != taskStatusNote {
		t.Fatalf("повторная Делегирована не открыла заметку (mode=%d)", s.mode)
	}
	s.statusNote.SetValue("Мария Сидорова")
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyCtrlS})
	hist, _ = db.StatusHistory(conn, db.OwnerSubtask, st.ID)
	if len(hist) != 1 || hist[0].Note != "Мария Сидорова" ||
		hist[0].From != "Новая" || hist[0].To != "Делегирована" {
		t.Errorf("запись после замены имени: %+v", hist)
	}

	// повторный ввод того же имени — запись не создаётся
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyEnter})
	s.statusNote.SetValue("Мария Сидорова")
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyCtrlS})
	hist, _ = db.StatusHistory(conn, db.OwnerSubtask, st.ID)
	if len(hist) != 1 {
		t.Errorf("повторный ввод имени создал запись: %+v", hist)
	}
}

// TestTaskStatusSubtaskHistory — быстрый переход подзадачи пишется в
// status_history и виден в info-панели.

func TestTaskStatusSubtaskHistory(t *testing.T) {
	conn, s, _, st := tasksSeedProject(t)
	m := &model{tasks: s}
	selectFirstSubtask(m)

	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	hist, _ := db.StatusHistory(conn, db.OwnerSubtask, st.ID)
	if len(hist) != 1 || hist[0].From != "Новая" || hist[0].To != "В работе" {
		t.Fatalf("история: %+v", hist)
	}
	entries, _ := db.JournalEntries(conn, st.ID)
	if len(entries) != 0 {
		t.Fatalf("журнал не должен содержать переход статуса: %+v", entries)
	}
	plain := stripANSI(s.infoTop(20))
	for _, want := range []string{"История статусов:", "Новая → В работе"} {
		if !strings.Contains(plain, want) {
			t.Errorf("в info нет %q", want)
		}
	}
	// в колонке описания переход не дублируется в журнале
	desc := stripANSI(s.descBox())
	if strings.Contains(desc, "Статус: ") {
		t.Error("переход статуса виден в журнале колонки описания")
	}
}

// TestTaskStatusHistoryInfo — переходы задачи видны в info-панели; быстрые
// переходы (в пределах минуты) сливаются в одну запись.

func TestTaskStatusHistoryInfo(t *testing.T) {
	conn, s, task, _ := tasksSeedProject(t)
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	hist, _ := db.StatusHistory(conn, db.OwnerTask, task.ID)
	if len(hist) != 1 {
		t.Fatalf("история: %d записей, ожидалась 1 (слияние быстрых переходов)", len(hist))
	}
	if hist[0].From != "Новая" || hist[0].To != "На проверке" {
		t.Errorf("слитая запись: %+v", hist[0])
	}
	plain := stripANSI(s.infoTop(20))
	for _, want := range []string{"История статусов:", "Новая → На проверке"} {
		if !strings.Contains(plain, want) {
			t.Errorf("в info нет %q", want)
		}
	}
	if strings.Contains(plain, "В работе") {
		t.Error("в info остался промежуточный переход")
	}
}

// TestTaskStatusPickCloses — выбор статуса без обязательной заметки в
// модалке применяет статус и закрывает модалку.

func TestTaskStatusPickCloses(t *testing.T) {
	conn, s, task, _ := tasksSeedProject(t)
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	if s.mode != taskStatusPick {
		t.Fatalf("c не открыл выбор статуса (mode=%d)", s.mode)
	}
	// ↓ до «В работе» (без note_prompt) и Enter
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyDown})
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyDown})
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyEnter})
	if s.mode != taskBrowse {
		t.Fatalf("Enter не закрыл модалку (mode=%d)", s.mode)
	}
	var status string
	if err := conn.QueryRow("SELECT status FROM tasks WHERE id = ?", task.ID).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "В работе" {
		t.Errorf("статус после выбора: %q", status)
	}
}

// TestTaskCreateRefreshesInfo — после создания задачи/подзадачи info-панель
// показывает историю нового элемента, а не старого.

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
	s := newTasksScreen(conn)
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
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyEnter}) // раскрыть T1
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
	s := newTasksScreen(conn)
	s.load()

	// T0 → T1, раскрыть, встать на S1, опустить
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyDown})
	s.updateTasksMsg(tea.KeyMsg{Type: tea.KeyEnter})
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
