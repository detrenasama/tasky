package ui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/kalpamer/tasky/internal/db"
)

func TestSettingsForm(t *testing.T) {
	conn, err := db.Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	p1, err := db.CreateProject(conn, "P1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.CreateProject(conn, "P2"); err != nil {
		t.Fatal(err)
	}
	m := newReportsModel(conn)

	mm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	m = mm.(model)
	if m.screen != screenSettings {
		t.Fatalf("s не открыл настройки (screen=%d)", m.screen)
	}
	view := m.View()
	for _, want := range []string{"Настройки отчёта", "Период:  сегодня", "все проекты", "Журнал:  выкл", "Каталог: reports"} {
		if !strings.Contains(view, want) {
			t.Errorf("в настройках нет %q", want)
		}
	}

	// период: Enter → модалка списка, ↓ Enter → вчера
	m.updateSettings(tea.KeyMsg{Type: tea.KeyEnter})
	if m.settings.mode != settingsPeriodList {
		t.Fatalf("Enter на периоде не открыл модалку (mode=%d)", m.settings.mode)
	}
	if !strings.Contains(m.View(), "Период отчёта") {
		t.Error("модалка периода не отрендерилась")
	}
	m.updateSettings(tea.KeyMsg{Type: tea.KeyDown})
	m.updateSettings(tea.KeyMsg{Type: tea.KeyEnter})
	if m.settings.cfg.period != periodYesterday || !strings.Contains(m.View(), "Период:  вчера") {
		t.Errorf("период не сменился: %d", m.settings.cfg.period)
	}

	// свой период: модалка (курсор на текущем — «вчера»), ↓×3 до «свой…»
	m.updateSettings(tea.KeyMsg{Type: tea.KeyEnter})
	if m.settings.mode != settingsPeriodList {
		t.Fatalf("Enter на периоде не открыл модалку (mode=%d)", m.settings.mode)
	}
	for i := 0; i < 3; i++ {
		m.updateSettings(tea.KeyMsg{Type: tea.KeyDown})
	}
	m.updateSettings(tea.KeyMsg{Type: tea.KeyEnter})
	if m.settings.mode != settingsPeriodInput {
		t.Fatalf("«свой…» не открыл ввод (mode=%d)", m.settings.mode)
	}

	// неверный формат — ошибка, режим не закрывается
	m.settings.periodInput.SetValue("abc")
	m.updateSettings(tea.KeyMsg{Type: tea.KeyEnter})
	if m.settings.mode != settingsPeriodInput || m.settings.lastErr == nil {
		t.Fatal("неверная дата не показала ошибку")
	}
	if !strings.Contains(m.View(), "нужен формат ДД.ММ.ГГГГ") {
		t.Error("нет подсказки про формат")
	}
	m.updateSettings(tea.KeyMsg{Type: tea.KeyEsc})
	if m.settings.mode != settingsBrowse || m.settings.cfg.period != periodYesterday {
		t.Error("Esc после ошибки вёл себя неверно")
	}

	// верный диапазон: снова модалка (курсор на «вчера»), ↓×3 до «свой…»
	m.updateSettings(tea.KeyMsg{Type: tea.KeyEnter})
	for i := 0; i < 3; i++ {
		m.updateSettings(tea.KeyMsg{Type: tea.KeyDown})
	}
	m.updateSettings(tea.KeyMsg{Type: tea.KeyEnter})
	m.settings.periodInput.SetValue("01.08.2026-03.08.2026")
	m.updateSettings(tea.KeyMsg{Type: tea.KeyEnter})
	if m.settings.cfg.period != periodCustom {
		t.Fatal("диапазон не применился")
	}
	wantFrom := time.Date(2026, 8, 1, 0, 0, 0, 0, time.Local)
	wantTo := time.Date(2026, 8, 4, 0, 0, 0, 0, time.Local)
	if !m.settings.cfg.customFrom.Equal(wantFrom) || !m.settings.cfg.customTo.Equal(wantTo) {
		t.Errorf("границы: %v — %v", m.settings.cfg.customFrom, m.settings.cfg.customTo)
	}
	if !strings.Contains(m.View(), "Период:  свой · 01.08.2026 – 03.08.2026") {
		t.Error("строка периода не показывает диапазон")
	}
	if m.reports.periodLabel() != "Отчет · 01.08.2026 – 03.08.2026" {
		t.Errorf("заголовок отчёта: %q", m.reports.periodLabel())
	}

	// одиночная дата: модалка (курсор уже на «свой…»), сразу Enter
	m.updateSettings(tea.KeyMsg{Type: tea.KeyEnter})
	m.updateSettings(tea.KeyMsg{Type: tea.KeyEnter})
	m.settings.periodInput.SetValue("15.08.2026")
	m.updateSettings(tea.KeyMsg{Type: tea.KeyEnter})
	if m.reports.periodLabel() != "Отчет за день · 15.08.2026" {
		t.Errorf("заголовок одиночного дня: %q", m.reports.periodLabel())
	}
	if m.reports.saveFileName() != "2026-08-15.txt" {
		t.Errorf("имя файла: %q", m.reports.saveFileName())
	}

	// проект: ↓ Enter → модалка списка, ↓ Enter → P1, Esc — отмена
	m.updateSettings(tea.KeyMsg{Type: tea.KeyDown})
	m.updateSettings(tea.KeyMsg{Type: tea.KeyEnter})
	if m.settings.mode != settingsProjList {
		t.Fatalf("Enter на проекте не открыл модалку (mode=%d)", m.settings.mode)
	}
	if !strings.Contains(m.View(), "Фильтр по проекту") {
		t.Error("модалка проекта не отрендерилась")
	}
	m.updateSettings(tea.KeyMsg{Type: tea.KeyDown})
	m.updateSettings(tea.KeyMsg{Type: tea.KeyEnter})
	if m.settings.cfg.projectID != p1.ID || !strings.Contains(m.View(), "Проект:  P1") {
		t.Errorf("фильтр не перешёл на первый проект: %d", m.settings.cfg.projectID)
	}
	m.updateSettings(tea.KeyMsg{Type: tea.KeyEnter})
	if m.settings.mode != settingsProjList {
		t.Fatal("Enter на проекте не открыл модалку повторно")
	}
	m.updateSettings(tea.KeyMsg{Type: tea.KeyEsc})
	if m.settings.mode != settingsBrowse || m.settings.cfg.projectID != p1.ID {
		t.Error("Esc не отменил выбор проекта")
	}

	// журнал: ↓ Enter → вкл
	m.updateSettings(tea.KeyMsg{Type: tea.KeyDown})
	m.updateSettings(tea.KeyMsg{Type: tea.KeyEnter})
	if !m.settings.cfg.includeJournal || !strings.Contains(m.View(), "Журнал:  вкл") {
		t.Error("журнал не включился")
	}

	// каталог: ↓ Enter → модалка, ввод пути, Enter
	m.updateSettings(tea.KeyMsg{Type: tea.KeyDown})
	m.updateSettings(tea.KeyMsg{Type: tea.KeyEnter})
	if m.settings.mode != settingsDirInput {
		t.Fatalf("Enter на каталоге не открыл модалку (mode=%d)", m.settings.mode)
	}
	if !strings.Contains(m.View(), "Каталог сохранения отчётов") {
		t.Error("модалка каталога не отрендерилась")
	}
	m.settings.dirInput.SetValue("out/reports")
	m.updateSettings(tea.KeyMsg{Type: tea.KeyEnter})
	if m.settings.mode != settingsBrowse {
		t.Fatal("Enter не закрыл модалку каталога")
	}
	if m.settings.cfg.saveDir != "out/reports" || !strings.Contains(m.View(), "Каталог: out/reports") {
		t.Errorf("каталог не сохранился: %q", m.settings.cfg.saveDir)
	}

	// Esc в модалке отменяет
	m.updateSettings(tea.KeyMsg{Type: tea.KeyEnter})
	m.settings.dirInput.SetValue("другой")
	m.updateSettings(tea.KeyMsg{Type: tea.KeyEsc})
	if m.settings.mode != settingsBrowse || m.settings.cfg.saveDir != "out/reports" {
		t.Error("Esc не отменил изменение каталога")
	}
}

