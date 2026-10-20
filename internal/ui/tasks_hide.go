package ui

import (
	"time"

	"github.com/detrenasama/tasky/internal/db"
)

// defaultHideDays — порог скрытия завершённых элементов по умолчанию.
const defaultHideDays = 7

// hiddenCompleted — общее правило скрытия завершённых элементов: статус
// завершённого типа и завершение hideDays и более дней назад.
func (s *tasksScreen) hiddenCompleted(status string, completedAt *time.Time) bool {
	if s.hideDays <= 0 || completedAt == nil {
		return false
	}
	st, ok := s.statusDef(status)
	if !ok || st.Type != "done" {
		return false
	}
	return s.now.Sub(*completedAt) >= time.Duration(s.hideDays)*24*time.Hour
}

// hiddenDue возвращает true, если задачу нужно скрыть. Поиск
// (buildSearchItems) фильтр не применяет — скрытые задачи остаются доступны
// через поиск.
func (s *tasksScreen) hiddenDue(t db.Task) bool {
	return s.hiddenCompleted(t.Status, t.CompletedAt)
}

// hiddenSubtaskDue возвращает true, если подзадачу нужно скрыть. Подзадача с
// активной сессией учёта времени не скрывается: иначе таймер пропал бы из
// списка. Поиск скрытые подзадачи находит (см. buildSearchItems).
func (s *tasksScreen) hiddenSubtaskDue(st db.SubtaskWithTime) bool {
	if st.ActiveSince != nil {
		return false
	}
	return s.hiddenCompleted(st.Status, st.CompletedAt)
}

// toggleHide переключает показ завершённых элементов (ctrl+h): выключенное
// скрытие действует до перезапуска приложения и не меняет настройку в БД.
// Текущий порог запоминается в hideSaved, чтобы ctrl+h вернул его назад; если
// в настройках 0, включение берёт порог по умолчанию.
func (s *tasksScreen) toggleHide() {
	if s.hideDays > 0 {
		s.hideSaved = s.hideDays
		s.hideDays = 0
	} else {
		s.hideDays = s.hideSaved
		if s.hideDays <= 0 {
			s.hideDays = defaultHideDays
		}
	}
	s.loadData()
}

// statusLabel — режим списка для заголовка страницы: поиск важнее скрытия,
// потому что при поиске фильтр скрытия не применяется.
func (s *tasksScreen) statusLabel() string {
	switch {
	case s.searchQuery != "":
		return `Поиск: "` + s.searchQuery + `"`
	case s.hideDays == 0:
		return "Все"
	default:
		return "Активные"
	}
}
