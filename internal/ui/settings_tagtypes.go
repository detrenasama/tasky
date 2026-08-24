package ui

import (
	"fmt"
	"strings"

	"github.com/detrenasama/tasky/internal/ui/theme"
)

// focusTagTypeField переводит фокус textinput на поле имени типа тега.
func (s *settingsScreen) focusTagTypeField() {
	if s.editFocus == 0 {
		s.editName.Focus()
	} else {
		s.editName.Blur()
	}
}

// openTagTypeEdit открывает редактор типа тега: id=0 — новый, иначе — правка.
func (s *settingsScreen) openTagTypeEdit(id int64) {
	s.tagTypeEditID = id
	s.lastErr = nil
	s.editName.SetValue("")
	s.editKind, s.editColor = 0, 0
	for _, tt := range s.tagTypes {
		if tt.ID != id {
			continue
		}
		s.editName.SetValue(tt.Name)
		for i, k := range tagKindCodes {
			if k == tt.Kind {
				s.editKind = i
			}
		}
		for i, c := range theme.StatusPalette {
			if c == tt.Color {
				s.editColor = i
			}
		}
	}
	s.editFocus = 0
	s.editName.Focus()
	s.mode = settingsTagTypeEdit
}

// saveTagTypeEdit сохраняет тип тега из редактора.
func (s *settingsScreen) saveTagTypeEdit() {
	name := strings.TrimSpace(s.editName.Value())
	if name == "" {
		s.lastErr = fmt.Errorf("имя не может быть пустым")
		return
	}
	var err error
	if s.tagTypeEditID == 0 {
		_, err = s.store.CreateTagType(name, tagKindCodes[s.editKind],
			theme.StatusPalette[s.editColor])
	} else {
		err = s.store.UpdateTagType(s.tagTypeEditID, name, tagKindCodes[s.editKind],
			theme.StatusPalette[s.editColor])
	}
	if err != nil {
		s.lastErr = err
		return
	}
	s.lastErr = nil
	s.load()
	s.mode = settingsTagTypeList
}
