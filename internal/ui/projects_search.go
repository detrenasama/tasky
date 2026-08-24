package ui

import (
	"strings"
)

func (s *projectsScreen) buildItems() {
	selID := s.selectedProjectID()
	s.items = nil
	if q := strings.ToLower(strings.TrimSpace(s.searchQuery)); q != "" {
		for _, p := range s.projects {
			if strings.Contains(strings.ToLower(p.Name), q) ||
				strings.Contains(strings.ToLower(p.Desc), q) ||
				strings.Contains(strings.ToLower(s.linkTexts[p.ID]), q) {
				s.items = append(s.items, projectItem{p})
			}
		}
	} else {
		for _, p := range s.projects {
			s.items = append(s.items, projectItem{p})
		}
	}
	s.list.SetItems(s.items)
	if len(s.items) > 0 {
		idx := -1
		for i, item := range s.items {
			if it, ok := item.(projectItem); ok && it.p.ID == selID {
				idx = i
				break
			}
		}
		if idx < 0 {
			idx = 0
		}
		s.list.Select(idx)
	}
}

// loadDesc подгружает описание и ссылки выбранного проекта и пересобирает
// контент колонки описания.
func (s *projectsScreen) clearSearch() {
	s.searchQuery = ""
	s.buildItems()
	s.loadDesc()
}
