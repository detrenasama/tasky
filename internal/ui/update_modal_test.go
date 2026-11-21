package ui

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbletea"

	"github.com/detrenasama/tasky/internal/update"
)

// nextUpdateMsg выполняет команду ожидания события обновления с таймаутом.
func nextUpdateMsg(t *testing.T, cmd tea.Cmd) tea.Msg {
	t.Helper()
	if cmd == nil {
		t.Fatal("нет команды ожидания события")
	}
	done := make(chan tea.Msg, 1)
	go func() { done <- cmd() }()
	select {
	case msg := <-done:
		return msg
	case <-time.After(2 * time.Second):
		t.Fatal("событие обновления не пришло")
		return nil
	}
}

// findPaletteCmd ищет команду палитры по названию.
func findPaletteCmd(m *model, title string) *cmdItem {
	for _, g := range m.paletteGroups() {
		for i := range g.cmds {
			if g.cmds[i].title == title {
				return &g.cmds[i]
			}
		}
	}
	return nil
}

func TestUpdatePaletteCommand(t *testing.T) {
	m, _ := newChangelogTestModel(t, "v0.5.4", "v0.5.4")
	m.updateVer = "v0.6.0"

	c := findPaletteCmd(m, "Обновить")
	if c == nil {
		t.Fatal("в палитре нет команды «Обновить»")
	}
	if !strings.Contains(c.desc, "доступно v0.6.0") {
		t.Fatalf("описание = %q", c.desc)
	}
	if c.runCmd == nil {
		t.Fatal("команда обновления должна запускать tea.Cmd")
	}
}

func TestUpdatePaletteHiddenForDev(t *testing.T) {
	m, _ := newChangelogTestModel(t, "dev", "")
	if c := findPaletteCmd(m, "Обновить"); c != nil {
		t.Fatal("для dev-сборки команда «Обновить» не должна показываться")
	}
}

func TestUpdatePaletteStartsModal(t *testing.T) {
	m, _ := newChangelogTestModel(t, "v0.5.4", "v0.5.4")
	m.updateVer = "v0.6.0"

	old := updateRun
	updateRun = func(ctx context.Context, current string, rep update.Reporter) (string, bool, error) {
		rep(update.StepCheck, "Проверка обновления…", -1)
		<-ctx.Done()
		return "", false, ctx.Err()
	}
	defer func() { updateRun = old }()

	c := findPaletteCmd(m, "Обновить")
	cmd := c.runCmd(m)
	if cmd == nil {
		t.Fatal("команда не вернула tea.Cmd запуска")
	}
	if !m.modal.Active() {
		t.Fatal("модалка обновления не открылась")
	}
	if u, ok := m.modal.active.(*updateModal); ok && u.cancel != nil {
		u.cancel()
	}
}

func TestUpdateModalCancelOnDownload(t *testing.T) {
	canceled := make(chan struct{})
	run := func(ctx context.Context, current string, rep update.Reporter) (string, bool, error) {
		rep(update.StepCheck, "Проверка обновления…", -1)
		rep(update.StepDownload, "Загрузка", 0)
		rep(update.StepDownload, "", 0.5)
		<-ctx.Done()
		close(canceled)
		return "", false, ctx.Err()
	}
	u, cmd := newUpdateModal("v0.5.4", "v0.6.0", nil, run)

	for {
		msg := nextUpdateMsg(t, cmd)
		pm, ok := msg.(updateProgressMsg)
		if !ok {
			t.Fatalf("неожиданное сообщение %T", msg)
		}
		mm, c := u.Update(pm)
		u = mm.(*updateModal)
		cmd = c
		if pm.step == update.StepDownload && pm.frac == 0.5 {
			break
		}
	}
	view := stripANSI(u.View())
	if !strings.Contains(view, "Загрузка") || !strings.Contains(view, "█") {
		t.Fatalf("нет прогресса загрузки:\n%s", view)
	}

	// Первый Esc только предупреждает.
	mm, _ := u.Update(tea.KeyMsg{Type: tea.KeyEsc})
	u = mm.(*updateModal)
	if !u.confirmCancel {
		t.Fatal("первый Esc не включил подтверждение отмены")
	}
	if !strings.Contains(stripANSI(u.View()), "Нажмите Esc ещё раз") {
		t.Fatalf("нет предупреждения об отмене:\n%s", stripANSI(u.View()))
	}

	// Второй Esc прерывает загрузку и закрывает модалку.
	mm, cmd = u.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if mm != nil {
		t.Fatal("модалка не закрылась после второго Esc")
	}
	if cmd != nil {
		t.Fatal("после отмены не должно быть команды")
	}
	select {
	case <-canceled:
	case <-time.After(2 * time.Second):
		t.Fatal("контекст обновления не отменён")
	}
}

