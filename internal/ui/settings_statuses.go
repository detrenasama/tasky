package ui

import (
	"fmt"
	"strings"

	"github.com/detrenasama/tasky/internal/ui/theme"
)

// focusEditField переводит фокус textinput на текущее поле редактора
// статуса (имя или подсказка).
func (s *settingsScreen) focusEditField() {
	switch s.editFocus {
	case 0:
		s.editName.Focus()
		s.editNote.Blur()
	case 4:
		s.editName.Blur()
		s.editNote.Focus()
	default:
		s.editName.Blur()
		s.editNote.Blur()
	}
}

// openStatusEdit открывает редактор статуса: id=0 — новый, иначе — правка.
func (s *settingsScreen) openStatusEdit(id int64) {
	s.statusEditID = id
	s.lastErr = nil
	s.editName.SetValue("")
	s.editNote.SetValue("")
	s.editType, s.editColor, s.editQuick = 0, 0, false
	for _, st := range s.statuses {
		if st.ID != id {
			continue
		}
		s.editName.SetValue(st.Name)
		s.editNote.SetValue(st.NotePrompt)
		s.editQuick = st.IsQuick
		for i, t := range statusTypeCodes {
			if t == st.Type {
				s.editType = i
			}
		}
		for i, c := range theme.StatusPalette {
			if c == st.Color {
				s.editColor = i
			}
		}
	}
	s.editFocus = 0
	s.editName.Focus()
	s.editNote.Blur()
	s.mode = settingsStatusEdit
}

// saveStatusEdit сохраняет статус из редактора.
func (s *settingsScreen) saveStatusEdit() {
	name := strings.TrimSpace(s.editName.Value())
	if name == "" {
		s.lastErr = fmt.Errorf("имя не может быть пустым")
		return
	}
	var err error
	if s.statusEditID == 0 {
		_, err = s.store.CreateStatus(name, statusTypeCodes[s.editType],
			theme.StatusPalette[s.editColor], strings.TrimSpace(s.editNote.Value()), s.editQuick)
	} else {
		err = s.store.UpdateStatus(s.statusEditID, name, statusTypeCodes[s.editType],
			theme.StatusPalette[s.editColor], strings.TrimSpace(s.editNote.Value()), s.editQuick)
	}
	if err != nil {
		s.lastErr = err
		return
	}
	s.lastErr = nil
	s.load()
	s.mode = settingsStatusList
}
