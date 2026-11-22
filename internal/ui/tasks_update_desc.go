package ui

import (
	"github.com/charmbracelet/bubbletea"
)

// updateTasksDesc обрабатывает модалку описания задачи/подзадачи.
func (s *tasksScreen) updateTasksDesc(m *model, msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	switch s.dmState {
	case dmView, dmSelect:
		switch msg.String() {
		case "up", "k":
			if s.dmState == dmView {
				s.descViewer.scrollUp(1)
			} else {
				s.descViewer.moveUp()
			}
		case "down", "j":
			if s.dmState == dmView {
				s.descViewer.scrollDown(1)
			} else {
				s.descViewer.moveDown()
			}
		case "pgup":
			s.descViewer.scrollUp(s.descViewer.height)
		case "pgdown":
			s.descViewer.scrollDown(s.descViewer.height)
		case "left", "h":
			if s.dmState == dmSelect {
				s.descViewer.moveLeft()
			}
		case "right", "l":
			if s.dmState == dmSelect {
				s.descViewer.moveRight()
			}
		case "e":
			s.descText.SetValue(s.descWork)
			s.descText.Focus()
			s.dmState = dmEdit
		case "E", "shift+e":
			path, cmd, err := openInEditor(s.descWork)
			if err != nil {
				s.notice = "Не удалось открыть редактор: " + err.Error()
				return m, nil, true
			}
			s.extEditPath = path
			s.extEditMode = 2
			return m, cmd, true
		case "alt+enter":
			path, cmd, err := openInEditor(s.descWork)
			if err != nil {
				s.notice = "Не удалось открыть редактор: " + err.Error()
				return m, nil, true
			}
			s.extEditPath = path
			s.extEditMode = 2
			return m, cmd, true
		case "v":
			if s.dmState == dmView {
				lines := s.descViewer.layout()
				top := s.descViewer.scroll
				if top < 0 || top >= len(lines) {
					top = 0
				}
				s.descViewer.plain = false
				s.descViewer.cursor = lines[top].start
				s.descViewer.anchor = -1
				s.dmState = dmSelect
			} else {
				s.descViewer.plain = true
				s.descViewer.anchor = -1
				s.descViewer.scroll = s.descViewer.lineOfCursor(s.descViewer.layout())
				s.dmState = dmView
			}
		case " ":
			if s.dmState == dmSelect {
				s.descViewer.anchor = s.descViewer.cursor
			}
		case "enter":
			if s.dmState == dmSelect {
				s.copyDescSelection()
			}
		case "y":
			s.copyDescSelection()
		case "d":
			if s.dmState == dmSelect && s.descViewer.anchor >= 0 {
				s.deleteDescSelection()
				s.dmState = dmView
			}
		case "ctrl+s":
			s.saveDescWork()
			s.dmState = dmView
			s.refreshDescViewer()
		case "esc":
			dirty := s.descWork != s.desc
			if !s.esc.Press(dirty) {
				return m, nil, true
			}
			if dirty {
				s.descWork = s.desc
				s.refreshDescViewer()
				return m, nil, true
			}
			s.mode = taskBrowse
		}
		return m, nil, true
	case dmEdit:
		var cmd tea.Cmd
		s.descText, cmd = s.descText.Update(msg)
		switch msg.String() {
		case "ctrl+s":
			s.descWork = s.descText.Value()
			s.saveDescWork()
			s.descText.Blur()
			s.esc.Reset()
			s.dmState = dmView
			s.refreshDescViewer()
		case "esc":
			if !s.esc.Press(s.descText.Value() != s.descWork) {
				return m, cmd, true
			}
			s.descText.Blur()
			s.dmState = dmView
		}
		return m, cmd, true
	}
	return m, nil, false
}