// TestSettingsListScroll — при большом числе проектов модалка показывает
// лишь видимую часть и плавно прокручивается за курсором.

func TestSettingsListScroll(t *testing.T) {
	conn, err := db.Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	for i := 0; i < 15; i++ {
		if _, err := db.CreateProject(conn, fmt.Sprintf("Проект %02d", i)); err != nil {
			t.Fatal(err)
		}
	}
	m := newReportsModel(conn)
	m.settings.resize(100, 27) // visible = 12

	mm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	m = mm.(model)
	m.updateSettings(tea.KeyMsg{Type: tea.KeyDown}) // строка «Проект»
	m.updateSettings(tea.KeyMsg{Type: tea.KeyEnter})
	if m.settings.mode != settingsProjList {
		t.Fatalf("модалка проекта не открылась (mode=%d)", m.settings.mode)
	}
	view := m.View()
	if !strings.Contains(view, "все проекты") || strings.Contains(view, "Проект 14") {
		t.Error("в начале видны не те элементы списка")
	}

	for i := 0; i < 15; i++ {
		m.updateSettings(tea.KeyMsg{Type: tea.KeyDown})
	}
	view = m.View()
	if !strings.Contains(view, "Проект 14") || strings.Contains(view, "Проект 00") {
		t.Error("список не прокрутился за курсором до конца")
	}

	m.updateSettings(tea.KeyMsg{Type: tea.KeyEnter})
	if m.settings.cfg.projectID == 0 {
		t.Error("Enter не выбрал проект из прокрученного списка")
	}
}

