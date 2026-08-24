package ui

import (
	"github.com/charmbracelet/bubbletea"
)

// startNewTask открывает ввод названия новой задачи (гвард canCreate).
func (s *tasksScreen) startNewTask() {
	if s.canCreate(kindTask) {
		s.inputKind = kindTask
		s.lastErr = nil
		s.mode = taskInput
		s.input.Focus()
	}
}

// startNewSubtask открывает ввод названия новой подзадачи (гвард canCreate).
func (s *tasksScreen) startNewSubtask() {
	if s.canCreate(kindSubtask) {
		s.inputKind = kindSubtask
		s.lastErr = nil
		s.mode = taskInput
		s.input.Focus()
	}
}

// startDelete открывает подтверждение удаления выбранного элемента.
func (s *tasksScreen) startDelete() {
	if !s.canDelete() {
		return
	}
	s.confirmKind = s.selectedKind()
	switch s.confirmKind {
	case kindTask:
		if item, ok := s.list.SelectedItem().(taskItem); ok {
			s.confirmID = item.t.ID
		}
	case kindSubtask:
		if item, ok := s.list.SelectedItem().(subItem); ok {
			s.confirmID = item.st.ID
		}
	}
	s.mode = taskConfirm
}

// startEditTitle открывает модалку изменения названия выбранного элемента:
// textinput префиллен текущим названием, Enter сохраняет, Esc отменяет.
func (s *tasksScreen) startEditTitle() {
	if !s.canDelete() {
		return
	}
	s.inputKind = s.selectedKind()
	switch s.inputKind {
	case kindTask:
		if item, ok := s.list.SelectedItem().(taskItem); ok {
			s.editID = item.t.ID
			s.input.SetValue(item.t.Title)
		}
	case kindSubtask:
		if item, ok := s.list.SelectedItem().(subItem); ok {
			s.editID = item.st.ID
			s.input.SetValue(item.st.Title)
		}
	}
	if s.editID == 0 {
		return
	}
	s.lastErr = nil
	s.input.CursorEnd()
	s.mode = taskTitleEdit
	s.input.Focus()
}

// startEditDescription открывает инлайн-редактирование описания выбранной
// задачи/подзадачи (Enter в списке — то же, что раньше «e» в колонке описания).
func (s *tasksScreen) startEditDescription() {
	s.lastErr = nil
	s.descText.SetValue(s.desc)
	s.mode = taskDescEdit
	s.descText.Focus()
}

// startLinkEdit открывает модалку ссылки выбранного элемента: id=0 — новая,
// иначе — правка существующей (поля префилливаются из списка ссылок).
func (s *tasksScreen) startLinkEdit(id int64) {
	s.lastErr = nil
	s.editLinkID = id
	s.linkName.SetValue("")
	s.linkInput.SetValue("")
	for _, l := range s.links {
		if l.ID == id {
			s.linkName.SetValue(l.Name)
			s.linkInput.SetValue(l.URL)
		}
	}
	s.mode = taskLinkEdit
	s.linkName.Focus()
	s.linkInput.Blur()
}

// openLinks открывает список ссылок выбранного элемента.
func (s *tasksScreen) openLinks() {
	s.lastErr = nil
	s.mode = taskLinks
}

// startSearch открывает модалку поиска по проекту.
func (s *tasksScreen) startSearch() {
	s.lastErr = nil
	s.searchInput.SetValue(s.searchQuery)
	s.searchInput.Focus()
	s.mode = taskSearch
}

// updateTasks — диспетчер экран задач: сначала маршрутизация модальных
// режимов (updateTasksModal), затем базовая навигация (updateTasksBase).
func (m *model) updateTasks(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if mm, cmd, handled := m.updateTasksModal(msg); handled {
		return mm, cmd
	}
	return m.updateTasksBase(msg)
}
