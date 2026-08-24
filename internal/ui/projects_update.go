package ui

import (
	"strings"

	"github.com/charmbracelet/bubbletea"
)

func (m *model) updateProjects(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	s := m.proj
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
	case projDescEdit:
		var cmd tea.Cmd
		s.descText, cmd = s.descText.Update(msg)
		switch msg.String() {
		case "ctrl+s":
			s.store.UpdateProjectDescription(s.selectedProjectID(), s.descText.Value())
			s.descText.Blur()
			s.mode = projBrowse
			s.loadDesc()
		case "esc":
			s.descText.Blur()
			s.mode = projBrowse
		}
		return m, cmd
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
	}

	var cmd tea.Cmd
	beforeID := s.selectedProjectID()
	s.list, cmd = s.list.Update(msg)
	if s.selectedProjectID() != beforeID {
		s.loadDesc()
		s.descV.GotoTop()
	}
	return m, cmd
}
