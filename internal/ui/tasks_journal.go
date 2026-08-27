package ui

import (
	"fmt"
	"github.com/detrenasama/tasky/internal/db"
	"time"
)

// startJournal открывает модалку новой записи журнала выбранной подзадачи.
func (s *tasksScreen) startJournal() {
	kind, id := s.selectedKindID()
	if kind != kindSubtask || id == 0 {
		return
	}
	s.lastErr = nil
	s.journalText.SetValue("")
	s.journalOrig = ""
	s.journalEditID = 0
	s.mode = taskJournal
	s.journalText.Focus()
}

// editTodayJournal открывает модалку редактирования самой свежей записи
// журнала текущего дня выбранной подзадачи.
func (s *tasksScreen) editTodayJournal() {
	kind, id := s.selectedKindID()
	if kind != kindSubtask || id == 0 {
		return
	}
	var target *db.JournalEntry
	for i := range s.journal {
		e := &s.journal[i]
		if sameDay(e.CreatedAt, s.now) {
			target = e
		}
	}
	if target == nil {
		s.lastErr = fmt.Errorf("редактировать можно только записи текущего дня")
		return
	}
	s.lastErr = nil
	s.journalEditID = target.ID
	s.journalText.SetValue(target.Text)
	s.journalOrig = target.Text
	s.journalText.CursorEnd()
	s.mode = taskJournal
	s.journalText.Focus()
}

// sameDay проверяет, что два момента времени приходятся на один календарный
// день (локальное время).
func sameDay(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}
