package ui

import (
	"github.com/charmbracelet/bubbletea"
)

// updateTasksModal маршрутизирует клавиши по модальным режимам экрана задач.
// Возвращает (_, _, false), если текущий режим — не модальный (навигация
// обрабатывается в updateTasksBase).
func (m *model) updateTasksModal(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	s := m.tasks
	// Любая клавиша, кроме Esc, снимает предупреждение «нажмите Esc ещё раз».
	if msg.String() != "esc" {
		s.esc.Reset()
	}
	switch s.mode {
	case taskInput, taskTitleEdit, taskConfirm, taskStatusPick, taskStatusNote, taskSearch:
		return s.updateTasksInput(m, msg)
	case taskDescModal:
		return s.updateTasksDesc(m, msg)
	case taskLinkEdit, taskLinks, taskLinkConfirm:
		return s.updateTasksLinks(m, msg)
	case taskJournal:
		return s.updateTasksJournal(m, msg)
	case taskTags, taskTagEdit, taskTagTypePick, taskTagConfirm:
		return s.updateTasksTags(m, msg)
	case taskChecklist, taskChecklistConfirm:
		return s.updateTasksChecklist(m, msg)
	case taskTimeList, taskTimeEdit, taskTimeDelete:
		return s.updateTasksTime(m, msg)
	}
	return m, nil, false
}