// TestSettingsHideInput — строка «Скрытие»: ввод числа дней, 0 — выкл,
// неверный формат — ошибка без изменения значения.

func TestSettingsHideInput(t *testing.T) {
	conn, err := db.Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	m := newReportsModel(conn)
	m.switchScreen(screenSettings)

	// по умолчанию — 7 дн
	if !strings.Contains(m.View(), "Скрытие: 7 дн") {
		t.Error("нет строки скрытия по умолчанию (7 дн)")
	}

	open := func() {
		for m.settings.sel != 4 {
			m.updateSettings(tea.KeyMsg{Type: tea.KeyDown})
		}
		m.updateSettings(tea.KeyMsg{Type: tea.KeyEnter})
		if m.settings.mode != settingsHideInput {
			t.Fatalf("Enter на скрытии не открыл модалку (mode=%d)", m.settings.mode)
		}
	}

	// неверный формат — ошибка, значение не меняется
	open()
	m.settings.hideInput.SetValue("abc")
	m.updateSettings(tea.KeyMsg{Type: tea.KeyEnter})
	if m.settings.mode != settingsHideInput || m.settings.lastErr == nil {
		t.Fatal("мусор в пороге не дал ошибку")
	}
	if !strings.Contains(m.View(), "целое число дней") {
		t.Error("нет подсказки про формат")
	}
	if v, _, _ := db.GetSetting(conn, "hide_days"); v != "" {
		t.Errorf("ошибочный ввод записал настройку: %q", v)
	}
	m.updateSettings(tea.KeyMsg{Type: tea.KeyEsc})
	if m.settings.mode != settingsBrowse || m.settings.lastErr != nil {
		t.Error("Esc после ошибки вёл себя неверно")
	}

	// отрицательное число — тоже ошибка
	open()
	m.settings.hideInput.SetValue("-3")
	m.updateSettings(tea.KeyMsg{Type: tea.KeyEnter})
	if m.settings.mode != settingsHideInput || m.settings.lastErr == nil {
		t.Fatal("отрицательное число не дало ошибку")
	}
	m.updateSettings(tea.KeyMsg{Type: tea.KeyEsc})

	// 14 → сохранилось в БД и в строке
	open()
	m.settings.hideInput.SetValue("14")
	m.updateSettings(tea.KeyMsg{Type: tea.KeyEnter})
	if m.settings.mode != settingsBrowse {
		t.Fatal("Enter не закрыл модалку скрытия")
	}
	if v, _, _ := db.GetSetting(conn, "hide_days"); v != "14" {
		t.Errorf("порог не сохранён: %q", v)
	}
	if !strings.Contains(m.View(), "Скрытие: 14 дн") {
		t.Error("строка не показывает 14 дн")
	}

	// 0 → выкл
	open()
	m.settings.hideInput.SetValue("0")
	m.updateSettings(tea.KeyMsg{Type: tea.KeyEnter})
	if v, _, _ := db.GetSetting(conn, "hide_days"); v != "0" {
		t.Errorf("0 не сохранён: %q", v)
	}
	if !strings.Contains(m.View(), "Скрытие: выкл") {
		t.Error("строка не показывает выкл")
	}

	// Esc отменяет изменение
	open()
	m.settings.hideInput.SetValue("30")
	m.updateSettings(tea.KeyMsg{Type: tea.KeyEsc})
	if v, _, _ := db.GetSetting(conn, "hide_days"); v != "0" {
		t.Errorf("Esc не отменил изменение: %q", v)
	}

	// load() снова читает из БД
	m.settings.load()
	if m.settings.hideDays != 0 {
		t.Errorf("load() не прочитал настройку: %d", m.settings.hideDays)
	}
}

// TestTaskStatusQuickCycle — x/z двигают статус по быстрой цепочке
// Новая → В работе → На проверке → Выполнена без зацикливания.

