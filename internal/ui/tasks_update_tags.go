package ui

import (
	"strings"

	"github.com/charmbracelet/bubbletea"
)

// updateTasksTags обрабатывает модалки тегов задачи.
func (s *tasksScreen) updateTasksTags(m *model, msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	switch s.mode {
	case taskTags:
		switch msg.String() {
		case "up":
			s.tagPick.move(-1)
		case "down":
			s.tagPick.move(1)
		case "pgup":
			s.tagPick.move(-s.tagPick.visible)
		case "pgdown":
			s.tagPick.move(s.tagPick.visible)
		case "n":
			s.openTagEdit(0)
		case "enter":
			if it, ok := s.tagPick.selected(); ok {
				s.openTagEdit(it.value)
			}
		case "d":
			if it, ok := s.tagPick.selected(); ok {
				s.tagConfirmID = it.value
				s.mode = taskTagConfirm
			}
		case "esc":
			s.esc.Reset()
			s.lastErr = nil
			s.mode = taskBrowse
		}
		return m, nil, true
	case taskTagEdit:
		switch msg.String() {
		case "up":
			s.tagEditFocus = (s.tagEditFocus + 2) % 3
			s.focusTagEditField()
		case "down", "tab":
			s.tagEditFocus = (s.tagEditFocus + 1) % 3
			s.focusTagEditField()
		case "enter":
			switch s.tagEditFocus {
			case 0:
				s.tagTypePick.sel = 0
				for i, tt := range s.tagTypes {
					if tt.ID == s.tagEditType {
						s.tagTypePick.sel = i
					}
				}
				s.tagTypePick.clampScroll()
				s.mode = taskTagTypePick
			case 1:
				s.tagEditFocus = 2
				s.focusTagEditField()
			case 2:
				s.saveTagEdit()
			}
		case "ctrl+s":
			s.saveTagEdit()
		case "esc":
			if !s.esc.Press(s.tagEditDirty()) {
				return m, nil, true
			}
			s.lastErr = nil
			s.mode = taskTags
		default:
			switch s.tagEditFocus {
			case 1:
				s.tagEditText, _ = s.tagEditText.Update(msg)
			case 2:
				s.tagEditURL, _ = s.tagEditURL.Update(msg)
			}
		}
		return m, nil, true
	case taskTagTypePick:
		switch msg.String() {
		case "up":
			s.tagTypePick.move(-1)
		case "down":
			s.tagTypePick.move(1)
		case "pgup":
			s.tagTypePick.move(-s.tagTypePick.visible)
		case "pgdown":
			s.tagTypePick.move(s.tagTypePick.visible)
		case "enter":
			if it, ok := s.tagTypePick.selected(); ok {
				s.tagEditType = it.value
			}
			s.mode = taskTagEdit
		case "esc":
			s.mode = taskTagEdit
		}
		return m, nil, true
	case taskTagConfirm:
		switch msg.String() {
		case "y", "enter":
			s.store.DeleteTag(s.tagConfirmID)
			s.loadTags()
			s.loadData()
			s.mode = taskTags
		case "n", "esc":
			s.mode = taskTags
		}
		return m, nil, true
	}
	return m, nil, false
}

// tagEditDirty — изменились ли поля редактора тега.
func (s *tasksScreen) tagEditDirty() bool {
	return strings.TrimSpace(s.tagEditText.Value()) != strings.TrimSpace(s.tagTextOrig) ||
		strings.TrimSpace(s.tagEditURL.Value()) != strings.TrimSpace(s.tagURLOrig) ||
		s.tagEditType != s.tagTypeOrig
}
