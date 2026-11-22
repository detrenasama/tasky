package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbletea"
)

// updateTasksInput обрабатывает модалки ввода экрана задач: создание
// задачи/подзадачи, переименование, подтверждение удаления, выбор статуса,
// заметку к статусу и поиск.
func (s *tasksScreen) updateTasksInput(m *model, msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	switch s.mode {
	case taskInput:
		var cmd tea.Cmd
		s.input, cmd = s.input.Update(msg)
		switch msg.String() {
		case "enter":
			title := strings.TrimSpace(s.input.Value())
			if title != "" {
				created, err := s.createItem(s.inputKind, title)
				s.lastErr = err
				if err == nil {
					if s.inputKind == kindSubtask {
						s.expanded[s.selectedTaskID()] = true
					}
					s.loadData()
					s.selectByKindID(s.inputKind, created)
					s.loadInfo()
					s.loadDesc()
					s.descV.GotoTop()
				}
			}
			s.input.SetValue("")
			s.inputOrig = ""
			s.input.Blur()
			s.esc.Reset()
			s.mode = taskBrowse
		case "esc":
			if !s.esc.Press(strings.TrimSpace(s.input.Value()) != strings.TrimSpace(s.inputOrig)) {
				return m, cmd, true
			}
			s.input.SetValue("")
			s.inputOrig = ""
			s.input.Blur()
			s.mode = taskBrowse
		}
		return m, cmd, true
	case taskTitleEdit:
		var cmd tea.Cmd
		s.input, cmd = s.input.Update(msg)
		switch msg.String() {
		case "enter":
			title := strings.TrimSpace(s.input.Value())
			if title == "" {
				s.lastErr = fmt.Errorf("название не может быть пустым")
				return m, cmd, true
			}
			var err error
			if s.inputKind == kindTask {
				err = s.store.UpdateTaskTitle(s.editID, title)
			} else {
				err = s.store.UpdateSubtaskTitle(s.editID, title)
			}
			s.lastErr = err
			if err == nil {
				s.loadData()
				s.selectByKindID(s.inputKind, s.editID)
				s.loadInfo()
				s.loadDesc()
				s.descV.GotoTop()
			}
			s.input.SetValue("")
			s.inputOrig = ""
			s.input.Blur()
			s.esc.Reset()
			s.mode = taskBrowse
		case "esc":
			if !s.esc.Press(strings.TrimSpace(s.input.Value()) != strings.TrimSpace(s.inputOrig)) {
				return m, cmd, true
			}
			s.input.SetValue("")
			s.inputOrig = ""
			s.input.Blur()
			s.mode = taskBrowse
		}
		return m, cmd, true
	case taskConfirm:
		switch msg.String() {
		case "y", "enter":
			s.deleteItem(s.confirmKind, s.confirmID)
			s.mode = taskBrowse
			s.loadData()
		case "n", "esc":
			s.mode = taskBrowse
		}
		return m, nil, true
	case taskStatusPick:
		switch msg.String() {
		case "up":
			s.statusPick.move(-1)
		case "down":
			s.statusPick.move(1)
		case "pgup":
			s.statusPick.move(-s.statusPick.visible)
		case "pgdown":
			s.statusPick.move(s.statusPick.visible)
		case "enter":
			if it, ok := s.statusPick.selected(); ok {
				for _, st := range s.statuses {
					if st.ID == it.value {
						s.applyStatus(s.statusKind, s.statusID, st)
						break
					}
				}
			}
		case "esc":
			s.mode = taskBrowse
		}
		return m, nil, true
	case taskStatusNote:
		var cmd tea.Cmd
		s.statusNote, cmd = s.statusNote.Update(msg)
		switch msg.String() {
		case "ctrl+s":
			note := strings.TrimSpace(s.statusNote.Value())
			if s.statusTarget != nil {
				if err := s.store.SetStatus(dbOwner(s.statusKind), s.statusID,
					s.statusTarget.Name, note, time.Now()); err == nil {
					s.now = time.Now()
					s.loadData()
				} else {
					s.lastErr = err
				}
			}
			s.statusNote.Blur()
			s.esc.Reset()
			s.mode = taskBrowse
		case "esc":
			if !s.esc.Press(strings.TrimSpace(s.statusNote.Value()) != strings.TrimSpace(s.statusNoteOrig)) {
				return m, cmd, true
			}
			s.statusNote.Blur()
			s.statusNoteOrig = ""
			s.mode = taskBrowse
		}
		return m, cmd, true
	case taskSearch:
		var cmd tea.Cmd
		s.searchInput, cmd = s.searchInput.Update(msg)
		switch msg.String() {
		case "enter":
			s.searchQuery = strings.TrimSpace(s.searchInput.Value())
			s.searchInput.Blur()
			s.mode = taskBrowse
			s.buildItems()
			s.loadInfo()
			s.loadDesc()
		case "esc":
			s.esc.Reset()
			s.searchQuery = ""
			s.searchInput.Blur()
			s.mode = taskBrowse
			s.buildItems()
			s.loadInfo()
			s.loadDesc()
		default:
			// живой фильтр по мере ввода
			s.searchQuery = strings.TrimSpace(s.searchInput.Value())
			s.buildItems()
			s.loadInfo()
			s.loadDesc()
		}
		return m, cmd, true
	}
	return m, nil, false
}
