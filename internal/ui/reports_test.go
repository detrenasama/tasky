package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/detrenasama/tasky/internal/db"
)

func TestReportsScreenRender(t *testing.T) {
	conn, task, _ := reportsSeedProject(t)
	m := newReportsModel(conn)

	m = paletteNav(t, m, tea.KeyCtrlR)
	if m.screen != screenReports {
		t.Fatalf("экран %d, ожидался screenReports", m.screen)
	}
	view := m.View()
	for _, want := range []string{"Отчет за сегодня", "Общее время:", task.Title, "S · "} {
		if !strings.Contains(view, want) {
			t.Errorf("в отчёте нет %q", want)
		}
	}
}

// TestReportsEmptyDay — без записей за сегодня отчёт показывает подсказку.

func TestReportsEmptyDay(t *testing.T) {
	conn, err := db.Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	m := newReportsModel(conn)

	m = paletteNav(t, m, tea.KeyCtrlR)
	if !strings.Contains(m.View(), "Времени за период ещё не учтено") {
		t.Error("пустой отчёт не показывает подсказку")
	}
}

// TestReportsSwitchWithRunningSession — при запущенном учёте времени переход
// на отчёты показывает предупреждение: Enter останавливает подзадачу и
// переходит, Esc отменяет.

func TestReportsSwitchWithRunningSession(t *testing.T) {
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
	if err := db.StartSession(conn, st.ID, time.Now().Add(-30*time.Minute)); err != nil {
		t.Fatal(err)
	}
	m := newReportsModel(conn)

	m = paletteNav(t, m, tea.KeyCtrlR)
	if !m.reportConfirm || m.screen != screenTasks {
		t.Errorf("предупреждение не выставлено: confirm=%v screen=%d", m.reportConfirm, m.screen)
	}
	if !strings.Contains(m.View(), "сформировать отчёт") {
		t.Error("в предупреждении нет вопроса о формировании отчёта")
	}

	m = upd(m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.reportConfirm {
		t.Error("Esc не отменил предупреждение")
	}
	if run, _ := db.RunningSession(conn); run == nil {
		t.Error("отмена остановила сессию")
	}

	m = paletteNav(t, m, tea.KeyCtrlR)
	if !m.reportConfirm {
		t.Fatal("r не показал предупреждение повторно")
	}
	m = upd(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.reportConfirm || m.screen != screenReports {
		t.Errorf("Enter не перешёл к отчётам: confirm=%v screen=%d", m.reportConfirm, m.screen)
	}
	if run, _ := db.RunningSession(conn); run != nil {
		t.Error("переход к отчётам не остановил сессию")
	}
	if !strings.Contains(m.View(), "S · ") {
		t.Error("остановленная сессия не попала в отчёт")
	}
}

// TestReportsPeriods — период «вчера» показывает данные за вчера и свой
// заголовок; «неделя» и «месяц» — свои заголовки.

func TestReportsPeriods(t *testing.T) {
	conn, err := db.Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	p, err := db.CreateProject(conn, "P")
	if err != nil {
		t.Fatal(err)
	}
	task, err := db.CreateTask(conn, p.ID, "T вчера")
	if err != nil {
		t.Fatal(err)
	}
	st, err := db.CreateSubtask(conn, task.ID, "S вчера")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	if err := db.StartSession(conn, st.ID, now.Add(-26*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := db.StopSession(conn, st.ID, now.Add(-25*time.Hour)); err != nil {
		t.Fatal(err)
	}
	m := newReportsModel(conn)

	// сегодня: записей нет
	m = paletteNav(t, m, tea.KeyCtrlR)
	if !strings.Contains(m.View(), "Времени за период ещё не учтено") {
		t.Error("за сегодня отчёт не пуст")
	}

	// вчера
	m.reports.cfg.period = periodYesterday
	m = paletteNav(t, m, tea.KeyCtrlR)
	view := m.View()
	for _, want := range []string{"Отчет за вчера", "T вчера", "S вчера"} {
		if !strings.Contains(view, want) {
			t.Errorf("за вчера нет %q", want)
		}
	}

	// неделя и месяц — заголовки
	m.reports.cfg.period = periodWeek
	m = paletteNav(t, m, tea.KeyCtrlR)
	if !strings.Contains(m.View(), "Отчет за неделю") {
		t.Error("нет заголовка недели")
	}
	m.reports.cfg.period = periodMonth
	m = paletteNav(t, m, tea.KeyCtrlR)
	if !strings.Contains(m.View(), "Отчет за "+monthNames[time.Now().Month()-1]) {
		t.Error("нет заголовка месяца")
	}
}

// TestReportsSave — клавиша s на экране отчётов сохраняет файл с заголовком,
// задачами и общим временем; на экране появляется подтверждение.

func TestReportsSave(t *testing.T) {
	conn, task, _ := reportsSeedProject(t)
	m := newReportsModel(conn)
	dir := t.TempDir()
	m.settings.cfg.saveDir = dir
	sv := tea.KeyMsg{Type: tea.KeyCtrlS}

	m = paletteNav(t, m, tea.KeyCtrlR)
	m = upd(m, sv)

	name := m.reports.saveFileName()
	data, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Fatalf("файл отчёта не создан: %v", err)
	}
	for _, want := range []string{"Отчет за сегодня", task.Title, "Общее время:"} {
		if !strings.Contains(string(data), want) {
			t.Errorf("в файле отчёта нет %q", want)
		}
	}
	if !strings.Contains(m.View(), "Отчёт сохранён") {
		t.Error("на экране нет подтверждения сохранения")
	}
}

// TestReportsSaveCreatesDir — каталог сохранения создаётся автоматически.

func TestReportsSaveCreatesDir(t *testing.T) {
	conn, _, _ := reportsSeedProject(t)
	m := newReportsModel(conn)
	dir := t.TempDir() + "/deep/reports"
	m.settings.cfg.saveDir = dir

	m = paletteNav(t, m, tea.KeyCtrlR)
	m = upd(m, tea.KeyMsg{Type: tea.KeyCtrlS})

	if _, err := os.Stat(filepath.Join(dir, m.reports.saveFileName())); err != nil {
		t.Fatalf("файл в созданном каталоге: %v", err)
	}
}

// TestReportsJournalInReport — при включённом журнале записи за период
// показываются под подзадачей и попадают в сохраняемый файл.

func TestReportsJournalInReport(t *testing.T) {
	conn, _, st := reportsSeedProject(t)
	if _, err := db.CreateJournalEntry(conn, st.ID, "запись в журнале"); err != nil {
		t.Fatal(err)
	}
	m := newReportsModel(conn)
	m.settings.cfg.includeJournal = true
	dir := t.TempDir()
	m.settings.cfg.saveDir = dir

	m = paletteNav(t, m, tea.KeyCtrlR)
	if !strings.Contains(m.View(), "запись в журнале") {
		t.Error("запись журнала не показана в отчёте")
	}

	m = upd(m, tea.KeyMsg{Type: tea.KeyCtrlS})
	data, err := os.ReadFile(filepath.Join(dir, m.reports.saveFileName()))
	if err != nil {
		t.Fatalf("файл отчёта не создан: %v", err)
	}
	if !strings.Contains(string(data), "запись в журнале") {
		t.Error("запись журнала не попала в файл")
	}
}

// TestReportsChecklist — в отчёте под подзадачей выводятся чек-листы: сначала
// «Выполнены» (done за период отчёта), затем «В работе» (все in_progress).
// new и done вне периода не выводятся; индикатор заменён на bullet «•».

func TestReportsChecklist(t *testing.T) {
	conn, _, st := reportsSeedProject(t)

	done, err := db.CreateChecklistItem(conn, st.ID, "сделано сегодня")
	if err != nil {
		t.Fatal(err)
	}
	if err := db.SetChecklistItemStatus(conn, done.ID, "done"); err != nil {
		t.Fatal(err)
	}
	ip, err := db.CreateChecklistItem(conn, st.ID, "в работе")
	if err != nil {
		t.Fatal(err)
	}
	if err := db.SetChecklistItemStatus(conn, ip.ID, "in_progress"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.CreateChecklistItem(conn, st.ID, "новое"); err != nil {
		t.Fatal(err)
	}
	out, err := db.CreateChecklistItem(conn, st.ID, "сделано вчера")
	if err != nil {
		t.Fatal(err)
	}
	if err := db.SetChecklistItemStatus(conn, out.ID, "done"); err != nil {
		t.Fatal(err)
	}
	yesterday := time.Now().AddDate(0, 0, -1)
	if _, err := conn.Exec("UPDATE checklist_items SET status_changed_at = ? WHERE id = ?",
		yesterday.Unix(), out.ID); err != nil {
		t.Fatal(err)
	}

	m := newReportsModel(conn)
	m = paletteNav(t, m, tea.KeyCtrlR)
	view := m.View()
	for _, want := range []string{"Выполнены:", "сделано сегодня", "В работе:", "в работе"} {
		if !strings.Contains(view, want) {
			t.Errorf("в отчёте нет %q", want)
		}
	}
	for _, notWant := range []string{"новое", "сделано вчера"} {
		if strings.Contains(view, notWant) {
			t.Errorf("в отчёте не должно быть %q", notWant)
		}
	}

	dir := t.TempDir()
	m.settings.cfg.saveDir = dir
	m = upd(m, tea.KeyMsg{Type: tea.KeyCtrlS})
	data, err := os.ReadFile(filepath.Join(dir, m.reports.saveFileName()))
	if err != nil {
		t.Fatalf("файл отчёта не создан: %v", err)
	}
	content := string(data)
	for _, want := range []string{"Выполнены:", "сделано сегодня", "В работе:", "в работе"} {
		if !strings.Contains(content, want) {
			t.Errorf("в файле отчёта нет %q", want)
		}
	}
	for _, notWant := range []string{"новое", "сделано вчера"} {
		if strings.Contains(content, notWant) {
			t.Errorf("в файле отчёта не должно быть %q", notWant)
		}
	}
}

// TestSettingsForm — настройки: период и проект выбираются в модалках,
// журнал — toggle, каталог — ввод пути; значения пишутся в общий cfg.