func TestUpdateModalNoCancelAfterDownload(t *testing.T) {
	stopped := make(chan struct{})
	run := func(ctx context.Context, current string, rep update.Reporter) (string, bool, error) {
		rep(update.StepDownload, "Загрузка", 1)
		rep(update.StepChecksum, "Проверка контрольной суммы…", -1)
		rep(update.StepExtract, "Распаковка…", -1)
		<-ctx.Done()
		close(stopped)
		return "", false, ctx.Err()
	}
	u, cmd := newUpdateModal("v0.5.4", "v0.6.0", nil, run)

	for {
		msg := nextUpdateMsg(t, cmd)
		pm, ok := msg.(updateProgressMsg)
		if !ok {
			t.Fatalf("неожиданное сообщение %T", msg)
		}
		mm, c := u.Update(pm)
		u = mm.(*updateModal)
		cmd = c
		if pm.step == update.StepExtract {
			break
		}
	}

	mm, _ := u.Update(tea.KeyMsg{Type: tea.KeyEsc})
	u = mm.(*updateModal)
	if u.confirmCancel {
		t.Fatal("после загрузки отмена не должна взводиться")
	}
	if !u.cancelHint {
		t.Fatal("нет сообщения о невозможности прервать обновление")
	}
	if !strings.Contains(stripANSI(u.View()), "прервать нельзя") {
		t.Fatalf("нет подсказки о запрете отмены:\n%s", stripANSI(u.View()))
	}
	u.cancel()
	<-stopped
}

func TestUpdateModalSuccess(t *testing.T) {
	run := func(ctx context.Context, current string, rep update.Reporter) (string, bool, error) {
		rep(update.StepDownload, "Загрузка", 1)
		rep(update.StepChecksum, "Проверка контрольной суммы…", -1)
		return "v0.6.0", true, nil
	}
	quitCalled := false
	u, cmd := newUpdateModal("v0.5.4", "v0.6.0", func() tea.Cmd {
		quitCalled = true
		return tea.Quit
	}, run)

	for {
		msg := nextUpdateMsg(t, cmd)
		mm, c := u.Update(msg)
		if mm == nil {
			t.Fatalf("модалка закрылась на сообщении %T", msg)
		}
		u = mm.(*updateModal)
		cmd = c
		if u.done {
			break
		}
	}

	view := stripANSI(u.View())
	for _, want := range []string{"Обновление установлено", "Перезапустите Tasky", "v0.6.0", "выйти из Tasky"} {
		if !strings.Contains(view, want) {
			t.Fatalf("в модалке нет %q:\n%s", want, view)
		}
	}

	mm, cmd := u.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if mm != nil {
		t.Fatal("модалка не закрылась по Enter")
	}
	if !quitCalled {
		t.Fatal("не вызван обработчик выхода")
	}
	if cmd == nil {
		t.Fatal("нет команды выхода")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatal("ожидалась команда tea.Quit")
	}
}

func TestUpdateModalAlreadyLatest(t *testing.T) {
	run := func(ctx context.Context, current string, rep update.Reporter) (string, bool, error) {
		rep(update.StepCheck, "Проверка обновления…", -1)
		return "v0.5.4", false, nil
	}
	quitCalled := false
	u, cmd := newUpdateModal("v0.5.4", "", func() tea.Cmd {
		quitCalled = true
		return tea.Quit
	}, run)

	for {
		msg := nextUpdateMsg(t, cmd)
		mm, c := u.Update(msg)
		u = mm.(*updateModal)
		cmd = c
		if u.done {
			break
		}
	}
	if !strings.Contains(stripANSI(u.View()), "Установлена последняя версия") {
		t.Fatalf("нет сообщения об актуальной версии:\n%s", stripANSI(u.View()))
	}
	mm, cmd := u.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if mm != nil {
		t.Fatal("модалка не закрылась")
	}
	if cmd != nil || quitCalled {
		t.Fatal("при актуальной версии приложение не должно закрываться")
	}
}

func TestUpdateModalError(t *testing.T) {
	run := func(ctx context.Context, current string, rep update.Reporter) (string, bool, error) {
		rep(update.StepCheck, "Проверка обновления…", -1)
		return "", false, context.DeadlineExceeded
	}
	u, cmd := newUpdateModal("v0.5.4", "", nil, run)

	for {
		msg := nextUpdateMsg(t, cmd)
		mm, c := u.Update(msg)
		u = mm.(*updateModal)
		cmd = c
		if u.done {
			break
		}
	}
	view := stripANSI(u.View())
	if !strings.Contains(view, "Не удалось обновить") || !strings.Contains(view, "deadline exceeded") {
		t.Fatalf("нет текста ошибки:\n%s", view)
	}
	mm, cmd := u.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if mm != nil {
		t.Fatal("модалка не закрылась по Esc")
	}
	if cmd != nil {
		t.Fatal("ошибка обновления не должна закрывать приложение")
	}
}

func TestUpdateModalIgnoresStaleRun(t *testing.T) {
	u := &updateModal{runID: 7, step: update.StepCheck}
	mm, _ := u.Update(updateProgressMsg{runID: 6, step: update.StepDownload, frac: 1})
	got := mm.(*updateModal)
	if got.step != update.StepCheck || got.frac != 0 {
		t.Fatalf("событие чужого запуска изменило состояние: step=%v frac=%v", got.step, got.frac)
	}
}

// TestUpdateProgressRoutedToModal — сообщения прогресса (не клавиши) должны
// доходить до активной модалки через корневую model.Update.
func TestUpdateProgressRoutedToModal(t *testing.T) {
	m, _ := newChangelogTestModel(t, "v0.5.4", "v0.5.4")
	u := &updateModal{runID: 1, step: update.StepCheck}
	m.modal.Show(u)

	mm, _ := m.Update(updateProgressMsg{runID: 1, step: update.StepDownload, frac: 0.5})
	got := mm.(model)
	um, ok := got.modal.active.(*updateModal)
	if !ok {
		t.Fatalf("модалка потерялась: %T", got.modal.active)
	}
	if um.step != update.StepDownload || um.frac != 0.5 {
		t.Fatalf("прогресс не дошёл: step=%v frac=%v", um.step, um.frac)
	}
}
