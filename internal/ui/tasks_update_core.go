package ui

import (
	"github.com/charmbracelet/bubbletea"
)

// updateTasksBase обрабатывает клавиши базовой навигации и списка задач
// (вне модальных режимов).
func (m *model) updateTasksBase(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	s := m.tasks

	switch msg.String() {
	case "tab":
		// переключение фокуса между панелями упразднено
		return m, nil
	case "enter":
		s.startEditDescription()
		return m, nil
	case "right":
		s.expandTask()
		return m, nil
	case "left":
		kind, id := s.selectedKindID()
		if kind == kindSubtask {
			s.selectByKindID(kindTask, s.selectedTaskID())
		} else if id != 0 {
			s.expanded[id] = false
			s.buildItems()
		}
		s.loadInfo()
		s.loadDesc()
		return m, nil
	case "pgup", "pgdown":
		s.descV, _ = s.descV.Update(msg)
		return m, nil
	case "e":
		s.startEditTitle()
		return m, nil
	case "o":
		s.openLinks()
		return m, nil
	case "ctrl+j":
		s.startJournal()
		return m, nil
	case "j":
		s.editTodayJournal()
		return m, nil
	case "/":
		s.lastErr = nil
		s.searchInput.SetValue(s.searchQuery)
		s.searchInput.Focus()
		s.mode = taskSearch
		return m, nil
	case "g":
		s.openTags()
		return m, nil
	case "i":
		s.openChecklist()
		return m, nil
	case "esc":
		if s.searchQuery != "" {
			s.searchQuery = ""
			s.buildItems()
			s.loadInfo()
			s.loadDesc()
		}
		return m, nil
	case "ctrl+l":
		s.toggleTimer()
		return m, nil
	case "[":
		s.switchProject(-1)
		return m, nil
	case "]":
		s.switchProject(1)
		return m, nil
	case "ctrl+up":
		s.moveSelected(-1)
		return m, nil
	case "ctrl+down":
		s.moveSelected(1)
		return m, nil
	case "n":
		s.startNewTask()
		return m, nil
	case "a":
		s.startNewSubtask()
		return m, nil
	case "d":
		s.startDelete()
		return m, nil
	case "x":
		s.shiftStatus(1)
		return m, nil
	case "z":
		s.shiftStatus(-1)
		return m, nil
	case "c":
		s.openStatusPick()
		return m, nil
	}

	// стрелки вверх/вниз — навигация по списку (через s.list.Update ниже);
	// прокрутка описания — PgUp/PgDn (независимо от фокуса, см. выше).

	var cmd tea.Cmd
	beforeKind, beforeID := s.selectedKindID()
	s.list, cmd = s.list.Update(msg)
	afterKind, afterID := s.selectedKindID()
	if beforeKind != afterKind || beforeID != afterID {
		s.loadInfo()
		s.loadDesc()
		s.descV.GotoTop()
	}
	return m, cmd
}