func TestSettingsStatusesManage(t *testing.T) {
	conn, err := db.Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	m := newReportsModel(conn)
	m.switchScreen(screenSettings)
	down := func() { m.updateSettings(tea.KeyMsg{Type: tea.KeyDown}) }

	// строка «Статусы» — пятое нажатие вниз (период→проект→журнал→каталог→скрытие→статусы)
	for i := 0; i < 5; i++ {
		down()
	}
	if m.settings.sel != 5 {
		t.Fatalf("sel=%d, ожидался статусы", m.settings.sel)
	}
	if !strings.Contains(m.View(), "Статусы: 8") {
		t.Error("строка статусов не показывает количество")
	}

	m.updateSettings(tea.KeyMsg{Type: tea.KeyEnter})
	if m.settings.mode != settingsStatusList {
		t.Fatalf("Enter не открыл список статусов (mode=%d)", m.settings.mode)
	}

	// создание нового статуса
	m.updateSettings(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	if m.settings.mode != settingsStatusEdit {
		t.Fatalf("n не открыл редактор (mode=%d)", m.settings.mode)
	}
	m.settings.editName.SetValue("Новый статус")
	m.updateSettings(tea.KeyMsg{Type: tea.KeyEnter}) // имя → тип
	m.updateSettings(tea.KeyMsg{Type: tea.KeyEnter}) // тип: Новый → В работе
	if m.settings.editType != 1 {
		t.Errorf("тип не переключился: %d", m.settings.editType)
	}
	m.updateSettings(tea.KeyMsg{Type: tea.KeyDown}) // тип → цвет
	if m.settings.editFocus != 2 {
		t.Fatalf("editFocus=%d, ожидался цвет", m.settings.editFocus)
	}
	m.updateSettings(tea.KeyMsg{Type: tea.KeyEnter}) // открыть палитру
	if m.settings.mode != settingsColorPick {
		t.Fatalf("Enter на цвете не открыл палитру (mode=%d)", m.settings.mode)
	}
	m.updateSettings(tea.KeyMsg{Type: tea.KeyDown})
	m.updateSettings(tea.KeyMsg{Type: tea.KeyDown})
	m.updateSettings(tea.KeyMsg{Type: tea.KeyEnter})
	if m.settings.editColor != 2 {
		t.Errorf("цвет не выбран: %d", m.settings.editColor)
	}
	m.updateSettings(tea.KeyMsg{Type: tea.KeyDown}) // цвет → быстрая цепочка
	if m.settings.editFocus != 3 {
		t.Fatalf("editFocus=%d, ожидался 3", m.settings.editFocus)
	}
	m.updateSettings(tea.KeyMsg{Type: tea.KeyEnter}) // быстрая цепочка: вкл
	if !m.settings.editQuick {
		t.Error("быстрая цепочка не включилась")
	}
	m.updateSettings(tea.KeyMsg{Type: tea.KeyCtrlS})
	if m.settings.mode != settingsStatusList {
		t.Fatalf("Ctrl+S не сохранил статус (mode=%d)", m.settings.mode)
	}
	sts, _ := db.Statuses(conn)
	if len(sts) != 9 {
		t.Fatalf("статусов %d, ожидалось 9", len(sts))
	}
	last := sts[len(sts)-1]
	if last.Name != "Новый статус" || last.Type != "in_progress" ||
		last.Color != "#569cd6" || !last.IsQuick {
		t.Errorf("созданный статус: %+v", last)
	}

	// удаление неиспользуемого статуса
	m.settings.statusPick.sel = len(m.settings.statusPick.items) - 1
	m.updateSettings(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	if m.settings.mode != settingsStatusConfirm {
		t.Fatalf("d не открыл подтверждение (mode=%d)", m.settings.mode)
	}
	m.updateSettings(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if m.settings.mode != settingsStatusList {
		t.Fatalf("подтверждение не закрылось (mode=%d)", m.settings.mode)
	}
	if sts, _ = db.Statuses(conn); len(sts) != 8 {
		t.Errorf("статусов после удаления: %d", len(sts))
	}

	// удаление используемого — ошибка
	p, _ := db.CreateProject(conn, "P")
	db.CreateTask(conn, p.ID, "T") // статус «Новая»
	m.settings.statusPick.sel = 0
	m.updateSettings(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	m.updateSettings(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if m.settings.lastErr == nil {
		t.Fatal("удаление используемого статуса не дало ошибку")
	}
	if !strings.Contains(m.View(), "статус используется") {
		t.Error("ошибка не показана в модалке")
	}
	if sts, _ = db.Statuses(conn); len(sts) != 8 {
		t.Errorf("используемый статус удалён: %d", len(sts))
	}
}
