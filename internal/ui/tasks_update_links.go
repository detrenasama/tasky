package ui

import (
	"strings"

	"github.com/charmbracelet/bubbletea"
)

// updateTasksLinks обрабатывает модалки ссылок задачи/подзадачи.
func (s *tasksScreen) updateTasksLinks(m *model, msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	switch s.mode {
	case taskLinkEdit:
		var cmd tea.Cmd
		if s.linkName.Focused() {
			s.linkName, cmd = s.linkName.Update(msg)
		} else {
			s.linkInput, cmd = s.linkInput.Update(msg)
		}
		switch msg.String() {
		case "tab":
			if s.linkName.Focused() {
				s.linkName.Blur()
				s.linkInput.Focus()
			} else {
				s.linkInput.Blur()
				s.linkName.Focus()
			}
		case "enter":
			if s.linkInput.Focused() {
				url := strings.TrimSpace(s.linkInput.Value())
				name := strings.TrimSpace(s.linkName.Value())
				if url != "" {
					var err error
					if s.editLinkID != 0 {
						kind, _ := s.selectedKindID()
						if kind == kindTask {
							err = s.store.UpdateTaskLink(s.editLinkID, name, url)
						} else {
							err = s.store.UpdateSubtaskLink(s.editLinkID, name, url)
						}
					} else {
						kind, id := s.selectedKindID()
						if kind == kindTask {
							_, err = s.store.CreateTaskLink(id, name, url)
						} else {
							_, err = s.store.CreateSubtaskLink(id, name, url)
						}
					}
					s.lastErr = err
					if err == nil {
						s.editLinkID = 0
						s.loadDesc()
					}
				}
				s.linkName.SetValue("")
				s.linkName.Blur()
				s.linkInput.SetValue("")
				s.linkInput.Blur()
				s.linkNameOrig, s.linkInputOrig = "", ""
				s.esc.Reset()
				if s.lastErr == nil {
					s.mode = taskLinks
				} else {
					s.mode = taskBrowse
				}
			} else {
				// Enter на названии — перейти к адресу
				s.linkName.Blur()
				s.linkInput.Focus()
			}
		case "esc":
			if !s.esc.Press(s.linkEditDirty()) {
				return m, cmd, true
			}
			s.editLinkID = 0
			s.linkName.SetValue("")
			s.linkName.Blur()
			s.linkInput.SetValue("")
			s.linkInput.Blur()
			s.linkNameOrig, s.linkInputOrig = "", ""
			s.mode = taskLinks
		}
		return m, cmd, true
	case taskLinks:
		var cmd tea.Cmd
		s.linkList, cmd = s.linkList.Update(msg)
		switch msg.String() {
		case "enter":
			if item, ok := s.linkList.SelectedItem().(linkItem); ok {
				if err := openURL(item.l.URL); err != nil {
					s.lastErr = err
				}
			}
			return m, nil, true
		case "e":
			if item, ok := s.linkList.SelectedItem().(linkItem); ok {
				s.startLinkEdit(item.l.ID)
			}
			return m, nil, true
		case "n":
			s.startLinkEdit(0)
			return m, nil, true
		case "d":
			if item, ok := s.linkList.SelectedItem().(linkItem); ok {
				s.confirmLinkID = item.l.ID
				s.mode = taskLinkConfirm
			}
			return m, nil, true
		case "esc":
			s.mode = taskBrowse
			s.lastErr = nil
		}
		return m, cmd, true
	case taskLinkConfirm:
		switch msg.String() {
		case "y", "enter":
			kind, _ := s.selectedKindID()
			if kind == kindTask {
				s.store.DeleteTaskLink(s.confirmLinkID)
			} else {
				s.store.DeleteSubtaskLink(s.confirmLinkID)
			}
			s.mode = taskLinks
			s.loadDesc()
		case "n", "esc":
			s.mode = taskLinks
		}
		return m, nil, true
	}
	return m, nil, false
}

// linkEditDirty — изменились ли поля модалки ссылки.
func (s *tasksScreen) linkEditDirty() bool {
	return strings.TrimSpace(s.linkName.Value()) != strings.TrimSpace(s.linkNameOrig) ||
		strings.TrimSpace(s.linkInput.Value()) != strings.TrimSpace(s.linkInputOrig)
}
