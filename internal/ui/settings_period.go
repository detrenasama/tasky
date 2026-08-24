package ui

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// openPeriodPick открывает модалку периода, курсор — на текущем значении.
func (s *settingsScreen) openPeriodPick() {
	s.periodPick.sel = int(s.cfg.period)
	if s.periodPick.sel >= len(s.periodPick.items) {
		s.periodPick.sel = len(s.periodPick.items) - 1
	}
	s.periodPick.clampScroll()
	s.mode = settingsPeriodList
}

// openProjPick открывает модалку проекта, курсор — на текущем фильтре.
func (s *settingsScreen) openProjPick() {
	s.projPick.sel = 0
	for i, it := range s.projPick.items {
		if it.value == s.cfg.projectID {
			s.projPick.sel = i
		}
	}
	s.projPick.clampScroll()
	s.mode = settingsProjList
}

// parseCustomPeriod разбирает «ДД.ММ.ГГГГ» или «ДД.ММ.ГГГГ–ДД.ММ.ГГГГ» в
// границы [from, to). Разделитель — «-», «–» или «—».
func parseCustomPeriod(v string) (time.Time, time.Time, error) {
	v = strings.TrimSpace(v)
	v = strings.ReplaceAll(v, "–", "-")
	v = strings.ReplaceAll(v, "—", "-")
	parseDay := func(s string) (time.Time, error) {
		return time.ParseInLocation("2.1.2006", strings.TrimSpace(s), time.Local)
	}
	dayStart := func(t time.Time) time.Time {
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	}
	if i := strings.IndexByte(v, '-'); i >= 0 {
		f, err := parseDay(v[:i])
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("неверная дата «%s», нужен формат ДД.ММ.ГГГГ", strings.TrimSpace(v[:i]))
		}
		t, err := parseDay(v[i+1:])
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("неверная дата «%s», нужен формат ДД.ММ.ГГГГ", strings.TrimSpace(v[i+1:]))
		}
		if t.Before(f) {
			return time.Time{}, time.Time{}, fmt.Errorf("начало периода позже конца")
		}
		return dayStart(f), dayStart(t).AddDate(0, 0, 1), nil
	}
	d, err := parseDay(v)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("неверная дата «%s», нужен формат ДД.ММ.ГГГГ", v)
	}
	return dayStart(d), dayStart(d).AddDate(0, 0, 1), nil
}

// customInputValue — подсказка своего периода из текущих границ.
func (s *settingsScreen) customInputValue() string {
	if s.cfg.customFrom.IsZero() {
		return ""
	}
	if s.cfg.customTo.Sub(s.cfg.customFrom) > 24*time.Hour {
		return s.cfg.customFrom.Format("02.01.2006") + "-" +
			s.cfg.customTo.AddDate(0, 0, -1).Format("02.01.2006")
	}
	return s.cfg.customFrom.Format("02.01.2006")
}

// parseHideDays разбирает порог скрытия: целое ≥ 0 (0 — скрытие выключено).
func parseHideDays(v string) (int, error) {
	v = strings.TrimSpace(v)
	n, err := strconv.Atoi(v)
	if err != nil || n < 0 {
		return 0, fmt.Errorf("нужно целое число дней (0 — выключить)")
	}
	return n, nil
}
