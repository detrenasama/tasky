package ui

import (
	"github.com/charmbracelet/bubbletea"
)

// updateTasksTime обрабатывает модалки записей времени подзадачи.
func (s *tasksScreen) updateTasksTime(m *model, msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	switch s.mode {
	case taskTimeList:
		switch msg.String() {
		case "up", "k":
			s.timePick.move(-1)
		case "down", "j":
			s.timePick.move(1)
		case "pgup":
			s.timePick.move(-s.timePick.visible)
		case "pgdown":
			s.timePick.move(s.timePick.visible)
		case "enter":
			s.startTimeEdit()
		case "d":
			s.startTimeDelete()
		case "esc":
			s.esc.Reset()
			s.mode = taskBrowse
		}
		return m, nil, true
	case taskTimeEdit:
		switch msg.String() {
		case "left", "h":
			base := (s.timeField / 5) * 5
			off := s.timeField % 5
			s.timeField = base + (off-1+5)%5
		case "right", "l":
			base := (s.timeField / 5) * 5
			off := s.timeField % 5
			s.timeField = base + (off+1)%5
		case "tab":
			if s.editHasEnd {
				if s.timeField < 5 {
					s.timeField += 5
				} else {
					s.timeField -= 5
				}
			}
		case "up":
			s.adjustTimeField(s.timeField, 1)
		case "down":
			s.adjustTimeField(s.timeField, -1)
		case "shift+up":
			s.adjustTimeField(s.timeField, s.shiftDelta())
		case "shift+down":
			s.adjustTimeField(s.timeField, -s.shiftDelta())
		case "enter":
			err := s.store.UpdateTimeEntry(s.editTimeID, s.editStart, s.editEnd)
			s.lastErr = err
			if err == nil {
				if kind, id := s.selectedKindID(); kind == kindSubtask {
					s.entries, _ = s.store.TimeEntriesBySubtask(id)
				}
				s.rebuildTimePick()
				s.timePick.sel = 0
				for i, it := range s.timePick.items {
					if it.value == s.editTimeID {
						s.timePick.sel = i
						break
					}
				}
				s.timePick.clampScroll()
				s.loadInfo()
				s.esc.Reset()
				s.mode = taskTimeList
			}
		case "esc":
			if !s.esc.Press(s.timeEditDirty()) {
				return m, nil, true
			}
			s.lastErr = nil
			s.mode = taskTimeList
		}
		return m, nil, true
	case taskTimeDelete:
		switch msg.String() {
		case "y", "enter":
			if err := s.store.DeleteTimeEntry(s.confirmTimeID); err != nil {
				s.lastErr = err
			} else {
				s.lastErr = nil
				if kind, id := s.selectedKindID(); kind == kindSubtask {
					s.entries, _ = s.store.TimeEntriesBySubtask(id)
				}
				s.rebuildTimePick()
			}
			s.mode = taskTimeList
		case "n", "esc":
			s.mode = taskTimeList
		}
		return m, nil, true
	}
	return m, nil, false
}

// timeEditDirty — изменились ли поля записи времени.
func (s *tasksScreen) timeEditDirty() bool {
	if s.editStart != s.editStartOrig || s.editHasEnd != s.editHasEndOrig {
		return true
	}
	if !s.editHasEnd {
		return false
	}
	if s.editEnd == nil || s.editEndOrig == nil {
		return s.editEnd != nil || s.editEndOrig != nil
	}
	return !s.editEnd.Equal(*s.editEndOrig)
}
