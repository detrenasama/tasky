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
	s.sizeList()
	s.syncScroll()
}

// sizeList «разворачивает» bubbles/list на все элементы, чтобы он не прыгал
// страницами, и обновляет верхнюю видимую строку.
func (s *projectsScreen) sizeList() {
	sizeListHeight(&s.list, s.listDelegate, len(s.items), s.listH)
}

// syncScroll удерживает курсор в отступе listScrollOff от краёв окна.
func (s *projectsScreen) syncScroll() {
	syncListTop(&s.list, &s.listTop, s.listDelegate, len(s.items), s.listH)
}

// listView возвращает только видимое окно списка (плавная прокрутка).
func (s *projectsScreen) listView() string {
	if len(s.items) == 0 {
		return strings.Repeat("\n", max(s.listH-1, 0))
	}
	return clipList(s.list, s.listTop, s.listH)
}

// loadDesc подгружает описание и ссылки выбранного проекта и пересобирает
// контент колонки описания.
func (s *projectsScreen) clearSearch() {
	s.searchQuery = ""
	s.buildItems()
	s.loadDesc()
}
