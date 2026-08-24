package ui

import (
	"fmt"
	"github.com/charmbracelet/lipgloss"
	"github.com/detrenasama/tasky/internal/db"
	"github.com/detrenasama/tasky/internal/ui/theme"
	"strings"
)

// tagChip — цветная метка тега `[текст]` (цвет от типа тега; при URL —
// подчёркивание, как у ссылки).
func tagChip(t db.Tag) string {
	st := lipgloss.NewStyle().Foreground(lipgloss.Color(t.Color))
	if t.URL != "" {
		st = st.Underline(true)
	}
	return st.Render("[" + t.Text + "]")
}

// tagsLine — метки тегов через пробел с ведущим пробелом (для списка и
// строк отчётов).
func tagsLine(tags []db.Tag) string {
	var parts []string
	for _, t := range tags {
		parts = append(parts, tagChip(t))
	}
	if len(parts) == 0 {
		return ""
	}
	return " " + strings.Join(parts, " ")
}

// tagsChips — метки тегов задачи через пробел (для строки списка).
func (s *tasksScreen) tagsChips(taskID int64) string {
	return tagsLine(s.tagsMap[taskID])
}

// tagTypeName возвращает имя и цвет типа тега по id (пустые, если не найден).
func (s *tasksScreen) tagTypeName(id int64) (string, string) {
	for _, tt := range s.tagTypes {
		if tt.ID == id {
			return tt.Name, tt.Color
		}
	}
	return "", ""
}

// openTags открывает модалку тегов выбранной задачи (для подзадачи —
// родительской).
func (s *tasksScreen) openTags() {
	taskID := s.selectedTaskID()
	if taskID == 0 {
		return
	}
	s.lastErr = nil
	s.tagTaskID = taskID
	s.loadTags()
	s.tagTypePick.sel = 0
	s.tagTypePick.clampScroll()
	s.mode = taskTags
}

// loadTags перечитывает теги текущей задачи и собирает список модалки.
func (s *tasksScreen) loadTags() {
	s.tags, _ = s.store.TaskTags(s.tagTaskID)
	items := make([]pickItem, 0, len(s.tags))
	for _, t := range s.tags {
		label := tagChip(t)
		if t.URL != "" {
			label += " " + theme.Faint(t.URL)
		}
		items = append(items, pickItem{value: t.ID, label: label})
	}
	s.tagPick.items = items
	s.tagPick.clampScroll()
}

// openTagEdit открывает редактор тега: id=0 — новый, иначе — правка.
func (s *tasksScreen) openTagEdit(id int64) {
	s.tagEditID = id
	s.lastErr = nil
	s.tagEditText.SetValue("")
	s.tagEditURL.SetValue("")
	s.tagEditType = 0
	if len(s.tagTypes) > 0 {
		s.tagEditType = s.tagTypes[0].ID
	}
	for _, t := range s.tags {
		if t.ID != id {
			continue
		}
		s.tagEditText.SetValue(t.Text)
		s.tagEditURL.SetValue(t.URL)
		s.tagEditType = t.TypeID
	}
	s.tagEditFocus = 0
	s.tagEditText.Blur()
	s.tagEditURL.Blur()
	s.mode = taskTagEdit
}

// focusTagEditField переводит фокус textinput на поле редактора тега.
func (s *tasksScreen) focusTagEditField() {
	switch s.tagEditFocus {
	case 1:
		s.tagEditText.Focus()
		s.tagEditURL.Blur()
	case 2:
		s.tagEditText.Blur()
		s.tagEditURL.Focus()
	default:
		s.tagEditText.Blur()
		s.tagEditURL.Blur()
	}
}

// saveTagEdit сохраняет тег из редактора и возвращается к списку тегов.
func (s *tasksScreen) saveTagEdit() {
	text := strings.TrimSpace(s.tagEditText.Value())
	if text == "" {
		s.lastErr = fmt.Errorf("значение тега не может быть пустым")
		return
	}
	var err error
	if s.tagEditID == 0 {
		_, err = s.store.CreateTag(s.tagTaskID, s.tagEditType, text,
			strings.TrimSpace(s.tagEditURL.Value()))
	} else {
		err = s.store.UpdateTag(s.tagEditID, s.tagEditType, text,
			strings.TrimSpace(s.tagEditURL.Value()))
	}
	if err != nil {
		s.lastErr = err
		return
	}
	s.lastErr = nil
	s.loadTags()
	s.loadData()
	s.mode = taskTags
}
