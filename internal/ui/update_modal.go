package ui

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/charmbracelet/bubbletea"

	"github.com/detrenasama/tasky/internal/ui/theme"
	"github.com/detrenasama/tasky/internal/update"
)

// updateRunFunc — запуск обновления; переопределяется в тестах, чтобы не
// ходить в сеть.
type updateRunFunc func(ctx context.Context, current string, rep update.Reporter) (string, bool, error)

// updateRun — функция обновления, используемая модалкой.
var updateRun updateRunFunc = update.UpgradeContext

// updateRunSeq выдаёт идентификаторы запусков: события из канала прошлого
// запуска не должны попадать в новую модалку.
var updateRunSeq atomic.Int64

type updateProgressMsg struct {
	runID int64
	step  update.Step
	msg   string
	frac  float64
}

type updateResultMsg struct {
	runID    int64
	version  string
	replaced bool
	err      error
}

// updateStages — этапы обновления в порядке выполнения (чек-лист модалки).
var updateStages = []update.Step{
	update.StepCheck,
	update.StepDownload,
	update.StepChecksum,
	update.StepExtract,
	update.StepInstall,
}

// updateStepName — подпись этапа обновления.
func updateStepName(s update.Step) string {
	switch s {
	case update.StepCheck:
		return "Проверка обновления"
	case update.StepDownload:
		return "Загрузка"
	case update.StepChecksum:
		return "Проверка контрольной суммы"
	case update.StepExtract:
		return "Распаковка"
	case update.StepInstall:
		return "Установка"
	case update.StepDone:
		return "Готово"
	}
	return ""
}

// updateModal — модалка обновления: прогресс-бар, чек-лист этапов, отмена по
// двойному Esc (только до проверки контрольной суммы) и выход после успешной
// установки.
type updateModal struct {
	runID   int64
	current string
	target  string

	step update.Step
	frac float64
	msg  string

	done     bool
	replaced bool
	version  string
	err      error

	// cancelable — обновление ещё можно прервать (проверка/загрузка).
	cancelable bool
	// confirmCancel — первый Esc нажат, ждём повторного подтверждения.
	confirmCancel bool
	// cancelHint — Esc нажат на этапе, который прерывать нельзя.
	cancelHint bool
	cancel     context.CancelFunc
	// onQuit — действие при закрытии после успешной установки (выход).
	onQuit func() tea.Cmd
	ch     chan tea.Msg
}

// newUpdateModal создаёт модалку и запускает обновление в горутине. События
// прогресса идут через канал, каждая wait() читает ровно одно событие.
func newUpdateModal(current, target string, onQuit func() tea.Cmd, run updateRunFunc) (*updateModal, tea.Cmd) {
	ctx, cancel := context.WithCancel(context.Background())
	u := &updateModal{
		runID:      updateRunSeq.Add(1),
		current:    current,
		target:     update.Canonical(target),
		step:       update.StepCheck,
		cancelable: true,
		cancel:     cancel,
		onQuit:     onQuit,
		ch:         make(chan tea.Msg, 64),
	}
	go func() {
		ver, replaced, err := run(ctx, current, func(step update.Step, msg string, frac float64) {
			// Прогресс — косметика: если интерфейс не успевает читать,
			// событие теряем, но загрузку не блокируем.
			select {
			case u.ch <- updateProgressMsg{runID: u.runID, step: step, msg: msg, frac: frac}:
			default:
			}
		})
		select {
		case u.ch <- updateResultMsg{runID: u.runID, version: ver, replaced: replaced, err: err}:
		case <-ctx.Done():
			// Модалка уже закрыта отменой — результат никому не нужен.
		}
		close(u.ch)
	}()
	return u, u.wait()
}

// wait возвращает команду чтения следующего события обновления.
func (u *updateModal) wait() tea.Cmd {
	ch := u.ch
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return nil
		}
		return msg
	}
}

func (u *updateModal) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case updateProgressMsg:
		if msg.runID != u.runID || u.done {
			return u, nil
		}
		if msg.step != update.StepCheck && msg.step != update.StepDownload {
			u.cancelable = false
			u.confirmCancel = false
		}
		if msg.step != u.step {
			u.cancelHint = false
		}
		u.step = msg.step
		if msg.frac >= 0 {
			u.frac = msg.frac
		}
		if msg.msg != "" {
			u.msg = msg.msg
		}
		return u, u.wait()

	case updateResultMsg:
		if msg.runID != u.runID {
			return u, nil
		}
		u.done = true
		u.replaced = msg.replaced
		u.version = msg.version
		u.err = msg.err
		u.cancelable = false
		u.confirmCancel = false
		u.cancelHint = false
		return u, nil

	case tea.KeyMsg:
		return u.updateKey(msg)
	}
	return u, nil
}

