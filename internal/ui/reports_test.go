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

	mm, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	m = mm.(model)
	if cmd != nil {
		t.Fatal("переход на отчёты не должен возвращать команду")
	}
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

	mm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	m = mm.(model)
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
	r := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}}

	mm, cmd := m.Update(r)
	m = mm.(model)
	if cmd != nil {
		t.Fatal("r с запущенной сессией не должен переключать сразу")
	}
	if !m.reportConfirm || m.screen != screenTasks {
		t.Errorf("предупреждение не выставлено: confirm=%v screen=%d", m.reportConfirm, m.screen)
	}
	if !strings.Contains(m.View(), "сформировать отчёт") {
		t.Error("в предупреждении нет вопроса о формировании отчёта")
	}

	mm, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = mm.(model)
	if m.reportConfirm {
		t.Error("Esc не отменил предупреждение")
	}
	if run, _ := db.RunningSession(conn); run == nil {
		t.Error("отмена остановила сессию")
	}

	mm, _ = m.Update(r)
	m = mm.(model)
	if !m.reportConfirm {
		t.Fatal("r не показал предупреждение повторно")
	}
	mm, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = mm.(model)
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
	r := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}}

	// сегодня: записей нет
	mm, _ := m.Update(r)
	m = mm.(model)
	if !strings.Contains(m.View(), "Времени за период ещё не учтено") {
		t.Error("за сегодня отчёт не пуст")
	}

	// вчера
	m.reports.cfg.period = periodYesterday
	mm, _ = m.Update(r)
	m = mm.(model)
	view := m.View()
	for _, want := range []string{"Отчет за вчера", "T вчера", "S вчера"} {
		if !strings.Contains(view, want) {
			t.Errorf("за вчера нет %q", want)
		}
	}

	// неделя и месяц — заголовки
	m.reports.cfg.period = periodWeek
	mm, _ = m.Update(r)
	m = mm.(model)
	if !strings.Contains(m.View(), "Отчет за неделю") {
		t.Error("нет заголовка недели")
	}
	m.reports.cfg.period = periodMonth
	mm, _ = m.Update(r)
	m = mm.(model)
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
	r := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}}
	sv := tea.KeyMsg{Type: tea.KeyCtrlS}

	mm, _ := m.Update(r)
	m = mm.(model)
	mm, _ = m.Update(sv)
	m = mm.(model)

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

	mm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	m = mm.(model)
	mm, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	m = mm.(model)

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

	mm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	m = mm.(model)
	if !strings.Contains(m.View(), "запись в журнале") {
		t.Error("запись журнала не показана в отчёте")
	}

	mm, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	m = mm.(model)
	data, err := os.ReadFile(filepath.Join(dir, m.reports.saveFileName()))
	if err != nil {
		t.Fatalf("файл отчёта не создан: %v", err)
	}
	if !strings.Contains(string(data), "запись в журнале") {
		t.Error("запись журнала не попала в файл")
	}
}

// TestSettingsForm — настройки: период и проект выбираются в модалках,
// журнал — toggle, каталог — ввод пути; значения пишутся в общий cfg.
