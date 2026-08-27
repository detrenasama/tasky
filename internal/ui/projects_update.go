package ui

import (
	"strings"

	"github.com/charmbracelet/bubbletea"
)

func (m *model) updateProjects(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	s := m.proj
	s.notice = ""
	switch s.mode {
	case projInput:
		var cmd tea.Cmd
		s.input, cmd = s.input.Update(msg)
		switch msg.String() {
		case "enter":
			name := strings.TrimSpace(s.input.Value())
			if name != "" {
				_, err := s.store.CreateProject(name)
				s.lastErr = err
				if err == nil {
					s.load()
				}
			}
			s.input.SetValue("")
			s.input.Blur()
			s.mode = projBrowse
		case "esc":
			s.input.SetValue("")
			s.input.Blur()
			s.mode = projBrowse
		}
		return m, cmd
	case projConfirm:
		switch msg.String() {
		case "y", "enter":
			s.store.DeleteProject(s.confirmID)
			s.mode = projBrowse
			s.load()
		case "n", "esc":
			s.mode = projBrowse
		}
		return m, nil
	case projDescModal:
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
					return m, nil
				}
				s.extEditPath = path
				s.extEditMode = 2
				return m, cmd
			case "alt+enter":
				path, cmd, err := openInEditor(s.descWork)
				if err != nil {
					s.notice = "Не удалось открыть редактор: " + err.Error()
					return m, nil
				}
				s.extEditPath = path
				s.extEditMode = 2
				return m, cmd
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
				s.descWork = s.descText.Value()
				s.saveDescWork()
				s.dmState = dmView
				s.refreshDescViewer()
			case "esc":
				if s.descWork != s.desc {
					s.dmPrev = s.dmState
					s.dmState = dmDiscard
				} else {
					s.mode = projBrowse
				}
			}
			return m, nil
		case dmEdit:
			var cmd tea.Cmd
			s.descText, cmd = s.descText.Update(msg)
			switch msg.String() {
			case "ctrl+s":
				s.descWork = s.descText.Value()
				s.saveDescWork()
				s.descText.Blur()
				s.dmState = dmView
				s.refreshDescViewer()
			case "esc":
				if s.descText.Value() != s.descWork {
					s.dmPrev = dmEdit
					s.dmState = dmDiscard
				} else {
					s.descText.Blur()
					s.dmState = dmView
				}
			}
			return m, cmd
		case dmDiscard:
			switch msg.String() {
			case "y":
				s.descWork = s.desc
				s.refreshDescViewer()
				s.dmState = dmView
			case "n", "esc":
				s.dmState = s.dmPrev
			}
			return m, nil
		}
	case projLinkEdit:
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
						err = s.store.UpdateProjectLink(s.editLinkID, name, url)
					} else {
						_, err = s.store.CreateProjectLink(s.selectedProjectID(), name, url)
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
				if s.lastErr == nil {
					s.mode = projLinks
				} else {
					s.mode = projBrowse
				}
			} else {
				// Enter на названии — перейти к адресу
				s.linkName.Blur()
				s.linkInput.Focus()
			}
		case "esc":
			s.editLinkID = 0
			s.linkName.SetValue("")
			s.linkName.Blur()
			s.linkInput.SetValue("")
			s.linkInput.Blur()
			s.mode = projLinks
		}
		return m, cmd
	case projLinks:
		var cmd tea.Cmd
		s.linkList, cmd = s.linkList.Update(msg)
		switch msg.String() {
		case "enter":
			if item, ok := s.linkList.SelectedItem().(linkItem); ok {
				if err := openURL(item.l.URL); err != nil {
					s.lastErr = err
				}
			}
			return m, nil
		case "e":
			if item, ok := s.linkList.SelectedItem().(linkItem); ok {
				s.startLinkEdit(item.l.ID)
			}
			return m, nil
		case "n":
			s.startLinkEdit(0)
			return m, nil
		case "d":
			if item, ok := s.linkList.SelectedItem().(linkItem); ok {
				s.confirmLinkID = item.l.ID
				s.mode = projLinkConfirm
			}
			return m, nil
		case "esc":
			s.mode = projBrowse
			s.lastErr = nil
		}
		return m, cmd
	case projLinkConfirm:
		switch msg.String() {
		case "y", "enter":
			s.store.DeleteProjectLink(s.confirmLinkID)
			s.mode = projLinks
			s.loadDesc()
		case "n", "esc":
			s.mode = projLinks
		}
		return m, nil
	case projSearch:
		var cmd tea.Cmd
		s.searchInput, cmd = s.searchInput.Update(msg)
		switch msg.String() {
		case "enter":
			s.searchQuery = strings.TrimSpace(s.searchInput.Value())
			s.searchInput.Blur()
			s.mode = projBrowse
			s.buildItems()
			s.loadDesc()
		case "esc":
			s.searchQuery = ""
			s.searchInput.Blur()
			s.mode = projBrowse
			s.buildItems()
			s.loadDesc()
		default:
			// живой фильтр по мере ввода
			s.searchQuery = strings.TrimSpace(s.searchInput.Value())
			s.buildItems()
			s.loadDesc()
		}
		return m, cmd
	}

	switch msg.String() {
	case "tab":
		// переключение фокуса между панелями упразднено
		return m, nil
	case "enter":
		s.startEditDescription()
		return m, nil
	case "pgup", "pgdown":
		s.descV, _ = s.descV.Update(msg)
		return m, nil
	case "left", "right":
		return m, nil
	case "e":
		// описание теперь открывается по Enter; «e» оставляем без действия
		return m, nil
	case "o":
		s.openLinks()
		return m, nil
	case "/":
		s.startSearch()
		return m, nil
	case "n":
		s.startNew()
		return m, nil
	case "d":
		s.startDelete()
		return m, nil
	case "y":
		if s.focus == projFocusDesc {
			if err := copyToClipboard(s.desc); err != nil {
				s.notice = "Не удалось скопировать: " + err.Error()
			} else {
				s.notice = "Описание скопировано в буфер обмена"
			}
			return m, nil
		}
	case "v":
		if s.focus == projFocusDesc {
			s.openDescModal(true)
			return m, nil
		}
	case "alt+enter":
		if s.focus == projFocusDesc {
			path, cmd, err := openInEditor(s.desc)
			if err != nil {
				s.notice = "Не удалось открыть редактор: " + err.Error()
				return m, nil
			}
			s.extEditPath = path
			s.extEditMode = 0
			return m, cmd
		}
	}

	var cmd tea.Cmd
	beforeID := s.selectedProjectID()
	s.list, cmd = s.list.Update(msg)
	s.syncScroll()
	if s.selectedProjectID() != beforeID {
		s.loadDesc()
		s.descV.GotoTop()
	}
	return m, cmd
}