// updateKey обрабатывает клавиши модалки. Во время работы доступен только Esc
// (двойное нажатие прерывает загрузку), после завершения — закрытие.
func (u *updateModal) updateKey(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	if u.done {
		switch k.String() {
		case "enter":
			if u.err == nil && u.replaced && u.onQuit != nil {
				return nil, u.onQuit()
			}
			return nil, nil
		case "esc", "q", "ctrl+c":
			return nil, nil
		}
		return u, nil
	}
	switch k.String() {
	case "esc":
		if !u.cancelable {
			u.cancelHint = true
			return u, nil
		}
		if !u.confirmCancel {
			u.confirmCancel = true
			return u, nil
		}
		// Повторный Esc до этапа проверки суммы — прерываем загрузку.
		if u.cancel != nil {
			u.cancel()
		}
		return nil, nil
	}
	// q/ctrl+c и остальные клавиши во время обновления игнорируются, чтобы
	// случайно не закрыть приложение.
	return u, nil
}

// Init заглушка для удовлетворения tea.Model.
func (u *updateModal) Init() tea.Cmd { return nil }

func (u *updateModal) View() string {
	d := dialog{title: "Обновление Tasky", body: u.body()}
	switch {
	case !u.done:
		// Подсказок нет: во время работы обрабатывается только Esc.
	case u.err != nil:
		d.primary = "Enter — закрыть"
		d.esc = "Esc — закрыть"
	case u.replaced:
		d.primary = "Enter — выйти из Tasky"
		d.esc = "Esc — остаться"
	default:
		d.primary = "Enter — закрыть"
		d.esc = "Esc — закрыть"
	}
	return d.render()
}

// body собирает тело модалки для текущего состояния.
func (u *updateModal) body() string {
	var b strings.Builder
	b.WriteString(theme.Faint(u.versionLine()))
	b.WriteString("\n\n")

	switch {
	case !u.done:
		b.WriteString(u.progressView())
		if u.confirmCancel {
			b.WriteString("\n" + theme.AccentText("Нажмите Esc ещё раз, чтобы прервать обновление."))
		}
		if u.cancelHint {
			b.WriteString("\n" + theme.ErrorStyle.Render("Обновление уже устанавливается — прервать нельзя."))
		}
	case u.err != nil:
		b.WriteString(theme.ErrorStyle.Render("✗ Не удалось обновить Tasky."))
		b.WriteString("\n\n")
		b.WriteString(wrapText(u.err.Error(), 44))
	case !u.replaced:
		b.WriteString(theme.SaveOKStyle.Render("✓ Установлена последняя версия " + u.versionLine()))
	default:
		b.WriteString(theme.SaveOKStyle.Render("✓ Обновление установлено."))
		b.WriteString("\n\n")
		b.WriteString("Перезапустите Tasky, чтобы использовать " + u.version + ".")
	}
	return b.String()
}

// progressView — прогресс-бар и чек-лист этапов.
func (u *updateModal) progressView() string {
	const barW = 24
	p := u.progress()
	pct := fmt.Sprintf("%3.0f%%", p*100)
	lines := []string{
		"  [" + updateBar(p, barW) + "] " + pct,
		"  " + u.stageLabel(),
		"",
	}
	for _, st := range updateStages {
		mark := "·"
		style := theme.Faint
		switch {
		case u.step == update.StepDone || st < u.step:
			mark = "✓"
			style = func(s string) string { return theme.SaveOKStyle.Render(s) }
		case st == u.step:
			mark = "›"
			style = theme.AccentText
		}
		lines = append(lines, "  "+style(mark+" "+updateStepName(st)))
	}
	return strings.Join(lines, "\n")
}

// stageLabel — подпись текущего этапа; для загрузки URL не показываем.
func (u *updateModal) stageLabel() string {
	if u.step != update.StepDownload && u.msg != "" {
		return u.msg
	}
	return updateStepName(u.step) + "…"
}

// progress — общая доля выполнения по этапам; загрузка занимает 10–70%.
func (u *updateModal) progress() float64 {
	switch u.step {
	case update.StepDownload:
		f := u.frac
		if f < 0 {
			f = 0
		}
		if f > 1 {
			f = 1
		}
		return 0.1 + 0.6*f
	case update.StepChecksum:
		return 0.8
	case update.StepExtract:
		return 0.9
	case update.StepInstall:
		return 0.95
	case update.StepDone:
		return 1
	}
	return 0.05
}

// updateBar рисует полосу прогресса шириной width (без скобок).
func updateBar(frac float64, width int) string {
	if frac < 0 {
		frac = 0
	}
	if frac > 1 {
		frac = 1
	}
	filled := int(frac * float64(width))
	return theme.AccentText(strings.Repeat("█", filled)) +
		theme.Faint(strings.Repeat("░", width-filled))
}

// versionLine — «v0.5.4 → v0.6.0» (или только текущая версия).
func (u *updateModal) versionLine() string {
	cur := "v" + update.TrimV(u.current)
	if u.target != "" && update.Compare(u.target, u.current) > 0 {
		return cur + " → " + u.target
	}
	return cur
}

// startUpdate открывает модалку обновления и запускает загрузку. Перед
// выходом после установки активная сессия останавливается — как при обычном
// закрытии приложения.
func (m *model) startUpdate() tea.Cmd {
	if !update.IsRelease(m.version) {
		return nil
	}
	onQuit := func() tea.Cmd {
		if run, err := m.store.RunningSession(); err == nil && run != nil {
			_ = m.store.StopSession(run.ID, time.Now())
		}
		return tea.Quit
	}
	u, cmd := newUpdateModal(m.version, m.updateVer, onQuit, updateRun)
	m.modal.Show(u)
	return cmd
}
